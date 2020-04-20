/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package flavor

import (
	"encoding/json"
	cLog "intel/isecl/lib/common/v2/log"
	pinfo "intel/isecl/lib/platform-info/v2/platforminfo"
	wlsclient "intel/isecl/wlagent/v2/clients"
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

// Fetch method is used to fetch image flavor key from workload-service
// Input Parameters: imageID string, Hardware UUID
// Return: returns a boolean value to the secure docker plugin.
// true if the flavorkey is fetched successfully, else return false.
func Fetch(imageID string) (string, bool) {
	log.Trace("flavor/flavor:Fetch Entering")
	defer log.Trace("flavor/flavor:Fetch Leaving")
	var flavorKeyInfo wlsclient.FlavorKey

	log.Debug("Retrieving host hardware UUID...")
	hardwareUUID, err := pinfo.HardwareUUID()
	if err != nil {
		log.Error("flavor/key_retrieval.go:Fetch() unable to get the host hardware UUID")
		log.Tracef("%+v", err)
		return "", false
	}
	log.Debugf("The host hardware UUID is :%s", hardwareUUID)
	// get image flavor key from workload service
	flavorKeyInfo, err = wlsclient.GetImageFlavorKey(imageID, hardwareUUID)
	if err != nil {
		secLog.WithError(err).Error("flavor/flavor.go:Fetch() Error while retrieving the image flavor")
		return "", false
	}

	if flavorKeyInfo.Flavor.Meta.ID == "" {
		log.Infof("Flavor does not exist for the image: %s", imageID)
		return "", true
	}

	if flavorKeyInfo.Flavor.EncryptionRequired {
		keyID := getKeyID(flavorKeyInfo.Flavor.Encryption.KeyURL)
		imageKeyID[keyID] = imageID
		if len(flavorKeyInfo.Key) == 0{
			secLog.Error("Could not retrieve flavor Key, Host is untrusted or key doesnt exist with associated flavor")
			return "", false
		}
	}

	f, err := json.Marshal(flavorKeyInfo.Flavor)
	if err != nil {
                log.WithError(err).Error("flavor/flavor.go:Fetch() Error while marshalling flavor")
                return "", false
        }

	return string(f), true
}
