/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package setup

import (
	csetup "intel/isecl/lib/common/setup"
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/common"
	"intel/isecl/wlagent/config"

	"github.com/pkg/errors"
)

type SigningKey struct {
	T tpm.Tpm
}

func (sk SigningKey) Run(c csetup.Context) error {
	log.Trace("setup/create_signing_key:Run() Entering")
	defer log.Trace("setup/create_signing_key:Run() Leaving")

	if config.Configuration.ConfigComplete == false {
		return ErrMessageSetupIncomplete
	}
	if sk.Validate(c) == nil {
		log.Info("setup/create_signing_key:Run() Signing key already created, skipping ...")
		return nil
	}
	log.Info("setup/create_signing_key:Run() Creating signing key.")

	err := common.GenerateKey(tpm.Signing, sk.T)
	if err != nil {
		return errors.Wrap(err, "setup/create_singing_key:Run() Error while generating tpm certified signing key")
	}
	return nil
}

func (sk SigningKey) Validate(c csetup.Context) error {
	log.Trace("setup/create_signing_key:Validate() Entering")
	defer log.Trace("setup/create_signing_key:Validate() Leaving")

	log.Info("setup/create_signing_key:Validate() Validation for signing key.")

	err := common.ValidateKey(tpm.Signing)
	if err != nil {
		return errors.Wrap(err, "setup/create_singing_key:Validate() Error while validating signing key")
	}

	return nil
}
