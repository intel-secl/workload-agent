package wlaconfig

import (
	"encoding/hex"
	"fmt"
	"log"

	"intel/isecl/wlagent/osutil"
)

// WlaConfig is to be used for storing configuration of workloadagent
type WlaConfig struct {
	WlsUserName string
	WlsUserPass string
	WlsShaSize  int
	WlsTlsSha   string
}

const workloadAgentConfigDir string = "WORKLOAD_AGENT_CONFIGURATION"
const taConfigExportCmd string = "tagent export-config --stdout"
const aikSecretKeyName string = "aik.secret"
const bindingKeyFileName string = "bindingkey.json"
const signingKeyFileName string = "signingkey.json"

func GetConfigDir() string {
	return workloadAgentConfigDir
}

func GetBindingKeyFileName() string {
	return bindingKeyFileName
}

func GetSigningKeyFileName() string {
	return signingKeyFileName
}

// This function returns the AIK Secret as a byte array running the tagent export config command
func GetAikSecret() ([]byte, error) {
	tagentConfig, stderr, err := osutil.RunCommandWithTimeout(taConfigExportCmd, 2)
	if err != nil {
		log.Println("Error: GetAikSecret: Command Failed. Details follow")
		log.Printf("Issued Command: \n%s\n", taConfigExportCmd)
		log.Printf("StdOut:\n%s\n", tagentConfig)
		log.Printf("StdError:\n%s\n", stderr)
		return nil, err
	}

	// log.Printf("Debug: Trust Agent Config: \n%s\n", tagentConfig)
	secret, err := osutil.GetMapValueFromConfigFileContent(tagentConfig, aikSecretKeyName)
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(secret)

}

// LoadConfig loads the configuration from the configuration file
// workloadagent.properties under the configuration directory
// TODO: Should read this as an encrypted file. For now, we are going
// to use a plain file. Need to sort out requirements around encrypting
// this file. This process runs under the context of the launching user
// So will probably need to set ownership of this file appropriately
func LoadConfig() error {

	return fmt.Errorf("LoadConfig Method - not yet implemented")
}
