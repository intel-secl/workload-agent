/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package config

import (
	"encoding/hex"
	"fmt"
	"intel/isecl/lib/common/exec"
	cLog "intel/isecl/lib/common/log"
	cLogInt "intel/isecl/lib/common/log/setup"
	csetup "intel/isecl/lib/common/setup"
	"intel/isecl/wlagent/consts"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
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
	Mtwilson struct {
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
	ConfigComplete                  bool
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
func GetAikSecret() ([]byte, error) {
	log.Trace("config/config:GetAikSecret() Entering")
	defer log.Trace("config/config:GetAikSecret() Leaving")

	log.Info("config/config:GetAikSecret() Getting AIK secret from trustagent configuration.")
	aikSecret, stderr, err := exec.RunCommandWithTimeout(consts.TAConfigAikSecretCmd, 2)
	if err != nil {
		log.WithError(&CommandError{IssuedCommand: consts.TAConfigAikSecretCmd, StdError: stderr}).Error("GetAikSecret: Command Failed. Details follow")
		return nil, err
	}
	return hex.DecodeString(strings.TrimSpace(aikSecret))
}

// Save method saves the changes in configuration file made by any of the setup tasks
func Save() error {
	file, err := os.OpenFile(configFilePath, os.O_CREATE|os.O_RDWR, 0)
	defer file.Close()
	if err != nil {
		return errors.Wrapf(err, "config/config:Save() Unable to save into config file %s", configFilePath)
	}
	return yaml.NewEncoder(file).Encode(Configuration)
}

func init() {
	// load from config
	file, err := os.Open(configFilePath)
	if err == nil {
		defer file.Close()
		yaml.NewDecoder(file).Decode(&Configuration)
	}
	LogWriter = os.Stdout
}

// SaveConfiguration is used to save configurations that are provided in environment during setup tasks
// This is called when setup tasks are called
func SaveConfiguration(c csetup.Context) error {
	log.Trace("config/config:SaveConfiguration() Entering")
	defer log.Trace("config/config:SaveConfiguration() Leaving")
	var err error

	//clear the ConfigComplete flag and save the file. We will mark it complete on at the end.
	// we can use the ConfigComplete field to check if the configuration is complete before
	// running the other tasks.
	Configuration.ConfigComplete = false
	err = Save()
	if err != nil {
		return errors.Wrap(err, "config/config:SaveConfiguration() unable to save configuration file")
	}

	// we are going to check and set the required configuration variables
	// however, we do not want to error out after each one. We want to provide
	// entries in the log file indicating which ones are missing. At the
	// end of this section we will error out. Will use a flag to keep track

	requiredConfigsPresent := true

	requiredConfigs := [...]csetup.EnvVars{
		{
			consts.CmsTlsCertDigestEnv,
			&Configuration.CmsTlsCertDigest,
			"CMS TLS Cert SHA384 digest",
			false,
		},
		{
			consts.MTWILSON_API_URL,
			&Configuration.Mtwilson.APIURL,
			"Mtwilson URL",
			false,
		},
		{
			consts.WLS_API_URL,
			&Configuration.Wls.APIURL,
			"Workload Service URL",
			false,
		},
		{
			consts.WLA_USERNAME,
			&Configuration.Wla.APIUsername,
			"Workload Agent Service Username",
			false,
		},
		{
			consts.WLA_PASSWORD,
			&Configuration.Wla.APIPassword,
			"Workload Agent Service Password",
			false,
		},
		{
			consts.TAUserNameEnvVar,
			&Configuration.TrustAgent.User,
			"Trust Agent User Name",
			false,
		},
		{
			consts.TAConfigDirEnvVar,
			&Configuration.TrustAgent.ConfigDir,
			"Trust Agent Configuration Directory",
			false,
		},
		{
			consts.SkipFlavorSignatureVerification,
			&Configuration.SkipFlavorSignatureVerification,
			"Flavor Signature Verification Skip",
			true,
		},
		{
			consts.AAS_URL,
			&Configuration.Aas.BaseURL,
			"AAS URL",
			false,
		},
		{
			consts.CMS_BASE_URL,
			&Configuration.Cms.BaseURL,
			"CMS URL",
			false,
		},
	}

	for _, cv := range requiredConfigs {
		_, _, err = c.OverrideValueFromEnvVar(cv.Name, cv.ConfigVar, cv.Description, cv.EmptyOkay)
		if err != nil {
			requiredConfigsPresent = false
			fmt.Fprintf(os.Stderr, "environment variable %s required - but not set", cv.Name)
			fmt.Fprintln(os.Stderr, err)
		}
	}
	ll, err := c.GetenvString(consts.LogLevelEnvVar, "Logging Level")
	if err != nil {
		fmt.Fprintln(os.Stderr, "No logging level specified, using default logging level: Error")
		Configuration.LogLevel = logrus.ErrorLevel
	}
	Configuration.LogLevel, err = logrus.ParseLevel(ll)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Invalid logging level specified, using default logging level: Error")
		Configuration.LogLevel = logrus.ErrorLevel
	}
	if requiredConfigsPresent {
		Configuration.TrustAgent.AikPemFile = filepath.Join(Configuration.TrustAgent.ConfigDir, consts.TAAikPemFileName)
		Configuration.ConfigComplete = true
		return Save()
	}
	return errors.New("one or more required environment variables for setup not present. log file has details")
}

// LogConfiguration is used to save log configurations
func LogConfiguration(stdOut, logFile, dLogFile bool) {
	// creating the log file if not preset
	var ioWriterDefault io.Writer
	secLogFile, _ := os.OpenFile(consts.SecurityLogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
	defaultLogFile, _ := os.OpenFile(consts.DefaultLogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
	daemonLogFile, _ := os.OpenFile(consts.DaemonLogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)

	ioWriterDefault = defaultLogFile
	if stdOut {
		ioWriterDefault = os.Stdout
	}

	if stdOut && logFile {
		ioWriterDefault = io.MultiWriter(os.Stdout, defaultLogFile)
	}

	if dLogFile {
		ioWriterDefault = daemonLogFile
	}
	ioWriterSecurity := io.MultiWriter(ioWriterDefault, secLogFile)

	cLogInt.SetLogger(cLog.DefaultLoggerName, Configuration.LogLevel, nil, ioWriterDefault, false)
	cLogInt.SetLogger(cLog.SecurityLoggerName, Configuration.LogLevel, nil, ioWriterSecurity, false)
	secLog.Trace("Security log initiated")
	log.Trace("Loggers setup finished")
}
