package wlsclient

import (
	"crypto/tls"
	"encoding/hex"
	t "intel/isecl/lib/common/tls"
	"intel/isecl/wlagent/config"
	"io/ioutil"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
)

//SendRequest method is used to create an http client object and send the request to the server
func SendRequest(req *http.Request, insecureConnection bool) ([]byte, error) {
	var certificateDigest [32]byte

	cert, err := hex.DecodeString(config.Configuration.Wls.TLSSha256)
	if err != nil {
		log.Error("Error decoding WLS TLS.")
		return nil, err
	}

	copy(certificateDigest[:], cert)
	var tlsConfig tls.Config

	if insecureConnection {
		tlsConfig = tls.Config{
			InsecureSkipVerify: true,
			//VerifyPeerCertificate: t.VerifyCertBySha256(certificateDigest),
		}
	} else {
		tlsConfig = tls.Config{
			InsecureSkipVerify:    true,
			VerifyPeerCertificate: t.VerifyCertBySha256(certificateDigest),
		}
	}

	transport := http.Transport{
		TLSClientConfig: &tlsConfig,
	}
	client := &http.Client{
		Transport: &transport,
	}
	response, err := client.Do(req)
	if err != nil {
		log.Error("Error in sending request.", err)
		return nil, err
	}
	defer response.Body.Close()

	//create byte array of HTTP response body
	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return nil, err
	}

	log.Info("status code returned : ", strconv.Itoa(response.StatusCode))
	return body, nil
}
