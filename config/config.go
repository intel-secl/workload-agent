/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package config

import (
	"fmt"
	cLog "intel/isecl/lib/common/v3/log"
	"intel/isecl/lib/common/v3/log/message"
	cLogInt "intel/isecl/lib/common/v3/log/setup"
	csetup "intel/isecl/lib/common/v3/setup"
	"intel/isecl/wlagent/v3/consts"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Configuration is the global configuration struct that is marshalled/unmarshaled to a persisted yaml file
var Configuration struct {
	BindingKeySecret string
	SigningKeySecret string
	CmsTlsCertDigest string
	Hvs              struct {
		APIURL string
	}
	Wls struct {
		APIURL string
	}
	Wla struct {
		APIUsername string
		APIPassword string
	}

	TrustAgent struct {
		ConfigDir  string
		AikPemFile string
		User       string
	}
	Aas struct {
		BaseURL string
	}
	Cms struct {
		BaseURL string
	}
	SkipFlavorSignatureVerification bool
	LogLevel                        logrus.Level
	LogMaxLength                    int
	ConfigComplete                  bool
	LogEnableStdout                 bool
}

var (
	configFilePath = consts.ConfigDirPath + consts.ConfigFileName
)

var secLog = cLog.GetSecurityLogger()
var log = cLog.GetDefaultLogger()

func getFileContentFromConfigDir(fileName string) ([]byte, error) {
	filePath := consts.ConfigDirPath + fileName
	// check if key file exists
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("File does not exist - %s", filePath)
	}

	return ioutil.ReadFile(filePath)
}

func GetSigningKeyFromFile() ([]byte, error) {
	log.Trace("config/config:GetSigningKeyFromFile() Entering")
	defer log.Trace("config/config:GetSigningKeyFromFile() Leaving")

	return getFileContentFromConfigDir(consts.SigningKeyFileName)
}

func GetBindingKeyFromFile() ([]byte, error) {
	log.Trace("config/config:GetBindingKeyFromFile() Entering")
	defer log.Trace("config/config:GetBindingKeyFromFile() Leaving")

	return getFileContentFromConfigDir(consts.BindingKeyFileName)
}

func GetSigningCertFromFile() (string, error) {
	log.Trace("config/config:GetSigningCertFromFile() Entering")
	defer log.Trace("config/config:GetSigningCertFromFile() Leaving")

	f, err := getFileContentFromConfigDir(consts.SigningKeyPemFileName)
	if err != nil {
		return "", errors.Wrapf(err, "config/config:GetSigningCertFromFile() Error while getting contents of File :%s", consts.SigningKeyPemFileName)
	}
	return string(f), nil
}

// GetAikSecret function returns the AIK Secret as a byte array running the tagent export config command
func GetAikSecret() (string, error) {
	log.Trace("config/config:GetAikSecret() Entering")
	defer log.Trace("config/config:GetAikSecret() Leaving")

	log.Info("config/config:GetAikSecret() Getting AIK secret from trustagent configuration.")
	aikSecret, err := ioutil.ReadFile(consts.DefaultTrustagentConfiguration + "/aiksecretkey")
	if err != nil {
		log.WithError(err).Error("Error while reading from aiksecret key file")
		return "", err
	}
	return string(aikSecret), nil
}

// Save method saves the changes in configuration file made by any of the setup tasks
func Save() error {
	file, err := os.OpenFile(configFilePath, os.O_RDWR, 0)
	defer func() {
		derr := file.Close()
		if derr != nil {
			log.WithError(derr).Error("Error closing file")
		}
	}()
	if err != nil {
		// we have an error
		if os.IsNotExist(err) {
			// error is that the config doesnt yet exist, create it
			log.Debug("config/config:Save() File does not exist, creating a file... ")
			file, err = os.OpenFile(configFilePath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0600)
			if err != nil {
				return errors.Wrap(err, "config/config:Save() Error in file creation")
			}
		} else {
			// someother I/O related error
			return errors.Wrap(err, "config/config:Save() I/O related error")
		}
	}
	return yaml.NewEncoder(file).Encode(Configuration)
}

func init() {
	// load from config
	file, err := os.Open(configFilePath)
	if err == nil {
		defer func() {
			derr := file.Close()
			if derr != nil {
				log.WithError(derr).Error("Error closing file")
			}
		}()
		err = yaml.NewDecoder(file).Decode(&Configuration)
		if err != nil {
			log.WithError(err).Error("Error decoding configuration")
		}
	}
}

