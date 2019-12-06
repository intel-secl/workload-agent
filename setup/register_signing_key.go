/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package setup

import (
	csetup "intel/isecl/lib/common/setup"
	hvsclient "intel/isecl/wlagent/clients"
	"intel/isecl/wlagent/common"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/consts"
	"os"

	"github.com/pkg/errors"
)

type RegisterSigningKey struct {
}

func (rs RegisterSigningKey) Run(c csetup.Context) error {
	log.Trace("setup/register_signing_key:Run() Entering")
	defer log.Trace("setup/register_signing_key:Run() Leaving")

	if config.Configuration.ConfigComplete == false {
		return ErrMessageSetupIncomplete
	}

	if rs.Validate(c) == nil {
		log.Info("setup/register_signing_key:Run() Signing key already registered. Skipping this setup task.")
		return nil
	}

	log.Info("setup/register_signing_key:Run() Registering signing key with host verification service.")
	signingKey, err := config.GetSigningKeyFromFile()
	if err != nil {
		return errors.Wrap(err, "setup/register_signing_key.go:Run() error reading signing key from  file ")
	}

	httpRequestBody, err := common.CreateRequest(signingKey)
	if err != nil {
		return errors.Wrap(err, "setup/register_signing_key.go:Run() error registering signing key ")
	}
	
	registerKey, err := hvsclient.CertifyHostSigningKey(httpRequestBody)
	if err != nil {
		secLog.WithError(err).Error("setup/register_signing_key.go:Run() error while certify host signing key from hvs")
		return errors.Wrap(err, "setup/register_signing_key.go:Run() error while certify host signing key from hvs")
	}

	err = common.WriteKeyCertToDisk(consts.ConfigDirPath+consts.SigningKeyPemFileName, registerKey.SigningKeyCertificate)
	if err != nil {
		return errors.New("setup/register_signing_key.go:Run() error writing signing key certificate to file")
	}
	return nil
}

// Validate checks whether or not the Register Signing Key task was completed successfully
func (rs RegisterSigningKey) Validate(c csetup.Context) error {
	log.Trace("setup/register_signing_key:Validate() Entering")
	defer log.Trace("setup/register_signing_key:Validate() Leaving")

	log.Info("setup/register_signing_key:Validate() Validation for registering signing key.")
	signingKeyCertPath := consts.ConfigDirPath + consts.SigningKeyPemFileName
	_, err := os.Stat(signingKeyCertPath)
	if os.IsNotExist(err) {
		return errors.New("setup/register_signing_key.go:Validate() Signing key certificate file does not exist")
	}
	return nil
}
