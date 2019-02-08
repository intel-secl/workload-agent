// +build linux

package wlavm

import (
	//"log"
	"os"
	"strings"
	"intel/isecl/wlagent/wlsclient"
	"intel/isecl/lib/vml"
	pinfo "intel/isecl/lib/platform-info"
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/util"
	"intel/isecl/wlagent/consts"
	"intel/isecl/lib/common/exec"
	osutil "intel/isecl/lib/common/os"
	"encoding/base64"
	"io/ioutil"
	"encoding/json"
	"strconv"
	"os/user"
	log "github.com/sirupsen/logrus"
	xmlpath "gopkg.in/xmlpath.v2"
)

// Start method is used perform the VM confidentiality check before lunching the VM
// Input Parameters: domainXML content string
// Return : Returns an int value to the libvirt hook.
// 0 if the instance is launched sucessfully, else return 1. 
func Start(domainXMLContent string) int {

	var skipImageVolumeCreation = false
	var err error

	domainXML, err := xmlpath.Parse(strings.NewReader(domainXMLContent))
	if err != nil {
		log.Infof("Error while parsing domaXML: %s", err)
		return 1
	}

	// get instance UUID from domain XML
	instanceUUID, err := util.GetInstanceUUID(domainXML)
	if err != nil {
		log.Infof("%s", err)
		return 1
	}

	// get instance path from domain XML
	instancePath, err := util.GetInstancePath(domainXML)
	if err != nil {
		log.Infof("%s", err)
		return 1
	}

	// get image UUID from domain XML
	imageUUID, err := util.GetImageUUID(domainXML)
	if err != nil {
		log.Infof("%s", err)
		return 1
	}

	// get image path from domain XML
	imagePath, err := util.GetImagePath(domainXML)
	if err != nil {
		log.Infof("%s", err)
		return 1
	}

	// get disk size from domain XML
	diskSize, err := util.GetDiskSize(domainXML)
	if err != nil {
		log.Infof("%s", err)
		return 1
	}

	//check if image exists in the given location
	_, err = os.Stat(imagePath)
	if os.IsNotExist(err) {
		log.Infof("image does not exist in location %s", imagePath)
		return 1
	}
	
	// check if the image is a symlink, if it is avoid creating image dm-crypt volume 
	log.Info("Check if the image file is a symlink")
	symLinkOut, err := os.Readlink(imagePath)
	if len(strings.TrimSpace(symLinkOut)) > 0 {
		log.Info("The image is a symlink, so will be skipping the image dm-crypt volume creation")
		skipImageVolumeCreation = true
	} else {
		// check if image is encrypted
		log.Info("Image is not a symlink, so checking is image is encrypted...")
		isImageEncrypted, err := util.IsImageEncrypted(imagePath)
		if !isImageEncrypted {
			log.Info("Image is not encrypted, returning to the hook")
			return 0
		}
		if err != nil {
			log.Info("Error while trying to check if the image is encrypted")
			return 1
		}
	}

	//check if the key is cached by filtercriteria imageUUID
	var keyID string
	var flavorKeyInfo wlsclient.FlavorKey

	keyID, err = getKeyIDFromCache(imageUUID)
	if err != nil {
		log.Info("Error while checking if the key exists in the cache and retrieving the keyID")
		return 1
	}

	// get host hardware UUID
	log.Info("Getting host hardware UUID...")
	hardwareUUID,err := pinfo.HardwareUUID()
	if err != nil {
		log.Info("Unable to get the host hardware UUID")
		return 1
	}
	log.Infof("The host hardware UUID is :%s", hardwareUUID)

	//get flavor-key from workload service
	flavorKeyInfo, err = wlsclient.GetImageFlavorKey(imageUUID, hardwareUUID, keyID)
	if err != nil {
		log.Info("Error while retrieving the image flavor and key")
		return 1
	}

	if flavorKeyInfo.ImageFlavor.Image.Meta.ID == "" {
		log.Info("Flavor does not exist for the image ", imageUUID)
		return 0
	}

	if (flavorKeyInfo.Image.Encryption.EncryptionRequired) {

			// if key not cached, cache the key
		if (len(strings.TrimSpace(keyID)) <= 0) {
			// get key from flavor and store it in the cache
			keyURLSplit := strings.Split(flavorKeyInfo.Image.Encryption.KeyURL, "/")
			keyID := keyURLSplit[len(keyURLSplit)-2]
			cacheErr := cacheKeyInMemory(imageUUID, keyID, flavorKeyInfo.Key)
			if cacheErr != nil {
				log.Info("Error while storing the key in cache")
			}
		}

		// unwrap key
		log.Info("Unwrapping the key...")
		key, unWrapErr := unwrapKey(flavorKeyInfo.Key)
		if unWrapErr != nil {
			log.Info("Error while unwrapping the key")
			return 1
		}

		size, _ := strconv.Atoi(diskSize)

		// get the nova user info and change image and instance file owner to nova
		userInfo, err := user.Lookup("nova")
		if err != nil {
			log.Info("Error while trying to look up nova gid")
			return 1
		}

		userID, _ := strconv.Atoi(userInfo.Uid)
		groupID, _ := strconv.Atoi(userInfo.Gid)

		if !skipImageVolumeCreation {
				// create image dm-crypt volume
			log.Info("Creating a dm-crypt volume for the image")
			imageDeviceMapperPath := consts.DevMapperDirPath + imageUUID
			sparseFilePath := imagePath + "_sparseFile"
			err = vml.CreateVolume(sparseFilePath, imageDeviceMapperPath, key, size)
			if err != nil {
				log.Infof("Error while creating image dm-crypt volume for image: %s", imageUUID)
				log.Infof("Error: %s", err.Error())
				return 1
			}

			//check if the image device mapper is mount path exists, if not create it
			imageDeviceMapperMountPath := consts.MountPath + imageUUID
			err := util.CheckMountPathExistsAndMountVolume(imageDeviceMapperMountPath, imageDeviceMapperPath)
			if err != nil {
				log.Infof("Error: %s", err.Error())
				return 1
			}

			// read image file contents
			log.Info("Reading the encrypted image")
			encryptedImage, ioReadErr := ioutil.ReadFile(imagePath)
			if ioReadErr != nil {
				log.Info("Error while reading the image file")
				return 1
			}

			//decrypt the image
			log.Info("Decrypting the image")		
			decryptedImage, err := vml.Decrypt(encryptedImage, key)
			if err != nil {
				log.Infof("Error while decrypting the image. %s", err.Error())
				return 1
			}

			// write the decrypted data into a file in image mount path
			decryptedImagePath := imageDeviceMapperMountPath + "/" + imageUUID
			ioWriteErr := ioutil.WriteFile(decryptedImagePath, decryptedImage, 0644)
			if ioWriteErr != nil {
				log.Info("error during writing the decrypted image to file")
				return 1
			}

			// remove the encrypted image file and create a symlink with the dm-crypt volume
			log.Infof("Deleting the enc image file from :%s", imagePath)
			rmErr := os.RemoveAll(imagePath)
			if rmErr != nil {
				log.Infof("Error while deleting the encrypted image from disk: %s", imagePath)
				return 1
			}

			log.Info("Creating a symlink between the image and the volume")
			// create symlink between the image and the dm-crypt volume
			err = os.Symlink(decryptedImagePath, imagePath)
			if err != nil {
				log.Infof("Error while creating symbolic link. %s", err)
				return 1
			}

			// change the image symlink file ownership to nova 
			log.Info("Changing image symlink ownership to nova")
			err = os.Lchown(imagePath, userID, groupID)
			if err != nil {
				log.Info("Error while trying to change image symlink owner to nova")
				return 1
			}

			// change the image mount path directory ownership to nova
			log.Info("Changing the decrypted image file ownership to nova")
			log.Info("image device mapper path: ", imageDeviceMapperMountPath)
			err = osutil.ChownR(imageDeviceMapperMountPath, userID, groupID)
			if err != nil {
				log.Info("Error while trying to change decrypted image owner to nova")
				return 1
			}
		}

		// create instance volume
		instanceDeviceMapperPath := consts.DevMapperDirPath + instanceUUID
		instanceSparseFilePath := strings.Replace(instancePath, "disk", instanceUUID + "_sparse", -1)

		log.Infof("Creating dm-crypt volume for the instance: %s", instanceUUID)
		err = vml.CreateVolume(instanceSparseFilePath, instanceDeviceMapperPath, key, size)
		if err != nil {
			log.Infof("Error while creating instance dm-crypt volume. %s", err)
			return 1
		}

		// mount the instance dmcrypt volume on to a mount path
		instanceDeviceMapperMountPath := consts.MountPath + instanceUUID
		err = util.CheckMountPathExistsAndMountVolume(instanceDeviceMapperMountPath, instanceDeviceMapperPath)
		if err != nil {
			log.Info("Error: ", err)
			return 1
		}

		// copy the files from instance path and create a symlink
		args := []string{instancePath, instanceDeviceMapperMountPath}
		_, err = exec.ExecuteCommand("cp", args)
		if err != nil {
			log.Infof("Error while copying the instance change disk: %s", instanceUUID)
			return 1
		}

		// remove the encrypted image file and create a symlink with the dm-crypt volume
		// log.Info("Deleting change disk :", instancePath)
		// _, err = exec.Command("rm", "-rf", instancePath).Output()
		// if err != nil {
		// 	log.Info("Error while deleting the change disk: ", imagePath)
		// 	return 1
		// }

		log.Info("Creating a symlink between the instance and the volume")
		// create symlink between the image and the dm-crypt volume
		instanceSymLinkFile := strings.Replace(instancePath, "disk", instanceUUID, -1)
		err = os.Symlink(instanceDeviceMapperMountPath, instanceSymLinkFile)
		if err != nil {
			log.Info("Instance : Error while creating symbolic link")
			log.Infof("Error: %s", err.Error())
			return 1
		}

		//change the instance symlink file ownership to nova
		log.Info("Changing instance symlink ownership to nova")
		err = os.Lchown(instanceSymLinkFile, userID, groupID)
		if err != nil {
			log.Info("Error while trying to change image symlink owner to nova")
			return 1
		}

		// change the instance mount path directory ownership to nova
		log.Info("Changing the instance change disk file ownership to nova")
		err = osutil.ChownR(instanceDeviceMapperMountPath, userID, groupID)
		if err != nil {
			log.Info("Error while trying to change decrypted image owner to nova")
			return 1
		}
		
	}

	// create VM manifest

	// create VM trust reports

	// sign VM trust report

	// post VM Trust report

	// Updating image-instance count association
	log.Info("Updating the image-instance count file")
	err = imageInstanceCountAssociation(imageUUID, imagePath)
	if err != nil {
		log.Infof("Error while updating the image-instance count file. %s", err.Error())
		return 1
	}

	log.Infof("Instance %s started", instanceUUID)
	return 0
}

