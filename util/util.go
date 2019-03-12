package util

import (
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/consts"
	"io/ioutil"
	"os"
	"sync"
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
	imageVMAssociationFilePath := consts.ConfigDirPath + consts.ImageVmCountAssociationFileName
	// Read from a file and store it in a string
	log.Info("Reading image vm association file.")
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

var fileMutex sync.Mutex

// SaveImageVMAssociation method saves vm image association to yaml file
func SaveImageVMAssociation() error {
	imageVMAssociationFilePath := consts.ConfigDirPath + consts.ImageVmCountAssociationFileName
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

var vmStartTpm tpm.Tpm

func GetTpmInstance() (tpm.Tpm, error) {
	var err error = nil
	if vmStartTpm == nil {
		log.Debug("Opening a new connection to the tpm")
		vmStartTpm, err = tpm.Open()
	}
	return vmStartTpm, err
}

func CloseTpmInstance() {
	if vmStartTpm != nil {
		log.Debug("Closing connection to the tpm")
		vmStartTpm.Close()
	}
}
