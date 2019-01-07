package common

import (
	"encoding/json"
	"fmt"
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/osutil"
	"log"
	"os"
	"strings"
)

const secretKeyLength int = 20

// CeritifiedKey is class that represents setup for a Signing or bindingkey
type CertifiedKey struct {
	keyUsage tpm.Usage
}

// tpmCertifiedKeySetup calls the TPM helper library to export a binding or signing keypair
func createKey(usage tpm.Usage) (tpmck *tpm.CertifiedKey, err error) {
	if usage != tpm.Binding && usage != tpm.Signing {
		return nil, fmt.Errorf("Function tpmCertifiedKeySetup - incorrect KeyUsage parameter - needs to be signing or binding")
	}
	t, err := tpm.Open()

	if t != nil {
		defer t.Close()
		secretbytes, err := osutil.GetRandomBytes(secretKeyLength)
		if err != nil {
			return nil, err
		}
		//get the aiksecret. This will return a byte array.
		aiksecret, err := config.GetAikSecret()
		if err != nil {
			return nil, err
		}
		log.Println(aiksecret)
		tpmck, err = t.CreateCertifiedKey(usage, secretbytes, aiksecret)
		if err != nil {
			return nil, err
		}
	}
	return tpmck, nil
}

//Todo: for now, this will always overwrite the file. Should be a parameter
// that forces overwrite of file.

func writeCertifiedKeyToDisk(tpmck *tpm.CertifiedKey, filepath string) error {

	if tpmck == nil {
		fmt.Errorf("CertifiedKey struct is empty")
	}

	json, err := json.MarshalIndent(*tpmck, "", "    ")
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

func NewCertifiedKey(certusage string) (*CertifiedKey, error) {

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
	return nil, fmt.Errorf("Unknown type of Setup CertifiedKey task - must be Signing or Binding")
}

// Execute method of BindingKey installs a binding key. It uses the AiKSecret
// that is obtained from the trust agent, a randomn secret and uses the TPM
// to generate a keypair that is tied to the TPM
func Run(ck *CertifiedKey) error {

	certKey, err := createKey(ck.keyUsage)
	if err != nil {
		log.Printf(err.Error())
		return err
	}
	var filename string
	switch ck.keyUsage {
	case tpm.Binding:
		filename = config.GetBindingKeyFileName()
	case tpm.Signing:
		filename = config.GetSigningKeyFileName()
	}
	filepath, err := osutil.MakeFilePathFromEnvVariable(config.GetConfigDir(), filename, true)
	if err != nil {
		log.Printf(err.Error())
		return err
	}
	log.Printf("Debug: Key store file path : %s", filepath)
	if ck == nil {
		return fmt.Errorf("Certified key not returned from TPM library")
	}
	err = writeCertifiedKeyToDisk(certKey, filepath)

	fmt.Println(filename)
	return nil

}

// Installed method of the CertifiedKey checks if there is a key already installed.
// For now, this only checks for the existence of the file and does not check if
// contents of the file are indeed correct
func Validate(ck *CertifiedKey) bool {
	var filename string

	switch ck.keyUsage {
	case tpm.Binding:
		filename = config.GetBindingKeyFileName()
	case tpm.Signing:
		filename = config.GetSigningKeyFileName()
	}

	filepath, _ := osutil.MakeFilePathFromEnvVariable(config.GetConfigDir(), filename, true)
	if fi, err := os.Stat(filepath); err == nil && fi != nil && fi.Mode().IsRegular() {
		return true
	}
	return false
}
