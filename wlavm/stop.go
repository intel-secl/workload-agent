// +build linux

package wlavm

import (
	"fmt"
	"intel/isecl/lib/common/exec"
	"intel/isecl/lib/vml"
	"intel/isecl/wlagent/consts"
	"intel/isecl/wlagent/libvirt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"intel/isecl/lib/common/exec"
	xmlpath "gopkg.in/xmlpath.v2"
)

var (
	isVmVolume bool
	isLastVm   bool
	imagePath        string
)

// Stop is called from the libvirt hook. Everytime stop cycle is called
// in any of the VM lifecycle events, this method will be called.
// e.g. shutdown, reboot, stop etc.
// Input Parameters: domainXML content string
// Return : Returns a boolean value to the main method.
// true if the vm is launched sucessfully, else returns false. 
func Stop(domainXMLContent string) bool {

	log.Info("Stop call intercepted")
	log.Info("Parsing domain XML to get image UUID, VM UUId and VM path")
	domainXML, err := xmlpath.Parse(strings.NewReader(domainXMLContent))
	if err != nil {
		log.Errorf("Error while parsing domaXML: %s", err)
		return false
	}

	// get vm UUID from domain XML
	vmUUID, err := libvirt.GetVMUUID(domainXML)
	if err != nil {
		log.Errorf(err.Error())
		return false
	}

	// get vm path from domain XML
	vmPath, err := libvirt.GetVMPath(domainXML)
	if err != nil {
		log.Errorf(err.Error())
		return false
	}

	// get image UUID from domain XML
	imageUUID, err := libvirt.GetImageUUID(domainXML)
	if err != nil {
		log.Errorf(err.Error())
		return false
	}

	// check if vm exists at given path
	log.Infof("Checking if VM exists in %s", vmPath)
	if _, err := os.Stat(vmPath); os.IsNotExist(err) {
		log.Error("VM does not exist")
		return false
	}

	// check if the vm volume is encrypted
	log.Info("Checking if vm volume is encrypted.")
	isVmVolume, err := isVmVolumeEncrypted(vmUUID)
	if err != nil {
		log.Error("Error while checking if a dm-crypt volume is created for the VM and is active")
		return false
	}
	// if vm volume is encrypted, close the volume
	if isVmVolume {
		var vmMountPath = consts.MountPath + vmUUID
		// Unmount the image
		log.Info("vm volume is encrypted, deleting the vm volume.")
		vml.Unmount(vmMountPath)
		vml.DeleteVolume(vmUUID)
		err := os.RemoveAll(vmMountPath)
		if err != nil {
			log.Error("Error while deleting the vm mount point")
			return false
		}
	}

	// check if this is the last vm associated with the image
	log.Info("Checking if this is the last vm using the image...")
	iAssoc := ImageVMAssocociation{imageUUID, ""}
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
	if !isLastvm {
		log.Infof("VM % stopped", vmUUID)
		log.Info("Not deleting the image volume as this is not the last vm using the image. Exiting with success.")
		return true
	}

	log.Info("Unmounting and deleting the image volume as this is the last vm using the image")
	var imageMountPath = consts.MountPath + imageUUID
	// Unmount the image
	vml.Unmount(imageMountPath)

	// Close the image volume
	vml.DeleteVolume(imageUUID)
	err = os.RemoveAll(imageMountPath)
	if err != nil {
		log.Error("Error while deleting the vm mount point")
		return false
	}

	log.Infof("VM % stopped", vmUUID)
	return true
}

func isVmVolumeEncrypted(vmUUID string) (bool, error) {
	// check the status of the device mapper
	log.Debug("Checking the status of the device mapper.")
	deviceMapperLocation := consts.DevMapperDirPath + vmUUID
	args := []string{"status", deviceMapperLocation}
	cmdOutput, err := exec.ExecuteCommand("cryptsetup", args)

	if cmdOutput != "" && strings.Contains(cmdOutput, "inactive") {
		log.Debug("The device mapper is inactive.")
		return false, nil
	}

	if err != nil {
		log.Error(err.Error())
	}
	log.Debug("The device mapper is encrypted and active.")
	return true, nil
}

var fileMutex sync.Mutex

func isLastvmAssociatedWithImage(imageUUID string) (bool, string) {
	var imagePath = ""
	imagevmAssociationFile := consts.ConfigDirPath + consts.ImageInstanceCountAssociationFileName

	// Read from a file and store it in a string
	// FORMAT OF THE FILE:
	// <image UUID> <vms running of that image>
	// eg: 6c55cf8fe339a52a798796d9ba0e765daharshitha	/var/lib/nova/vms/_base/6c55cf8fe339a52a798796d9ba0e765dac55aef7	count:2
	log.Debug("Reading image vm association file.")
	str, err := ioutil.ReadFile(imagevmAssociationFile)
	if err != nil {
		log.Error(err.Error())
	}

	log.Info("Recursively checking if image uuid exists in file. If it does, reduce the vm count by 1.")
	lines := strings.Split(string(str), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			break
		}
		// Split words of each line by space character into an array
		words := strings.Fields(line)
		imagePath = words[1]
		// To check the if this is the last vm running of that image
		// check if the first part of the line matches given image uuid and
		// then check if there is only 1 vm running of that image (which is the current one)
		count := strings.Split(words[2], ":")
		// Reduce the number of vm by 1 and if it is zero; delete that entry
		if strings.Contains(words[0], imageUUID) {
			log.Debug("Image ID found in image vm association file. Reducing vm count by 1.")
			cnt, _ := strconv.Atoi(count[1])
			replaceLine := strings.Replace(string(line), "count:"+count[1], "count:"+strconv.Itoa(cnt-1), 1)
			lines[i] = replaceLine
		}
		if strings.Contains(words[0], imageUUID) && count[1] == "1" {
			log.Debugf("Deleting image entry %s as this was last vm to use the image.", imageUUID)
			lines[i] = lines[len(lines)-1]

			// After modifying contents, store it back to the file
			log.Debug("Outputting modified text back to file.")

			// Add mutex lock so that at one time only one process can write to a file
			fileMutex.Lock()
			// Release the mutext lock
			defer fileMutex.Unlock()
			outputToFile := strings.Join(lines[:len(lines)-1], "\n")
			err = ioutil.WriteFile(imagevmAssociationFile, []byte(outputToFile), 0644)
			if err != nil {
				return false, imagePath, fmt.Errorf("Error occured while writing to a file. %s" + err.Error())
			}
			return true, imagePath, nil
		}
	}
	// Add mutex lock so that at one time only one process can write to a file
	fileMutex.Lock()
	// Release the mutext lock
	defer fileMutex.Unlock()
	outputToFile := strings.Join(lines, "\n")
	err = ioutil.WriteFile(imagevmAssociationFile, []byte(outputToFile), 0644)
	if err != nil {
		return false, imagePath, fmt.Errorf("Error occured while writing to a file. %s" + err.Error())
	}
	return false, imagePath, nil
}

func CleanUp() {
	
}

func CleanUp() {
	
}
