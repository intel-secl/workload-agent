package config

import (
	"encoding/hex"
	"intel/isecl/wlagent/osutil"
	"io"
	"os"

	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// Configuration is the global configuration struct that is marshalled/unmarshaled to a persisted yaml file
var Configuration struct {
	Mtwilson struct {
		APIURL      string
		APIUsername string
		APIPassword string
		TLSSha256   string
	}
	Wls struct {
		UserName string
		UserPass string
		ShaSize  int
		TlsSha   string
	}
	LogRotate struct {
		MaxRotateSize int // maximum megabytes before log is rotated
		MaxDays       int // maximum number of old log files to keep
		MaxBackups    int // maximum number of days to retain log files
	}
}

// MTWILSON_API_URL is a string environment variable for specifying the
// mtwilson API URL and is used to  connect to mtwilson
const MTWILSON_API_URL = "MTWILSON_API_URL"

// MTWILSON_API_USERNAME is a string environment variable for specifying the
// mtwilson API URL and is used to connect to mtwilson
const MTWILSON_API_USERNAME = "MTWILSON_API_USERNAME"

// MTWILSON_API_PASSWORD is a string environment variable for specifying
// the mtwilson API password and is used to connect to mtwilson
const MTWILSON_API_PASSWORD = "MTWILSON_API_PASSWORD"

// MTWILSON_TLS_SHA256 is a string environment variable for specifying
// the mtwilson TLS sha256 and is used to connect to mtwilson
const MTWILSON_TLS_SHA256 = "MTWILSON_TLS_SHA256"

const workloadAgentConfigDir string = "WORKLOAD_AGENT_CONFIGURATION"
const trustAgentConfigDir string = "TRUST_AGENT_CONFIGURATION"
const taConfigExportCmd string = "tagent export-config --stdout"
const aikSecretKeyName string = "aik.secret"
const bindingKeyFileName string = "bindingkey.json"
const signingKeyFileName string = "signingkey.json"
const bindingKeyPemFileName string = "bindingkey.pem"
const signingKeyPemFileName string = "signingkey.pem"
const numberOfInstancesPerImageFileName string = "no_of_instances_per_image"
const devMapperPath string = "/dev/mapper/"
const configFilePath = "workloadagent.env"

var LogFilePath string = os.Getenv("WORKLOAD_AGENT_LOGS") + "/workloadagent.log"

func GetConfigDir() string {
	return workloadAgentConfigDir
}

func GetTrustAgentConfigDir() string {
	return trustAgentConfigDir
}

func GetNumberOfInstancesPerImageFileName() string {
	return numberOfInstancesPerImageFileName
}

func GetDevMapperDir() string {
	return devMapperPath
}

func GetBindingKeyFileName() string {
	return bindingKeyFileName
}

func GetSigningKeyFileName() string {
	return signingKeyFileName
}

func GetBindingKeyPemFileName() string {
	return bindingKeyPemFileName
}

func GetSigningKeyPemFileName() string {
	return signingKeyPemFileName
}

// This function returns the AIK Secret as a byte array running the tagent export config command
func GetAikSecret() ([]byte, error) {
	log.Info("Getting AIK secret from trustagent configuration.")
	tagentConfig, stderr, err := osutil.RunCommandWithTimeout(taConfigExportCmd, 2)
	if err != nil {
		log.Info("Error: GetAikSecret: Command Failed. Details follow")
		log.Info("Issued Command: \n%s\n", taConfigExportCmd)
		log.Info("StdOut:\n%s\n", tagentConfig)
		log.Info("StdError:\n%s\n", stderr)
		return nil, err
	}

	secret, err := osutil.GetMapValueFromConfigFileContent(tagentConfig, aikSecretKeyName)
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(secret)
}

var LogWriter io.Writer
var configYamlFile = os.Getenv(workloadAgentConfigDir) + "/config.yml"

// Save the configuration struct into configuration directory
func Save() error {
	file, err := os.OpenFile(configYamlFile, os.O_RDWR, 0)
	if err != nil {
		// we have an error
		if os.IsNotExist(err) {
			// error is that the config doesnt yet exist, create it
			file, err = os.Create(configYamlFile)
			if err != nil {
				return err
			}
		} else {
			// someother I/O related error
			return err
		}
	}
	defer file.Close()
	return yaml.NewEncoder(file).Encode(Configuration)
}

func init() {
	// load from config
	file, err := os.Open(configYamlFile)
	if err == nil {
		defer file.Close()
		yaml.NewDecoder(file).Decode(&Configuration)
	}
	LogWriter = os.Stdout
}
