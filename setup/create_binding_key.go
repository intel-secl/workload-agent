/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package setup

import (
	"flag"
	"fmt"
	cLog "intel/isecl/lib/common/v3/log"
	csetup "intel/isecl/lib/common/v3/setup"
	"intel/isecl/lib/tpmprovider/v3"
	"intel/isecl/wlagent/v3/common"
	"intel/isecl/wlagent/v3/config"
	"os"

	"github.com/pkg/errors"
)

type BindingKey struct {
	T     tpmprovider.TpmFactory
	Flags []string
}

var log = cLog.GetDefaultLogger()
var secLog = cLog.GetSecurityLogger()
var ErrMessageSetupIncomplete = errors.New("configuration is not complete - setup tasks can be completed only after configuration")

func (bk BindingKey) Run(c csetup.Context) error {
	log.Trace("setup/create_binding_key:Run() Entering")
	defer log.Trace("setup/create_binding_key:Run() Leaving")
	fs := flag.NewFlagSet("BindingKey", flag.ContinueOnError)
	force := fs.Bool("force", false, "force recreation, will overwrite any existing signing key")
	err := fs.Parse(bk.Flags)
	if err != nil {
		return errors.Wrap(err, "setup/create_binding_key:Run() Unable to parse flags")
	}
	if config.Configuration.ConfigComplete == false {
		return ErrMessageSetupIncomplete
	}
	if !*force && bk.Validate(c) == nil {
		fmt.Fprintln(os.Stdout, "Binding key already created, skipping ...")
		log.Info("setup/create_binding_key:Run() Binding key already created, skipping ...")
		return nil
	}
	log.Info("setup/create_binding_key:Run() Creating binding key.")

	err = common.GenerateKey(tpmprovider.Binding, bk.T)
	if err != nil {
		return errors.Wrap(err, "setup/create_binding_key:Run() Error while generating tpm certified binding key")
	}
	return nil
}

func (bk BindingKey) Validate(c csetup.Context) error {
	log.Trace("setup/create_binding_key:Validate() Entering")
	defer log.Trace("setup/create_binding_key:Validate() Leaving")

	log.Info("setup/create_binding_key:Validate() Validation for binding key.")

	err := common.ValidateKey(tpmprovider.Binding)
	if err != nil {
		return errors.Wrap(err, "setup/create_binding_key:Validate() Error while validating binding key")
	}

	return nil
}
