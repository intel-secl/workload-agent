package common

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"intel/isecl/lib/common/crypt"
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/consts"
	"os"

	log "github.com/sirupsen/logrus"
)

const secretKeyLength int = 20

// tpmCertifiedKeySetup calls the TPM helper library to export a binding or signing keypair
func createKey(usage tpm.Usage, t tpm.Tpm) (tpmck *tpm.CertifiedKey, err error) {
	log.Info("Creation of signing or binding key.")
	if usage != tpm.Binding && usage != tpm.Signing {
		return nil, errors.New("incorrect KeyUsage parameter - needs to be signing or binding")
	}
	secretbytes, err := crypt.GetRandomBytes(secretKeyLength)
	if err != nil {
		return nil, err
	}
	// get the aiksecret. This will return a byte array.
	log.Debug("Getting aik secret from trusagent configuration.")
	aiksecret, err := config.GetAikSecret()
	if err != nil {
		return nil, err
	}
	log.Debug("Calling CreateCertifiedKey of tpm library to create and certify signing or binding key.")
	tpmck, err = t.CreateCertifiedKey(usage, secretbytes, aiksecret)
	if err != nil {
		return nil, err
	}

	config.Configuration.BindingKeySecret = base64.StdEncoding.EncodeToString(secretbytes)
	config.Save()

	log.Println("The binding key secret is:", base64.StdEncoding.EncodeToString(secretbytes))
	log.Println("The binding key secret from the config var is:", config.Configuration.BindingKeySecret)
	return tpmck, nil
}

//Todo: for now, this will always overwrite the file. Should be a parameter
// that forces overwrite of file.
func writeCertifiedKeyToDisk(tpmck *tpm.CertifiedKey, filepath string) error {
	log.Debug("Writing certified signing or binding key to specified location on disk.")
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

// GenerateKey creates a TPM binding or signing key
// It uses the AiKSecret that is saved in the Workload Agent configuration
// that is obtained from the trust agent, a randomn secret and uses the TPM
// to generate a keypair that is tied to the TPM
func GenerateKey(usage tpm.Usage, t tpm.Tpm) error {
	if t == nil || (usage != tpm.Binding && usage != tpm.Signing) {
		return errors.New("certified key or connection to TPM library failed")
	}

	// Create and certify the signing or binding key
	certKey, err := createKey(usage, t)
	if err != nil {
		return err
	}

	// Get the name of signing or binding key files depending on input parameter
	var filename string
	switch usage {
	case tpm.Binding:
		filename = consts.BindingKeyFileName
	case tpm.Signing:
		filename = consts.SigningKeyFileName
	}

	// Join configuration path and signing or binding file name
	filepath := consts.ConfigDirPath + filename

	// Writing certified key value to file path
	err = writeCertifiedKeyToDisk(certKey, filepath)
	if err != nil {
		return err
	}

	log.Info("Key is stored at file path : ", filepath)
	return nil
}

// ValidateKey validates if a key of type binding or signing is actually configured in
// the Workload Agent
// Installed method of the CertifiedKey checks if there is a key already installed.
// For now, this only checks for the existence of the file and does not check if
// contents of the file are indeed correct
func ValidateKey(usage tpm.Usage) error {
	// Get the name of signing or binding key files depending on input parameter
	var filename string
	switch usage {
	case tpm.Binding:
		filename = consts.BindingKeyFileName
	case tpm.Signing:
		filename = consts.SigningKeyFileName
	}

	// Join configuration path and signing or binding file name
	filepath := consts.ConfigDirPath + filename
	fi, err := os.Stat(filepath)
	if err != nil {
		return err
	}
	if fi == nil && !fi.Mode().IsRegular() {
		return errors.New("key file path is incorrect")
	}
	return nil
}
