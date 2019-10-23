/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package flavor

import (
	"encoding/json"
	cLog "intel/isecl/lib/common/log"
	flvr "intel/isecl/lib/flavor"
	wlsclient "intel/isecl/wlagent/clients"
	"strings"
)

var log = cLog.GetDefaultLogger()
var secLog = cLog.GetSecurityLogger()

// imageKeyID is a map of keyID and imageUUID, the secureoverlay2 driver is unaware of image uuid.
// Secureoverlay has only information of keyID of each layer.
// The secure docker daemon passes the keyid to workload agent for fetching the key
// which in turn usees the image uuid for fetching the flavor key
var imageKeyID map[string]string

// OutFlavor is an struct containing return code and image flavor as output from RPC call
type OutFlavor struct {
	ReturnCode  bool
	ImageFlavor string
}

func getKeyID(keyURL string) string {

	keyURLSplit := strings.Split(keyURL, "/")
	keyID := keyURLSplit[len(keyURLSplit)-2]
	return keyID
}

func init() {
	imageKeyID = make(map[string]string)
}

// Fetch method is used to fetch image flavor from workload-service
// Input Parameters: imageID string, flavorPart string
// Return: returns a boolean value to the docker plugin.
// true if the flavor is fetched successfully, else return false.
func Fetch(imageID, flavorPart string) (string, bool) {
	log.Trace("flavor/flavor:Fetch Entering")
	defer log.Trace("flavor/flavor:Fetch Leaving")
	var flavor flvr.SignedImageFlavor

	// get image flavor from workload service
	flavor, err := wlsclient.GetImageFlavor(imageID, flavorPart)
	if err != nil {
		secLog.WithError(err).Error("flavor/flavor.go:Fetch() Error while retrieving the image flavor")
		return "", false
	}

	if flavor.ImageFlavor.Meta.ID == "" {
		log.Infof("Flavor does not exist for the image: %s", imageID)
		return "", true
	}
	if flavor.ImageFlavor.EncryptionRequired {
		keyID := getKeyID(flavor.ImageFlavor.Encryption.KeyURL)
		imageKeyID[keyID] = imageID
	}
	f, _ := json.Marshal(flavor)
	return string(f), true
}
