// +build linux

package wlavm

import (
	//"log"
	"intel/isecl/lib/vml"
	"intel/isecl/wlagent/filewatch"
	"intel/isecl/wlagent/wlsclient"
	"os"
	"strings"

	//"intel/isecl/lib/verifier"
	pinfo "intel/isecl/lib/platform-info"
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/config"

	//"intel/isecl/lib/flavor"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strconv"

	"github.com/fsnotify/fsnotify"
)

const mountPath = "/mnt/crypto/"

// Start method is used perform the VM confidentiality check before lunching the VM
func Start(instanceUUID, imageUUID, imagePath, instancePath, diskSize string, filewatcher *filewatch.Watcher) int {
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

	// check if image is encrypted
	fmt.Println("Checking is image is encrypted")
	isImageEncrypted, err := IsImageEncrypted(imagePath)
	if !isImageEncrypted {
		return 0
	}
	if err != nil {
		fmt.Println("Error while trying to check if the image is encrypted")
		return 1
	}

	//check if the key is cached by filtercriteria imageUUID
	var keyID string
	var flavorKeyInfo wlsclient.FlavorKey

	keyID, err = getKeyIDFromCache(imageUUID)
	if err != nil {
		fmt.Println("Error while checking if the key exists in the cache and retrieving the keyID")
		return 1
	}

	// get host hardware UUID
	hardwareUUID, err := pinfo.HardwareUUID()
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

	if flavorKeyInfo.ImageFlavor.Image.Meta.ID == "" {
		fmt.Println("Flavor does not exist for the image ", imageUUID)
		return 0
	}

	if flavorKeyInfo.Image.Encryption.EncryptionRequired {

		// if key not cached, cache the key
		if len(strings.TrimSpace(keyID)) <= 0 {
			// get key from flavor and store it in the cache
			keyURLSplit := strings.Split(flavorKeyInfo.Image.Encryption.KeyURL, "/")
			keyID := keyURLSplit[len(keyURLSplit)-2]
			cacheErr := cacheKeyInMemory(imageUUID, keyID, flavorKeyInfo.Key)
			if cacheErr != nil {
				fmt.Println("Error while storing the key in cache")
			}
		}

		// unwrap key
		fmt.Println("Unwrapping the key...")
		key, unWrapErr := unwrapKey(flavorKeyInfo.Key)
		if unWrapErr != nil {
			fmt.Println("Error while unwrapping the key")
			return 1
		}

		// create image dm-crypt volume
		fmt.Println("Creating a dm-crypt volume for the image")
		imageDeviceMapperPath := config.GetDevMapperDir() + imageUUID
		sparseFilePath := imagePath + "_sparseFile"
		size, _ := strconv.Atoi(diskSize)
		err = vml.CreateVolume(sparseFilePath, imageDeviceMapperPath, key, size)
		if err != nil {
			fmt.Println("Error while creating image dm-crypt volume for image:", imageUUID)
			fmt.Println("Error: ", err)
			return 1
		}

		//check if the image device mapper is mount path exists, if not create it
		imageDeviceMapperMountPath := mountPath + imageUUID
		err := CheckMountPathExistsAndMountVolume(imageDeviceMapperMountPath, imageDeviceMapperPath)
		if err != nil {
			fmt.Println("Error: ", err)
			return 1
		}

		// read image file contents
		fmt.Println("Reading the encrypted image")
		encryptedImage, ioReadErr := ioutil.ReadFile(imagePath)
		if ioReadErr != nil {
			fmt.Println("Error while reading the image file")
			return 1
		}

		//decrypt the image
		fmt.Println("Decrypting the image")
		decryptedImage, err := vml.Decrypt(encryptedImage, key)
		if err != nil {
			fmt.Println("Error while decrypting the image")
			fmt.Println("Error: ", err)
			return 1
		}

		// write the decrypted data into a file in image mount path
		decryptedImagePath := imageDeviceMapperMountPath + "/" + imageUUID
		ioWriteErr := ioutil.WriteFile(decryptedImagePath, decryptedImage, 0600)
		if ioWriteErr != nil {
			fmt.Println("error during writing the decrypted image to file")
			return 1
		}

		// remove the encrypted image file and create a symlink with the dm-crypt volume
		fmt.Println("Deleting the enc image file from :", imagePath)
		_, rmErr := exec.Command("rm", "-rf", imagePath).Output()
		if rmErr != nil {
			fmt.Println("Error while deleting the encrypted image from disk: ", imagePath)
			return 1
		}

		fmt.Println("Creating a symlink between the image and the volume")
		// create symlink between the image and the dm-crypt volume
		err = os.Symlink(imageDeviceMapperPath, imagePath)
		if err != nil {
			fmt.Println("Error while creating symbolic link")
			fmt.Println("Error: ", err)
			return 1
		}
		// create instance volume
		instanceDeviceMapperPath := config.GetDevMapperDir() + instanceUUID
		instanceSparseFilePath := strings.Replace(instancePath, "disk", instanceUUID+"_sparse", -1)

		fmt.Println("Creating dm-crypt volume for the instance: ", instanceUUID)
		err = vml.CreateVolume(instanceSparseFilePath, instanceDeviceMapperPath, key, size)
		if err != nil {
			fmt.Println("Error while creating instance dm-crypt volume")
			fmt.Println("Error: ", err)
			return 1
		}

		// Watch the symlink for deletion, and remove the _sparseFile if it is
		filewatcher.HandleEvent(imagePath, func(e fsnotify.Event) {
			if e.Op&fsnotify.Remove == fsnotify.Remove {
				os.Remove(instanceSparseFilePath)
			}
		})

		// mount the instance dmcrypt volume on to a mount path
		instanceDeviceMapperMountPath := mountPath + instanceUUID
		err = CheckMountPathExistsAndMountVolume(instanceDeviceMapperMountPath, instanceDeviceMapperPath)
		if err != nil {
			fmt.Println("Error: ", err)
			return 1
		}

		// copy the files from instance path and create a symlink
		_, err = exec.Command("cp", instancePath, instanceDeviceMapperMountPath).Output()
		if err != nil {
			fmt.Println("Error while copying the instance change disk: ", instanceUUID)
			return 1
		}

		// remove the encrypted image file and create a symlink with the dm-crypt volume
		// fmt.Println("Deleting change disk :", instancePath)
		// _, err = exec.Command("rm", "-rf", instancePath).Output()
		// if err != nil {
		// 	fmt.Println("Error while deleting the change disk: ", imagePath)
		// 	return 1
		// }

		fmt.Println("Creating a symlink between the instance and the volume")
		// create symlink between the image and the dm-crypt volume
		instanceSymLinkFile := strings.Replace(instancePath, "disk", instanceUUID, -1)
		err = os.Symlink(instanceDeviceMapperMountPath, instanceSymLinkFile)
		if err != nil {
			fmt.Println("Instance : Error while creating symbolic link")
			fmt.Println("Error: ", err)
			return 1
		}

		fmt.Println("Successfully created instance path")
		fmt.Println("Updating the image-instance count file")
		err = imageInstanceCountAssociation(imageUUID, imagePath)
		if err != nil {
			fmt.Println("Error while updating the image-instance count file")
			fmt.Println("Error: ", err)
			return 1
		}
	}
	return 0
}

