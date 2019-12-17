/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package setup

/**
** @author srege
**/

import (
	"flag"
	"fmt"
	csetup "intel/isecl/lib/common/setup"
	hvsclient "intel/isecl/wlagent/clients"
	"intel/isecl/wlagent/common"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/consts"
	"os"
	"os/user"
	"strconv"

	"github.com/pkg/errors"
)

type RegisterBindingKey struct {
	Flags []string
}

func (rb RegisterBindingKey) Run(c csetup.Context) error {
	log.Trace("setup/register_binding_key:Run() Entering")
	defer log.Trace("setup/register_binding_key:Run() Leaving")
	fs := flag.NewFlagSet("RegisterBindingKey", flag.ContinueOnError)
	force := fs.Bool("force", false, "Re-register binding key with Verification service")
	err := fs.Parse(rb.Flags)
	if err != nil {
		return errors.Wrap(err, "setup/register_binding_key:Run() Unable to parse flags")
	}
	if config.Configuration.ConfigComplete == false {
		return ErrMessageSetupIncomplete
	}

	if !*force && rb.Validate(c) == nil {
		fmt.Fprintln(os.Stdout, "Binding key already registered. Skipping this setup task.")
		log.Info("setup/register_binding_key:Run() Binding key already registered. Skipping this setup task.")
		return nil
	}

	log.Info("setup/register_binding_key:Run() Registering binding key with host verification service.")
	bindingKey, err := config.GetBindingKeyFromFile()
	if err != nil {
		return errors.Wrap(err, "setup/register_binding_key:Run() error reading binding key from  file. ")
	}

	httpRequestBody, err := common.CreateRequest(bindingKey)
	if err != nil {
		return errors.Wrap(err, "setup/register_binding_key:Run() error registering binding key. ")
	}

	registerKey, err := hvsclient.CertifyHostBindingKey(httpRequestBody)
	if err != nil {
		secLog.WithError(err).Error("setup/register_binding_key.go:Run() error while certifying host binding key with hvs")
		return errors.Wrap(err, "setup/register_binding_key:Run() error while certifying host binding key with hvs")
	}

	err = common.WriteKeyCertToDisk(consts.ConfigDirPath+consts.BindingKeyPemFileName, registerKey.BindingKeyCertificate)
	if err != nil {
		return errors.New("setup/register_binding_key:Run() error writing binding key certificate to file")
	}

	return rb.setBindingKeyPemFileOwner()
}

// Validate checks whether or not the register binding key task was completed successfully
func (rb RegisterBindingKey) Validate(c csetup.Context) error {
	log.Trace("setup/register_binding_key:Validate() Entering")
	defer log.Trace("setup/register_binding_key:Validate() Leaving")

	log.Info("setup/register_binding_key:Validate() Validation for registering binding key.")
	bindingKeyCertFilePath := consts.ConfigDirPath + consts.BindingKeyPemFileName
	_, err := os.Stat(bindingKeyCertFilePath)
	if os.IsNotExist(err) {
		return errors.New("setup/register_binding_key:Validate() binding key certificate file does not exist")
	}
	return nil
}

// setBindingKeyFileOwner sets the owner of the binding key file to the trustagent user
// This is necessary for the TrustAgent to add the binding key to the manifest.
func (rb RegisterBindingKey) setBindingKeyPemFileOwner() (err error) {
	log.Trace("setup/register_binding_key:setBindingKeyPemFileOwner() Entering")
	defer log.Trace("setup/register_binding_key:setBindingKeyPemFileOwner() Leaving")
	var usr *user.User
	err = nil
	// get the user id from the configuration variable that we have set
	if config.Configuration.TrustAgent.User == "" {
		return errors.New("setup/register_binding_key:setBindingKeyPemFileOwner() trust agent user name cannot be empty in configuration")
	}

	if usr, err = user.Lookup(config.Configuration.TrustAgent.User); err != nil {
		return errors.Wrapf(err, "setup/register_binding_key:setBindingKeyPemFileOwner() could not lookup up user id of trust agent user : %s", config.Configuration.TrustAgent.User)
	}

	uid, _ := strconv.Atoi(usr.Uid)
	gid, _ := strconv.Atoi(usr.Gid)
	// no need to check errors for the above two call since had just looked up the user
	// using the user.Lookup call
	err = os.Chown(consts.ConfigDirPath+consts.BindingKeyPemFileName, uid, gid)
	if err != nil {
		return errors.Wrapf(err, "setup/register_binding_key:setBindingKeyPemFileOwner() Could not set permission for File %s", consts.ConfigDirPath+consts.BindingKeyPemFileName)
	}

	return nil
}
