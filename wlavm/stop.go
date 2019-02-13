// +build linux

package wlavm

import (
	"intel/isecl/lib/common/exec"
	"intel/isecl/lib/vml"
	"intel/isecl/wlagent/consts"
	"intel/isecl/wlagent/filewatch"
	"intel/isecl/wlagent/util"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	xmlpath "gopkg.in/xmlpath.v2"
)

// Stop is called from the libvirt hook. Everytime stop cycle is called
// in any of the VM lifecycle events, this method will be called.
// e.g. shutdown, reboot, stop etc.
//
// Input Parameter:
//
// 	iunstanceUUID – Instace uuid or VM uuid
//  imageUUID - Image uuid
//  instancePath - Absolute path of the instance
func Stop(domainXMLContent string, filewatcher *filewatch.Watcher) int {
	log.Info("Stop call intercepted.")

	domainXML, err := xmlpath.Parse(strings.NewReader(domainXMLContent))
	if err != nil {
		log.Infof("Error while parsing domaXML: %s", err)
		return 1
	}

	// get instance UUID from domain XML
	instanceUUID, err := util.GetInstanceUUID(domainXML)
	if err != nil {
		log.Infof("%s", err)
		return 1
	}

	// get instance path from domain XML
	instancePath, err := util.GetInstancePath(domainXML)
	if err != nil {
		log.Infof("%s", err)
		return 1
	}

	// get image UUID from domain XML
	imageUUID, err := util.GetImageUUID(domainXML)
	if err != nil {
		log.Infof("%s", err)
		return 1
	}

	// check if instance exists at given path
	log.Info("Checking if instance eixsts at given instance path.")
	if _, err := os.Stat(instancePath); os.IsNotExist(err) {
		return 1
	}

	// check if the instance volume is encrypted
	log.Info("Checking if instance volume is encrypted.")
	var isInstanceVolume = isInstanceVolumeEncrypted(instanceUUID)
	// if instance volume is encrypted, close the volume
	if isInstanceVolume {
		var instanceMountPath = consts.MountPath + instanceUUID
		// Unmount the image
		log.Info("Instance volume is encrypted, deleting the instance volume.")
		vml.Unmount(instanceMountPath)
		vml.DeleteVolume(instanceUUID)
		err := os.RemoveAll(instanceMountPath)
		if err != nil {
			log.Error("Error while deleting the instance mount point")
			return 1
		}
	}

	// check if this is the last instance associated with the image
	log.Info("Checking if this is the last instance using the image.")
	iAssoc := ImageVMAssocociation{imageUUID, ""}
	var isLastInstance, imagePath = iAssoc.Delete()
	// as the original image is deleted during the VM start process, there is no way
	// to check if original image is encrypted. Instead we check if sparse file of image
	// exists at given path, if it does that means the image was enrypted and volumes were created
	if _, err := os.Stat(imagePath + "_sparseFile"); os.IsNotExist(err) {
		log.Info("The base image is not ecrypted. Exiting with success.")
		return 0
	}

	// check if this is the last instance associated with the image
	if !isLastInstance {
		log.Info("Not deleting the image volume as this is not the last instance using the image. Exiting with success.")
		return 0
	}

	log.Info("Unmounting and deleting the image volume as this is the last instance using the image.")
	var imageMountPath = consts.MountPath + imageUUID
	// Unmount the image
	vml.Unmount(imageMountPath)

	// Close the image volume
	vml.DeleteVolume(imageUUID)
	err = os.RemoveAll(imageMountPath)
	if err != nil {
		log.Info("Error while deleting the instance mount point")
		return 1
	}
	return 0
}

func isInstanceVolumeEncrypted(instanceUUID string) bool {
	// check the status of the device mapper
	log.Info("Checking the status of the device mapper.")
	deviceMapperLocation := consts.DevMapperDirPath + instanceUUID
	args := []string{"status", deviceMapperLocation}
	cmdOutput, err := exec.ExecuteCommand("cryptsetup", args)

	if cmdOutput != "" && strings.Contains(cmdOutput, "inactive") {
		log.Debug("The device mapper is inactive.")
		return false
	}

	if err != nil {
		log.Error(err)
	}
	log.Debug("The device mapper is encrypted and active.")
	return true
}
