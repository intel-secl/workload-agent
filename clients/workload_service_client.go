/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package clients

import (
	"github.com/intel-secl/intel-secl/v3/pkg/clients/wlsclient"
	wlsModel "github.com/intel-secl/intel-secl/v3/pkg/model/wls"
	"intel/isecl/wlagent/v2/config"
	"intel/isecl/wlagent/v2/consts"
	"net/url"

	"github.com/pkg/errors"
)

// GetImageFlavorKey method is used to get the image flavor-key from the workload service
func GetImageFlavorKey(imageUUID, hardwareUUID string) (wlsModel.FlavorKey, error) {
	log.Trace("clients/workload_service_client:GetImageFlavorKey() Entering")
	defer log.Trace("clients/workload_service_client:GetImageFlavorKey() Leaving")
	var flavorKeyInfo wlsModel.FlavorKey
	wlsClientFactory, err := wlsclient.NewWLSClientFactory(config.Configuration.Wls.APIURL, config.Configuration.Aas.BaseURL,
		config.Configuration.Wla.APIUsername, config.Configuration.Wla.APIPassword, consts.TrustedCaCertsDir)
	if err != nil {
		return flavorKeyInfo, errors.Wrap(err, "Error while instantiating WLSClientFactory")

	}

	flavorsClient, err := wlsClientFactory.FlavorsClient()
	if err != nil {
		return flavorKeyInfo, errors.Wrap(err, "Error while instantiating FlavorsClient")
	}

	flavorKeyInfo, err = flavorsClient.GetImageFlavorKey(imageUUID, hardwareUUID)
	if err != nil {
		return flavorKeyInfo, errors.Wrap(err, "Error while retrieving Flavor-Key")
	}

	log.Debug("client/workload_service_client:GetImageFlavorKey() Successfully retrieved Flavor-Key")
	return flavorKeyInfo, nil
}

// GetImageFlavor method is used to get the image flavor from the workload service
func GetImageFlavor(imageID, flavorPart string) (wlsModel.SignedImageFlavor, error) {
	log.Trace("clients/workload_service_client:GetImageFlavor() Entering")
	defer log.Trace("clients/workload_service_client:GetImageFlavor() Leaving")
	var flavor wlsModel.SignedImageFlavor

	wlsClientFactory, err := wlsclient.NewWLSClientFactory(config.Configuration.Wls.APIURL, config.Configuration.Aas.BaseURL,
		config.Configuration.Wla.APIUsername, config.Configuration.Wla.APIPassword, consts.TrustedCaCertsDir)
	if err != nil {
		return flavor, errors.Wrap(err, "Error while instantiating WLSClientFactory")

	}

	flavorsClient, err := wlsClientFactory.FlavorsClient()
	if err != nil {
		return flavor, errors.Wrap(err, "Error while instantiating FlavorsClient")
	}

	flavor, err = flavorsClient.GetImageFlavor(imageID, flavorPart)
	if err != nil {
		return flavor, errors.Wrap(err, "Error while getting ImageFlavor")
	}

	return flavor, nil
}

//PostVMReport method is used to upload the VM trust report to workload service
func PostVMReport(report []byte) error {
	log.Trace("clients/workload_service_client:PostVMReport() Entering")
	defer log.Trace("clients/workload_service_client:PostVMReport() Leaving")
	var err error

	wlsClientFactory, err := wlsclient.NewWLSClientFactory(config.Configuration.Wls.APIURL, config.Configuration.Aas.BaseURL,
		config.Configuration.Wla.APIUsername, config.Configuration.Wla.APIPassword, consts.TrustedCaCertsDir)
	if err != nil {
		return errors.Wrap(err, "Error while instantiating WLSClientFactory")

	}

	reportsClient, err := wlsClientFactory.ReportsClient()
	if err != nil {
		return errors.Wrap(err, "Error while instantiating ReportsClient")
	}

	err = reportsClient.PostVMReport(report)
	if err != nil {
		return errors.Wrap(err, "Error creating instance trust report")
	}
	return nil
}

// GetKeyWithURL method is used to get the image flavor-key from the workload service
func GetKeyWithURL(keyUrl string, hardwareUUID string) (wlsModel.ReturnKey, error) {
	log.Trace("clients/workload_service_client:GetKeyWithURL() Entering")
	defer log.Trace("clients/workload_service_client:GetKeyWithURL() Leaving")
	var retKey wlsModel.ReturnKey

	requestURL, err := url.Parse(config.Configuration.Wls.APIURL)
	if err != nil {
		return retKey, errors.New("client/workload_service_client:GetKeyWithURL() error retrieving WLS API URL")
	}

	requestURL, err = url.Parse(requestURL.String() + "keys")
	if err != nil {
		return retKey, errors.New("client/workload_service_client:GetKeyWithURL() error forming GET key API URL")
	}
	wlsClientFactory, err := wlsclient.NewWLSClientFactory(config.Configuration.Wls.APIURL, config.Configuration.Aas.BaseURL,
		config.Configuration.Wla.APIUsername, config.Configuration.Wla.APIPassword, consts.TrustedCaCertsDir)
	if err != nil {
		return retKey, errors.Wrap(err, "Error while instantiating WLSClientFactory")

	}

	keysClient, err := wlsClientFactory.KeysClient()
	if err != nil {
		return retKey, errors.Wrap(err, "Error while instantiating KeysClient")
	}

	retKey, err = keysClient.GetKeyWithURL(keyUrl, hardwareUUID)
	if err != nil {
		return retKey,errors.Wrap(err, "Error while getting key")
	}
	log.Debug("client/workload_service_client:GetKeyWithURL() Successfully retrieved Key")
	return retKey, nil
}
