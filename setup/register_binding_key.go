package setup

/**
** @author srege
**/

import (
	"bytes"
	b "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	csetup "intel/isecl/lib/common/setup"
	"intel/isecl/wlagent/common"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/consts"
	"intel/isecl/lib/common/exec"
	"io/ioutil"

	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
)

const beginCert string = "-----BEGIN CERTIFICATE-----"
const endCert string = "-----END CERTIFICATE-----"

type RegisterBindingKey struct {
}

type BindingKeyInfo struct {
	Version        int    `json:"Version"`
	KeyAttestation string `json:"KeyAttestation"`
	PublicKey      string `json:"PublicKey"`
	KeySignature   string `json:"KeySignature"`
	KeyName        string `json:"KeyName"`
}
type BindingKeyCert struct {
	BindingKeyCertificate string `json:"binding_key_der_certificate"`
}

func (rb RegisterBindingKey) Run(c csetup.Context) error {

	var url string
	var requestBody []byte
	var bindingkey BindingKeyInfo
	var tpmVersion string
	var originalNameDigest []byte
	var bindingKeyCert BindingKeyCert
	var operatingSystem string

	if rb.Validate(c) == nil {
		log.Info("Binding key already registered. Skipping this setup task.")
		return nil
	}

	// save configuration from config.yml
	e := config.SaveConfiguration(c)
	if e != nil {
		log.Error(e.Error())
		return e
	}
	log.Info("Registering binding key with host verification service.")

	url = config.Configuration.Mtwilson.APIURL + "/rpc/certify-host-binding-key"

	// join configuration path and binding key file name
	bindingkeyFilePath := consts.ConfigDirPath + consts.BindingKeyFileName

	// check if binding key file exists
	_, err := os.Stat(bindingkeyFilePath)
	if os.IsNotExist(err) {
		return errors.New("bindingkey file does not exist")
	}

	// read contents of binding key file and store in BindingKeyInfo struct
	file, _ := os.Open(bindingkeyFilePath)
	defer file.Close()
	byteValue, _ := ioutil.ReadAll(file)

	_ = json.Unmarshal(byteValue, &bindingkey)

	// remove first two bytes from KeyAttestation. These are extra bytes written.
	tpmCertifyKeyBytes, _ := b.StdEncoding.DecodeString(bindingkey.KeyAttestation)
	tpmCertifyKey := b.StdEncoding.EncodeToString(tpmCertifyKeyBytes[2:])

	// remove first byte from the value written to KeyName. This is an extra byte written.
	originalNameDigest, _ = b.StdEncoding.DecodeString(bindingkey.KeyName)
	originalNameDigest = originalNameDigest[1:]

	//append 0 added as padding
	for i := 0; i < 34; i++ {
		originalNameDigest = append(originalNameDigest, 0)
	}

	nameDigest := b.StdEncoding.EncodeToString(originalNameDigest)

	//get trustagent aik cert location
	aikCertName, _ := exec.MkDirFilePathFromEnvVariable(consts.TrustAgentConfigDirEnv, consts.AIKPemFileName, true)

	// set operating system
	if runtime.GOOS == "linux" {
		operatingSystem = "Linux"
	} else {
		operatingSystem = "Windows"
	}

	//set tpm version
	if bindingkey.Version == 2 {
		tpmVersion = "2.0"
	} else {
		tpmVersion = "1.2"
	}

	//getAikCert removes the begin / end certificate tags and newline characters
	aik := getAikCert(aikCertName)

	//construct request body
	requestBody = []byte(`{
		 "public_key_modulus":"` + bindingkey.PublicKey + `",
	 	 "tpm_certify_key":"` + tpmCertifyKey + `",
	     "tpm_certify_key_signature":"` + bindingkey.KeySignature + `",
	 	 "aik_der_certificate":"` + aik + `",
	 	 "name_digest":"` + nameDigest + `",
	     "tpm_version":"` + tpmVersion + `",
		 "operating_system":"` + operatingSystem + `"}`)

	// set POST request Accept, Content-Type and Authorization headers
	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.SetBasicAuth(config.Configuration.Mtwilson.APIUsername, config.Configuration.Mtwilson.APIPassword)

	fmt.Println("HVS Username: ", config.Configuration.Mtwilson.APIUsername)
	fmt.Println("HVS Password: ", config.Configuration.Mtwilson.APIPassword)
	fmt.Println("HVS URL: ", url)

	httpResponse, err := common.SendRequest(httpRequest)
	if err != nil {
		return errors.New("error in binding key registration.")
	}
	_ = json.Unmarshal([]byte(httpResponse), &bindingKeyCert)

	if len(strings.TrimSpace(bindingKeyCert.BindingKeyCertificate)) <= 0 {
		return errors.New("error in binding key certificate registration.")
	}

	//construct the certificate by adding begin and end certificate tags
	aikPem := beginCert + "\n" + bindingKeyCert.BindingKeyCertificate + "\n" + endCert + "\n"

	//write the binding key certificate to file
	bindingKeyCertPath := consts.ConfigDirPath + consts.BindingKeyPemFileName

	file, _ = os.Create(bindingKeyCertPath)
	_, err = file.Write([]byte(aikPem))
	if err != nil {
		return errors.New("error in writing to file.")
	}
	return nil
}

//getAikCert method removes begin and end certificate tag and newline character.
func getAikCert(aikCertName string) string {
	aikfile, err := os.Open(aikCertName)
	if err != nil {
		fmt.Println(err)
	}
	aikCert, _ := ioutil.ReadAll(aikfile)
	aik := string(aikCert)
	aik = strings.Replace(aik, beginCert, "", -1)
	aik = strings.Replace(aik, endCert, "", -1)

	re := regexp.MustCompile(`\r?\n`)
	aik = re.ReplaceAllString(aik, "")
	return aik
}

// Validate checks whether or not the register binding key task was completed successfully
func (rb RegisterBindingKey) Validate(c csetup.Context) error {
	log.Info("Validation for registering binding key.")
	bindingKeyCertFilePath := consts.ConfigDirPath + consts.BindingKeyPemFileName
	_, err := os.Stat(bindingKeyCertFilePath)
	if os.IsNotExist(err) {
		return errors.New("Binding key certificate file does not exist")
	}
	return nil
}
