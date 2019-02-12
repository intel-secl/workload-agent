package wlsclient

import (
	"bytes"
	"encoding/json"
	f "intel/isecl/lib/flavor"
	"intel/isecl/lib/verifier"
	"intel/isecl/wlagent/config"
	"net/http"
	"strings"
	log "github.com/sirupsen/logrus"
	"net/url"
)

//FlavorKey is a representation of flavor-key information
type FlavorKey struct {
	f.ImageFlavor
	Key []byte `json:"key"`
}

// GetImageFlavorKey method is used to get the image flavor-key from the workload service
func GetImageFlavorKey(imageUUID, hardwareUUID, keyID string) (FlavorKey, error) {
	var flavorKeyInfo FlavorKey
	
	requestURL, err := url.Parse(config.Configuration.Wls.APIURL + "images/" + imageUUID + "/flavor-key?hardware_uuid=" + hardwareUUID)
	if err != nil {
		return flavorKeyInfo, err
	}

	var flavorKeyInfo FlavorKey
	if len(strings.TrimSpace(keyID)) > 0 {
		requestURL, err = url.Parse("&&keyId=" + keyID)
		if err != nil {
			return flavorKeyInfo, err
		}
	}

	httpRequest, err := http.NewRequest("GET", requestURL.String(), nil)
	if err != nil {
		return flavorKeyInfo, err
	}

	log.Debugf("WLS image-flavor-key retrieval GET request URL: %s", requestURL.String())
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Content-Type", "application/json")
	//httpRequest.SetBasicAuth(config.WlaConfig.WlsAPIUsername, config.WlaConfig.WlsAPIPassword)

	httpResponse, err := SendRequest(httpRequest, true)
	if err != nil {
		return flavorKeyInfo, err
	}

	//deserialize the response to UserInfo response
	err = json.Unmarshal(httpResponse, &flavorKeyInfo)
	if err != nil {
		return flavorKeyInfo, err
	}

	log.Debugf("response from API: %s", string(httpResponse))

	return flavorKeyInfo, nil

}

//PostVMReport method is used to upload the VM trust report to workload service
func PostVMReport(report []byte) error {
	var err error

	//Add client here
	requestURL, err := url.Parse(config.Configuration.Wls.APIURL + "reports")
	if err != nil {
		return err
	}

	log.Debugf("WLS VM reports POST Request URL: %s", requestURL.String())
	// set POST request Accept and Content-Type headers
	httpRequest, err := http.NewRequest("POST", requestURL.String(), bytes.NewBuffer(report))
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Content-Type", "application/json")
	//httpRequest.SetBasicAuth(config.WlaConfig.WlsAPIUsername, config.WlaConfig.WlsAPIPassword)

	_, err = SendRequest(httpRequest, true)
	if err != nil {
		return err
	}
	return nil
}
