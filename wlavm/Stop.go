// +build linux

package wlavm

import (
	"intel/isecl/lib/common/exec"
	"intel/isecl/lib/vml"
	"intel/isecl/wlagent/consts"

	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Stop is called from the libvirt hook. Everytime stop cycle is called
// in any of the VM lifecycle events, this method will be called.
// e.g. shutdown, reboot, stop etc.
//
// Input Parameter:
//
// 	vmUUID – Instace uuid or VM uuid
//  imageUUID - Image uuid
//  instancePath - Absolute path of the instance
func Stop(vmUUID string, imageUUID string,
	instancePath string, filewatcher *filewatch.Watcher) int {
	log.Info("Stop call intercepted.")
	var mntLocation = consts.MountDirPath

	// check if instance exists at given path
	log.Info("Checking if instance eixsts at given instance path.")
	if _, err := os.Stat(instancePath); os.IsNotExist(err) {
		return 1
	}

	// check if the instance volume is encrypted
	log.Info("Checking if instance volume is encrypted.")
	var isInstanceVolume = isInstanceVolumeEncrypted(vmUUID)
	// if instance volume is encrypted, close the volume
	if isInstanceVolume {
		var instanceMntPath = mntLocation + vmUUID
		// Unmount the image
		log.Info("Instance volume is encrypted, deleting the instance volume.")
		vml.Unmount(instanceMntPath)
		vml.DeleteVolume(vmUUID)
	}

	// check if this is the last instance associated with the image
	log.Info("Checking if this is the last instance using the image.")
	var isLastInstance, imagePath = isLastInstanceAssociatedWithImage(imageUUID)
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
	var imageMntPath = mntLocation + imageUUID
	// Unmount the image
	vml.Unmount(imageMntPath)

	// Close the image volume
	vml.DeleteVolume(imageUUID)
	log.Info("Successfully stopped instance.")
	return 0
}

func isInstanceVolumeEncrypted(vmUUID string) bool {
	var deviceMapperPath = consts.DevMapperDirPath

	// check the status of the device mapper
	log.Info("Checking the status of the device mapper.")
	deviceMapperLocation := deviceMapperPath + vmUUID
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

func isLastInstanceAssociatedWithImage(imageUUID string) (bool, string) {
	var imagePath = ""
	err := UnmarshalImageInstanceAssociation()
	if err != nil {
		log.Error(err)
	}
	for i, item := range ImageInstanceAssociations {
		imagePath = item.ImagePath
		if strings.Contains(item.ImageID, imageUUID) {
			log.Debug("Image ID already exist in file, decreasing the count of instance by 1.")
			item.InstanceCount = item.InstanceCount - 1
			if item.InstanceCount == 0 {
				log.Debug("Instance count is 0, hence deleting the entry with image id ", imageUUID)
				ImageInstanceAssociations[i] = ImageInstanceAssociations[0]
				ImageInstanceAssociations = ImageInstanceAssociations[1:]
				err = MarshalImageInstanceAssociation()
				if err != nil {
					log.Error(err)
				}
				return true, imagePath
			}
		}
	}
	log.Debug("Image ID not found in the file.")
	err = MarshalImageInstanceAssociation()
	if err != nil {
		log.Error(err)
	}
	return false, imagePath
}
