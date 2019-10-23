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

type BindingKey struct {
	T tpm.Tpm
}

func (bk BindingKey) Run(c csetup.Context) error {
	log.Trace("setup/create_binding_key:Run() Entering")
	defer log.Trace("setup/create_binding_key:Run() Leaving")
	if config.Configuration.ConfigComplete == false {
		return ErrMessageSetupIncomplete
	}
	if bk.Validate(c) == nil {
		log.Info("setup/create_binding_key:Run() Binding key already created, skipping ...")
		return nil
	}
	log.Info("setup/create_binding_key:Run() Creating binding key.")

	err := common.GenerateKey(tpm.Binding, bk.T)
	if err != nil {
		return errors.Wrap(err, "setup/create_binding_key:Run() Error while generating tpm certified binding key")
	}
	return nil
}

func (bk BindingKey) Validate(c csetup.Context) error {
	log.Trace("setup/create_binding_key:Validate() Entering")
	defer log.Trace("setup/create_binding_key:Validate() Leaving")

	log.Info("setup/create_binding_key:Validate() Validation for binding key.")

	err := common.ValidateKey(tpm.Binding)
	if err != nil {
		return errors.Wrap(err, "setup/create_binding_key:Validate() Error while validating binding key")
	}

	return nil
}
