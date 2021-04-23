/*
 * Copyright (C) 2021 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package setup

import (
	"flag"
	"fmt"
	csetup "intel/isecl/lib/common/v3/setup"
	"intel/isecl/wlagent/v3/config"
	"intel/isecl/wlagent/v3/consts"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type Update_Service_Config struct {
	Flags []string
}

func (uc Update_Service_Config) Run(c csetup.Context) error {
	log.Trace("setup/update_service_config:Run() Entering")
	defer log.Trace("setup/update_service_config:Run() Leaving")
	fmt.Println("Running setup task: update_service_config")
	fs := flag.NewFlagSet("Update_Service_Config", flag.ContinueOnError)

	force := fs.Bool("force", false, "force recreation, will overwrite any existing signing key")
	err := fs.Parse(uc.Flags)
	if err != nil {
		fmt.Println("update_service_config setup: Unable to parse flags")
		return fmt.Errorf("update_service_config setup: Unable to parse flags")
	}

	if !*force && uc.Validate(c) == nil {
		fmt.Println("setup update_service_config: update_service_config config variables already set, so skipping update_service_config setup task...")
		log.Info("setup/update_service_config:Run() WLS update_service_config setup already complete, skipping ...")
		return nil
	}

	wlsAPIUrl, err := c.GetenvString(consts.WlsApiUrlEnv, "Workload Service URL")
	if err == nil && wlsAPIUrl != "" {
		config.Configuration.Wls.APIURL = wlsAPIUrl
	} else if strings.TrimSpace(config.Configuration.Wls.APIURL) == "" {
		return errors.Wrapf(err, "%s is not defined in environment or configuration file", consts.WlsApiUrlEnv)
	}

	wlaAASUser, err := c.GetenvString(consts.WlaUsernameEnv, "WLA Service Username")
	if err == nil && wlaAASUser != "" {
		config.Configuration.Wla.APIUsername = wlaAASUser
	} else if config.Configuration.Wla.APIUsername == "" {
		return errors.Wrapf(err, "%s is not defined in environment or configuration file", consts.WlaUsernameEnv)
	}

	wlaAASPassword, err := c.GetenvSecret(consts.WlaPasswordEnv, "WLA Service Password")
	if err == nil && wlaAASPassword != "" {
		config.Configuration.Wla.APIPassword = wlaAASPassword
	} else if strings.TrimSpace(config.Configuration.Wla.APIPassword) == "" {
		return errors.Wrapf(err, " is not defined in environment or configuration file", consts.WlaPasswordEnv)
	}

	if skipFlavorSignatureVerification, err := c.GetenvString(consts.SkipFlavorSignatureVerificationEnv,
		"Skip flavor signature verification"); err == nil {
		config.Configuration.SkipFlavorSignatureVerification, err = strconv.ParseBool(skipFlavorSignatureVerification)
		if err != nil {
			log.Warn(consts.SkipFlavorSignatureVerificationEnv, " is set to invalid value (should be true/false). "+
				"Setting it to true by default")
			config.Configuration.SkipFlavorSignatureVerification = true
		}
	} else {
		log.Info(consts.SkipFlavorSignatureVerificationEnv, " is not set. Setting it to true by default")
		config.Configuration.SkipFlavorSignatureVerification = true
	}

	logEntryMaxLength, err := c.GetenvInt(consts.LogEntryMaxlengthEnv, "Maximum length of each entry in a log")
	if err == nil && logEntryMaxLength >= consts.MinLogEntryMaxlength {
		config.Configuration.LogMaxLength = logEntryMaxLength
	} else if config.Configuration.LogMaxLength != 0 {
		log.Info("No change in Log Entry Max Length")
	} else {
		log.Info("Invalid Log Entry Max Length defined (should be > ", consts.MinLogEntryMaxlength, "), using default value:", consts.DefaultLogEntryMaxlength)
		config.Configuration.LogMaxLength = consts.DefaultLogEntryMaxlength
	}

	config.Configuration.LogEnableStdout = false
	logEnableStdout, err := c.GetenvString(consts.EnableConsoleLogEnv, "Workload Agent Enable standard output")
	if err == nil && logEnableStdout != "" {
		config.Configuration.LogEnableStdout, err = strconv.ParseBool(logEnableStdout)
		if err != nil {
			log.Info("Error while parsing the variable ", consts.EnableConsoleLogEnv, " setting to default value false")
		}
	}

	return config.Save()
}

func (u Update_Service_Config) Validate(c csetup.Context) error {
	log.Trace("setup/update_service_config:Validate() Entering")
	defer log.Trace("setup/update_service_config:Validate() Leaving")

	log.Info("setup/update_service_config:Validate() Validation for update_service_config")

	wla := &config.Configuration.Wla
	if wla.APIUsername == "" {
		return errors.New("setup/update_service_config:Validate() WLA User is not set")
	}
	if wla.APIPassword == "" {
		return errors.New("setup/update_service_config:Validate() WLA Password is not set ")
	}
	return nil
}
