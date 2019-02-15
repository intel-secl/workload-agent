// +build linux

package wlavm

import (
	"fmt"
	"intel/isecl/lib/common/exec"
	"intel/isecl/lib/vml"
	"intel/isecl/wlagent/consts"
	"intel/isecl/wlagent/util"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	xmlpath "gopkg.in/xmlpath.v2"
)

var (
	isInstanceVolume bool
	isLastInstance   bool
	imagePath        string
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
	isInstanceVolume, err = isInstanceVolumeEncrypted(instanceUUID)
	if err != nil {
		log.Error(err)
		return 1
	}
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
	isLastInstance, imagePath, err = iAssoc.Delete()
	if err != nil {
		log.Error(err)
		return 1
	}
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
	// err = os.RemoveAll(imageMountPath)
	// if err != nil {
	// 	log.Info("Error while deleting the instance mount point")
	// 	return 1
	// }
	return 0
}

func isInstanceVolumeEncrypted(instanceUUID string) (bool, error) {
	// check the status of the device mapper
	log.Info("Checking the status of the device mapper.")
	deviceMapperLocation := consts.DevMapperDirPath + instanceUUID
	args := []string{"status", deviceMapperLocation}
	cmdOutput, err := exec.ExecuteCommand("cryptsetup", args)

	if cmdOutput != "" && strings.Contains(cmdOutput, "inactive") {
		log.Debug("The device mapper is inactive.")
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("Error occured while executing cryptsetup status command. %s" + err.Error())
	}
	log.Debug("The device mapper is encrypted and active.")
	return true, nil
}

var fileMutex sync.Mutex

func isLastInstanceAssociatedWithImage(imageUUID string) (bool, string, error) {
	var imagePath = ""
	imageInstanceAssociationFile := consts.ConfigDirPath + consts.ImageInstanceCountAssociationFileName

	// Read from a file and store it in a string
	// FORMAT OF THE FILE:
	// <image UUID> <instances running of that image>
	// eg: 6c55cf8fe339a52a798796d9ba0e765daharshitha	/var/lib/nova/instances/_base/6c55cf8fe339a52a798796d9ba0e765dac55aef7	count:2
	log.Info("Reading image instance association file.")
	str, err := ioutil.ReadFile(imageInstanceAssociationFile)
	if err != nil {
		return false, imagePath, fmt.Errorf("Error occured while reading a file. %s" + err.Error())
	}

	log.Info("Recursively checking if imageid exists in file. If it does, reduce the instance count by 1.")
	lines := strings.Split(string(str), "\n")
	for i, line := range lines {
		log.Debug("line: ", line)
		if strings.TrimSpace(line) == "" {
			break
		}
		// Split words of each line by space character into an array
		words := strings.Fields(line)
		imagePath = words[1]
		// To check the if this is the last instance running of that image
		// check if the first part of the line matches given image uuid and
		// then check if there is only 1 instance running of that image (which is the current one)
		count := strings.Split(words[2], ":")
		// Reduce the number of instance by 1 and if it is zero; delete that entry
		if strings.Contains(words[0], imageUUID) {
			log.Debug("Image ID found in image instance association file. Reducing instance count by 1.")
			cnt, _ := strconv.Atoi(count[1])
			replaceLine := strings.Replace(string(line), "count:"+count[1], "count:"+strconv.Itoa(cnt-1), 1)
			lines[i] = replaceLine
		}
		if strings.Contains(words[0], imageUUID) && count[1] == "1" {
			log.Debugf("Deleting image entry %s as this was last instance to use the image.", imageUUID)
			lines[i] = lines[len(lines)-1]

			// After modifying contents, store it back to the file
			log.Debug("Outputting modified text back to file.")

			// Add mutex lock so that at one time only one process can write to a file
			fileMutex.Lock()
			// Release the mutext lock
			defer fileMutex.Unlock()
			outputToFile := strings.Join(lines[:len(lines)-1], "\n")
			err = ioutil.WriteFile(imageInstanceAssociationFile, []byte(outputToFile), 0644)
			if err != nil {
				return false, imagePath, fmt.Errorf("Error occured while writing to a file. %s" + err.Error())
			}
			return true, imagePath, nil
		}
	}
	log.Debug("Image ID not found in image instance association file.")
	// Add mutex lock so that at one time only one process can write to a file
	fileMutex.Lock()
	// Release the mutext lock
	defer fileMutex.Unlock()
	outputToFile := strings.Join(lines, "\n")
	err = ioutil.WriteFile(imageInstanceAssociationFile, []byte(outputToFile), 0644)
	if err != nil {
		return false, imagePath, fmt.Errorf("Error occured while writing to a file. %s" + err.Error())
	}
	return false, imagePath, nil
}
