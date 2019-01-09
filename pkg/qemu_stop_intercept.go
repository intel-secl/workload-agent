package pkg

import (
	"bufio"
	"intel/isecl/lib/vml"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/osutil"
	"io"
	"io/ioutil"

	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

// QemuStopIntercept is called from the libvirt hook. Everytime stop cycle is called
// in any of the VM lifecycle events, this method will be called.
// e.g. shutdown, reboot, stop etc.
//
// Input Parameter:
//
// 	vmUUID – Instace uuid or VM uuid
//  imageUUID - Image uuid
//  instancePath - Absolute path of the instance
//  imagePath - Absolute path of the image
func QemuStopIntercept(vmUUID string, imageUUID string,
	instancePath string, imagePath string) int {
	var deviceMapperPath = config.GetDevMapperDir()

	// check if instance exists at given path
	if _, err := os.Stat(instancePath); os.IsNotExist(err) {
		return 1
	}

	// check if the instance volume is encrypted
	var isInstanceVolume = isInstanceVolumeEncrypted(vmUUID)
	// if instance volume is encrypted, close the volume
	if isInstanceVolume {
		vml.DeleteVolume(vmUUID)
	}

	// check if image exists at given path
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		log.Info("The encrypted file does not exist.")
		return 1
	}

	// check if the image is salted and store the result in a variable
	var isImgEncrypted = isImageEncrypted(imagePath)
	if !isImgEncrypted {
		log.Info("The base image is not ecrypted. Exiting with success.")
		return 0
	}

	// check if this is the last instance associated with the image
	var isLastInstance = isLastInstanceAssociatedWithImage(imageUUID)
	// if instance volume is encrypted, close the volume
	if !isLastInstance {
		log.Info("Not deleting the image volume as this is not the last instance using the image. Exiting with success.")
		return 0
	}

	log.Info("Deleting the image volume as this is the last instance using the image.")
	var deviceMapper = deviceMapperPath + imageUUID
	// Unmount the image
	vml.Unmount(deviceMapper)

	// Close the image volume
	vml.DeleteVolume(imageUUID)
	return 0
}

func isInstanceVolumeEncrypted(vmUUID string) bool {
	var deviceMapperPath = config.GetDevMapperDir()

	// check the status of the device mapper
	log.Debug("Checking the status of the device mapper ...")
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

func isLastInstanceAssociatedWithImage(imageUUID string) bool {
	// Join the file name and config file path
	var fileName = config.GetNumberOfInstancesPerImageFileName()
	file, err := osutil.MakeFilePathFromEnvVariable(config.GetConfigDir(), fileName, true)
	if err != nil {
		log.Debug(err)
	}

	// Open the file
	// FORMAT OF THE FILE:
	// <image UUID> <instances running of that image>
	// eg: 6c55cf8fe339a52a798796d9ba0e765daharshitha	/var/lib/nova/instances/_base/6c55cf8fe339a52a798796d9ba0e765dac55aef7	count:2
	f, err := os.Open(file)
	if err != nil {
		log.Debugf("error opening file: %v\n", err)
	}

	// Read the file
	r := bufio.NewReader(f)
	for {
		// Read each line in loop until \n character occurs
		read_line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Debug(err)
		}

		// Split words of each line by space character into an array
		words := strings.Fields(read_line)
		// To check the if this is the last instance running of that image
		// check if the first part of the line matches given image uuid and
		// then check if there is only 1 instance running of that image (which is the current one)
		count := strings.Split(words[2], ":")
		if strings.Contains(words[0], imageUUID) && count[1] == "1" {
			return true
		}

		// TODO: Reduce the number of instance by 1 and if it is zero; delete that entry
	}
	return false
}

func isImageEncrypted(imagePath string) bool {
	log.Info("Checking if the image is encrypted...")
	content, err := ioutil.ReadFile(imagePath)
	if err != nil {
		log.Debug(err)
	}
	str := string(content)
	// check whether image content contains Salted in the text
	return strings.Contains(str, "Salted_")
}

func runCommand(cmd string, args []string) (string, error) {
	out, err := exec.Command(cmd, args...).Output()
	return string(out), err
}
