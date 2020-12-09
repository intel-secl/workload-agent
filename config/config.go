/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package config

import (
	"fmt"
	"intel/isecl/lib/common/v3/exec"
	cLog "intel/isecl/lib/common/v3/log"
	"intel/isecl/lib/common/v3/log/message"
	cLogInt "intel/isecl/lib/common/v3/log/setup"
	csetup "intel/isecl/lib/common/v3/setup"
	"intel/isecl/wlagent/v3/consts"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
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
	configFilePath string = consts.ConfigDirPath + consts.ConfigFileName
	LogWriter      io.Writer
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

func GetBindingCertFromFile() (string, error) {
	log.Trace("config/config:GetBindingCertFromFile() Entering")
	defer log.Trace("config/config:GetBindingCertFromFile() Leaving")

	f, err := getFileContentFromConfigDir(consts.BindingKeyPemFileName)
	if err != nil {
		return "", errors.Wrapf(err, "config/config:BindingKeyPemFileName() Error while getting contents of File :%s", consts.BindingKeyPemFileName)
	}
	return string(f), nil
}

type CommandError struct {
	IssuedCommand string
	StdError      string
}

func (e CommandError) Error() string {
	return fmt.Sprintf("config/config Command Error %s: %s", e.IssuedCommand, e.StdError)
}

// GetAikSecret function returns the AIK Secret as a byte array running the tagent export config command
func GetAikSecret() (string, error) {
	log.Trace("config/config:GetAikSecret() Entering")
	defer log.Trace("config/config:GetAikSecret() Leaving")

	log.Info("config/config:GetAikSecret() Getting AIK secret from trustagent configuration.")
	aikSecret, stderr, err := exec.RunCommandWithTimeout(consts.TAConfigAikSecretCmd, 2)
	if err != nil {
		log.WithError(&CommandError{IssuedCommand: consts.TAConfigAikSecretCmd, StdError: stderr}).Error("GetAikSecret: Command Failed. Details follow")
		return "", err
	}
	return strings.TrimSpace(aikSecret), nil
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
	LogWriter = os.Stdout
}

// SaveConfiguration is used to save configurations that are provided in environment during setup tasks
// This is called when setup tasks are called
func SaveConfiguration(c csetup.Context) error {
	log.Trace("config/config:SaveConfiguration() Entering")
	defer log.Trace("config/config:SaveConfiguration() Leaving")
	var err error

	tlsCertDigest, err := c.GetenvString(consts.CmsTlsCertDigestEnv, "CMS TLS certificate digest")
	if err == nil && tlsCertDigest != "" {
		Configuration.CmsTlsCertDigest = tlsCertDigest
	} else if strings.TrimSpace(Configuration.CmsTlsCertDigest) == "" {
		return errors.Wrap(err, "CMS_TLS_CERT_SHA384 is not defined in environment or configuration file")
	}

	cmsBaseUrl, err := c.GetenvString(consts.CMS_BASE_URL, "CMS Base URL")
	if err == nil && cmsBaseUrl != "" {
		Configuration.Cms.BaseURL = cmsBaseUrl
	} else if strings.TrimSpace(Configuration.Cms.BaseURL) == "" {
		return errors.Wrap(err, "CMS_BASE_URL is not defined in environment or configuration file")
	}

	aasAPIUrl, err := c.GetenvString(consts.AAS_URL, "AAS API URL")
	if err == nil && aasAPIUrl != "" {
		Configuration.Aas.BaseURL = aasAPIUrl
	} else if strings.TrimSpace(Configuration.Aas.BaseURL) == "" {
		return errors.Wrap(err, "AAS_API_URL is not defined in environment or configuration file")
	}

	wlsAPIUrl, err := c.GetenvString(consts.WLS_API_URL, "Workload Service URL")
	if err == nil && aasAPIUrl != "" {
		Configuration.Wls.APIURL = wlsAPIUrl
	} else if strings.TrimSpace(Configuration.Wls.APIURL) == "" {
		return errors.Wrap(err, "WLS_API_URL is not defined in environment or configuration file")
	}

	hvsUrl, err := c.GetenvString(consts.HVS_URL, "Verification Service URL")
	if err == nil && hvsUrl != "" {
		Configuration.Hvs.APIURL = hvsUrl
	} else if strings.TrimSpace(Configuration.Hvs.APIURL) == "" {
		return errors.Wrap(err, "HVS_URL is not defined in environment or configuration file")
	}

	wlaAASUser, err := c.GetenvString(consts.WLA_USERNAME, "WLA Service Username")
	if err == nil && wlaAASUser != "" {
		Configuration.Wla.APIUsername = wlaAASUser
	} else if Configuration.Wla.APIUsername == "" {
		return errors.Wrap(err, "WLA_SERVICE_USERNAME is not defined in environment or configuration file")
	}

	wlaAASPassword, err := c.GetenvSecret(consts.WLA_PASSWORD, "WLA Service Password")
	if err == nil && wlaAASPassword != "" {
		Configuration.Wla.APIPassword = wlaAASPassword
	} else if strings.TrimSpace(Configuration.Wla.APIPassword) == "" {
		return errors.Wrap(err, "WLA_SERVICE_PASSWORD is not defined in environment or configuration file")
	}

	// See if the 'TRUSTAGENT_USER' name has been exported to the env.  This is the name
	// of the Linux user that the trust-agent service runs under (and requires access
	// to /etc/workload-agent/bindingkey.pem to serve to hvs).  If the name is not in the
	// environment, assume the default value 'tagent'.
	taUser, err := c.GetenvString(consts.TAUserNameEnvVar, "Trust Agent User Name")
	if err == nil && taUser != "" {
		Configuration.TrustAgent.User = taUser
	} else if strings.TrimSpace(Configuration.TrustAgent.User) == "" {
		log.Info("TRUSTAGENT_USER is not defined in the environment, using default '%s'", consts.DEFAULT_TRUSTAGENT_USER)
		Configuration.TrustAgent.User = consts.DEFAULT_TRUSTAGENT_USER
	}

	taConfigDir, err := c.GetenvString(consts.TAConfigDirEnvVar, "Trust Agent Configuration Directory")
	if err == nil && taConfigDir != "" {
		Configuration.TrustAgent.ConfigDir = taConfigDir
	} else if strings.TrimSpace(Configuration.TrustAgent.ConfigDir) == "" {
		log.Info("TRUSTAGENT_CONFIGURATION is not defined in the environment, using default '%s'", consts.DEFAULT_TRUSTAGENT_CONFIGURATION)
		Configuration.TrustAgent.ConfigDir = consts.DEFAULT_TRUSTAGENT_CONFIGURATION
	}

	if skipFlavorSignatureVerification, err := c.GetenvString(consts.SkipFlavorSignatureVerification,
		"Skip flavor signature verification"); err == nil {
		Configuration.SkipFlavorSignatureVerification, err = strconv.ParseBool(skipFlavorSignatureVerification)
		if err != nil {
			log.Warn("SKIP_FLAVOR_SIGNATURE_VERIFICATION is set to invalid value (should be true/false). " +
				"Setting it to true by default")
			Configuration.SkipFlavorSignatureVerification = true
		}
	} else {
		log.Info("SKIP_FLAVOR_SIGNATURE_VERIFICATION is not set. Setting it to true by default")
		Configuration.SkipFlavorSignatureVerification = true
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

	logEntryMaxLength, err := c.GetenvInt(consts.LogEntryMaxlengthEnv, "Maximum length of each entry in a log")
	if err == nil && logEntryMaxLength >= 100 {
		Configuration.LogMaxLength = logEntryMaxLength
	} else if Configuration.LogMaxLength != 0 {
		log.Info("No change in Log Entry Max Length")
	} else {
		log.Info("Invalid Log Entry Max Length defined (should be > 100), using default value:", consts.DefaultLogEntryMaxlength)
		Configuration.LogMaxLength = consts.DefaultLogEntryMaxlength
	}

	Configuration.LogEnableStdout = false
	logEnableStdout, err := c.GetenvString("WLA_ENABLE_CONSOLE_LOG", "Workload Agent Enable standard output")
	if err == nil && logEnableStdout != "" {
		Configuration.LogEnableStdout, err = strconv.ParseBool(logEnableStdout)
		if err != nil {
			log.Info("Error while parsing the variable WLA_ENABLE_CONSOLE_LOG, setting to default value false")
		}
	}

	Configuration.TrustAgent.AikPemFile = filepath.Join(Configuration.TrustAgent.ConfigDir, consts.TAAikPemFileName)
	Configuration.ConfigComplete = true
	fmt.Println("Configuration Loaded")
	log.Info("config/config:SaveConfiguration() Saving Environment variables inside the configuration file")
	return Save()

}

// LogConfiguration is used to save log configurations
func LogConfiguration(isStdOut bool) {
	// creating the log file if not preset
	var ioWriterDefault io.Writer
	secLogFile, _ := os.OpenFile(consts.SecurityLogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	defaultLogFile, _ := os.OpenFile(consts.DefaultLogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)

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
