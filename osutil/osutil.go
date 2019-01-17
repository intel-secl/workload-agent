package osutil

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

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

// GetValueFromEnvBody return the value of a key from a config/environment
// file content. We are passing the contents of a file here and not the filename
// The type of file is a env file where the format is line seperated 'key=value'
// Todo : vcheeram : Move this to a common library. Keeping as exported for now
// Todo: vcheeram: this needs to be converted to some sort of io.reader instead
// passing the string.
//
// Unit test this with extra whitespace
func GetValueFromEnvBody(content, keyName string) (value string, err error) {
	if strings.TrimSpace(content) == "" || strings.TrimSpace(keyName) == "" {
		return "", errors.New("content and keyName cannot be empty")
	}
	//the config file should have the keyname as part of the beginning of line
	r, err := regexp.Compile(`(?im)^` + keyName + `\s*=\s*(.*)`)
	if err != nil {
		return
	}
	rs := r.FindStringSubmatch(content)
	if rs != nil {
		return rs[1], nil
	}
	return "", fmt.Errorf("Could not find Value for %s", keyName)
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

// GetHash returns a byte array to the hash of the data.
// alg indicates the hashing algorithm. Currently, the only supported hashing algorithms
// are SHA1, SHA256, SHA384 and SHA512
func GetHashData(data []byte, alg crypto.Hash) ([]byte, error) {

	if data == nil {
		return nil, fmt.Errorf("Error - data pointer is nil")
	}

	switch alg {
	case crypto.SHA1:
		s := sha1.Sum(data)
		return s[:], nil
	case crypto.SHA256:
		s := sha256.Sum256(data)
		return s[:], nil
	case crypto.SHA384:
		//SHA384 is implemented in the sha512 package
		s := sha512.Sum384(data)
		return s[:], nil
	case crypto.SHA512:
		s := sha512.Sum512(data)
		return s[:], nil
	}

	return nil, fmt.Errorf("Error - Unsupported hashing function %d requested. Only SHA1, SHA256, SHA384 and SHA512 supported", alg)
}

// ParseSetupTasks takes space seperated list of tasks along with any additional flags.
// Not used for now...
// TODO : to be implemented.
func ParseSetupTasks(commandargs ...[]string) []string {
	//TODO: This function for now takes a space seperated list of
	// setup arguments. We should parse this to check for the presence of --force
	//flags. This should be a common utility that is able to parse a list of
	// tasks as well as an associated flags
	if len(commandargs) > 1 {
		log.Println("Expecting a slice of string as argument.")
	}
	return commandargs[0]
}

// RunTasks - function to be implemented as part of the Common Installer module
func RunTasks(commandargs []string) {

}
