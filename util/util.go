/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package util

import (
	"encoding/hex"
	"encoding/json"
	cLog "intel/isecl/lib/common/log"
	"intel/isecl/lib/common/log/message"
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/consts"
	"io/ioutil"
	"os"
	"sync"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var log = cLog.GetDefaultLogger()
var secLog = cLog.GetSecurityLogger()
var ImageVMAssociations = make(map[string]*ImageVMAssociation)

type ImageVMAssociation struct {
	ImagePath string `yaml:"imagepath"`
	VMCount   int    `yaml:"vmcount"`
}

// LoadImageVMAssociation method loads image vm association from yaml file
func LoadImageVMAssociation() error {
	log.Trace("util/util: LoadImageVMAssociation Entering")
	defer log.Trace("util/util: LoadImageVMAssociation Leaving")
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
	log.Trace("util/util:SaveImageVMAssociation() Entering")
	defer log.Trace("util/util:SaveImageVMAssociation() Leaving")

	imageVMAssociationFilePath := consts.ConfigDirPath + consts.ImageVmCountAssociationFileName
	log.Infof("util/util:SaveImageVMAssociation() Writing to image vm association file %s", imageVMAssociationFilePath)
	associations, err := yaml.Marshal(&ImageVMAssociations)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(imageVMAssociationFilePath, []byte(string(associations)), 0644)
	if err != nil {
		return errors.Wrapf(err, "util/util:SaveImageVMAssociation() Error while writing file:%s", imageVMAssociationFilePath)
	}
	return nil
}

var vmStartTpm tpm.Tpm

// GetTpmInstance method is used to get an instance of TPM to perform various tpm operations
func GetTpmInstance() (tpm.Tpm, error) {
	log.Trace("util/util:GetTpmInstance() Entering")
	defer log.Trace("util/util:GetTpmInstance() Leaving")
	if vmStartTpm != nil {
		log.Debug("util/util:GetTpmInstance() Returning an existing connection to the tpm")
		return vmStartTpm, nil
	}
	return nil, errors.New("util/util:GetTpmInstance() Connection to TPM does not exist")
}

func GetNewTpmInstance() (tpm.Tpm, error){
	log.Trace("util/util:GetNewTpmInstance() Entering")
	defer log.Trace("util/util:GetNewTpmInstance() Leaving")
	var err error
	secLog.Infof("util/util:GetTpmInstance() Opening a new connection to the tpm %s", message.SU)
	vmStartTpm, err = tpm.Open()
	return vmStartTpm, err
}

// CloseTpmInstance method is used to close an instance of TPM
func CloseTpmInstance() {
	log.Trace("util/util:CloseTpmInstance() Entering")
	defer log.Trace("util/util:CloseTpmInstance() Leaving")


	if vmStartTpm != nil {
		secLog.Infof("util/util:CloseTpmInstance() Closing connection to the tpm %s", message.SU)
		vmStartTpm.Close()
	}
}

// UnwrapKey method is used to unbind a key using TPM
func UnwrapKey(tpmWrappedKey []byte) ([]byte, error) {
	log.Trace("util/util:UnwrapKey() Entering")
	defer log.Trace("util/util:UnwrapKey() Leaving")

	var certifiedKey tpm.CertifiedKey
	t, err := GetTpmInstance()
	if err != nil {
		return nil, errors.Wrap(err, "util/util:UnwrapKey() Could not establish connection to TPM ")
	}

	log.Debug("util/util:UnwrapKey() Reading the binding key certificate")
	bindingKeyFilePath := consts.ConfigDirPath + consts.BindingKeyFileName
	bindingKeyCert, fileErr := ioutil.ReadFile(bindingKeyFilePath)
	if fileErr != nil {
		return nil, errors.New("util/util:UnwrapKey() Error while reading the binding key certificate")
	}

	log.Debug("util/util:UnwrapKey() Unmarshalling the binding key certificate file contents to TPM CertifiedKey object")
	jsonErr := json.Unmarshal(bindingKeyCert, &certifiedKey)
	if jsonErr != nil {
		return nil, errors.New("util/util:UnwrapKey() Error unmarshalling the binding key file contents to TPM CertifiedKey object")
	}

	log.Debug("util/util:UnwrapKey() Binding key deserialized")
	keyAuth, _ := hex.DecodeString(config.Configuration.BindingKeySecret)
	secLog.Infof("util/util:UnwrapKey() Binding key getting decrypted, %s", message.SU)
	key, unbindErr := t.Unbind(&certifiedKey, keyAuth, tpmWrappedKey)
	if unbindErr != nil {
		return nil, errors.Wrap(unbindErr, "util/util:UnwrapKey() error while unbinding the tpm wrapped key ")
	}
	log.Debug("util/util:UnwrapKey() Unbinding TPM wrapped key was successful, return the key")
	return key, nil
}
