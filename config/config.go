package config

import (
	"encoding/hex"
	"fmt"
	"intel/isecl/wlagent/osutil"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	csetup "intel/isecl/lib/common/setup"

	log "github.com/sirupsen/logrus"
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
		TLSSha256   string
	}
}

// Define constants to be accessed in other packages
const (
	MTWILSON_API_URL                      = "MTWILSON_API_URL"
	MTWILSON_API_USERNAME                 = "MTWILSON_API_USERNAME"
	MTWILSON_API_PASSWORD                 = "MTWILSON_API_PASSWORD"
	MTWILSON_TLS_SHA256                   = "MTWILSON_TLS_SHA256"
	WLS_API_URL                           = "WLS_API_URL"
	WLS_API_USERNAME                      = "WLS_API_USERNAME"
	WLS_API_PASSWORD                      = "WLS_API_PASSWORD"
	WLS_TLS_SHA256                        = "WLS_TLS_SHA256"
	aikSecretKeyName                      = "aik.secret"
	TrustAgentConfigDirEnv                = "TRUST_AGENT_CONFIGURATION"
	taConfigAikSecretCmd                  = "tagent config aik.secret"
	BindingKeyFileName                    = "bindingkey.json"
	SigningKeyFileName                    = "signingkey.json"
	BindingKeyPemFileName                 = "bindingkey.pem"
	SigningKeyPemFileName                 = "signingkey.pem"
	ImageInstanceCountAssociationFileName = "image_instance_association"
	EnvFileName                           = "workloadagent.env"
	DevMapperDirPath                      = "/dev/mapper/"
	LogDirPath                            = "/var/log/workloadagent/"
	LogFileName                           = "workloadagent.log"
	ConfigFilePath                        = "/etc/workloadagent/config.yml"
	ConfigDirPath                         = "/etc/workloadagent/"
	OptDirPath                            = "/opt/workloadagent/"
	LibvirtHookFilePath                   = "/etc/libvirt/hooks/qemu"
)

// GetAikSecret returns the AIK Secret as a byte array running the tagent config command
func GetAikSecret() ([]byte, error) {
	log.Info("Getting AIK secret from trustagent configuration.")
	aikSecret, stderr, err := osutil.RunCommandWithTimeout(taConfigAikSecretCmd, 2)
	if err != nil {
		log.WithFields(log.Fields{
			"Issued Command:": taConfigAikSecretCmd,
			"StdOut:":         aikSecret,
			"StdError:":       stderr,
		}).Error("GetAikSecret: Command Failed. Details follow")
		return nil, err
	}
	return hex.DecodeString(strings.TrimSpace(aikSecret))
}

// Save the configuration struct into configuration directory
func Save() error {
	file, err := os.OpenFile(ConfigFilePath, os.O_RDWR, 0)
	if err != nil {
		// we have an error
		if os.IsNotExist(err) {
			// error is that the config doesnt yet exist, create it
			file, err = os.Create(ConfigFilePath)
			if err != nil {
				return err
			}
		}
	}
	defer file.Close()
	return yaml.NewEncoder(file).Encode(Configuration)
}

var LogWriter io.Writer

func init() {
	// load from config
	file, err := os.Open(ConfigFilePath)
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
	// Configuration.Wls.TLSSha256, err = c.GetenvString(WLS_TLS_SHA256, "Workload Service TLS SHA256")
	// if err != nil {
	// 	return err
	// }
	return Save()
}

// LogConfiguration is used to setup log rotation configurations
func LogConfiguration() {
	// creating the log file if not preset
	LogFilePath := LogDirPath + LogFileName
	_, err := os.Stat(LogFilePath)
	if os.IsNotExist(err) {
		fmt.Println("Log file does not exist. Creating the file.")
		_, touchErr := exec.Command("touch", LogFilePath).Output()
		if touchErr != nil {
			fmt.Println("Error while creating the log file.", touchErr)
			return
		}
	}
	logFile, err := os.OpenFile(LogFilePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		fmt.Printf("unable to write file on filehook %v\n", err)
		return
	}
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true, TimestampFormat: time.RFC1123Z})
	logMultiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(logMultiWriter)
}
