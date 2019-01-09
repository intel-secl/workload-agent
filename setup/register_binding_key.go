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
	common "intel/isecl/wlagent/common"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/osutil"
	"io/ioutil"

	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
)

const aikCertName string = "/opt/trustagent/configuration/aik.pem"
const bindingKeyCertPath string = "/opt/workloadagent/configuration/bindingkeycert.pem" //GetTrustAgentConfigDir() +
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
	e := common.SaveConfiguration(c)
	if e != nil {
		log.Error(e.Error())
		return e
	}

	var url string
	var requestBody []byte
	var bindingkey BindingKeyInfo
	var tpmVersion string
	var originalNameDigest []byte
	var bindingKeyCert BindingKeyCert
	var operatingSystem string

	url = config.Configuration.Mtwilson.APIURL + "rpc/certify-host-binding-key"
	fileName := config.GetBindingKeyFileName()

	fileName := config.GetBindingKeyFileName()
	bindingkeyFilePath, err := osutil.MakeFilePathFromEnvVariable(config.GetConfigDir(), fileName, true)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	_, err = os.Stat(bindingkeyFilePath)
	if os.IsNotExist(err) {
		return errors.New("bindingkey file does not exist")
	}
	file, _ := os.Open(bindingkeyFilePath)
	defer file.Close()
	byteValue, _ := ioutil.ReadAll(file)

	_ = json.Unmarshal(byteValue, &bindingkey)

	aikCertFileName, err := osutil.MakeFilePathFromEnvVariable(config.GetConfigDir(), "aik.pem", true)
	if err != nil {
		log.Printf(err.Error())
		return err
	}

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
	aik := getAikCert(aikCertName)

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
func getAikCert(aikCertFileName string) string {
	aikfile, err := os.Open(aikCertFileName)
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

// Validate checks whether or not the Register Binding Key task was completed successfully
func (rb RegisterBindingKey) Validate(c csetup.Context) error {
	/*if BindingKeyCert.BindingKeyCertificate == "" {
		return errors.New("Register Bigning key: BindingKeyCertificate is not set")
	}*/
	return nil
}
