/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package setup

import (
	"flag"
	"fmt"
	csetup "intel/isecl/lib/common/v3/setup"
	hvsclient "intel/isecl/wlagent/v3/clients"
	"intel/isecl/wlagent/v3/common"
	"intel/isecl/wlagent/v3/config"
	"intel/isecl/wlagent/v3/consts"
	"os"

	"github.com/pkg/errors"
)

type RegisterSigningKey struct {
	Flags []string
}

func (rs RegisterSigningKey) Run(c csetup.Context) error {
	log.Trace("setup/register_signing_key:Run() Entering")
	defer log.Trace("setup/register_signing_key:Run() Leaving")
	fs := flag.NewFlagSet("SigningKey", flag.ContinueOnError)
	force := fs.Bool("force", false, "Re-register signing key with Verification service")
	err := fs.Parse(rs.Flags)
	if err != nil {
		return errors.Wrap(err, "setup/register_signing_key:Run(): Unable to parse flags")
	}
	if config.Configuration.ConfigComplete == false {
		return ErrMessageSetupIncomplete
	}

	if !*force && rs.Validate(c) == nil {
		fmt.Fprintln(os.Stdout, "Signing key already registered. Skipping this setup task.")
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
