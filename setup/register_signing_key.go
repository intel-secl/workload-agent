package setup

import (
	"bytes"
	b "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	csetup "intel/isecl/lib/common/setup"
	"intel/isecl/wlagent/common"
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

const beginCertTag string = "-----BEGIN CERTIFICATE-----"
const endCertTag string = "-----END CERTIFICATE-----"

type RegisterSigningKey struct {
}

type SigningKeyInfo struct {
	Version        int    `json:"Version"`
	KeyAttestation string `json:"KeyAttestation"`
	PublicKey      string `json:"PublicKey"`
	KeySignature   string `json:"KeySignature"`
	KeyName        string `json:"KeyName"`
}
type SigningKeyCert struct {
	SigningKeyCertificate string `json:"signing_key_der_certificate"`
}

func (rs RegisterSigningKey) Run(c csetup.Context) error {
	var url string
	var requestBody []byte
	var signingkey SigningKeyInfo
	var tpmVersion string
	var originalNameDigest []byte
	var signingKeyCert SigningKeyCert
	var operatingSystem string

	if rs.Validate(c) == nil {
		log.Info("Signing key already registered. Skipping this setup task.")
		return nil
	}

	// save configuration from config.yml
	e := config.SaveConfiguration(c)
	if e != nil {
		log.Error(e.Error())
		return e
	}

	log.Info("Registering signing key with host verification service.")

	url = config.Configuration.Mtwilson.APIURL + "/rpc/certify-host-signing-key"

	// join configuration path and signing key file name
	fileName := config.GetSigningKeyFileName()
	signingkeyFilePath, err := osutil.MakeFilePathFromEnvVariable(config.GetConfigDir(), fileName, true)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	// check if signing key file exists
	_, err = os.Stat(signingkeyFilePath)
	if os.IsNotExist(err) {
		return errors.New("signingkey file does not exist")
	}

	// read contents of signing key file and store in SigningKeyInfo struct
	jsonFile, err := os.Open(signingkeyFilePath)
	if err != nil {
		return errors.New("error opening signingkey file")
	}

	defer jsonFile.Close()
	byteValue, _ := ioutil.ReadAll(jsonFile)

	_ = json.Unmarshal(byteValue, &signingkey)

	// remove first two bytes from KeyAttestation. These are extra bytes being written.
	tpmCertifyKeyBytes, _ := b.StdEncoding.DecodeString(strings.TrimSpace(signingkey.KeyAttestation))
	tpmCertifyKey := b.StdEncoding.EncodeToString(tpmCertifyKeyBytes[2:])

	// remove first byte from the value written to KeyName. This is extra byte written.
	originalNameDigest, _ = b.StdEncoding.DecodeString(strings.TrimSpace(signingkey.KeyName))
	originalNameDigest = originalNameDigest[1:]

	//append 0 added as padding
	for i := 0; i < 34; i++ {
		originalNameDigest = append(originalNameDigest, 0)
	}

	nameDigest := b.StdEncoding.EncodeToString(originalNameDigest)

	//get trustagent aik cert location
	aikCertFileName, _ := osutil.MakeFilePathFromEnvVariable(config.GetTrustAgentConfigDir(), "aik.pem", true)

	//getAikCert removes the begin / end certificate tags and newline characters
	aik := getAikCert(aikCertFileName)

	// set operating system
	if runtime.GOOS == "linux" {
		operatingSystem = "Linux"
	} else {
		operatingSystem = "Windows"
	}

	//set tpm version
	if signingkey.Version == 2 {
		tpmVersion = "2.0"
	} else {
		tpmVersion = "1.2"
	}

	//construct request body
	requestBody = []byte(`{
		 "public_key_modulus":"` + signingkey.PublicKey + `",
	 	 "tpm_certify_key":"` + tpmCertifyKey + `",
	     "tpm_certify_key_signature":"` + signingkey.KeySignature + `",
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
		return errors.New("error in signing key registration")
	}

	_ = json.Unmarshal([]byte(httpResponse), &signingKeyCert)

	if len(strings.TrimSpace(signingKeyCert.SigningKeyCertificate)) <= 0 {
		return errors.New("error in signing key certificate creation.")
	}

	//construct the certificate by adding begin and end certificate tags
	aikPem := beginCertTag + "\n" + signingKeyCert.SigningKeyCertificate + "\n" + endCertTag + "\n"

	//write the signing key certificate to file
	signingKeyCertPath, err := osutil.MakeFilePathFromEnvVariable(config.GetConfigDir(), config.GetSigningKeyPemFileName(), true)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	file, _ := os.Create(signingKeyCertPath)
	_, err = file.Write([]byte(aikPem))
	if err != nil {
		return errors.New("error in writing to file")
	}
	return nil
}

//getAikCertFile method removes begin and end certificate tag and newline character.
func getAikCertFile(aikCertFileName string) string {
	aikfile, err := os.Open(aikCertFileName)
	if err != nil {
		fmt.Println(err)
	}
	aikCert, _ := ioutil.ReadAll(aikfile)
	aik := string(aikCert)
	aik = strings.Replace(aik, beginCertTag, "", -1)
	aik = strings.Replace(aik, endCertTag, "", -1)

	re := regexp.MustCompile(`\r?\n`)
	aik = re.ReplaceAllString(aik, "")
	return aik
}

// Validate checks whether or not the Register Signing Key task was completed successfully
func (rs RegisterSigningKey) Validate(c csetup.Context) error {

	log.Info("Validation for registering signing key.")

	signingKeyCertPath, err := osutil.MakeFilePathFromEnvVariable(config.GetConfigDir(), config.GetSigningKeyPemFileName(), true)
	_, err = os.Stat(signingKeyCertPath)
	if os.IsNotExist(err) {
		return errors.New("Signing key certificate file does not exist")
	}
	return nil
}
