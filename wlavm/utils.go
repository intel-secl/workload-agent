package wlavm

import (
	"errors"
	"fmt"
	"intel/isecl/lib/vml"
	"os"
	"os/exec"
	"strings"
)

//IsImageEncrypted method is used to check if the image is encryped and returns a boolean value.
func IsImageEncrypted(filePath string) (bool, error) {
	//check if image is encrypted
	// this has to be changed
	fileCmdOutput, err := exec.Command("file", filePath).Output()
	if err != nil {
		fmt.Println("Error while checking if the image is encrypted")
		return false, errors.New("error while checking if the image is encrypted")
	}

	outputFormat := strings.Split(string(fileCmdOutput), ":")
	imageFormat := outputFormat[len(outputFormat)-1]

	if strings.TrimSpace(imageFormat) != "data" {
		fmt.Println("image is not encrypted")
		return false, nil
	} 
	fmt.Println("image is encrypted")
	return true, nil
}

// CheckMountPathExistsAndMountVolume method is used to check if te mount path exists, 
// if it does not exists, the method creates the mount path and mounts the device mapper.
func CheckMountPathExistsAndMountVolume(mountPath, deviceMapperPath string) error{	
	fmt.Println("Mounting the device mapper: ", deviceMapperPath)
	_, err := os.Stat(mountPath)
	if os.IsNotExist(err) {
		_, mkdirErr := exec.Command("mkdir", "-p", mountPath).Output()
		if mkdirErr != nil {
			fmt.Println("Error while creating the mount point for the image device mapper")
			return mkdirErr
		}
	}

	mountErr := vml.Mount(deviceMapperPath, mountPath)
	if mountErr != nil {
		if !strings.Contains(mountErr.Error(), "device is already mounted") {
			fmt.Println("Error while mounting the image device mapper")
			return mountErr
		}
	}
	return nil
}

// Cleanup method is used to cleanup the image and instance sparsefile, mount path directory and the dm-crypt volumes
func Cleanup() error{
	// var args []string
	// var err error
	// // clean up image volume and device mapper mount point
	// _, err := os.Stat(deviceMapperPath)
	// if !os.IsNotExist(err) {
	// 	if !
	// 	fmt.Println("Unmounting the dm-crypt volume and closing the volume: ", imageDeviceMapperLocation)
	// 	moutnErr := vml.Unmount(imageMountPath)
	// 	if mountErr != nil {
	// 		fmt.Println("Error while unmount the device mapper from: ", deviceMapperLocation)
	// 	}

	// 	args
	// 	return 1
	// }
	return nil
}

// Execute command is used to execute a linux command line command and return the output of the command with an error if it exists.
func ExecuteCommand(cmd string, args []string) (string, error) {
	out, err := exec.Command(cmd, args...).Output()
	return string(out), err
}
