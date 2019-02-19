package config

import (
	"encoding/hex"
	"fmt"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
	"intel/isecl/lib/common/exec"
	csetup "intel/isecl/lib/common/setup"
	"intel/isecl/wlagent/consts"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"
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

const HashingAlgorithm crypto.Hash = crypto.SHA256

func getFileContentFromConfigDir(fileName string) ([]byte, error) {
	filePath := consts.ConfigDirPath + fileName
	// check if key file exists
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("File does not exist - %s", filePath)
	}

	// read contents of file
	file, _ := os.Open(filePath)
	defer file.Close()
	byteValue, _ := ioutil.ReadAll(file)
	return byteValue, nil
}

func GetSigningKeyFromFile() ([]byte, error) {

	return getFileContentFromConfigDir(GetSigningKeyFileName())
}

func GetBindingKeyFromFile() ([]byte, error) {

	return getFileContentFromConfigDir(consts.BindingKeyFileName)
}

func GetSigningCertFromFile() (string, error) {

	f, err := getFileContentFromConfigDir(GetSigningKeyPemFileName())
	if err != nil {
		return "", err
	}
	return string(f), nil
}

func GetBindingCertFromFile() (string, error) {

	f, err := getFileContentFromConfigDir(GetBindingKeyPemFileName())
	if err != nil {
		return "", err
	}
	return string(f), nil
}

// GetAikSecret function returns the AIK Secret as a byte array running the tagent export config command
func GetAikSecret() ([]byte, error) {
	log.Info("Getting AIK secret from trustagent configuration.")
	aikSecret, stderr, err := exec.RunCommandWithTimeout(consts.TAConfigAikSecretCmd, 2)
	if err != nil {
		log.WithFields(log.Fields{
			"Issued Command:": consts.TAConfigAikSecretCmd,
			"StdError:":       stderr,
		}).Error("GetAikSecret: Command Failed. Details follow")
		return nil, err
	}
	return hex.DecodeString(strings.TrimSpace(aikSecret))
}

func Save() error {
	file, err := os.OpenFile(configFilePath, os.O_CREATE|os.O_RDWR, 0)
	defer file.Close()
	if err != nil {
		return err
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

// LogConfiguration is used to save log configurations
func LogConfiguration(logFilePath string) {
	var succ bool
	Configuration.LogLevel, succ = os.LookupEnv("LOG_LEVEL")
	if !succ {
		fmt.Printf("Log level configuration variable not set.")
		Configuration.LogLevel = "debug"
	}
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModeAppend)
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
