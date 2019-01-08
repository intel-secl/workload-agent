package wlavm

import (
	"log"
	"os"
	"strings"
	"intel/isecl/wlagent/wlsclient"
	"intel/isecl/lib/vml"
	"intel/isecl/lib/verifier"
	pinfo "intel/isecl/lib/platform-info"
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/wlaconfig"
	"os/exec"
	"encoding/base64"
	"io/ioutil"
	"encoding/json"
	"strconv"
)

const (
	// move this wlaconfig as properties
	mountLocation = "/mnt/crypto/"
	devMapperLocation = "/dev/mapper/"
)

// Start method is used perform the VM confidentiality check before lunching the VM
func Start(instanceUUID, imageUUID, imagePath, instancePath, diskSize string) int {
	// validate input parameters
	if len(strings.TrimSpace(instancePath)) <= 0 {
		log.Println("instance path not given")
		return 1
	}

	if len(strings.TrimSpace(instanceUUID)) <= 0 {
		log.Println("instance UUID not given")
		return 1
	}

	if len(strings.TrimSpace(imagePath)) <= 0 {
		log.Println("image path not given")
		return 1
	}

	if len(strings.TrimSpace(imageUUID)) <= 0 {
		log.Println("image UUID not given")
		return 1
	}

	if len(strings.TrimSpace(diskSize)) <= 0 {
		log.Println("sparse file size not given")
		return 1
	}

	//check if image exists in the given location
	_, err := os.Stat(imagePath)
	if os.IsNotExist(err) {
		log.Println("image does not exist in location ", imagePath)
		return 1
	}

	//check if image is encrypted
	fileOutput, err := exec.Command("file", imagePath).Output()
	if err != nil {
		log.Println("Error while checking if the image is encrypted")
		return 1
	}
	
	outputFormat := strings.Split(string(fileOutput), ":")
    imageFormat := outputFormat[len(outputFormat)-1]
	
	if imageFormat != "data" {
		log.Println("The image is not encrypted")
		return 0
	}


	//check if the key is cached by filtercriteria imageUUID
	var keyID string
	var flavorKeyInfo wlsclient.FlavorKeyInfo
	var keyPath string

	// get host hardware UUID
	hardwareUUID,err := pinfo.HardwareUUID()
	if err != nil {
		log.Println("Unable to get the host hardware UUID")
		return 1
	}
	log.Println("The host hardware UUID is :", hardwareUUID)
	//get flavor-key from workload service
	flavorKeyInfo, err = wlsclient.GetImageFlavorKey(imageUUID, hardwareUUID, keyID)
	if err != nil {
		log.Println("Error while retrieving the image flavor and key")
		return 1
	}

	// if key not cached, cache the key

	if flavorKeyInfo.Flavor.Image.Meta.ID == "" {
		log.Println("Flavor does not exist for the image ", imageUUID)
		return 0
	}

	if (flavorKeyInfo.Flavor.Image.Encryption.EncryptionRequired) {
		// unwrap key
		unWrappedKey, err := unwrapKey(flavorKeyInfo.Key)
		if err != nil {
			log.Println("Error while unwrapping the key")
			return 1
		}

		// write the key to a temp file on disk
		keyPath = wlaconfig.GetConfigDir() + "key"
		_ = ioutil.WriteFile(keyPath, unWrappedKey, 0600)
		
		// create image volume
		deviceMapperLocation := devMapperLocation + imageUUID
		err = vml.CreateVolume(imagePath, deviceMapperLocation, keyPath, diskSize)
		if err != nil {
			log.Println("Error while creating image dm-crypt volume")
			return 1
		}

		//check if the image is mounted
		imageMountLocation := mountLocation + "imageUUID"
		_, err = os.Stat(imageMountLocation)
		if os.IsNotExist(err) {
			log.Println("Mounting the image")
			mountErr := vml.Mount(deviceMapperLocation, imageMountLocation)
			if mountErr != nil {
				log.Println("Error while mounting the image")
				return 1
			}
		}

		//decrypt the image
		err = vml.Decrypt(imagePath, deviceMapperLocation, keyPath)
		if err != nil {
			log.Println("Error while decrypting the image")
			return 1
		}

		// create symlink between the image and the dm-crypt volume
		err = os.Symlink(imagePath, deviceMapperLocation)
		if err != nil {
			log.Println("Error while creating symbolic link")
			return 1
		}

		// create instance volume
		instanceDeviceMapperLocation := devMapperLocation + instanceUUID		
		err = vml.CreateVolume(instancePath, instanceDeviceMapperLocation, keyPath, diskSize)
		if err != nil {
			log.Println("Error while creating instance dm-crypt volume")
			return 1
		}
	}

	//create VM manifest
	manifest, err := vml.CreateVMManifest(instanceUUID, hardwareUUID, imageUUID, true) 
	if err != nil {
		log.Println("Error while creating VM manifest")
		return 1
	}

	//create VM report
	vmTrustReport, err := verifier.Verify(&manifest, flavorKeyInfo.Flavor)
	if err != nil {
		log.Println("Error while creating VM manifest")
		return 1
	}

	//post VM report on to workload service
	err = wlsclient.PostVMReport(vmTrustReport.(verifier.VMTrustReport))
	if err!= nil {
		log.Println("Error while posting the VM trust report on to workload service")
		return 1
	}

	//associate instance UUID with the image UUID
	err = imageInstanceAssociation(imageUUID, imagePath)
	if err != nil {
		log.Println("Error while ")
		return 1
	}

	// delete the temp key file
	_ = os.Remove(keyPath)

	return 0
}

