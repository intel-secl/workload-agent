/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */

package flavor

import (
	pinfo "intel/isecl/lib/platform-info/v2/platforminfo"
	wlsclient "intel/isecl/wlagent/v2/clients"
)

// RetrieveKey retrieves an Image decryption key
// It uses the hardwareUUID that is fetched from the the Platform Info library
func RetrieveKey(keyID string) ([]byte, bool) {
	log.Trace("flavor/key_retrieval:RetrieveKey Entering")
	defer log.Trace("flavor/key_retrieval:RetrieveKey Leaving")
	//check if the key is cached by filtercriteria imageUUID
	var err error
	var flavorKeyInfo wlsclient.FlavorKey
	var tpmWrappedKey []byte

	if (imageKeyID[keyID] == "") {
		log.Errorf("flavor/key_retrieval.go:RetrieveKey() unable to get the image ID for given key ID %s", keyID)
                return nil, false
	}
	imageUUID := imageKeyID[keyID]

	// get host hardware UUID
	log.Debug("Retrieving host hardware UUID...")
	hardwareUUID, err := pinfo.HardwareUUID()
	if err != nil {
		log.Error("flavor/key_retrieval.go:RetrieveKey() unable to get the host hardware UUID")
		log.Tracef("%+v", err)
		return nil, false
	}
	log.Debugf("The host hardware UUID is :%s", hardwareUUID)

	//get flavor-key from workload service
	log.Infof("Retrieving image-flavor-key for image %s from WLS", imageUUID)
	flavorKeyInfo, err = wlsclient.GetImageFlavorKey(imageUUID, hardwareUUID)
	if err != nil {
		log.Errorf("flavor/key_retrieval.go:RetrieveKey() error retrieving the image flavor and key: %s", err.Error())
		log.Tracef("%+v", err)
		return nil, false
	}

	if flavorKeyInfo.Flavor.Meta.ID == "" {
		log.Infof("Flavor does not exist for the image %s", imageUUID)
		return nil, true
	}

	if flavorKeyInfo.Flavor.EncryptionRequired {
		// if the WLS response includes a key, cache the key on host
		if len(flavorKeyInfo.Key) > 0 {
			// get the key from WLS response
			tpmWrappedKey = flavorKeyInfo.Key
			return tpmWrappedKey, true
		}

		return nil, false
	}

	return nil, false
}
