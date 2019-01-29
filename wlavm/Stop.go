// +build linux

package wlavm

import (
	"intel/isecl/lib/vml"
	"intel/isecl/wlagent/config"
	"io/ioutil"
	"strconv"

	"os"
	"os/exec"
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
	instancePath string) int {
	log.Info("Stop call intercepted.")
	var mntLocation = config.GetMountDir()

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
	return 0
}

func isInstanceVolumeEncrypted(vmUUID string) bool {
	var deviceMapperPath = config.GetDevMapperDir()

	// check the status of the device mapper
	log.Info("Checking the status of the device mapper.")
	deviceMapperLocation := deviceMapperPath + vmUUID
	args := []string{"status", deviceMapperLocation}
	cmdOutput, err := runCommand("cryptsetup", args)

	if cmdOutput != "" && strings.Contains(cmdOutput, "inactive") {
		return false
	}

	if err != nil {
		log.Error(err)
	}
	return true
}

func isLastInstanceAssociatedWithImage(imageUUID string) (bool, string) {
	var imagePath = ""
	imageInstanceAssociationFile := "/etc/workloadagent/" + config.ImageInstanceCountAssociationFileName()

	// Read from a file and store it in a string
	// FORMAT OF THE FILE:
	// <image UUID> <instances running of that image>
	// eg: 6c55cf8fe339a52a798796d9ba0e765daharshitha	/var/lib/nova/instances/_base/6c55cf8fe339a52a798796d9ba0e765dac55aef7	count:2
	log.Info("Reading image instance association file.")
	str, err := ioutil.ReadFile(imageInstanceAssociationFile)
	if err != nil {
		log.Fatalln(err)
	}

	log.Info("Recursively checking if imageid exists in file. If it does, reduce the instance count by 1.")
	lines := strings.Split(string(str), "\n")
	for i, line := range lines {
		log.Info("line: ", line)
		// Split words of each line by space character into an array
		words := strings.Fields(line)
		imagePath = words[1]
		// To check the if this is the last instance running of that image
		// check if the first part of the line matches given image uuid and
		// then check if there is only 1 instance running of that image (which is the current one)
		count := strings.Split(words[2], ":")

		// Reduce the number of instance by 1 and if it is zero; delete that entry
		if strings.Contains(words[0], imageUUID) {
			cnt, _ := strconv.Atoi(count[1])
			replaceLine := strings.Replace(string(line), "count:"+count[1], "count:"+strconv.Itoa(cnt-1), 1)
			lines[i] = replaceLine
		}
		if strings.Contains(words[0], imageUUID) && count[1] == "1" {
			log.Infof("Deleting image entry %s as this was last instance to use the image.", imageUUID)
			lines[i] = lines[len(lines)-1]

			// After modifying contents, store it back to the file
			log.Info("Outputting modified text back to file.")
			outputToFile := strings.Join(lines[:len(lines)-1], "\n")
			err = ioutil.WriteFile(imageInstanceAssociationFile, []byte(outputToFile), 0644)
			if err != nil {
				log.Error(err)
			}
			return true, imagePath
		}
	}
	outputToFile := strings.Join(lines, "\n")
	err = ioutil.WriteFile(imageInstanceAssociationFile, []byte(outputToFile), 0644)
	if err != nil {
		log.Error(err)
	}
	return false, imagePath
}

func runCommand(cmd string, args []string) (string, error) {
	out, err := exec.Command(cmd, args...).Output()
	return string(out), err
}
