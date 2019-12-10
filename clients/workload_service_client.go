/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package clients

import (
	"bytes"
	"encoding/json"
	"intel/isecl/lib/flavor"
	"intel/isecl/wlagent/config"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

//FlavorKey is a representation of flavor-key information
type FlavorKey struct {
	Flavor    flavor.Image `json:"flavor"`
	Signature string       `json:"signature"`
	Key       []byte       `json:"key"`
}

// GetImageFlavorKey method is used to get the image flavor-key from the workload service
func GetImageFlavorKey(imageUUID, hardwareUUID string) (FlavorKey, error) {
	log.Trace("clients/workload_service_client:GetImageFlavorKey() Entering")
	defer log.Trace("clients/workload_service_client:GetImageFlavorKey() Leaving")
	var flavorKeyInfo FlavorKey

	requestURL, err := url.Parse(config.Configuration.Wls.APIURL)
	if err != nil {
		return flavorKeyInfo, errors.New("client/workload_service_client:GetImageFlavorKey() error retrieving WLS API URL")
	}

	requestURL, err = url.Parse(requestURL.String() + "images/" + imageUUID + "/flavor-key?hardware_uuid=" + hardwareUUID)
	if err != nil {
		return flavorKeyInfo, errors.New("client/workload_service_client:GetImageFlavorKey() error forming GET flavor-key for image API URL")
	}

	httpRequest, err := http.NewRequest("GET", requestURL.String(), nil)
	if err != nil {
		return flavorKeyInfo, err
	}

	log.Debugf("clients/workload_service_client:GetImageFlavorKey() WLS image-flavor-key retrieval GET request URL: %s", requestURL.String())
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Content-Type", "application/json")
	//httpRequest.SetBasicAuth(config.WlaConfig.WlsAPIUsername, config.WlaConfig.WlsAPIPassword)

	httpResponse, err := SendRequest(httpRequest, true)
	if err != nil {
		secLog.WithError(err).Error("client/workload_service_client:GetImageFlavorKey() Error while getting response from Get Image Flavor-Key from WLS API")
		return flavorKeyInfo, errors.Wrap(err, "client/workload_service_client:GetImageFlavorKey() Error while getting response from Get Image Flavor-Key from WLS API")
	}

	if httpResponse != nil {
		//deserialize the response to UserInfo response
		err = json.Unmarshal(httpResponse, &flavorKeyInfo)
		if err != nil {
			return flavorKeyInfo, errors.Wrap(err, "client/workload_service_client:GetImageFlavorKey() Failed to unmarshal response into flavor key info")
		}
	}
	log.Debug("client/workload_service_client:GetImageFlavorKey() Successfully retrieved Flavor-Key")       
	return flavorKeyInfo, nil
}

// GetImageFlavor method is used to get the image flavor from the workload service
func GetImageFlavor(imageID, flavorPart string) (flavor.SignedImageFlavor, error) {
	log.Trace("clients/workload_service_client:GetImageFlavor() Entering")
	defer log.Trace("clients/workload_service_client:GetImageFlavor() Leaving")
	var flavor flavor.SignedImageFlavor

	requestURL, err := url.Parse(config.Configuration.Wls.APIURL)
	if err != nil {
		return flavor, errors.New("client/workload_service_client:GetImageFlavor() error retrieving WLS API URL")
	}

	requestURL, err = url.Parse(requestURL.String() + "images/" + imageID + "/flavors?flavor_part=" + flavorPart)
	if err != nil {
		return flavor, errors.New("client/workload_service_client:GetImageFlavor() error forming GET flavors for image API URL")
	}

	httpRequest, err := http.NewRequest("GET", requestURL.String(), nil)
	if err != nil {
		return flavor, err
	}

	log.Debugf("clients/workload_service_client:GetImageFlavor() WLS image-flavor retrieval GET request URL: %s", requestURL.String())
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Content-Type", "application/json")

	httpResponse, err := SendRequest(httpRequest, true)
	if err != nil {
		secLog.WithError(err).Error("client/workload_service_client:GetImageFlavor() Error in response from WLS GetImageFlavor API")
		return flavor, errors.Wrap(err, "client/workload_service_client:GetImageFlavor() Error in response from WLS GetImageFlavor API")
	}

	// deserialize the response to ImageFlavor response
	if httpResponse != nil {
		err = json.Unmarshal(httpResponse, &flavor)
		if err != nil {
			return flavor, errors.Wrap(err, "client/workload_service_client:GetImageFlavor() Failed to unmarshal response into flavor")
		}
	}
	log.Debugf("clients/workload_service_client:GetImageFlavor() response from API: %s", string(httpResponse))

	return flavor, nil
}

//PostVMReport method is used to upload the VM trust report to workload service
func PostVMReport(report []byte) error {
	log.Trace("clients/workload_service_client:PostVMReport() Entering")
	defer log.Trace("clients/workload_service_client:PostVMReport() Leaving")
	var err error

	//Add client here
	requestURL, err := url.Parse(config.Configuration.Wls.APIURL)
	if err != nil {
		return errors.New("client/workload_service_client:PostVMReport() error retrieving WLS API URL")
	}

	requestURL, err = url.Parse(requestURL.String() + "reports")
	if err != nil {
		return errors.New("client/workload_service_client:PostVMReport() error forming reports POST API URL")
	}

	log.Debugf("clients/workload_service_client:PostVMReport() WLS VM reports POST Request URL: %s", requestURL.String())
	// set POST request Accept and Content-Type headers
	httpRequest, err := http.NewRequest("POST", requestURL.String(), bytes.NewBuffer(report))
	if err != nil {
		return errors.Wrap(err, "client/workload_service_client:PostVMReport() Failed to create WLS POST API request for vm reports")
	}
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Content-Type", "application/json")
	//httpRequest.SetBasicAuth(config.WlaConfig.WlsAPIUsername, config.WlaConfig.WlsAPIPassword)

	_, err = SendRequest(httpRequest, true)
	if err != nil {
		secLog.WithError(err).Error("client/workload_service_client:PostVMReport() Error while getting response for Post WLS VM reports API")
		return errors.Wrap(err, "client/workload_service_client:PostVMReport() Error while getting response for Post WLS VM reports API")
	}
	return nil
}
