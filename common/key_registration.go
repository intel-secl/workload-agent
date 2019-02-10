package common

import (
	"crypto/x509"
	b "encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	mtwilson "intel/isecl/lib/mtwilson-client"
	"intel/isecl/wlagent/consts"
	"intel/isecl/wlagent/osutil"
	"io/ioutil"
	"os"
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

func CreateRequest(keyfilePath string) (mtwilson.RegisterKeyInfo, error) {
	var httpRequestBody mtwilson.RegisterKeyInfo
	var keyInfo KeyInfo
	var tpmVersion string
	var originalNameDigest []byte
	var err error

	// check if binding key file exists
	_, err = os.Stat(keyfilePath)
	if os.IsNotExist(err) {
		return httpRequestBody, errors.New("key file does not exist")
	}
	// read contents of key file and store in KeyInfo struct
	file, err := os.Open(keyfilePath)
	if err != nil {
		return httpRequestBody, errors.New("error opening key file. " + err.Error())
	}

	defer file.Close()

	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		return httpRequestBody, errors.New("error reading file. ")
	}

	err = json.Unmarshal(byteValue, &keyInfo)
	if err != nil {
		return httpRequestBody, errors.New("error unmarshalling. " + err.Error())
	}

	// remove first two bytes from KeyAttestation. These are extra bytes written.
	tpmCertifyKeyBytes, _ := b.StdEncoding.DecodeString(keyInfo.KeyAttestation)
	tpmCertifyKey := b.StdEncoding.EncodeToString(tpmCertifyKeyBytes[2:])

	// remove first byte from the value written to KeyName. This is an extra byte written.
	originalNameDigest, err = b.StdEncoding.DecodeString(keyInfo.KeyName)
	if err != nil {
		return httpRequestBody, errors.New("errror decoding name digest. " + err.Error())
	}
	originalNameDigest = originalNameDigest[1:]

	//append 0 added as padding
	for i := 0; i < 34; i++ {
		originalNameDigest = append(originalNameDigest, 0)
	}

	nameDigest := b.StdEncoding.EncodeToString(originalNameDigest)

	//get trustagent aik cert location
	//TODO Vinil
	aikCertName, _ := osutil.MakeFilePathFromEnvVariable(consts.TrustAgentConfigDirEnv, "aik.pem", true)

	//set tpm version
	//TODO Vinil
	if keyInfo.Version == 2 {
		tpmVersion = "2.0"
	} else {
		tpmVersion = "1.2"
	}

	aikCert, err := ioutil.ReadFile(aikCertName)
	if err != nil {
		return httpRequestBody, errors.New("error reading certificate file. " + err.Error())
	}
	aikDer, _ := pem.Decode(aikCert)
	_, err = x509.ParseCertificate(aikDer.Bytes)
	if err != nil {
		return httpRequestBody, errors.New("error parsing certificate file. " + err.Error())
	}

	//construct request body
	httpRequestBody = mtwilson.RegisterKeyInfo{
		PublicKeyModulus:       keyInfo.PublicKey,
		TpmCertifyKey:          tpmCertifyKey,
		TpmCertifyKeySignature: keyInfo.KeySignature,
		AikDerCertificate:      aikDer.Bytes,
		NameDigest:             nameDigest,
		TpmVersion:             tpmVersion,
		OsType:                 strings.Title(runtime.GOOS),
	}

	return httpRequestBody, nil
}
func WriteKeyCertToDisk(keyCertPath string, aikPem string) error {
	file, err := os.Create(keyCertPath)
	if err != nil {
		return errors.New("error creating file. " + err.Error())
	}
	_, err = file.Write([]byte(aikPem))
	if err != nil {
		return errors.New("error writing to file. " + err.Error())
	}
	return nil

}
