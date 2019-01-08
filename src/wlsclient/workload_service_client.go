package wlsclient

import (
	"encoding/json"
	"errors"
	f "intel/isecl/lib/flavor"
	"intel/isecl/wlagent/wlaconfig"
	"intel/isecl/lib/verifier"
	"log"
	"net/http"
	"bytes"
	"strings"
)

//FlavorKeyInfo is a representation of flavor-key information
type FlavorKeyInfo struct {
	Flavor    f.ImageFlavor `json:"flavor"`
	Key       []byte `json:"key"`
}

// GetImageFlavorKey method is used to get the image flavor-key from the workload service
func GetImageFlavorKey(imageUUID, hardwareUUID, keyID string) (FlavorKeyInfo, error){
	requestURL := wlaconfig.WlaConfig.WlsURL + "images/" + imageUUID +"/flavor-key?hardware_uuid=" + hardwareUUID

	if len(strings.TrimSpace(keyID)) >0 {
		requestURL = requestURL + "&&keyId=" + keyID
	}

	httpRequest, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		log.Fatal(err)
	}
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Content-Type", "application/json")
	//httpRequest.Header.Set("Authorization", "Basic "+token)
	var flavorKeyInfo FlavorKeyInfo

	httpResponse, err := SendRequest(httpRequest)
	if err != nil {
		return flavorKeyInfo, errors.New("error while getting http response")
	}

	//deserialize the response to UserInfo response
	err = json.Unmarshal([]byte(httpResponse), &flavorKeyInfo)
	if err != nil {
		return flavorKeyInfo, errors.New("error while unmarshalling the http response to the type flavor-key")
	}
	return flavorKeyInfo, nil

}

//PostVMReport method is used to upload the VM trust report to workload service
func PostVMReport(vmTrustReport verifier.VMTrustReport) error {
	var err error
	var url string
	var requestBody bytes.Buffer

	//Add client here
	url = wlaconfig.WlaConfig.WlsURL + "/reports"

	//build request body using username and password from config
	requestBody.WriteString(``)
	
	// set POST request Accept and Content-Type headers
	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(requestBody.String())))
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Content-Type", "application/json")

	_, err = SendRequest(httpRequest)
	if err != nil {
		return err
	}

	return nil

}
