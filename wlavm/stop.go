// +build linux

package wlavm

import (
	"fmt"
	"intel/isecl/lib/common/exec"
	"intel/isecl/lib/vml"
	"intel/isecl/wlagent/consts"
	"intel/isecl/wlagent/filewatch"
	"intel/isecl/wlagent/libvirt"
	"os"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

var (
	isVmVolume bool
	isLastVm   bool
	imagePath  string
)

var (
	mtx sync.Mutex
)

// Stop is called from the libvirt hook. Everytime stop cycle is called
// in any of the VM lifecycle events, this method will be called.
// e.g. shutdown, reboot, stop etc.
// Input Parameters: domainXML content string
// Return : Returns a boolean value to the main method.
// true if the vm is launched sucessfully, else returns false.
func Stop(domainXMLContent string, filewatcher *filewatch.Watcher) bool {
	log.Info("Stop call intercepted")
	log.Info("Parsing domain XML to get image UUID, VM UUID and VM path")

	d, err := libvirt.NewDomainParser(domainXMLContent, libvirt.Stop)
	if err != nil {
		log.Error("Parsing error")
		return false
	}

	// check if vm exists at given path
	log.Infof("Checking if VM exists in %s", d.GetVMPath())
	if _, err := os.Stat(d.GetVMPath()); os.IsNotExist(err) {
		log.Error("VM does not exist")
		return false
	}

	// check if the vm volume is encrypted
	log.Info("Checking if a dm-crypt volume for the image is created")
	isVmVolume, err := isVmVolumeEncrypted(d.GetVMUUID())
	if err != nil {
		log.Error("Error while checking if a dm-crypt volume is created for the VM and is active")
		return false
	}
	// if vm volume is encrypted, close the volume
	if isVmVolume {
		var vmMountPath = consts.MountPath + d.GetVMUUID()
		// Unmount the image
		log.Info("A dm-crypt volume for the image is created, deleting the vm volume")
		vml.Unmount(vmMountPath)
		vml.DeleteVolume(d.GetVMUUID())
	}

	// check if this is the last vm associated with the image
	log.Info("Checking if this is the last vm using the image...")
	iAssoc := ImageVMAssociation{d.GetImageUUID(), ""}
	isLastVm, imagePath, err = iAssoc.Delete()
	if err != nil {
		log.Error(err)
		return false
	}
	// as the original image is deleted during the VM start process, there is no way
	// to check if original image is encrypted. Instead we check if sparse file of image
	// exists at given path, if it does that means the image was enrypted and volumes were created
	if _, err := os.Stat(imagePath + "_sparseFile"); os.IsNotExist(err) {
		log.Info("The base image is not ecrypted. Exiting with success.")
		return true
	}

	// check if this is the last vm associated with the image
	if !isLastVm {
		log.Infof("Not deleting the image volume as this is not the last vm using the image, VM %s stopped", d.GetVMUUID())
		return true
	}

	log.Info("Unmounting and deleting the image volume as this is the last vm using the image")

	mtx.Lock()
	defer mtx.Unlock()
	var imageMountPath = consts.MountPath + d.GetImageUUID()
	// Unmount the image
	vml.Unmount(imageMountPath)
	// Close the image volume
	vml.DeleteVolume(d.GetImageUUID())
	log.Infof("VM %s stopped", d.GetVMUUID())
	return true
}

func isVmVolumeEncrypted(vmUUID string) (bool, error) {
	// check the status of the device mapper
	log.Debug("Checking the status of the device mapper")
	deviceMapperLocation := consts.DevMapperDirPath + vmUUID
	args := []string{"status", deviceMapperLocation}
	cmdOutput, err := exec.ExecuteCommand("cryptsetup", args)

	if cmdOutput != "" && strings.Contains(cmdOutput, "inactive") {
		log.Debug("The device mapper is inactive")
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("error occured while executing cryptsetup status command: %s" + err.Error())
	}
	log.Debug("The device mapper is encrypted and active")
	return true, nil
}
