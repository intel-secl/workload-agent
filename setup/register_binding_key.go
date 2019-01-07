package setup

/**
** @author srege
**/

import (
	"bytes"
	b "encoding/base64"
	"encoding/json"
	"errors"
	csetup "intel/isecl/lib/common/setup"
	common "intel/isecl/wlagent/common"
	"intel/isecl/wlagent/config"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"
)

const aikCertName = "aik.pem"
const bindingKeyCertPath string = "/opt/workloadagent/configuration/bindingkey.pem"
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

	url = config.WlaConfig.MtwilsonAPIURL + "/rpc/certify-host-binding-key"

	fileName := "/opt/workloadagent/bindingkey.json"
	//config.GetBindingKeyFileName()

	_, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		return errors.New("bindingkey file does not exist")
	}
	file, _ := os.Open(fileName)
	defer file.Close()
	byteValue, _ := ioutil.ReadAll(file)

	_ = json.Unmarshal(byteValue, &bindingkey)

	tpmCertifyKeyBytes, _ := b.StdEncoding.DecodeString(bindingkey.KeyAttestation)
	tpmCertifyKey := b.StdEncoding.EncodeToString(tpmCertifyKeyBytes[2:])

	originalNameDigest, _ = b.StdEncoding.DecodeString(bindingkey.KeyName)
	originalNameDigest = originalNameDigest[1:]
	for i := 0; i < 34; i++ {
		originalNameDigest = append(originalNameDigest, 0)
	}

	nameDigest := b.StdEncoding.EncodeToString(originalNameDigest)

	if runtime.GOOS == "Linux" {
		operatingSystem = "Linux"
	} else {
		operatingSystem = "Windows"
	}

	if bindingkey.Version == 2 {
		tpmVersion = "2.0"
	} else {
		tpmVersion = "1.2"
	}
	aik := getAikCert()

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
	httpRequest.SetBasicAuth(config.WlaConfig.MtwilsonAPIUsername, config.WlaConfig.MtwilsonAPIPassword)

	httpResponse, err := common.SendRequest(httpRequest)
	if err != nil {
		return errors.New("error in binding key registration.")
	}
	_ = json.Unmarshal([]byte(httpResponse), &bindingKeyCert)

	aikPem := beginCert + "\n" + bindingKeyCert.BindingKeyCertificate + "\n" + endCert + "\n"
	file, _ = os.Create(bindingKeyCertPath)

	_, err = file.Write([]byte(aikPem))
	if err != nil {
		return errors.New("error in writing to file.")
	}
	return nil
}
func getAikCert() string {
	aikfile, err := os.Open(aikCertName)
	if err != nil {

	}
	aikCert, _ := ioutil.ReadAll(aikfile)
	aik := string(aikCert)
	aik = strings.Replace(aik, beginCert, "", -1)
	aik = strings.Replace(aik, endCert, "", -1)

	re := regexp.MustCompile(`\r?\n`)
	aik = re.ReplaceAllString(aik, "")
	return aik
}

// Validate checks whether or not the Register Binding Key task was completed successfully
func (rb RegisterBindingKey) Validate(c csetup.Context) error {
	/*if BindingKeyCert.BindingKeyCertificate == "" {
		return errors.New("Register Bigning key: BindingKeyCertificate is not set")
	}*/
	return nil
}
