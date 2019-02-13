package util

import (
	"intel/isecl/lib/common/crypt"
	"intel/isecl/wlagent/consts"
	"io/ioutil"
	"strings"
	"sync"
	"os"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// ImageVMAssociations is variable that consists of array of ImageVMAssociation struct
var ImageVMAssociations []ImageVMAssociation

// ImageVMAssociation is the global struct that is used to store the image vm count to yaml file
type ImageVMAssociation struct {
	ImageID   string
	ImagePath string
	VMCount   int
}

// LoadImageVMAssociation method loads image vm association from yaml file
func LoadImageVMAssociation() error {
	imageVMAssociationFile := consts.ConfigDirPath + consts.ImageVmCountAssociationFileName
	log.Info("Reading image vm association file.")
	file, err := os.OpenFile(imageVMAssociationFile, os.O_RDONLY|os.O_CREATE, 0644)
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

var fileMutex sync.Mutex

// SaveImageVMAssociation method saves vm image association to yaml file
func SaveImageVMAssociation() error {
	imageVMAssociationFile := consts.ConfigDirPath + consts.ImageVmCountAssociationFileName
	log.Info("Writing to image vm association file.")
	data, err := yaml.Marshal(&ImageVMAssociations)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(imageVMAssociationFilePath, []byte(string(associations)), 0644)
	if err != nil {
		return err
	}
	return nil
}

//IsFileEncrypted method is used to check if the image is encryped and returns a boolean value.
func IsFileEncrypted(encFilePath string) (bool, error) {

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
