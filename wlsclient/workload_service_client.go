package wlsclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"intel/isecl/lib/flavor"
	"intel/isecl/wlagent/config"
	"net/http"
	"net/url"
	"strings"

	log "github.com/sirupsen/logrus"
)

//FlavorKey is a representation of flavor-key information
type FlavorKey struct {
	Flavor    flavor.Image `json:"flavor"`
	Signature string       `json:"signature"`
	Key       []byte       `json:"key"`
}

// GetImageFlavorKey method is used to get the image flavor-key from the workload service
func GetImageFlavorKey(imageUUID, hardwareUUID, keyID string) (FlavorKey, error) {
	var flavorKeyInfo FlavorKey

	requestURL, err := url.Parse(config.Configuration.Wls.APIURL)
	if err != nil {
		return flavorKeyInfo, errors.New("error retrieving WLS API URL")
	}

	requestURL, err = url.Parse(requestURL.String() + "images/" + imageUUID + "/flavor-key?hardware_uuid=" + hardwareUUID)
	if err != nil {
		return flavorKeyInfo, errors.New("error forming GET flavor-key for image API URL")
	}

	if len(strings.TrimSpace(keyID)) > 0 {
		requestURL, err = url.Parse(requestURL.String() + "&&key_id=" + keyID)
		if err != nil {
			return flavorKeyInfo, errors.New("error forming GET flavor-key for image API URL")
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

// GetImageFlavor method is used to get the image flavor from the workload service
func GetImageFlavor(imageID, flavorPart string) (flavor.SignedImageFlavor, error) {

	var flavor flavor.SignedImageFlavor

	requestURL, err := url.Parse(config.Configuration.Wls.APIURL)
	if err != nil {
		return flavor, errors.New("error retrieving WLS API URL")
	}

	requestURL, err = url.Parse(requestURL.String() + "images/" + imageID + "/flavors?flavor_part=" + flavorPart)
	if err != nil {
		return flavor, errors.New("error forming GET flavors for image API URL")
	}

	httpRequest, err := http.NewRequest("GET", requestURL.String(), nil)
	if err != nil {
		return flavor, err
	}

	log.Debugf("WLS image-flavor retrieval GET request URL: %s", requestURL.String())
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Content-Type", "application/json")

	httpResponse, err := SendRequest(httpRequest, true)
	if err != nil {
		return flavor, err
	}
	
	// deserialize the response to ImageFlavor response
	if(httpResponse != nil){
		err = json.Unmarshal(httpResponse, &flavor)
		if err != nil {
			return flavor, err
		}
	}
	log.Debugf("response from API: %s", string(httpResponse))

	return flavor, nil
}

//PostVMReport method is used to upload the VM trust report to workload service
func PostVMReport(report []byte) error {
	var err error

	//Add client here
	requestURL, err := url.Parse(config.Configuration.Wls.APIURL)
	if err != nil {
		return errors.New("error retrieving WLS API URL")
	}

	requestURL, err = url.Parse(requestURL.String() + "reports")
	if err != nil {
		return errors.New("error forming reports POST API URL")
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
