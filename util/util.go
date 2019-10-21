/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package util

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/consts"
	"io/ioutil"
	"os"
	"sync"
)

var ImageVMAssociations = make(map[string]*ImageVMAssociation)

type ImageVMAssociation struct {
	ImagePath string `yaml:"imagepath"`
	VMCount   int `yaml:"vmcount"`
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

var vmStartTpm tpm.Tpm

// GetTpmInstance method is used to get an instance of TPM to perform various tpm operations
func GetTpmInstance() (tpm.Tpm, error) {
	if vmStartTpm == nil {
		log.Debug("Opening a new connection to the tpm")
		return tpm.Open()
	}
	return vmStartTpm, nil
}

// CloseTpmInstance method is used to close an instance of TPM
func CloseTpmInstance() {
	if vmStartTpm != nil {
		log.Debug("Closing connection to the tpm")
		vmStartTpm.Close()
	}
}

// UnwrapKey method is used to unbind a key using TPM
func UnwrapKey(tpmWrappedKey []byte) ([]byte, error) {
	var certifiedKey tpm.CertifiedKey
	t, err := GetTpmInstance()
	if err != nil {
		return nil, fmt.Errorf("could not establish connection to TPM: %s", err)
	}
	
	log.Debug("Reading the binding key certificate")
	bindingKeyFilePath := consts.ConfigDirPath + consts.BindingKeyFileName
	bindingKeyCert, fileErr := ioutil.ReadFile(bindingKeyFilePath)
	if fileErr != nil {
		return nil, errors.New("error while reading the binding key certificate")
	}

	log.Debug("Unmarshalling the binding key certificate file contents to TPM CertifiedKey object")
	jsonErr := json.Unmarshal(bindingKeyCert, &certifiedKey)
	if jsonErr != nil {
		return nil, errors.New("error unmarshalling the binding key file contents to TPM CertifiedKey object")
	}

	log.Debug("Binding key deserialized")
	keyAuth, _ := hex.DecodeString(config.Configuration.BindingKeySecret)
	key, unbindErr := t.Unbind(&certifiedKey, keyAuth, tpmWrappedKey)
	if unbindErr != nil {
		return nil, fmt.Errorf("error while unbinding the tpm wrapped key: %s", unbindErr.Error())
	}
	log.Debug("Unbinding TPM wrapped key was successful, return the key")
	return key, nil
}