func unwrapKey(tpmWrappedKey []byte) ([]byte, error) {

	var certifiedKey tpm.CertifiedKey
	t, err := tpm.Open()

	if err != nil {
		fmt.Println("Error while opening the TPM")
		fmt.Println("Err: ", err)
		return nil, err
	}

	defer t.Close()

	bindingKeyFilePath := "/etc/workloadagent/bindingkey.json"
	fmt.Println("Bindkey file name:", bindingKeyFilePath)
	bindingKeyCert, fileErr := ioutil.ReadFile(bindingKeyFilePath)
	if fileErr != nil {
		fmt.Println("Error while reading the binding key certificate")
		return nil, fileErr
	}
	fmt.Println("Binding key file read")
	jsonErr := json.Unmarshal(bindingKeyCert, &certifiedKey)
	if jsonErr != nil {
		fmt.Println("Error while unmarshalling the binding key file contents to TPM CertifiedKey object")
		return nil, jsonErr
	}

	fmt.Println("Binding key deserialized")
	keyAuth, _ := base64.StdEncoding.DecodeString(config.Configuration.BindingKeySecret)
	fmt.Println("Binding key secret value: ", keyAuth)
	fmt.Println("Value from config: ", config.Configuration.BindingKeySecret)
	key, unbindErr := t.Unbind(&certifiedKey, keyAuth, tpmWrappedKey)
	if unbindErr != nil {
		fmt.Println("Error while unbinding the tpm wrapped key")
		fmt.Println("Err: ", unbindErr.Error())
		return nil, unbindErr
	}
	fmt.Println("Unbind successful")
	fmt.Println("Unwrapped key length returned by TPM: ", len(key))
	return key, nil
}

func imageInstanceCountAssociation(imageUUID, imagePath string) error {

	imageUUIDFound := false
	imageInstanceCountAssociationFilePath := "/etc/workloadagent/" + config.ImageInstanceCountAssociationFileName()

	// creating the image-instance file if not preset
	_, err := os.Stat(imageInstanceCountAssociationFilePath)
	if os.IsNotExist(err) {
		fmt.Println("Image-instance count file doesnot exists. Creating the file")
		_, touchErr := exec.Command("touch", imageInstanceCountAssociationFilePath).Output()
		if touchErr != nil {
			fmt.Println("Error while trying to create the image-instance count association file")
			return touchErr
		}
	}

	// read the contents of the file
	output, err := exec.Command("cat", imageInstanceCountAssociationFilePath).Output()
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
			_, sedErr := exec.Command("sed", "-i", replaceString, imageInstanceCountAssociationFilePath).Output()
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

		f, err := os.OpenFile(imageInstanceCountAssociationFilePath, os.O_WRONLY|os.O_APPEND, 0600)
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

// This method is used to check if the key for an image file is cached.
// If the key is cached, the method you return the key ID.
func getKeyIDFromCache(imageUUID string) (string, error) {
	// checking if key is cached is not implementaed yet
	return "", nil
}

// This method is used add the key to cache and map it with the image UUID
func cacheKeyInMemory(imageUUID, keyID string, key []byte) error {
	// method not implemented yet
	return nil
}
