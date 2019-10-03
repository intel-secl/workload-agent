/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package common

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	exec "intel/isecl/lib/common/exec"
	hvsclient "intel/isecl/wlagent/clients"
	tpm "intel/isecl/lib/tpm"
	"intel/isecl/wlagent/consts"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
)

func CreateRequest(key []byte) (*hvsclient.RegisterKeyInfo, error) {
	var httpRequestBody *hvsclient.RegisterKeyInfo
	var keyInfo tpm.CertifiedKey
	var tpmVersion string
	var err error

	err = json.Unmarshal(key, &keyInfo)
	if err != nil {
		return httpRequestBody, errors.New("error unmarshalling. " + err.Error())
	}

	//get trustagent aik cert location
	//TODO Vinil
	aikCertName, _ := exec.MkDirFilePathFromEnvVariable(consts.TAConfigDirEnvVar, "aik.pem", true)

	//set tpm version
	//TODO Vinil
	if keyInfo.Version == 2 {
		tpmVersion = "2.0"
	} else {
		tpmVersion = "1.2"
	}

	aikCert, err := ioutil.ReadFile(aikCertName)
	if err != nil {
		return httpRequestBody, errors.New("error reading certificate file. " + err.Error())
	}
	aikDer, _ := pem.Decode(aikCert)
	_, err = x509.ParseCertificate(aikDer.Bytes)
	if err != nil {
		return httpRequestBody, errors.New("error parsing certificate file. " + err.Error())
	}

	// TODO remove hack below. This hack was added since key stored on disk needs to be modified
	// so that HVS can register the key.
	// ISECL - 3506 opened to address this issue later
	//construct request body
	httpRequestBody = &hvsclient.RegisterKeyInfo{
		PublicKeyModulus:       keyInfo.PublicKey,
		TpmCertifyKey:          keyInfo.KeyAttestation[2:],
		TpmCertifyKeySignature: keyInfo.KeySignature,
		AikDerCertificate:      aikDer.Bytes,
		NameDigest:             append(keyInfo.KeyName[1:], make([]byte, 34)...),
		TpmVersion:             tpmVersion,
		OsType:                 strings.Title(runtime.GOOS),
	}

	return httpRequestBody, nil
}
func WriteKeyCertToDisk(keyCertPath string, aikPem []byte) error {
	file, err := os.Create(keyCertPath)
	if err != nil {
		return errors.New("error creating file. " + err.Error())
	}
	if err = pem.Encode(file, &pem.Block{Type: consts.PemCertificateHeader, Bytes: aikPem}); err != nil {
		return errors.New("error writing certificate to file")
	}
	return nil

}
