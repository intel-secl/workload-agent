package common

import (
	"bytes"
	b "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/consts"
	"intel/isecl/wlagent/osutil"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strings"
)

type KeyInfo struct {
	Version        int    `json:"Version"`
	KeyAttestation string `json:"KeyAttestation"`
	PublicKey      string `json:"PublicKey"`
	KeySignature   string `json:"KeySignature"`
	KeyName        string `json:"KeyName"`
}
type BindingKeyCert struct {
	BindingKeyCertificate string `json:"binding_key_der_certificate"`
}
type SigningKeyCert struct {
	SigningKeyCertificate string `json:"signing_key_der_certificate"`
}
type HttpRequestBody struct {
	PublicKeyModulus       string `json:"public_key_modulus"`
	TpmCertifyKey          string `json:"tpm_certify_key"`
	TpmCertifyKeySignature string `json:"tpm_certify_key_signature"`
	AikDerCertificate      string `json:"aik_der_certificate"`
	NameDigest             string `json:"name_digest"`
	TpmVersion             string `json:"tpm_version"`
	OsType                 string `json:"operating_system"`
}

const beginCert string = "-----BEGIN CERTIFICATE-----"
const endCert string = "-----END CERTIFICATE-----"

func RegisterKey(usage tpm.Usage) error {
	var certifyKeyUrl *url.URL
	var keyInfo KeyInfo
	var keyFilePath string
	var originalNameDigest []byte
	var requestBody []byte
	var tpmVersion string
	var operatingSystem string
	var err error

	if usage != tpm.Binding && usage != tpm.Signing {
		return errors.New("incorrect KeyUsage parameter - needs to be signing or binding")
	}

	// join configuration path and binding key file name
	if usage == tpm.Binding {
		keyFilePath = consts.ConfigDirPath + consts.BindingKeyFileName
		certifyKeyUrl, err = url.Parse(config.Configuration.Mtwilson.APIURL + "/rpc/certify-host-binding-key")
		if err != nil {
			fmt.Println(err)
		}
	
	} else {
		keyFilePath = consts.ConfigDirPath + consts.SigningKeyFileName
		certifyKeyUrl, err = url.Parse(config.Configuration.Mtwilson.APIURL + "/rpc/certify-host-signing-key")
		if err != nil {
			fmt.Println(err)
		}
	}
	
	// check if binding key file exists
	_, err = os.Stat(keyFilePath)
	if os.IsNotExist(err) {
		return errors.New("key file does not exist")
	}
	// read contents of binding key file and store in KeyInfo struct
	file, _ := os.Open(keyFilePath)
	defer file.Close()

	byteValue, _ := ioutil.ReadAll(file)

	_ = json.Unmarshal(byteValue, &keyInfo)
	// remove first two bytes from KeyAttestation. These are extra bytes written.
	tpmCertifyKeyBytes, _ := b.StdEncoding.DecodeString(keyInfo.KeyAttestation)
	tpmCertifyKey := b.StdEncoding.EncodeToString(tpmCertifyKeyBytes[2:])

	// remove first byte from the value written to KeyName. This is an extra byte written.
	originalNameDigest, _ = b.StdEncoding.DecodeString(keyInfo.KeyName)
	originalNameDigest = originalNameDigest[1:]

	//append 0 added as padding
	for i := 0; i < 34; i++ {
		originalNameDigest = append(originalNameDigest, 0)
	}

	nameDigest := b.StdEncoding.EncodeToString(originalNameDigest)

	//get trustagent aik cert location
	aikCertName, _ := osutil.MakeFilePathFromEnvVariable(consts.TrustAgentConfigDirEnv, "aik.pem", true)

	// set operating system
	if runtime.GOOS == "linux" {
		operatingSystem = "Linux"
	} else {
		operatingSystem = "Windows"
	}

	//set tpm version
	if keyInfo.Version == 2 {
		tpmVersion = "2.0"
	} else {
		tpmVersion = "1.2"
	}

	//getAikCert removes the begin / end certificate tags and newline characters
	aik := getAikCert(aikCertName)

	//construct request body
	httpRequestBody := HttpRequestBody{
		PublicKeyModulus:       keyInfo.PublicKey,
		TpmCertifyKey:          tpmCertifyKey,
		TpmCertifyKeySignature: keyInfo.KeySignature,
		AikDerCertificate:      aik,
		NameDigest:             nameDigest,
		TpmVersion:             tpmVersion,
		OsType:                 operatingSystem,
	}
	requestBody, err = json.Marshal(httpRequestBody)
	if err != nil {
		fmt.Println(err)
	}
	// set POST request Accept, Content-Type and Authorization headers
	httpRequest, err := http.NewRequest("POST", certifyKeyUrl.String(), bytes.NewBuffer(requestBody))
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.SetBasicAuth(config.Configuration.Mtwilson.APIUsername, config.Configuration.Mtwilson.APIPassword)

	httpResponse, err := SendHttpRequest(httpRequest)
	if err != nil {
		return errors.New("error in key registration.")
	}
	switch usage {
	case tpm.Binding:
		{
			//construct the certificate by adding begin and end certificate tags
			//*****pem Decode and encode
			var bindingkeyCert BindingKeyCert
			err = json.Unmarshal([]byte(httpResponse), &bindingkeyCert)
			if err != nil {
				fmt.Println("Error Marshalling." + err.Error())
			}
			aikPem := beginCert + "\n" + bindingkeyCert.BindingKeyCertificate + "\n" + endCert + "\n"

			//write the binding key certificate to file
			keyCertPath := consts.ConfigDirPath + consts.BindingKeyPemFileName
			file, _ = os.Create(keyCertPath)
			_, err = file.Write([]byte(aikPem))
			if err != nil {
				return errors.New("error in writing to file.")
			}
		}
	case tpm.Signing:
		{
			var signingkeyCert SigningKeyCert
			err = json.Unmarshal([]byte(httpResponse), &signingkeyCert)
			if err != nil {
				fmt.Printf("error Marshalling. %s", err.Error())
			}
			//construct the certificate by adding begin and end certificate tags
			aikPem := beginCert + "\n" + signingkeyCert.SigningKeyCertificate + "\n" + endCert + "\n"

			//write the binding key certificate to file
			keyCertPath := consts.ConfigDirPath + consts.SigningKeyPemFileName

			file, _ = os.Create(keyCertPath)
			_, err = file.Write([]byte(aikPem))
			if err != nil {
				return errors.New("error in writing to file.")
			}
		}
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
