package config

import (
	"crypto"
	"encoding/hex"
	"fmt"
	"intel/isecl/wlagent/consts"
	"intel/isecl/wlagent/osutil"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"
	"fmt"

	csetup "intel/isecl/lib/common/setup"

	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// Configuration is the global configuration struct that is marshalled/unmarshaled to a persisted yaml file
var Configuration struct {
	BindingKeySecret string
	SigningKeySecret string
	
	Mtwilson struct {
		APIURL      string
		APIUsername string
		APIPassword string
		TLSSha256   string
	}
	Wls struct {
		APIURL      string
		APIUsername string
		APIPassword string
		TLSSha256   string
	}
	LogLevel string
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

//WLS vars
const WLS_API_URL = "WLS_API_URL"
const WLS_API_USERNAME = "WLS_API_USERNAME"
const WLS_API_PASSWORD = "WLS_API_PASSWORD"
const WLS_TLS_SHA256 = "WLS_TLS_SHA256"

//TODO - this should be moved elsewhere to some sort of global constants
const hashingAlgorithm crypto.Hash = crypto.SHA256

const workloadAgentConfigDir string = "WORKLOAD_AGENT_CONFIGURATION"
const trustAgentConfigDir string = "TRUST_AGENT_CONFIGURATION"
const taConfigExportCmd string = "tagent export-config --stdout"
const aikSecretKeyName string = "aik.secret"
const bindingKeyFileName string = "bindingkey.json"
const signingKeyFileName string = "signingkey.json"
const bindingKeyPemFileName string = "bindingkey.pem"
const signingKeyPemFileName string = "signingkey.pem"
const imageInstanceCountAssociationFileName string = "image_instance_association"
const devMapperPath string = "/dev/mapper/"
const configFilePath = "workloadagent.env"
const devMapperDirPath = "/dev/mapper/"

var LogFilePath string = os.Getenv("WORKLOAD_AGENT_LOGS") + "/workloadagent.log"

func GetConfigDir() string {
	return workloadAgentConfigDir
}

func GetTrustAgentConfigDir() string {
	return trustAgentConfigDir
}

func ImageInstanceCountAssociationFileName() string {
	return imageInstanceCountAssociationFileName
}

func GetDevMapperDir() string {
	return devMapperDirPath
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

func GetHashingAlgorithm()  crypto.Hash {
	return hashingAlgorithm
}

func GetHashingAlgorithmName() string {
	switch GetHashingAlgorithm() {
	case crypto.SHA256:
		return "SHA-256"
	case crypto.SHA384:
		return "SHA-384"
	}
	return ""
}

func getFileContents(fileName string) ([]byte, error){

	keyFilePath, err := osutil.MakeFilePathFromEnvVariable(GetConfigDir(), fileName, true)
	if err != nil {
		return nil, err
	}

	// check if key file exists
	_, err = os.Stat(keyFilePath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("File does not exist %s", fileName)
	}

	// read contents of file
	file, _ := os.Open(keyFilePath)
	defer file.Close()
	byteValue, _ := ioutil.ReadAll(file)
	return byteValue, nil 
}

func getFileContentFromConfigDir(fileName string) ([]byte, error){
		keyFilePath := "/etc/workloadagent/" + fileName
		// check if key file exists
		_, err := os.Stat(keyFilePath)
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("File does not exist %s", keyFilePath)
		}

		// read contents of file
		file, _ := os.Open(keyFilePath)
		defer file.Close()
		byteValue, _ := ioutil.ReadAll(file)
		return byteValue, nil
}

func GetSigningKeyFromFile() ([]byte, error) {

	return getFileContentFromConfigDir(GetSigningKeyFileName())
}

func GetBindingKeyFromFile() ([]byte, error) {
		
	return getFileContentFromConfigDir(GetBindingKeyFileName())
}

func GetSigningCertFromFile() (string, error){

	f, err := getFileContentFromConfigDir(GetSigningKeyPemFileName())
	if err != nil {
		return "", err
	}
	return string(f), nil 
}

func GetBindingCertFromFile() (string, error){

	f, err := getFileContentFromConfigDir(GetBindingKeyPemFileName())
	if err != nil {
		return "", err
	}
	return string(f), nil 
}

// This function returns the AIK Secret as a byte array running the tagent export config command
func GetAikSecret() ([]byte, error) {
	log.Info("Getting AIK secret from trustagent configuration.")
	tagentConfig, stderr, err := osutil.RunCommandWithTimeout(taConfigExportCmd, 2)
	if err != nil {
		log.Info("Error: GetAikSecret: Command Failed. Details follow")
		log.Infof("Issued Command: \n%s\n", taConfigExportCmd)
		log.Infof("StdOut:\n%s\n", tagentConfig)
		log.Infof("StdError:\n%s\n", stderr)
		return nil, err
	}

	secret, err := osutil.GetMapValueFromConfigFileContent(tagentConfig, aikSecretKeyName)
	if err != nil {
		log.WithFields(log.Fields{
			"Issued Command:": consts.TAConfigAikSecretCmd,
			"StdError:":       stderr,
		}).Error("GetAikSecret: Command Failed. Details follow")
		return nil, err
	}
	return hex.DecodeString(strings.TrimSpace(aikSecret))
}

// Save the configuration struct into configuration directory
func Save() error {
	file, err := os.OpenFile(consts.ConfigFilePath, os.O_RDWR, 0)
	defer file.Close()
	if err != nil {
		// we have an error
		if os.IsNotExist(err) {
			// error is that the config doesnt yet exist, create it
			file, err = os.Create(consts.ConfigFilePath)
			if err != nil {
				return err
			}
		}
	}
	return yaml.NewEncoder(file).Encode(Configuration)
}

var LogWriter io.Writer

func init() {
	// load from config
	file, err := os.Open(consts.ConfigFilePath)
	if err == nil {
		defer file.Close()
		yaml.NewDecoder(file).Decode(&Configuration)
	}
	LogWriter = os.Stdout
}

// SaveConfiguration is used to save configurations that are provided in environment during setup tasks
// This is called when setup tasks are called
func SaveConfiguration(c csetup.Context) error {
	var err error
	Configuration.Mtwilson.APIURL, err = c.GetenvString(consts.MTWILSON_API_URL, "Mtwilson URL")
	if err != nil {
		return err
	}
	Configuration.Mtwilson.APIUsername, err = c.GetenvString(consts.MTWILSON_API_USERNAME, "Mtwilson Username")
	if err != nil {
		return err
	}
	Configuration.Mtwilson.APIPassword, err = c.GetenvString(consts.MTWILSON_API_PASSWORD, "Mtwilson Password")
	if err != nil {
		return err
	}
	Configuration.Mtwilson.TLSSha256, err = c.GetenvString(consts.MTWILSON_TLS_SHA256, "Mtwilson TLS SHA256")
	if err != nil {
		return err
	}
	Configuration.Wls.APIURL, err = c.GetenvString(consts.WLS_API_URL, "Workload Service URL")
	if err != nil {
		return err
	}
	Configuration.Wls.APIUsername, err = c.GetenvString(consts.WLS_API_USERNAME, "Workload Service API Username")
	if err != nil {
		return err
	}
	Configuration.Wls.APIPassword, err = c.GetenvString(consts.WLS_API_PASSWORD, "Workload Service API Password")
	if err != nil {
		return err
	}
	// Configuration.Wls.TLSSha256, err = c.GetenvString(WLS_TLS_SHA256, "Workload Service TLS SHA256")
	// if err != nil {
	// 	return err
	// }
	return Save()
}

// LogConfiguration is used to setup log rotation configurations
func LogConfiguration() {
	var succ bool
	Configuration.LogLevel, succ = os.LookupEnv("LOG_LEVEL")
	if !succ {
		fmt.Printf("Log level configuration variable not set.")
		Configuration.LogLevel = "debug"
	}
	// creating the log file if not preset
	LogFilePath := consts.LogDirPath + consts.LogFileName
	logFile, err := os.OpenFile(LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		fmt.Printf("unable to write file on filehook %v\n", err)
		return
	}
	// parse string, this is built-in feature of logrus
	logLevel, err := log.ParseLevel(Configuration.LogLevel)
	if err != nil {
		logLevel = log.DebugLevel
	}
	// set global log level
	log.SetLevel(logLevel)

	// set formatting of logs
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true, TimestampFormat: time.RFC1123Z})

	// print logs to std out and logfile
	logMultiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(logMultiWriter)
}
