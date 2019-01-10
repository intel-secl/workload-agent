package wlavm

import (
	//"log"
	"os"
	"strings"
	"intel/isecl/wlagent/wlsclient"
	"intel/isecl/lib/vml"
	"intel/isecl/lib/verifier"
	pinfo "intel/isecl/lib/platform-info"
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/osutil"
	"os/exec"
	"encoding/base64"
	"io/ioutil"
	"encoding/json"
	"strconv"
	"fmt"
)

const mountLocation = "/mnt/crypto/"

// Start method is used perform the VM confidentiality check before lunching the VM
func Start(instanceUUID, imageUUID, imagePath, instancePath, diskSize string) int {
	// validate input parameters
	if len(strings.TrimSpace(instancePath)) <= 0 {
		fmt.Println("instance path not given")
		return 1
	}

	if len(strings.TrimSpace(instanceUUID)) <= 0 {
		fmt.Println("instance UUID not given")
		return 1
	}

	if len(strings.TrimSpace(imagePath)) <= 0 {
		fmt.Println("image path not given")
		return 1
	}

	if len(strings.TrimSpace(imageUUID)) <= 0 {
		fmt.Println("image UUID not given")
		return 1
	}

	if len(strings.TrimSpace(diskSize)) <= 0 {
		fmt.Println("sparse file size not given")
		return 1
	}

	//check if image exists in the given location
	_, err := os.Stat(imagePath)
	if os.IsNotExist(err) {
		fmt.Println("image does not exist in location ", imagePath)
		return 1
	}

	//check if image is encrypted
	fileOutput, err := exec.Command("file", imagePath).Output()
	if err != nil {
		fmt.Println("Error while checking if the image is encrypted")
		return 1
	}
	
	outputFormat := strings.Split(string(fileOutput), ":")
	imageFormat := outputFormat[len(outputFormat)-1]
	
	fmt.Println("ImageFormateValue: ", imageFormat)
	
	if strings.TrimSpace(imageFormat) != "data" {
		fmt.Println("The image is not encrypted")
		return 0
	}


	//check if the key is cached by filtercriteria imageUUID
	var keyID string
	var flavorKeyInfo wlsclient.FlavorKey
	var keyPath string

	// get host hardware UUID
	hardwareUUID,err := pinfo.HardwareUUID()
	if err != nil {
		fmt.Println("Unable to get the host hardware UUID")
		return 1
	}
	fmt.Println("The host hardware UUID is :", hardwareUUID)
	//get flavor-key from workload service
	flavorKeyInfo, err = wlsclient.GetImageFlavorKey(imageUUID, hardwareUUID, keyID)
	if err != nil {
		fmt.Println("Error while retrieving the image flavor and key")
		return 1
	}

	// if key not cached, cache the key

	if flavorKeyInfo.ImageFlavor.Image.Meta.ID == "" {
		fmt.Println("Flavor does not exist for the image ", imageUUID)
		return 0
	}

	if (flavorKeyInfo.Image.Encryption.EncryptionRequired) {
		// unwrap key
		fmt.Println("Unwrapping the key...")
		unWrappedKey, unWrapErr := unwrapKey(flavorKeyInfo.Key)
		if unWrapErr != nil {
			fmt.Println("Error while unwrapping the key")
			return 1
		}

		fmt.Println("Key unwrapping Done")
		// write the key to a temp file on disk
		keyPath = config.GetConfigDir() + "/key"
		err = ioutil.WriteFile(keyPath, unWrappedKey, 0600)
		if err != nil {
			fmt.Println("Error while writting the unwrapped key on disk")
			return 1
		}
		
		// create image volume
		deviceMapperLocation := config.GetDevMapperLocation() + imageUUID
		err = vml.CreateVolume(imagePath, deviceMapperLocation, keyPath, diskSize)
		if err != nil {
			fmt.Println("Error while creating image dm-crypt volume")
			return 1
		}

		//check if the image is mounted
		imageMountLocation := mountLocation + "imageUUID"
		_, err = os.Stat(imageMountLocation)
		if os.IsNotExist(err) {
			fmt.Println("Mounting the image")
			mountErr := vml.Mount(deviceMapperLocation, imageMountLocation)
			if mountErr != nil {
				fmt.Println("Error while mounting the image")
				return 1
			}
		}

		//decrypt the image
		err = vml.Decrypt(imagePath, deviceMapperLocation, keyPath)
		if err != nil {
			fmt.Println("Error while decrypting the image")
			return 1
		}

		// create symlink between the image and the dm-crypt volume
		err = os.Symlink(imagePath, deviceMapperLocation)
		if err != nil {
			fmt.Println("Error while creating symbolic link")
			return 1
		}

		// create instance volume
		instanceDeviceMapperLocation := config.GetDevMapperLocation() + instanceUUID		
		err = vml.CreateVolume(instancePath, instanceDeviceMapperLocation, keyPath, diskSize)
		if err != nil {
			fmt.Println("Error while creating instance dm-crypt volume")
			return 1
		}
	}

	//create VM manifest
	manifest, err := vml.CreateVMManifest(instanceUUID, hardwareUUID, imageUUID, true) 
	if err != nil {
		fmt.Println("Error while creating VM manifest")
		return 1
	}

	//create VM report
	vmTrustReport, err := verifier.Verify(&manifest, flavorKeyInfo.ImageFlavor)
	if err != nil {
		fmt.Println("Error while creating VM manifest")
		return 1
	}

	//post VM report on to workload service
	err = wlsclient.PostVMReport(vmTrustReport.(*verifier.VMTrustReport))
	if err!= nil {
		fmt.Println("Error while posting the VM trust report on to workload service")
		return 1
	}

	//associate instance UUID with the image UUID
	err = imageInstanceAssociation(imageUUID, imagePath)
	if err != nil {
		fmt.Println("Error while associating the image with the instance")
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

	if err != nil {
			fmt.Println("Error while opening the TPM")
			fmt.Println("Err: ", err)
			return key, err
		} 

	if t != nil {
		defer t.Close()
		
		bindingKeyFile := osutil.MakeFilePathFromEnvVariable(config.GetConfigDir(), config.GetBindingKeyFileName(), false)
		fmt.Println("Bindkey file name:", bindingKeyFile)
		fileContents, fileErr := ioutil.ReadFile(bindingKeyFile)
		if fileErr != nil {
			fmt.Println("Error while reading the binding key certificate")
			return key, fileErr
		}
		fmt.Println("Binding key file read")
		jsonErr := json.Unmarshal([]byte(fileContents), &certifiedKey)
		if jsonErr != nil {
			fmt.Println("Error while unmarshalling the binding key file contents to TPM CertifiedKey object")
			return key, jsonErr
		}

		fmt.Println("Binding key deserialized")
		keyAuth,_ := base64.StdEncoding.DecodeString(config.WlaConfig.WlsBindingKeySecret)
		fmt.Println("Binding key secret value: ", keyAuth)
		fmt.Println("Value from config: ", config.WlaConfig.WlsBindingKeySecret)
		key, unbindErr := t.Unbind(&certifiedKey, keyAuth, tpmWrappedKey)
		if unbindErr != nil {
			fmt.Println("Error while unbinding the tpm wrapped key")
			return key, unbindErr
		}
		fmt.Println("Unbind successful")
	}
	return key, nil
}

func imageInstanceAssociation(imageUUID, imagePath string) error {

	imageUUIDFound := false
	fileName := ""
	// read the contents of the file
	output, err := exec.Command("cat", fileName).Output()
	if err != nil {
		fmt.Println("Error while reading the contents of the file")
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
				fmt.Println("Error while replacing the count of the instance for an image")
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
			fmt.Println("Error while opening image-instance information")
			return err
		}

		defer f.Close()
		if _, err = f.WriteString(data); err != nil {
			fmt.Println("Error while writing image-instance information")
			return err
		}
	}

	return nil

}