/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package flavor

import (
	"encoding/json"
	flvr "intel/isecl/lib/flavor"
	wlsclient "intel/isecl/wlagent/clients"

	log "github.com/sirupsen/logrus"
)

// OutFlavor is an struct containing return code and image flavor as output from RPC call
type OutFlavor struct {
	ReturnCode  bool
	ImageFlavor string
}

// Fetch method is used to fetch image flavor from workload-service
// Input Parameters: imageID string, flavorPart string
// Return: returns a boolean value to the docker plugin.
// true if the flavor is fetched successfully, else return false.
func Fetch(imageID, flavorPart string) (string, bool) {

	var flavor flvr.SignedImageFlavor

	// get image flavor from workload service
	flavor, err := wlsclient.GetImageFlavor(imageID, flavorPart)
	if err != nil {
		log.Infof("Error while retrieving the image flavor : %s", err)
		return "", false
	}

	if flavor.ImageFlavor.Meta.ID == "" {
		log.Info("Flavor does not exist for the image ", imageID)
		return "", true
	}

	f, _ := json.Marshal(flavor)
	return string(f), true
}
