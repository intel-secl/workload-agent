/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package common

import (
	"encoding/hex"
	"encoding/json"
	"intel/isecl/lib/common/crypt"
	cLog "intel/isecl/lib/common/log"
	"intel/isecl/lib/common/log/message"
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/consts"
	"os"

	"github.com/pkg/errors"
)

const secretKeyLength int = 20

var log = cLog.GetDefaultLogger()
var secLog = cLog.GetSecurityLogger()

// tpmCertifiedKeySetup calls the TPM helper library to export a binding or signing keypair
func createKey(usage tpm.Usage, t tpm.Tpm) (tpmck *tpm.CertifiedKey, err error) {
	log.Trace("common/key_creation:createKey() Entering")
	defer log.Trace("common/key_creation:createKey() Leaving")
	if usage != tpm.Binding && usage != tpm.Signing {
		return nil, errors.New("common/key_creation:createKey()  Incorrect KeyUsage parameter - needs to be signing or binding")
	}
	secretbytes, err := crypt.GetRandomBytes(secretKeyLength)
	if err != nil {
		return nil, err
	}
	// get the aiksecret. This will return a byte array.
	log.Debug("common/key_creation:createKey() Getting aik secret from trusagent configuration.")
	aiksecret, err := config.GetAikSecret()
	if err != nil {
		return nil, err
	}
	secLog.Infof("common/key_creation:createKey() %s, Calling CreateCertifiedKey of tpm library to create and certify signing or binding key", message.SU)
	tpmck, err = t.CreateCertifiedKey(usage, secretbytes, aiksecret)
	if err != nil {
		return nil, err
	}

	switch usage {
	case tpm.Binding:
		config.Configuration.BindingKeySecret = hex.EncodeToString(secretbytes)
	case tpm.Signing:
		config.Configuration.SigningKeySecret = hex.EncodeToString(secretbytes)
	}

	config.Save()

	return tpmck, nil
}

//Todo: for now, this will always overwrite the file. Should be a parameter
// that forces overwrite of file.
func writeCertifiedKeyToDisk(tpmck *tpm.CertifiedKey, filepath string) error {
	log.Trace("common/key_creation:writeCertifiedKeyToDisk() Entering")
	defer log.Trace("common/key_creation:writeCertifiedKeyToDisk() Leaving")

	if tpmck == nil {
		return errors.New("common/key_creation:writeCertifiedKeyToDisk() certifiedKey struct is empty")
	}

	// Marshal the certified key to json
	json, err := json.MarshalIndent(tpmck, "", "    ")
	if err != nil {
		return errors.Wrap(err, "common/key_creation:writeCertifiedKeyToDisk() Error while marshalling tpm certified key to json")
	}

	// create a file and write the json value to it and finally close it
	f, err := os.Create(filepath)
	if err != nil {
		return errors.New("common/key_creation:writeCertifiedKeyToDisk() Could not create file Error:" + err.Error())
	}
	f.WriteString(string(json))
	f.WriteString("\n")
	defer f.Close()

	return nil
}

// GenerateKey creates a TPM binding or signing key
// It uses the AiKSecret that is saved in the Workload Agent configuration
// that is obtained from the trust agent, a randomn secret and uses the TPM
// to generate a keypair that is tied to the TPM
func GenerateKey(usage tpm.Usage, t tpm.Tpm) error {
	log.Trace("common/key_creation:GenerateKey() Entering")
	defer log.Trace("common/key_creation:GenerateKey() Leaving")

	if t == nil || (usage != tpm.Binding && usage != tpm.Signing) {
		return errors.New("common/key_creation:GenerateKey() Certified key or connection to TPM library failed")
	}

	// Create and certify the signing or binding key
	certKey, err := createKey(usage, t)
	if err != nil {
		return errors.Wrap(err, "common/key_creation:GenerateKey() Error while creating binding/signing key")
	}

	// Get the name of signing or binding key files depending on input parameter
	var filename string
	switch usage {
	case tpm.Binding:
		filename = consts.BindingKeyFileName
	case tpm.Signing:
		filename = consts.SigningKeyFileName
	}

	// Join configuration path and signing or binding file name
	filepath := consts.ConfigDirPath + filename

	// Writing certified key value to file path
	err = writeCertifiedKeyToDisk(certKey, filepath)
	if err != nil {
		return errors.Wrapf(err, "common/key_creation:GenerateKey() Error while writing key to the file %s", filepath)
	}

	log.Info("common/key_creation:GenerateKey() Key is stored at file path : ", filepath)
	return nil
}

// ValidateKey validates if a key of type binding or signing is actually configured in
// the Workload Agent
// Installed method of the CertifiedKey checks if there is a key already installed.
// For now, this only checks for the existence of the file and does not check if
// contents of the file are indeed correct
func ValidateKey(usage tpm.Usage) error {
	log.Trace("common/key_creation:ValidateKey() Entering")
	defer log.Trace("common/key_creation:ValidateKey() Leaving")

	// Get the name of signing or binding key files depending on input parameter
	var filename string
	switch usage {
	case tpm.Binding:
		filename = consts.BindingKeyFileName
	case tpm.Signing:
		filename = consts.SigningKeyFileName
	}

	// Join configuration path and signing or binding file name
	filepath := consts.ConfigDirPath + filename
	fi, err := os.Stat(filepath)
	if err != nil {
		return errors.Wrapf(err, "common/key_creation:ValidateKey() Could not find file %s", filepath)
	}
	if fi == nil && !fi.Mode().IsRegular() {
		return errors.New("common/key_creation:ValidateKey() Key file path is incorrect")
	}
	return nil
}
