/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package util

import (
	"encoding/json"
	cLog "intel/isecl/lib/common/v4/log"
	"intel/isecl/lib/common/v4/log/message"
	"intel/isecl/lib/tpmprovider/v4"
	"intel/isecl/wlagent/v4/config"
	"intel/isecl/wlagent/v4/consts"
	"io/ioutil"
	"os"
	"sync"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var log = cLog.GetDefaultLogger()
var secLog = cLog.GetSecurityLogger()
var ImageVMAssociations = make(map[string]*ImageVMAssociation)
var MapMtx sync.RWMutex

type ImageVMAssociation struct {
	ImagePath string `yaml:"imagepath"`
	VMCount   int    `yaml:"vmcount"`
}

// LoadImageVMAssociation method loads image vm association from yaml file
func LoadImageVMAssociation() error {
	log.Trace("util/util:LoadImageVMAssociation Entering")
	defer log.Trace("util/util:LoadImageVMAssociation Leaving")

	imageVMAssociationFilePath := consts.RunDirPath + consts.ImageVmCountAssociationFileName
	// Read from a file and store it in a string
	log.Info("Reading image vm association file.")
	MapMtx.RLock()
	imageVMAssociationFile, err := os.OpenFile(imageVMAssociationFilePath, os.O_RDWR|os.O_CREATE, 0600)
	MapMtx.RUnlock()
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
	err = yaml.Unmarshal(associations, &ImageVMAssociations)
	if err != nil {
		return err
	}
	return nil
}

// SaveImageVMAssociation method saves vm image association to yaml file
func SaveImageVMAssociation() error {
	log.Trace("util/util:SaveImageVMAssociation() Entering")
	defer log.Trace("util/util:SaveImageVMAssociation() Leaving")

	imageVMAssociationFilePath := consts.RunDirPath + consts.ImageVmCountAssociationFileName
	log.Infof("util/util:SaveImageVMAssociation() Writing to image vm association file %s", imageVMAssociationFilePath)
	associations, err := yaml.Marshal(&ImageVMAssociations)
	if err != nil {
		return err
	}
	fInfo, err := os.Stat(imageVMAssociationFilePath)
	if fInfo != nil && fInfo.Mode().Perm() != 0600 {
		return errors.Errorf("Invalid file permission on %s", imageVMAssociationFilePath)
	}

	MapMtx.Lock()
	defer MapMtx.Unlock()
	err = ioutil.WriteFile(imageVMAssociationFilePath, associations, 0600)
	if err != nil {
		return errors.Wrapf(err, "util/util:SaveImageVMAssociation() Error while writing file:%s", imageVMAssociationFilePath)
	}
	return nil
}

// GetTpmInstance method is used to get an instance of TPM to perform various tpm operations
func GetTpmInstance() (tpmprovider.TpmProvider, error) {
	log.Trace("util/util:GetTpmInstance() Entering")
	defer log.Trace("util/util:GetTpmInstance() Leaving")
	tpmFactory, err := tpmprovider.NewTpmFactory()
	if err != nil {
		return nil, errors.Wrap(err, "util/util:GetTpmInstance() Could not create TPM Factory ")
	}

	vmStartTpm, err := tpmFactory.NewTpmProvider()

	return vmStartTpm, nil
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
	defer t.Close()
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