// SaveConfiguration is used to save configurations that are provided in environment during setup tasks
// This is called when setup tasks are called
func SaveConfiguration(c csetup.Context, taskName string) error {
	log.Trace("config/config:SaveConfiguration() Entering")
	defer log.Trace("config/config:SaveConfiguration() Leaving")

	var err error

	checkCmsConfig := func(c csetup.Context) error {
		cmsBaseUrl, err := c.GetenvString(consts.CmsBaseUrl, "CMS Base URL")
		if err == nil && cmsBaseUrl != "" {
			Configuration.Cms.BaseURL = cmsBaseUrl
		} else if strings.TrimSpace(Configuration.Cms.BaseURL) == "" {
			return errors.Wrap(err, "CMS_BASE_URL is not defined in environment or configuration file")
		}

		tlsCertDigest, err := c.GetenvString(consts.CmsTlsCertDigestEnv, "CMS TLS certificate digest")
		if err == nil && tlsCertDigest != "" {
			Configuration.CmsTlsCertDigest = tlsCertDigest
		} else if strings.TrimSpace(Configuration.CmsTlsCertDigest) == "" {
			return errors.Wrap(err, "CMS_TLS_CERT_SHA384 is not defined in environment or configuration file")
		}
		return nil
	}

	checkHvsConfig := func(c csetup.Context) error {
		hvsUrl, err := c.GetenvString(consts.HvsUrlEnv, "Verification Service URL")
		if err == nil && hvsUrl != "" {
			Configuration.Hvs.APIURL = hvsUrl
		} else if strings.TrimSpace(Configuration.Hvs.APIURL) == "" {
			return errors.Wrap(err, "HVS_URL is not defined in environment or configuration file")
		}
		return nil
	}

	switch taskName {
	case consts.SetupAllCommand:
		aasAPIUrl, err := c.GetenvString(consts.AasUrl, "AAS API URL")
		if err == nil && aasAPIUrl != "" {
			Configuration.Aas.BaseURL = aasAPIUrl
		} else if strings.TrimSpace(Configuration.Aas.BaseURL) == "" {
			return errors.Wrap(err, "AAS_API_URL is not defined in environment or configuration file")
		}

		// See if the 'TRUSTAGENT_USER' name has been exported to the env.  This is the name
		// of the Linux user that the trust-agent service runs under (and requires access
		// to /etc/workload-agent/bindingkey.pem to serve to hvs).  If the name is not in the
		// environment, assume the default value 'tagent'.
		taUser, err := c.GetenvString(consts.TAUserNameEnvVar, "Trust Agent User Name")
		if err == nil && taUser != "" {
			Configuration.TrustAgent.User = taUser
		} else if strings.TrimSpace(Configuration.TrustAgent.User) == "" {
			log.Infof("TRUSTAGENT_USER is not defined in the environment, using default '%s'", consts.DefaultTrustagentUser)
			Configuration.TrustAgent.User = consts.DefaultTrustagentUser
		}

		taConfigDir, err := c.GetenvString(consts.TAConfigDirEnvVar, "Trust Agent Configuration Directory")
		if err == nil && taConfigDir != "" {
			Configuration.TrustAgent.ConfigDir = taConfigDir
		} else if strings.TrimSpace(Configuration.TrustAgent.ConfigDir) == "" {
			log.Infof("TRUSTAGENT_CONFIGURATION is not defined in the environment, using default '%s'", consts.DefaultTrustagentConfiguration)
			Configuration.TrustAgent.ConfigDir = consts.DefaultTrustagentConfiguration
		}

		ll, err := c.GetenvString(consts.LogLevelEnvVar, "Logging Level")
		if err != nil {
			log.Info("No logging level specified, using default logging level: Info")
			Configuration.LogLevel = logrus.InfoLevel
		} else if Configuration.LogLevel != 0 {
			log.Info("No change in logging level")
		} else {
			Configuration.LogLevel, err = logrus.ParseLevel(ll)
			if err != nil {
				log.Info("Invalid logging level specified, using default logging level: Info")
				Configuration.LogLevel = logrus.InfoLevel
			}
		}

		Configuration.TrustAgent.AikPemFile = filepath.Join(Configuration.TrustAgent.ConfigDir, consts.TAAikPemFileName)

		err = checkCmsConfig(c)
		if err != nil {
			return err
		}
		err = checkHvsConfig(c)
		if err != nil {
			return err
		}

		Configuration.ConfigComplete = true

	case consts.DownloadRootCACertCommand:
		err = checkCmsConfig(c)
		if err != nil {
			return err
		}

	case consts.RegisterBindingKeyCommand, consts.RegisterSigningKeyCommand:
		err = checkHvsConfig(c)
		if err != nil {
			return err
		}
	}

	fmt.Println("Configuration Loaded")
	log.Info("config/config:SaveConfiguration() Saving Environment variables inside the configuration file")
	return Save()

}

// LogConfiguration is used to save log configurations
func LogConfiguration(isStdOut bool) {
	// creating the log file if not preset
	var ioWriterDefault io.Writer
	secLogFile, _ := os.OpenFile(consts.SecurityLogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0640)
	defaultLogFile, _ := os.OpenFile(consts.DefaultLogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0640)

	ioWriterDefault = defaultLogFile

	if isStdOut {
		ioWriterDefault = io.MultiWriter(os.Stdout, ioWriterDefault)
	}

	ioWriterSecurity := io.MultiWriter(ioWriterDefault, secLogFile)
	cLogInt.SetLogger(cLog.DefaultLoggerName, Configuration.LogLevel, &cLog.LogFormatter{MaxLength: Configuration.LogMaxLength}, ioWriterDefault, false)
	cLogInt.SetLogger(cLog.SecurityLoggerName, Configuration.LogLevel, &cLog.LogFormatter{MaxLength: Configuration.LogMaxLength}, ioWriterSecurity, false)
	secLog.Info(message.LogInit)
	log.Info(message.LogInit)
}
