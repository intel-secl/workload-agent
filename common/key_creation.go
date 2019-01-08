package common

import (
	"encoding/json"
	"errors"
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/osutil"
	"log"
	"os"
	"strings"
)

const secretKeyLength int = 20

// CeritifiedKey is class that represents setup for a signing or bindingkey
type CertifiedKey struct {
	keyUsage tpm.Usage
}

// tpmCertifiedKeySetup calls the TPM helper library to export a binding or signing keypair
func createKey(usage tpm.Usage, t tpm.Tpm) (tpmck *tpm.CertifiedKey, err error) {
	log.Println("Creation of signing or binding key.")
	if usage != tpm.Binding && usage != tpm.Signing {
		return nil, errors.New("incorrect KeyUsage parameter - needs to be signing or binding")
	}
	secretbytes, err := osutil.GetRandomBytes(secretKeyLength)
	if err != nil {
		return nil, err
	}
	// get the aiksecret. This will return a byte array.
	log.Println("Getting aik secret from trusagent configuration.")
	aiksecret, err := config.GetAikSecret()
	if err != nil {
		return nil, err
	}
	log.Println("Calling CreateCertifiedKey of tpm library to create and certify signing or binding key.")
	tpmck, err = t.CreateCertifiedKey(usage, secretbytes, aiksecret)
	if err != nil {
		return nil, err
	}
	return tpmck, nil
}

//Todo: for now, this will always overwrite the file. Should be a parameter
// that forces overwrite of file.
func writeCertifiedKeyToDisk(tpmck *tpm.CertifiedKey, filepath string) error {
	log.Println("Writing certified signing or binding key to specified location on disk.")
	if tpmck == nil {
		return errors.New("certifiedKey struct is empty")
	}

	// Marshal the certified key to json
	json, err := json.MarshalIndent(tpmck, "", "    ")
	if err != nil {
		return err
	}

	// create a file and write the json value to it and finally close it
	f, err := os.Create(filepath)
	if err != nil {
		return errors.New("could not create file Error:" + err.Error())
	}
	f.WriteString(string(json))
	f.WriteString("\n")
	defer f.Close()

	return nil
}

func NewCertifiedKey(certusage string) (*CertifiedKey, error) {
	log.Println("Returning object of CertifiedKey depending on input parameter.")
	switch strings.ToLower(strings.TrimSpace(certusage)) {
	case "signing", "sign":
		return &CertifiedKey{
			keyUsage: tpm.Signing,
		}, nil

	case "binding", "bind":
		return &CertifiedKey{
			keyUsage: tpm.Binding,
		}, nil
	}
	return nil, errors.New("unknown type of Setup CertifiedKey task - must be Signing or Binding")
}

// Execute method of BindingKey installs a binding key. It uses the AiKSecret
// that is obtained from the trust agent, a randomn secret and uses the TPM
// to generate a keypair that is tied to the TPM
func KeyGeneration(ck *CertifiedKey, t tpm.Tpm) error {
	if t == nil || ck == nil {
		return errors.New("certified key or connection to TPM library failed")
	}

	// Create and certify the signing or binding key
	certKey, err := createKey(ck.keyUsage, t)
	if err != nil {
		return err
	}

	// Get the name of signing or binding key files depending on input parameter
	var filename string
	switch ck.keyUsage {
	case tpm.Binding:
		filename = config.GetBindingKeyFileName()
	case tpm.Signing:
		filename = config.GetSigningKeyFileName()
	}

	// Join configuration path and signing or binding file name
	filepath, err := osutil.MakeFilePathFromEnvVariable(config.GetConfigDir(), filename, true)
	if err != nil {
		return err
	}

	// Writing certified key value to file path
	err = writeCertifiedKeyToDisk(certKey, filepath)
	if err != nil {
		return err
	}

	log.Printf("Key is stored at file path : %s", filepath)
	return nil
}

// Installed method of the CertifiedKey checks if there is a key already installed.
// For now, this only checks for the existence of the file and does not check if
// contents of the file are indeed correct
func KeyValidation(ck *CertifiedKey) error {
	// Get the name of signing or binding key files depending on input parameter
	var filename string
	switch ck.keyUsage {
	case tpm.Binding:
		filename = config.GetBindingKeyFileName()
	case tpm.Signing:
		filename = config.GetSigningKeyFileName()
	}

	// Join configuration path and signing or binding file name
	filepath, _ := osutil.MakeFilePathFromEnvVariable(config.GetConfigDir(), filename, true)
	fi, err := os.Stat(filepath)
	if err != nil {
		return err
	}
	if fi == nil && !fi.Mode().IsRegular() {
		return errors.New("key file path is incorrect")
	}
	return nil
}
