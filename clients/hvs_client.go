package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	csetup "intel/isecl/lib/common/setup"
	cLog "intel/isecl/lib/common/log"
	"intel/isecl/lib/clients"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/consts"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/pkg/errors"
)

var log = cLog.GetDefaultLogger()
var secLog = cLog.GetSecurityLogger()

// Error is an error struct that contains error information thrown by the actual HVS
type Error struct {
	StatusCode int
	Message    string
}

type RegisterKeyInfo struct {
	PublicKeyModulus       []byte `json:"public_key_modulus,omitempty"`
	TpmCertifyKey          []byte `json:"tpm_certify_key,omitempty"`
	TpmCertifyKeySignature []byte `json:"tpm_certify_key_signature,omitempty"`
	AikDerCertificate      []byte `json:"aik_der_certificate,omitempty"`
	NameDigest             []byte `json:"name_digest,omitempty"`
	TpmVersion             string `json:"tpm_version,omitempty"`
	OsType                 string `json:"operating_system,omitempty"`
}

type BindingKeyCert struct {
	BindingKeyCertificate []byte `json:"binding_key_der_certificate,omitempty"`
}
type SigningKeyCert struct {
	SigningKeyCertificate []byte `json:"signing_key_der_certificate,omitempty"`
}

func (e Error) Error() string {
	return fmt.Sprintf("hvs-client: failed (HTTP Status Code: %d)\nMessage: %s", e.StatusCode, e.Message)
}

// CertifyHostSigningKey sends a POST to /certify-host-signing-key to register signing key with HVS
func CertifyHostSigningKey(key *RegisterKeyInfo) (*SigningKeyCert, error) {
	log.Trace("clients/hvs_client:CertifyHostSigningKey() Entering")
	defer log.Trace("clients/hvs_client:CertifyHostSigningKey() Leaving")
	var keyCert SigningKeyCert

	rsp, err := certifyHostKey(key, "/rpc/certify-host-signing-key", "signing")
	if err != nil {
		return nil, errors.Wrap(err, "clients/hvs_client.go:CertifyHostSigningKey()  error registering signing key with HVS")
	}
	err = json.Unmarshal(rsp, &keyCert)
	if err != nil {
		return nil, errors.Wrap(err, "clients/hvs_client.go:CertifyHostSigningKey() error decoding signing key certificate")
	}
	return &keyCert, nil
}

// CertifyHostBindingKey sends a POST to /certify-host-binding-key to register binding key with HVS
func CertifyHostBindingKey(key *RegisterKeyInfo) (*BindingKeyCert, error) {
	log.Trace("clients/hvs_client:CertifyHostBindingKey Entering")
	defer log.Trace("clients/hvs_client:CertifyHostBindingKey Leaving")
	var keyCert BindingKeyCert
	rsp, err := certifyHostKey(key, "/rpc/certify-host-binding-key", "binding")
	if err != nil {
		return nil, errors.Wrap(err, "clients/hvs_client.go:CertifyHostBindingKey() error registering binding key with HVS")
	}
	err = json.Unmarshal(rsp, &keyCert)
	if err != nil {
		return nil, errors.Wrap(err, "clients/hvs_client.go:CertifyHostBindingKey() error decoding binding key certificate.")
	}
	return &keyCert, nil
}

func certifyHostKey(key *RegisterKeyInfo, endPoint string, keyUsage string) ([]byte, error) {
	log.Trace("clients/hvs_client:certifyHostKey Entering")
	defer log.Trace("clients/hvs_client:certifyHostKey Leaving")

	kiJSON, err := json.Marshal(key)
	if err != nil {
		return nil, errors.Wrapf(err, "clients/hvs_client.go:certifyHostKey() error marshalling %s key. ", keyUsage)
	}

	certifyKeyURL, err := url.Parse(config.Configuration.Mtwilson.APIURL)
	if err != nil {
		return nil, errors.Wrap(err, "clients/hvs_client.go:certifyHostKey() error parsing base url")
	}

	certifyKeyURL.Path = path.Join(certifyKeyURL.Path, endPoint)

	req, err := http.NewRequest("POST", certifyKeyURL.String(), bytes.NewBuffer(kiJSON))
	if err != nil {
		return nil, errors.Wrap(err, "clients/hvs_client.go:certifyHostKey() Failed to create request for certifying Binding/Signing Key")
	}

	var c csetup.Context
	jwtToken, err := c.GetenvSecret(consts.BEARER_TOKEN_ENV, "BEARER_TOKEN")
	if jwtToken == "" || err != nil {
		fmt.Fprintln(os.Stderr, "BEARER_TOKEN is not defined in environment")
		return nil, errors.Wrap(err, "BEARER_TOKEN is not defined in environment")
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ jwtToken)
	client, err := clients.HTTPClientWithCADir(consts.TrustedCaCertsDir)
	if err != nil {
		return nil, errors.Wrap(err, "clients/hvs_client.go:certifyHostKey() Failed to create http client")
	}
	rsp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "clients/hvs_client.go:certifyHostKey() Error from response")
	}
	if rsp == nil {
		return nil, &Error{Message: fmt.Sprintf("clients/hvs_client.go:certifyHostKey() Failed to register host %s key with HVS . Error : %s", keyUsage, err.Error())}
	}
	defer rsp.Body.Close()
	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "clients/send_http_request.go:SendRequest() Error from response")
	}

	return body, nil

}
