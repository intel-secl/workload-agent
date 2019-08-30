/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package key

import (
	"intel/isecl/wlagent/common"
        "intel/isecl/wlagent/util"
	log "github.com/sirupsen/logrus"
)

// Cache method is used to fetch decryption key from workload-service and save it in keycache
// Input parameters: imageID string, keyID string
// Return: returns a boolean value to the docker plugin.
// true if the key is fetched and saved successfully, else return false.
func Cache(imageID, keyID string) bool {

	// Check for key in key cache, if the key is already present then will not be stored again
        _, err := util.GetKeyFromCache(keyID)
	if err != nil {
		// Fetch key from workload service
		key, err := common.RetrieveKey(imageID)
		if err != nil {
			log.Info("Error while retrieving the image decryption key")
			return false
		}

		if key != nil {
		        err := util.CacheKeyInMemory(keyID, key)
			if err != nil {
				log.Infof("Unable to store the key in the key cache: %s", err)
				return false
			}
		}
	}

	log.Info("Key is present in KeyCache")
	return true
}
