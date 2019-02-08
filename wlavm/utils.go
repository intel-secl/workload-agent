package wlavm

import (
	"errors"
	"fmt"
	"intel/isecl/lib/common/exec"
	"intel/isecl/lib/vml"
	"intel/isecl/wlagent/consts"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// ImageVMAssociations is variable that consists of array of ImageVMAssociation struct
var ImageVMAssociations []ImageVMAssociation

// ImageVMAssociation is the global struct that is used to store the image instance count to yaml file
type ImageVMAssociation struct {
	ImageID   string
	ImagePath string
	VMCount   int
}

// LoadImageVMAssociation method loads image instance association from yaml file
func LoadImageVMAssociation() error {
	imageVMAssociationFile := consts.ConfigDirPath + consts.ImageInstanceCountAssociationFileName
	// Read from a file and store it in a string
	// FORMAT OF THE FILE:
	// <image UUID> <instances running of that image>
	// eg: 6c55cf8fe339a52a798796d9ba0e765daharshitha	/var/lib/nova/instances/_base/6c55cf8fe339a52a798796d9ba0e765dac55aef7	count:2
	log.Info("Reading image instance association file.")
	data, err := ioutil.ReadFile(imageVMAssociationFile)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal([]byte(data), &ImageVMAssociations)
	if err != nil {
		return err
	}
	return nil
}

var fileMutex sync.Mutex

// SaveImageVMAssociation method saves instance image association to yaml file
func SaveImageVMAssociation() error {
	imageVMAssociationFile := consts.ConfigDirPath + consts.ImageInstanceCountAssociationFileName
	// FORMAT OF THE FILE:
	// <image UUID> <instances running of that image>
	// eg: 6c55cf8fe339a52a798796d9ba0e765daharshitha	/var/lib/nova/instances/_base/6c55cf8fe339a52a798796d9ba0e765dac55aef7	count:2
	log.Info("Writing to image instance association file.")
	data, err := yaml.Marshal(&ImageVMAssociations)
	if err != nil {
		return err
	}
	// Apply mutex lock to yaml file
	fileMutex.Lock()
	// Release the mutext lock
	defer fileMutex.Unlock()
	err = ioutil.WriteFile(imageVMAssociationFile, []byte(string(data)), 0644)
	if err != nil {
		return err
	}
	return nil
}

//IsImageEncrypted method is used to check if the image is encryped and returns a boolean value.
func IsImageEncrypted(filePath string) (bool, error) {
	//check if image is encrypted
	// this has to be changed
	fileCmdOutput, err := exec.ExecuteCommand("file", []string{filePath})
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
func CheckMountPathExistsAndMountVolume(mountPath, deviceMapperPath string) error {
	fmt.Println("Mounting the device mapper: ", deviceMapperPath)
	_, err := os.Stat(mountPath)
	if os.IsNotExist(err) {
		args := []string{"-p", mountPath}
		_, mkdirErr := exec.ExecuteCommand("mkdir", args)
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
func Cleanup() error {
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
