package util

import (
	"fmt"
	"intel/isecl/lib/common/crypt"
	"intel/isecl/lib/vml"
	"intel/isecl/wlagent/consts"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	xmlpath "gopkg.in/xmlpath.v2"
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
	imageVMAssociationFilePath := consts.ConfigDirPath + consts.ImageInstanceCountAssociationFileName
	// Read from a file and store it in a string
	// FORMAT OF THE FILE:
	// <image UUID> <instances running of that image>
	// eg: 6c55cf8fe339a52a798796d9ba0e765daharshitha	/var/lib/nova/instances/_base/6c55cf8fe339a52a798796d9ba0e765dac55aef7	count:2
	log.Info("Reading image instance association file.")
	imageVMAssociationFileContent, err := os.OpenFile(imageVMAssociationFile, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	associations, err := ioutil.ReadAll(imageVMAssociationFileContent)
	err = yaml.Unmarshal([]byte(associations), &ImageVMAssociations)
	if err != nil {
		return err
	}
	return nil
}

var fileMutex sync.Mutex

// SaveImageVMAssociation method saves instance image association to yaml file
func SaveImageVMAssociation() error {
	imageVMAssociationFilePath := consts.ConfigDirPath + consts.ImageInstanceCountAssociationFileName
	// FORMAT OF THE FILE:
	// <image UUID> <instances running of that image>
	// eg: 6c55cf8fe339a52a798796d9ba0e765daharshitha	/var/lib/nova/instances/_base/6c55cf8fe339a52a798796d9ba0e765dac55aef7	count:2
	log.Info("Writing to image instance association file.")
	associations, err := yaml.Marshal(&ImageVMAssociations)
	if err != nil {
		return err
	}
	// Apply mutex lock to yaml file
	fileMutex.Lock()
	// Release the mutext lock
	defer fileMutex.Unlock()
	err = ioutil.WriteFile(imageVMAssociationFile, []byte(string(associations)), 0644)
	if err != nil {
		return err
	}
	return nil
}

//IsImageEncrypted method is used to check if the image is encryped and returns a boolean value.
func IsImageEncrypted(encImagePath string) (bool, error) {

	var encryptionHeader crypt.EncryptionHeader
	//check if image is encrypted
	encImageContent, err := ioutil.ReadFile(encImagePath)
	if err != nil {
		log.Info("Error while reading the file contents")
		return false, err
	}

	magicText := encImageContent[:len(encryptionHeader.MagicText)]
	if !strings.Contains(string(magicText), crypt.EncryptionHeaderMagicText) {
		log.Infof("Image file located in %s is not encrypted", encImagePath)
		return false, nil
	}

	log.Infof("Image file located in %s is encrypted", encImagePath)
	return true, nil
}

// CheckMountPathExistsAndMountVolume method is used to check if te mount path exists,
// if it does not exists, the method creates the mount path and mounts the device mapper.
func CheckMountPathExistsAndMountVolume(mountPath, deviceMapperPath string) error {
	log.Infof("Mounting the device mapper: %s", deviceMapperPath)
	mkdirErr := os.MkdirAll(mountPath, 0655)
	if mkdirErr != nil {
		log.Info("Error while creating the mount point for the image device mapper")
		return mkdirErr
	}

	mountErr := vml.Mount(deviceMapperPath, mountPath)
	if mountErr != nil {
		if !strings.Contains(mountErr.Error(), "device is already mounted") {
			log.Info("Error while mounting the image device mapper")
			return mountErr
		}
	}
	return nil
}

func getItemFromDomainXML(domainXML *xmlpath.Node, xmlPath string, item string) (string, error) {

	// parse the item in xml path from domainXMl
	parseItem := xmlpath.MustCompile(xmlPath)
	itemValue, ok := parseItem.String(domainXML)
	if !ok {
		log.Infof("Error while getting %s from domainXMl", item)
		return "", fmt.Errorf("Error while getting %s from domainXMl", item)
	}
	return itemValue, nil
}

// GetInstanceUUID method is used to get the instance UUID value from the domain XML
func GetInstanceUUID(domainXML *xmlpath.Node) (string, error) {
	return getItemFromDomainXML(domainXML, "/domain/uuid", "instanceUUID")
}

// GetInstancePath method is used to get the instance path value from the domain XML
func GetInstancePath(domainXML *xmlpath.Node) (string, error) {
	return getItemFromDomainXML(domainXML, "/domain/devices/disk/source/@file", "instancePath")
}

// GetImageUUID method is used to get the image UUID value from the domain XML
func GetImageUUID(domainXML *xmlpath.Node) (string, error) {
	return getItemFromDomainXML(domainXML, "/domain/metadata//node()[@type='image']/@uuid", "imageUUID")
}

// GetImagePath method is used to get the image path value from the domain XML
func GetImagePath(domainXML *xmlpath.Node) (string, error) {
	imagePath, err := getItemFromDomainXML(domainXML, "/domain/devices/disk/backingStore/source/@file", "imagePath")
	if err != nil {
		return getItemFromDomainXML(domainXML, "/domain/devices/disk/backingStore/source/@dev", "imagePath")
	}
	return imagePath, nil
}

// GetDiskSize method is used to get the disk size value from the domain XML
func GetDiskSize(domainXML *xmlpath.Node) (string, error) {
	return getItemFromDomainXML(domainXML, "/domain/metadata//disk", "diskSize")
}
