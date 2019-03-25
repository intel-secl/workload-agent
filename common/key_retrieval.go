package common

import (
	pinfo "intel/isecl/lib/platform-info"
	"intel/isecl/wlagent/wlsclient"
        
	"strings"

	log "github.com/sirupsen/logrus"
)

// RetrieveKey retrieves an Image decryption key
// It uses the hardwareUUID that is fetched from the the Platform Info library
func RetrieveKey(imageUUID string) ([]byte, error) {

	//check if the key is cached by filtercriteria imageUUID
	var err error
	var keyID string
	var flavorKeyInfo wlsclient.FlavorKey
	var tpmWrappedKey []byte

	// get host hardware UUID
	log.Debug("Retrieving host hardware UUID...")
	hardwareUUID, err := pinfo.HardwareUUID()
	if err != nil {
		log.Error("Unable to get the host hardware UUID")
		return nil, err
	}
	log.Debugf("The host hardware UUID is :%s", hardwareUUID)

	//get flavor-key from workload service
	log.Infof("Retrieving image-flavor-key for image %s from WLS", imageUUID)
	flavorKeyInfo, err = wlsclient.GetImageFlavorKey(imageUUID, hardwareUUID, keyID)
	if err != nil {
		log.Errorf("Error retrieving the image flavor and key: %s", err.Error())
		return nil, err
	}

	if flavorKeyInfo.Image.Meta.ID == "" {
		log.Infof("Flavor does not exist for the image %s", imageUUID)
		// check with Ryan
		return nil, nil
	}

	if flavorKeyInfo.Image.EncryptionRequired {
		// if key not cached, cache the key
		keyURLSplit := strings.Split(flavorKeyInfo.Image.Encryption.KeyURL, "/")
		keyID = keyURLSplit[len(keyURLSplit)-2]
		// if the WLS response includes a key, cache the key on host
		if len(flavorKeyInfo.Key) > 0 {
			// get the key from WLS response
			tpmWrappedKey = flavorKeyInfo.Key
			return tpmWrappedKey, nil
		}

		return nil, nil
	}

	return nil, nil
}

