package wlagent

import (
	"encoding/json"
	"fmt"
	"intel/isecl/lib/tpm"
	"log"
	"os"
	"reflect"
	"strings"
)


const taConfigExportCmd string = "tagent export-config --stdout"
const aikSecretKeyName string = "aik.secret"
const secretKeyLength int = 20

// SetupTask is an generic interface for a setup task.
// The Execute() method of the class executes the setup task
// The Installed() method check if the SetupTask is already completed
// The sequence of operation for a setup task should be to check if
// it has already installed and if not installed, then call the execute method
type SetupTask interface {
	Execute() error
	Installed() bool
}



// certifiedKeySetup calls the TPM helper library to export a binding or signing keypair
func certifiedKeySetup(keyUsage tpm.Usage) (ck *tpm.CertifiedKey, err error) {


	if keyUsage != tpm.Binding && keyUsage != tpm.Signing {

		return nil, fmt.Errorf("Function CertifiedKeySetup - incorrect KeyUsage parameter - needs to be signing or binding")
	}
	t, err := tpm.Open()
	//todo: remove

	if t != nil {
		defer t.Close()
		secretbytes, err := GetRandomBytes(secretKeyLength)
		if err != nil {
			return nil, err
		}

		//get the aiksecret. This will return a byte array. 
		aiksecret, err := getAikSecret()
		if err != nil {
			return nil, err
		}
		log.Println(aiksecret)
		ck, err = t.CreateCertifiedKey(keyUsage, secretbytes, aiksecret)
		if err != nil {
			return nil, err
		}

	}
	return ck, nil
}

//Todo: for now, this will always overwrite the file. Should be a parameter
// that forces overwrite of file.

func writeCertifiedKeyToDisk(ck *tpm.CertifiedKey, filepath string) error {

	if ck == nil {
		fmt.Errorf("CertifiedKey struct is empty")
	}

	json, err := json.MarshalIndent(*ck, "", "    ")
	if err != nil {
		return err
	}

	f, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("Could not create file Error:" + err.Error())
	}
	f.WriteString(string(json))
	f.WriteString("\n")

	defer f.Close()

	return nil
}

func setupKey(keyUsage tpm.Usage, filename string ) error {

	certKey, err := certifiedKeySetup(keyUsage)
	if err != nil {
		log.Printf(err.Error())
		return err
	}

	filepath, err := MakeFilePathFromEnvVariable(getConfigDIr(), filename, true)
	if err != nil {
		log.Printf(err.Error())
		return err
	}
	log.Printf("Debug: Key store file path : %s", filepath)
	if certKey == nil {
		return fmt.Errorf("Certified key not returned from TPM library")
	}
	err = writeCertifiedKeyToDisk(certKey, filepath)

	fmt.Println(filename)
	return nil

}

// BindingKey represents a class for SigningKey bound to the TPM
type BindingKey struct{}

// Execute method of BindingKey installs a binding key. It uses the AiKSecret
// that is obtained from the trust agent, a randomn secret and uses the TPM
// to generate a keypair that is tied to the TPM
func (BindingKey) Execute() error {

	return setupKey(tpm.Binding, getBindingKeyFileName())
}

// Installed method of the BindingKey check if there is a key already installed.
// For now, this only checks for the existence of the file and does not check if
// contents of the file are indeed correct
func (BindingKey) Installed() bool {
	filepath, _ := MakeFilePathFromEnvVariable(getConfigDIr(), getBindingKeyFileName(), true)
	if fi,err := os.Stat(filepath); err == nil && fi != nil && fi.Mode().IsRegular(){
		return true
	}
	return false
}

// SigningKey represents a class for SigningKey bound to the TPM
type SigningKey struct{}

// Execute method of SigningKey installs a binding key. It uses the AiKSecret
// that is obtained from the trust agent, a randomn secret and uses the TPM
// to generate a keypair that is tied to the TPM
func (SigningKey) Execute() error {
	return setupKey(tpm.Signing, getSigningKeyFileName())
}
// Installed method of the SigningKey check if there is a key already installed.
// For now, this only checks for the existence of the file and does not check if
// contents of the file are indeed correct
func (SigningKey) Installed() bool {
	filepath, _ := MakeFilePathFromEnvVariable(getConfigDIr(), getSigningKeyFileName(), true)
	if fi,err := os.Stat(filepath); err == nil && fi != nil && fi.Mode().IsRegular(){
		return true
	}
	return false
}

// GetSetupTasks returns a map with SetupTasks in the module. These are all the struct/ class
// that implements the SetupTask interface. 
// If there is a specific task(s) being requested, we will return only these task
func GetSetupTasks(commandargs []string) map[string]SetupTask {

	//tasks = ParseSetupTasks(commandargs)
	if len(commandargs) < 1 || strings.ToLower(commandargs[0]) != "setup" {
		panic (fmt.Errorf("method GetSetupTasks need at least one parameter with command \"setup\". Arguments : %v", commandargs))
	}
	
	m := make(map[string]SetupTask)

	if len(commandargs) > 1  {
		// Todo - we should be able to find structs using reflection in this 
		// package that implements the SetupTask Interface and add elements to the
		//  map. For now, we are just going to hardcode the setup tasks that we have

		// First argument is "setup" - the rest should be list of tasks
		for _, task := range commandargs[1:] {

			switch strings.ToLower(task) {
			case "signingkey":
				m["SigningKey"] = SigningKey{}
			case "bindingkey":
				m["BindingKey"] = BindingKey{}
			default:
				log.Printf("Unknown Setup Task in list : %s", task)
			}
		}
		
	} else {
		fmt.Println("No arguments passed in")
		// no specific tasks passed in. We will return a list of all tasks
		m[reflect.TypeOf(SigningKey{}).Name()] = SigningKey{}
		m[reflect.TypeOf(BindingKey{}).Name()] = BindingKey{}
	
	}
	
	//for key, obj := range m {
	//	fmt.Println("Key=" + key + "\tValue=" + reflect.TypeOf(obj).Name())
	//}
	
	return m
}

// ParseSetupTasks takes space seperated list of tasks along with any additional flags. 
// Not used for now... 
// TODO : to be implemented. 
func ParseSetupTasks(commandargs ...[]string) []string{
	//TODO: This function for now takes a space seperated list of
	// setup arguments. We should parse this to check for the presence of --force
	//flags. This should be a common utility that is able to parse a list of 
	// tasks as well as an associated flags
	if len(commandargs) > 1{
		log.Println("Expecting a slice of string as argument.")
	}
	fmt.Println(commandargs)
	return commandargs[0]
}

// RunTasks - function to be implemented as part of the Common Installer module
func RunTasks(commandargs []string ){

}