func unwrapKey(tpmWrappedKey []byte) ([]byte, error) {
	
	var certifiedKey tpm.CertifiedKey
	t, err := tpm.Open()

	if err != nil {
			log.Info("Error while opening the TPM")
			log.Info("Err: ", err)
			return nil, err
		} 

	defer t.Close()
	
	bindingKeyFilePath := consts.ConfigDirPath + consts.BindingKeyFileName
	log.Info("Bindkey file name:", bindingKeyFilePath)
	bindingKeyCert, fileErr := ioutil.ReadFile(bindingKeyFilePath)
	if fileErr != nil {
		log.Info("Error while reading the binding key certificate")
		return nil, fileErr
	}
	log.Info("Binding key file read")
	jsonErr := json.Unmarshal(bindingKeyCert, &certifiedKey)
	if jsonErr != nil {
		log.Info("Error while unmarshalling the binding key file contents to TPM CertifiedKey object")
		return nil, jsonErr
	}

	log.Info("Binding key deserialized")
	keyAuth,_ := base64.StdEncoding.DecodeString(config.Configuration.BindingKeySecret)
	key, unbindErr := t.Unbind(&certifiedKey, keyAuth, tpmWrappedKey)
	if unbindErr != nil {
		log.Info("Error while unbinding the tpm wrapped key")
		log.Infof("Err: %s", unbindErr.Error())
		return nil, unbindErr
	}
	log.Info("Unbind successful")
	return key, nil
}

