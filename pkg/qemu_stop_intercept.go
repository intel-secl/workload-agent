package wagent

import (
	"bufio"
	"intel/isecl/lib/vml"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"workload-agent/config"
	"workload-agent/osutil"
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
func QemuStopIntercept(vmUUID string, imageUUID string,
	instancePath string, imagePath string) int {
	var deviceMapperPath = "/dev/mapper"

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
		log.Fatal("encrypted file does not exist")
		return 1
	}

	// check if the image is salted and store the result in a variable
	var isImgEncrypted = isImageEncrypted(imagePath)
	if !isImgEncrypted {
		log.Println("The base image is not ecrypted. Exiting with success.")
		return 0
	}

	// check if this is the last instance associated with the image
	var isLastInstance = isLastInstanceAssociatedWithImage(imageUUID)

	// if instance volume is encrypted, close the volume
	if !isLastInstance {
		return 0
	}

	var deviceMapper = deviceMapperPath + imageUUID

	// Unmount the image
	vml.Unmount(deviceMapper)

	// Close the image volume
	vml.DeleteVolume(imageUUID)

	return 0
}

func isInstanceVolumeEncrypted(vmUUID string) bool {
	var deviceMapperPath = "/dev/mapper"

	deviceMapperLocation := deviceMapperPath + vmUUID
	// check the status of the device mapper
	log.Println("Checking the status of the device mapper ...")
	args := []string{"status", deviceMapperLocation}
	cmdOutput, err := runCommand("cryptsetup", args)
	if err != nil {
		panic(err)
	}
	if strings.Contains(cmdOutput, "inactive") {
		return false
	}
	return true
}

func isLastInstanceAssociatedWithImage(imageUUID string) bool {

	var fileName = config.GetNumberOfInstancesPerImageFileName()
	file, err := osutil.MakeFilePathFromEnvVariable(config.GetConfigDir(), fileName, true)
	if err != nil {
		return err
	}
	f := bufio.NewReader(file)

	for {
		read_line, err := f.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}
		words := strings.Fields(read_line)
		if words[0] == imageUUID && words[1] == "1" {
			return true
		}
	}
	return false
}

func isImageEncrypted(imagePath string) bool {
	log.Println("Checking if the image is encrypted...")
	content, err := ioutil.ReadFile(imagePath)
	if err != nil {
		log.Fatal(err)
	}
	str := string(content)
	// check whether image content contains Salted in the text
	return strings.Contains(str, "Salted_")
}

func runCommand(cmd string, args []string) (string, error) {
	out, err := exec.Command(cmd, args...).Output()
	return string(out), err
}
