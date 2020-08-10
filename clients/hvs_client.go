/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package clients

import (
	"encoding/json"
	"fmt"
	"github.com/intel-secl/intel-secl/v3/pkg/clients/hvsclient"
	wlaModel "github.com/intel-secl/intel-secl/v3/pkg/model/wlagent"
	"github.com/pkg/errors"
	cLog "intel/isecl/lib/common/v2/log"
	csetup "intel/isecl/lib/common/v2/setup"
	"intel/isecl/wlagent/v2/config"
	"intel/isecl/wlagent/v2/consts"
	"os"
)

var log = cLog.GetDefaultLogger()
var secLog = cLog.GetSecurityLogger()

// Error is an error struct that contains error information thrown by the actual HVS
type Error struct {
	StatusCode int
	Message    string
}

func (e Error) Error() string {
	return fmt.Sprintf("hvs-client: failed (HTTP Status Code: %d)\nMessage: %s", e.StatusCode, e.Message)
}

// CertifyHostSigningKey sends a POST to /certify-host-signing-key to register signing key with HVS
func CertifyHostSigningKey(key *wlaModel.RegisterKeyInfo) (*wlaModel.SigningKeyCert, error) {
	log.Trace("clients/hvs_client:CertifyHostSigningKey() Entering")
	defer log.Trace("clients/hvs_client:CertifyHostSigningKey() Leaving")
	var keyCert wlaModel.SigningKeyCert

	rsp, err := certifyHostKey(key, "signing")
	if err != nil {
		return nil, errors.Wrap(err, "clients/hvs_client.go:CertifyHostSigningKey()  error registering signing key with HVS")
	}
	err = json.Unmarshal(rsp, &keyCert)
	if err != nil {
		log.Debugf("Could not unmarshal json from /rpc/certify-host-signing-key: %s", string(rsp))
		return nil, errors.Wrap(err, "clients/hvs_client.go:CertifyHostSigningKey() error decoding signing key certificate")
	}
	return &keyCert, nil
}

// CertifyHostBindingKey sends a POST to /certify-host-binding-key to register binding key with HVS
func CertifyHostBindingKey(key *wlaModel.RegisterKeyInfo) (*wlaModel.BindingKeyCert, error) {
	log.Trace("clients/hvs_client:CertifyHostBindingKey Entering")
	defer log.Trace("clients/hvs_client:CertifyHostBindingKey Leaving")
	var keyCert wlaModel.BindingKeyCert
	rsp, err := certifyHostKey(key, "binding")
	if err != nil {
		return nil, errors.Wrap(err, "clients/hvs_client.go:CertifyHostBindingKey() error registering binding key with HVS")
	}
	err = json.Unmarshal(rsp, &keyCert)
	if err != nil {
		log.Debugf("Could not unmarshal json from /rpc/certify-host-binding-key: %s", string(rsp))
		return nil, errors.Wrap(err, "clients/hvs_client.go:CertifyHostBindingKey() error decoding binding key certificate.")
	}
	return &keyCert, nil
}

func certifyHostKey(keyInfo *wlaModel.RegisterKeyInfo, keyUsage string) ([]byte, error) {
	log.Trace("clients/hvs_client:certifyHostKey Entering")
	defer log.Trace("clients/hvs_client:certifyHostKey Leaving")

	var c csetup.Context
	jwtToken, err := c.GetenvSecret(consts.BEARER_TOKEN_ENV, "BEARER_TOKEN")
	if jwtToken == "" || err != nil {
		fmt.Fprintln(os.Stderr, "BEARER_TOKEN is not defined in environment")
		return nil, errors.Wrap(err, "BEARER_TOKEN is not defined in environment")
	}

	vsClientFactory, err := hvsclient.NewVSClientFactory(config.Configuration.Mtwilson.APIURL, jwtToken, consts.TrustedCaCertsDir)
	if err != nil {
		return nil, errors.Wrap(err, "Error while instantiating VSClientFactory")
	}

	certifyHostKeysClient, err := vsClientFactory.CertifyHostKeysClient()
	if err != nil {
		return nil, errors.Wrap(err, "Error while instantiating CertifyHostKeysClient")
	}

	var responseData []byte
	if keyUsage == "signing" {
		responseData, err = certifyHostKeysClient.CertifyHostSigningKey(keyInfo)
	} else {
		responseData, err = certifyHostKeysClient.CertifyHostBindingKey(keyInfo)
	}
	if err != nil {
		return nil, errors.Wrap(err, "Error from response")
	}

	return responseData, nil
}
