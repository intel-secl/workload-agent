package util

import (
	"intel/isecl/lib/common/crypt"
	"intel/isecl/wlagent/consts"
	"io/ioutil"
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
	imageVMAssociationFile := consts.ConfigDirPath + consts.ImageVmCountAssociationFileName
	// Read from a file and store it in a string
	// FORMAT OF THE FILE:
	// <image UUID> <instances running of that image>
	// eg: 6c55cf8fe339a52a798796d9ba0e765daharshitha	/var/lib/nova/instances/_base/6c55cf8fe339a52a798796d9ba0e765dac55aef7	count:2
	log.Info("Reading image instance association file.")
	imageVMAssociationFile, err := os.OpenFile(imageVMAssociationFilePath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer imageVMAssociationFile.Close()
	associations, err := ioutil.ReadAll(imageVMAssociationFile)
	err = yaml.Unmarshal([]byte(associations), &ImageVMAssociations)
	if err != nil {
		return err
	}
	return nil
}

// SaveImageVMAssociation method saves instance image association to yaml file
func SaveImageVMAssociation() error {
	imageVMAssociationFile := consts.ConfigDirPath + consts.ImageVmCountAssociationFileName
	// FORMAT OF THE FILE:
	// <image UUID> <instances running of that image>
	// eg: 6c55cf8fe339a52a798796d9ba0e765daharshitha	/var/lib/nova/instances/_base/6c55cf8fe339a52a798796d9ba0e765dac55aef7	count:2
	log.Info("Writing to image instance association file.")
	associations, err := yaml.Marshal(&ImageVMAssociations)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(imageVMAssociationFilePath, []byte(string(associations)), 0644)
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
		return false, err
	}

	magicText := encImageContent[:len(encryptionHeader.MagicText)]
	if !strings.Contains(string(magicText), crypt.EncryptionHeaderMagicText) {
		return false, nil
	}

	return true, nil
}
