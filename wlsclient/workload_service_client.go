package wlsclient

import (
	"encoding/json"
	"errors"
	f "intel/isecl/lib/flavor"
	"intel/isecl/wlagent/config"
	"log"
	"net/http"
	"bytes"
	"strings"
	"fmt"
)

//FlavorKey is a representation of flavor-key information
type FlavorKey struct {
	f.ImageFlavor
	Key []byte `json:"key"`
}

// GetImageFlavorKey method is used to get the image flavor-key from the workload service
func GetImageFlavorKey(imageUUID, hardwareUUID, keyID string) (FlavorKey, error) {
	requestURL := config.Configuration.Wls.APIURL + "images/" + imageUUID +"/flavor-key?hardware_uuid=" + hardwareUUID

	if len(strings.TrimSpace(keyID)) > 0 {
		requestURL = requestURL + "&&keyId=" + keyID
	}

	httpRequest, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("RequestURL: ", requestURL)
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Content-Type", "application/json")
	//httpRequest.SetBasicAuth(config.WlaConfig.WlsAPIUsername, config.WlaConfig.WlsAPIPassword)

	var flavorKeyInfo FlavorKey

	httpResponse, err := SendRequest(httpRequest, true)
	if err != nil {
		return flavorKeyInfo, errors.New("error while getting http response")
	}

	//deserialize the response to UserInfo response
	err = json.Unmarshal(httpResponse, &flavorKeyInfo)
	if err != nil {
		return flavorKeyInfo, errors.New("error while unmarshalling the http response to the type flavor-key")
	}

	fmt.Println("response from API: ", string(httpResponse))

	return flavorKeyInfo, nil

}

//PostVMReport method is used to upload the VM trust report to workload service
func PostVMReport(report []byte) error {
	var err error
	var requestURL string

	//Add client here
	requestURL = config.Configuration.Wls.APIURL + "reports"

	fmt.Println("RequestURL: ", requestURL)
	// set POST request Accept and Content-Type headers
	httpRequest, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(report))
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Content-Type", "application/json")
	//httpRequest.SetBasicAuth(config.WlaConfig.WlsAPIUsername, config.WlaConfig.WlsAPIPassword)

	_, err = SendRequest(httpRequest, true)
	if err != nil {
		return err
	}

	return nil

}
