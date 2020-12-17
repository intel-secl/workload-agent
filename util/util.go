/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package util

import (
	"encoding/json"
	cLog "intel/isecl/lib/common/v3/log"
	"intel/isecl/lib/common/v3/log/message"
	"intel/isecl/lib/tpmprovider/v3"
	"intel/isecl/wlagent/v3/config"
	"intel/isecl/wlagent/v3/consts"
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
	imageVMAssociationFile, err := os.OpenFile(imageVMAssociationFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer func() {
		derr := imageVMAssociationFile.Close()
		if derr != nil {
			log.WithError(derr).Error("Error closing file")
		}
	}()
	associations, err := ioutil.ReadAll(imageVMAssociationFile)
	if err != nil {
		return err
	}
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
	fInfo, err := os.Stat(imageVMAssociationFilePath)
	if fInfo != nil && fInfo.Mode().Perm() != 0644 {
		return errors.Errorf("Invalid file permission on %s", imageVMAssociationFilePath)
	}

	err = ioutil.WriteFile(imageVMAssociationFilePath, []byte(string(associations)), 0644)
	if err != nil {
		return errors.Wrapf(err, "util/util:SaveImageVMAssociation() Error while writing file:%s", imageVMAssociationFilePath)
	}
	return nil
}

var vmStartTpm tpmprovider.TpmProvider

// GetTpmInstance method is used to get an instance of TPM to perform various tpm operations
func GetTpmInstance() (tpmprovider.TpmProvider, error) {
	log.Trace("util/util:GetTpmInstance() Entering")
	defer log.Trace("util/util:GetTpmInstance() Leaving")
	if vmStartTpm == nil {
		tpmFactory, err := tpmprovider.NewTpmFactory()
		if err != nil {
			return nil, errors.Wrap(err, "util/util:GetTpmInstance() Could not create TPM Factory ")
		}

		vmStartTpm, err = tpmFactory.NewTpmProvider()
		if err != nil {
			return nil, errors.Wrap(err, "util/util:GetTpmInstance() Could not create TPM ")
		}
	} else {
		log.Debug("util/util:GetTpmInstance() Returning an existing connection to the tpm")
	}

	return vmStartTpm, nil
}

// func GetNewTpmInstance() (tpm.Tpm, error){
// 	log.Trace("util/util:GetNewTpmInstance() Entering")
// 	defer log.Trace("util/util:GetNewTpmInstance() Leaving")
// 	var err error
// 	secLog.Infof("util/util:GetTpmInstance() %s, Opening a new connection to the tpm", message.SU)
// 	vmStartTpm, err = tpm.Open()
// 	return vmStartTpm, err
// }

// CloseTpmInstance method is used to close an instance of TPM
func CloseTpmInstance() {
	log.Trace("util/util:CloseTpmInstance() Entering")
	defer log.Trace("util/util:CloseTpmInstance() Leaving")

	if vmStartTpm != nil {
		secLog.Infof("util/util:CloseTpmInstance() %s, Closing connection to the tpm", message.SU)
		vmStartTpm.Close()
	}
}

// UnwrapKey method is used to unbind a key using TPM
func UnwrapKey(tpmWrappedKey []byte) ([]byte, error) {
	log.Trace("util/util:UnwrapKey() Entering")
	defer log.Trace("util/util:UnwrapKey() Leaving")

	if len(tpmWrappedKey) == 0 {
		return nil, errors.New("util/util:UnwrapKey() tpm wrapped key is empty")
	}

	var certifiedKey tpmprovider.CertifiedKey
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
	secLog.Infof("util/util:UnwrapKey() %s, Binding key getting decrypted", message.SU)
	key, unbindErr := t.Unbind(&certifiedKey, config.Configuration.BindingKeySecret, tpmWrappedKey)
	if unbindErr != nil {
		return nil, errors.Wrap(unbindErr, "util/util:UnwrapKey() error while unbinding the tpm wrapped key ")
	}
	log.Debug("util/util:UnwrapKey() Unbinding TPM wrapped key was successful, return the key")
	return key, nil
}
