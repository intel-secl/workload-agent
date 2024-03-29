/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package setup

import (
	"flag"
	"fmt"
	"github.com/pkg/errors"
	csetup "intel/isecl/lib/common/v4/setup"
	"intel/isecl/lib/tpmprovider/v4"
	"intel/isecl/wlagent/v4/common"
	"intel/isecl/wlagent/v4/config"
	"intel/isecl/wlagent/v4/consts"
	"os"
)

type SigningKey struct {
	T     tpmprovider.TpmFactory
	Flags []string
}

func (sk SigningKey) Run(c csetup.Context) error {
	log.Trace("setup/create_signing_key:Run() Entering")
	defer log.Trace("setup/create_signing_key:Run() Leaving")
	fmt.Println("Running setup task: SigningKey")
	fs := flag.NewFlagSet(consts.CreateSigningKey, flag.ContinueOnError)
	force := fs.Bool("force", false, "force recreation, will overwrite any existing signing key")
	err := fs.Parse(sk.Flags)
	if err != nil {
		return errors.Wrap(err, "setup/create_signing_key:Run() Unable to parse flags")
	}

	if config.Configuration.ConfigComplete == false {
		return ErrMessageSetupIncomplete
	}
	if !*force && sk.Validate(c) == nil {
		fmt.Fprintln(os.Stdout, "Signing key already created, skipping ...")
		log.Info("setup/create_signing_key:Run() Signing key already created, skipping ...")
		return nil
	}
	log.Info("setup/create_signing_key:Run() Creating signing key.")

	err = common.GenerateKey(tpmprovider.Signing, sk.T)
	if err != nil {
		return errors.Wrap(err, "setup/create_singing_key:Run() Error while generating tpm certified signing key")
	}
	return nil
}

func (sk SigningKey) Validate(c csetup.Context) error {
	log.Trace("setup/create_signing_key:Validate() Entering")
	defer log.Trace("setup/create_signing_key:Validate() Leaving")

	log.Info("setup/create_signing_key:Validate() Validation for signing key.")

	err := common.ValidateKey(tpmprovider.Signing)
	if err != nil {
		return errors.Wrap(err, "setup/create_singing_key:Validate() Error while validating signing key")
	}

	return nil
}
