package config

import (
	"encoding/hex"
	"intel/isecl/wlagent/osutil"
	"io"
	"os"
	"strconv"
	"time"

	csetup "intel/isecl/lib/common/setup"

	log "github.com/sirupsen/logrus"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
	yaml "gopkg.in/yaml.v2"
)

// Configuration is the global configuration struct that is marshalled/unmarshaled to a persisted yaml file
var Configuration struct {
	BindingKeySecret string
	Mtwilson         struct {
		APIURL      string
		APIUsername string
		APIPassword string
		TLSSha256   string
	}
	Wls struct {
		APIURL      string
		APIUsername string
		APIPassword string
		TlsSha256   string
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

//WLS vars
const WLS_API_URL = "WLS_API_URL"
const WLS_API_USERNAME = "WLS_API_USERNAME"
const WLS_API_PASSWORD = "WLS_API_PASSWORD"
const WLS_TLS_SHA256 = "WLS_TLS_SHA256"

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

// This function returns the AIK Secret as a byte array running the tagent export config command
func GetAikSecret() ([]byte, error) {
	log.Info("Getting AIK secret from trustagent configuration.")
	// Change to `tagent config aik.secret`
	tagentConfig, stderr, err := osutil.RunCommandWithTimeout(taConfigExportCmd, 2)
	if err != nil {
		log.Info("Error: GetAikSecret: Command Failed. Details follow")
		log.Info("Issued Command: \n%s\n", taConfigExportCmd)
		log.Info("StdOut:\n%s\n", tagentConfig)
		log.Info("StdError:\n%s\n", stderr)
		return nil, err
	}

	secret, err := osutil.GetValueFromEnvBody(tagentConfig, aikSecretKeyName)
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(secret)
}

var LogWriter io.Writer

// Save the configuration struct into configuration directory
func Save() error {
	file, err := os.OpenFile("/etc/workloadagent/config.yml", os.O_RDWR, 0)
	if err != nil {
		// we have an error
		if os.IsNotExist(err) {
			// error is that the config doesnt yet exist, create it
			file, err = os.Create("/etc/workloadagent/config.yml")
			if err != nil {
				return err
			}
		}
	}
	defer file.Close()
	return yaml.NewEncoder(file).Encode(Configuration)
}

func init() {
	// load from config
	file, err := os.Open("/etc/workloadagent/config.yml")
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
	Configuration.Mtwilson.APIURL, err = c.GetenvString(MTWILSON_API_URL, "Mtwilson URL")
	if err != nil {
		return err
	}
	Configuration.Mtwilson.APIUsername, err = c.GetenvString(MTWILSON_API_USERNAME, "Mtwilson Username")
	if err != nil {
		return err
	}
	Configuration.Mtwilson.APIPassword, err = c.GetenvString(MTWILSON_API_PASSWORD, "Mtwilson Password")
	if err != nil {
		return err
	}
	Configuration.Mtwilson.TLSSha256, err = c.GetenvString(MTWILSON_TLS_SHA256, "Mtwilson TLS SHA256")
	if err != nil {
		return err
	}
	Configuration.Wls.APIURL, err = c.GetenvString(WLS_API_URL, "Workload Service URL")
	if err != nil {
		return err
	}
	Configuration.Wls.APIUsername, err = c.GetenvString(WLS_API_USERNAME, "Workload Service API Username")
	if err != nil {
		return err
	}
	Configuration.Wls.APIPassword, err = c.GetenvString(WLS_API_PASSWORD, "Workload Service API Password")
	if err != nil {
		return err
	}
	// Configuration.Wls.TlsSha256, err = c.GetenvString(WLS_TLS_SHA256, "Workload Service TLS SHA256")
	// if err != nil {
	// 	return err
	// }
	return Save()
}

// SaveSetupConfiguration is used to save configurations that are provided during setup
// for example logger rotate configurations
func SaveSetupConfiguration() error {
	// Don't need rotation anymore
	Configuration.LogRotate.MaxBackups, _ = strconv.Atoi(os.Getenv("LOG_ROTATE_MAX_BACKUPS"))
	Configuration.LogRotate.MaxRotateSize, _ = strconv.Atoi(os.Getenv("LOG_ROTATE_MAX_SIZE"))
	Configuration.LogRotate.MaxDays, _ = strconv.Atoi(os.Getenv("LOG_ROTATE_MAX_DAYS"))
	return Save()
}

// LogConfiguration is used to setup log rotation configurations
func LogConfiguration() {
	// Remove log rotation, only use Logrus package.
	lumberjackLogrotate := &lumberjack.Logger{
		Filename:   LogFilePath,
		MaxSize:    Configuration.LogRotate.MaxRotateSize,
		MaxBackups: Configuration.LogRotate.MaxBackups,
		MaxAge:     Configuration.LogRotate.MaxDays,
		Compress:   true,
	}

	log.SetFormatter(&log.TextFormatter{FullTimestamp: true, TimestampFormat: time.RFC1123Z})
	logMultiWriter := io.MultiWriter(os.Stdout, lumberjackLogrotate)
	log.SetOutput(logMultiWriter)
}