func unwrapKey(tpmWrappedKey []byte) ([]byte, error) {
	
	var certifiedKey tpm.CertifiedKey
	var key []byte
	var err error
	t, err := tpm.Open()

	if t != nil {
		defer t.Close()
		
		bindingKeyFile := wlaconfig.GetConfigDir() + wlaconfig.GetBindingKeyFileName()
		fileContents, fileErr := ioutil.ReadFile(bindingKeyFile)
		if fileErr != nil {
			log.Println("Error while reading the binding key certificate")
			return key, err
		}
		jsonErr := json.Unmarshal([]byte(fileContents), &certifiedKey)
		if jsonErr != nil {
			log.Println("Error while unmarshalling the binding key file contents to TPM CertifiedKey object")
			return key, err
		}

		keyAuth,_ := base64.StdEncoding.DecodeString(wlaconfig.WlaConfig.WlsBindingKeySecret)
		key, unbindErr := t.Unbind(&certifiedKey, keyAuth, tpmWrappedKey)
		if unbindErr != nil {
			log.Println("Error while unbinding the tpm wrapped key")
			return key, err
		}
	}
	return key, nil
}

func imageInstanceAssociation(imageUUID, imagePath string) error {

	imageUUIDFound := false
	fileName := ""
	// read the contents of the file
	output, err := exec.Command("cat", fileName).Output()
	if err != nil {
		log.Println("Error while reading the contents of the file")
		return err
	}

	fileContents := strings.Split(string(output), "\n")
	for i, lineContent := range fileContents {
		if strings.Contains(lineContent, imageUUID) {
			// increment the count and replace the count in the string
			contentArray := strings.Split(lineContent, "\t")
			countSection := contentArray[len(contentArray)-1]
			splitCountSection := strings.Split(countSection, ":")
			currentCount, _ := strconv.Atoi(splitCountSection[len(splitCountSection)-1])
			replaceString := strconv.Itoa(i+1) + " s/count:" + strconv.Itoa(currentCount) + "/count:" + strconv.Itoa(currentCount+1) + "/"
			_, sedErr := exec.Command("sed", "-i", replaceString, fileName).Output()
			if sedErr != nil {
				log.Println("Error while replacing the count of the instance for an image")
				return err
			}
			imageUUIDFound = true
			break
		}

	}

	if !imageUUIDFound {
		data := imageUUID + "\t" + imagePath + "\t" + "count:" + strconv.Itoa(1) + "\n"

		f, err := os.OpenFile(fileName, os.O_APPEND, 0600)
		if err != nil {
			log.Println("Error while opening image-instance information")
			return err
		}

		defer f.Close()
		if _, err = f.WriteString(data); err != nil {
			log.Println("Error while writing image-instance information")
			return err
		}
	}

	return nil

}