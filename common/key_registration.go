package common

import (
	"crypto/x509"
	b "encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	exec "intel/isecl/lib/common/exec"
	mtwilson "intel/isecl/lib/mtwilson-client"
	"intel/isecl/wlagent/consts"
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

func CreateRequest(keyfilePath string) (*mtwilson.RegisterKeyInfo, error) {
	var httpRequestBody *mtwilson.RegisterKeyInfo
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

	// TODO remove hack below. This hack was added since key stored on disk needs to be modified
	// so that HVS can register the key.
	// ISECL - 3506 opened to address this issue later

	// remove first two bytes from KeyAttestation. These are extra bytes written.
	tpmCertifyKeyBytes, err := b.StdEncoding.DecodeString(keyInfo.KeyAttestation)
	if err != nil {
		return httpRequestBody, errors.New("errror decoding name digest. " + err.Error())
	}
	tpmCertifyKey := b.StdEncoding.EncodeToString(tpmCertifyKeyBytes[2:])

	// remove first byte from the value written to KeyName. This is an extra byte written.
	originalNameDigest, err = b.StdEncoding.DecodeString(keyInfo.KeyName)
	if err != nil {
		return httpRequestBody, errors.New("errror decoding key name. " + err.Error())
	}
	originalNameDigest = originalNameDigest[1:]

	//pad zeroes
	zeroes := make([]byte, 34)
	originalNameDigest = append(originalNameDigest, zeroes[:]...)

	nameDigest := b.StdEncoding.EncodeToString(originalNameDigest)

	//get trustagent aik cert location
	//TODO Vinil
	aikCertName, _ := exec.MkDirFilePathFromEnvVariable(consts.TrustAgentConfigDirEnv, "aik.pem", true)

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
	httpRequestBody = &mtwilson.RegisterKeyInfo{
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
func WriteKeyCertToDisk(keyCertPath string, aikPem []byte) error {
	file, err := os.Create(keyCertPath)
	if err != nil {
		return errors.New("error creating file. " + err.Error())
	}
	if err = pem.Encode(file, &pem.Block{Type: consts.PemPublicKeyHeader, Bytes: aikPem}); err != nil {
		return errors.New("error writing certificate to file")
	}
	return nil

}
