package config

import (
	"encoding/hex"
	"fmt"
	"intel/isecl/lib/common/exec"
	csetup "intel/isecl/lib/common/setup"
	"intel/isecl/wlagent/consts"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

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
		TLSSha384   string
	}
	Wls struct {
		APIURL      string
		APIUsername string
		APIPassword string
		TLSSha384   string
	}
	TrustAgent struct {
		ConfigDir  string
		AikPemFile string
		User       string
	}
	Aas struct {
		BaseURL string
	}
	FlavorSignatureVerificationSkip bool
	LogLevel                        string
	ConfigComplete                  bool
}

const HashingAlgorithm crypto.Hash = crypto.SHA256

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

	//clear the ConfigComplete flag and save the file. We will mark it complete on at the end.
	// we can use the ConfigComplete field to check if the configuration is complete before
	// running the other tasks.
	Configuration.ConfigComplete = false
	err = Save()
	if err != nil {
		return fmt.Errorf("unable to save configuration file")
	}

	// we are going to check and set the required configuration variables
	// however, we do not want to error out after each one. We want to provide
	// entries in the log file indicating which ones are missing. At the
	// end of this section we will error out. Will use a flag to keep track

	requiredConfigsPresent := true

	requiredConfigs := [...]csetup.EnvVars{
		{
			consts.MTWILSON_API_URL,
			&Configuration.Mtwilson.APIURL,
			"Mtwilson URL",
			false,
		},
		{
			consts.MTWILSON_API_USERNAME,
			&Configuration.Mtwilson.APIUsername,
			"Mtwilson Username",
			false,
		},
		{
			consts.MTWILSON_API_PASSWORD,
			&Configuration.Mtwilson.APIPassword,
			"Mtwilson Password",
			false,
		},
		{
			consts.MTWILSON_TLS_SHA384,
			&Configuration.Mtwilson.TLSSha384,
			"Mtwilson TLS SHA384",
			false,
		},
		{
			consts.WLS_API_URL,
			&Configuration.Wls.APIURL,
			"Workload Service URL",
			false,
		},
		{
			consts.WLS_API_USERNAME,
			&Configuration.Wls.APIUsername,
			"Workload Service API Username",
			false,
		},
		{
			consts.WLS_API_PASSWORD,
			&Configuration.Wls.APIPassword,
			"Workload Service API Password",
			false,
		},
		//{
		//	consts.WLS_TLS_SHA384,
		//	&Configuration.Wls.TLSSha384,
		//	"Workload Service TLS SHA384",
		//	false,
		//},
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
			consts.LogLevelEnvVar,
			&Configuration.LogLevel,
			"Log Level",
			false,
		},
		{
			consts.FlavorSignatureVerificationSkip,
			&Configuration.FlavorSignatureVerificationSkip,
			"Flavor Signature Verification Skip",
			true,
		},
		{	consts.AAS_URL,
			&Configuration.Aas.BaseURL,
			"AAS URL",
			false,
		},
	}

	for _, cv := range requiredConfigs {
		_, _, err = c.OverrideValueFromEnvVar(cv.Name, cv.ConfigVar, cv.Description, cv.EmptyOkay)
		if err != nil {
			requiredConfigsPresent = false
			log.Errorf("environment variable %s required - but not set", cv.Name)
		}
	}

	if requiredConfigsPresent {
		Configuration.TrustAgent.AikPemFile = filepath.Join(Configuration.TrustAgent.ConfigDir, consts.TAAikPemFileName)
		Configuration.ConfigComplete = true
		return Save()
	}
	return fmt.Errorf("one or more required environment variables for setup not present. log file has details")
}

// LogConfiguration is used to save log configurations
func LogConfiguration(logFilePath string) {
	// creating the log file if not preset
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		fmt.Printf("unable to write file on filehook %v\n", err)
		return
	}
	// parse string, this is built-in feature of logrus
	logLevel, err := log.ParseLevel(Configuration.LogLevel)
	if err != nil {
		logLevel = log.InfoLevel
	}
	// set global log level
	log.SetLevel(logLevel)

	// set formatting of logs
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true, TimestampFormat: time.RFC1123Z})

	// print logs to std out and logfile
	logWriter := io.Writer(logFile)
	log.SetOutput(logWriter)
}
