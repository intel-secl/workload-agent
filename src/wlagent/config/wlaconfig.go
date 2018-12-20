package wlagent

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// WlaConfig is to be used for storing the configuration of workloadagent
type WlaConfig struct {
	WlsUserName string
	WlsUserPass string
	WlsShaSize  int
	WlsTlsSha   string
}

const workloadAgentConfigDir string = "WORKLOAD_AGENT_CONFIGURATION"
const bindingKeyFileName string = "bindingkey.json"
const signingKeyFileName string = "signingkey.json"

func getConfigDIr() string {
	return workloadAgentConfigDir
}

func getBindingKeyFileName() string {
	return bindingKeyFileName
}

func getSigningKeyFileName() string {
	return signingKeyFileName
}

// RunCommandWithTimeout takes a command line and returs the stdout and stderr output
// If command does not terminate within 'timeout', it returns an error
//Todo : vcheeram : Move this to a common library. Keeping as exported for now
func RunCommandWithTimeout(commandLine string, timeout int) (stdout, stderr string, err error) {

	// Create a new context and add a timeout to it
	// log.Println(commandLine)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel() // The cancel should be deferred so resources are cleaned up

	r := csv.NewReader(strings.NewReader(commandLine))
	r.Comma = ' '
	records, err := r.Read()
	if records == nil {
		return "", "", fmt.Errorf("No command to execute - commandLine - %s", commandLine)
	}

	var cmd *exec.Cmd
	if len(records) > 1 {
		cmd = exec.CommandContext(ctx, records[0], records[1:]...)
	} else {
		cmd = exec.CommandContext(ctx, records[0])
	}

	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	err = cmd.Run()
	stdout = outb.String()
	stderr = errb.String()

	return stdout, stderr, err

}

// MakeFilePathFromEnvVariable creates a filepath given an environment variable and the filename
// createDir will create a directory if one does not exist
func MakeFilePathFromEnvVariable(dirEnvVar, filename string, createDir bool) (string, error) {

	if filename == "" {
		return "", fmt.Errorf("Filename is empty")
	}
	dir := os.Getenv(dirEnvVar)
	if dir == "" {
		return "", fmt.Errorf("Environment variable %s not set", dirEnvVar)
	}
	dir = strings.TrimSpace(dir)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return "", fmt.Errorf("Directory %s does not exist", dir)
	}

	return filepath.Join(dir, filename), nil

}

// GetMapValueFromConfigFileContent return the value of a key from a config/environment
// file content. We are passing the contents of a file here and not the filename
// The type of file is a env file where the format is line seperated 'key=value'
// Todo : vcheeram : Move this to a common library. Keeping as exported for now
// Todo: vcheeram: this needs to be converted to some sort of io.reader instead
//  passing the string.
func GetMapValueFromConfigFileContent(fileContent, keyName string) (value string, err error) {
	if strings.TrimSpace(fileContent) == "" || strings.TrimSpace(keyName) == "" {
		return "", errors.New("FileContent and KeyName cannot be empty")
	}
	//the config file should have the keyname as part of the beginning of line
	r := regexp.MustCompile(`(?im)^` + keyName + `\s*=\s*(.*)`)
	rs := r.FindStringSubmatch(fileContent)
	if rs != nil {
		return rs[1], nil
	}
	return "", fmt.Errorf("Could not find Value for %s", keyName)
}

// This function returns the AIK Secret as a byte array running the tagent export config command
func getAikSecret() ([]byte, error) {
	tagentConfig, stderr, err := RunCommandWithTimeout(taConfigExportCmd, 2)
	if err != nil {
		log.Println("Error: getAikSecret: Command Failed. Details follow")
		log.Printf("Issued Command: \n%s\n", taConfigExportCmd)
		log.Printf("StdOut:\n%s\n", tagentConfig)
		log.Printf("StdError:\n%s\n", stderr)
		return nil, err
	}

	// log.Printf("Debug: Trust Agent Config: \n%s\n", tagentConfig)
	secret, err := GetMapValueFromConfigFileContent(tagentConfig, aikSecretKeyName)
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

	return nil
}

// GetHexRandomString return a random string of 'length'
func GetHexRandomString(length int) (string, error) {

	bytes, err := GetRandomBytes(length)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

// GetRandomBytes retrieves a byte array of 'length'
func GetRandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	return bytes, nil
}