func imageInstanceCountAssociation(imageUUID, imagePath string) error {

	imageUUIDFound := false
	imageInstanceCountAssociationFilePath := consts.ConfigDirPath + consts.ImageInstanceCountAssociationFileName
	
	// creating the image-instance file if not preset
	_, err := os.Stat(imageInstanceCountAssociationFilePath)
	if os.IsNotExist(err) {
		log.Info("Image-instance count file doesnot exists. Creating the file")
		args := []string{imageInstanceCountAssociationFilePath}
		_, touchErr := exec.ExecuteCommand("touch", args)
		if touchErr != nil {
			log.Info("Error while trying to create the image-instance count association file")
			return touchErr
		}
	}
	
	// read the contents of the file
	args := []string{imageInstanceCountAssociationFilePath}
	output, err := exec.ExecuteCommand("cat", args)
	if err != nil {
		log.Info("Error while reading the contents of the file")
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
			args := []string{ "-i", replaceString, imageInstanceCountAssociationFilePath}
			_, sedErr := exec.ExecuteCommand("sed", args)
			if sedErr != nil {
				log.Info("Error while replacing the count of the instance for an image")
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
			log.Info("Error while opening image-instance information")
			return err
		}

		defer f.Close()
		if _, err = f.WriteString(data); err != nil {
			log.Info("Error while writing image-instance information")
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