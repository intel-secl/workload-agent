// +build linux

package wlavm

import (
	//"log"
	"crypto"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"intel/isecl/lib/common/crypt"
	"intel/isecl/lib/common/exec"
	osutil "intel/isecl/lib/common/os"
	pinfo "intel/isecl/lib/platform-info"
	"intel/isecl/lib/tpm"
	"intel/isecl/lib/verifier"
	"intel/isecl/lib/vml"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/libvirt"
	"intel/isecl/wlagent/consts"
	"intel/isecl/wlagent/filewatch"
	"intel/isecl/wlagent/util"
	"intel/isecl/wlagent/wlsclient"
	"io/ioutil"
	"os"
	"os/user"
	"fmt"
	"errors"
	log "github.com/sirupsen/logrus"
	xmlpath "gopkg.in/xmlpath.v2"
)

// Todo: ISECL-3352 Move the TPM initialization to deamon start

var vmStartTpm tpm.Tpm

func GetTpmInstance()(tpm.Tpm, error){
	if vmStartTpm == nil {
		return tpm.Open()
	}
	return vmStartTpm, nil
}

func CloseTpmInstance(){
	if vmStartTpm != nil {
		vmStartTpm.Close()
	}
}

// Start method is used perform the VM confidentiality check before lunching the VM
// Input Parameters: domainXML content string
// Return : Returns a boolean value to the main method.
// true if the vm is launched sucessfully, else returns false. 
func Start(domainXMLContent string) bool {

	log.Info("VM start call intercepted")
	var skipImageVolumeCreation = false
	var err error

	log.Info("Parsing domain XML to get image UUID, image path, VM UUID, VM path and disk size")
	var parser *libvirt.DomainParser

	domainXML, err := xmlpath.Parse(strings.NewReader(domainXMLContent))
	if err != nil {
		log.Error("Error trying to parse domain xml")
		return false
	}
	parser = &libvirt.DomainParser{
		XML : domainXML,      
		QemuInterceptCall : libvirt.Start,
	}
	
	parsedValue, err := libvirt.NewDomainParser(parser)
	
	vmUUID := parsedValue.VMUUID
	vmPath := parsedValue.VMPath
	imageUUID := parsedValue.ImageUUID
	imagePath := parsedValue.ImagePath
	size := parsedValue.Size

	_, err = os.Stat(imagePath)
	if os.IsNotExist(err) {
		log.Errorf("Image does not exist in location %s", imagePath)
		return false
	}
	
	// check if the image is a symlink, if it is avoid creating image dm-crypt volume 
	log.Info("Checking if the image file is a symlink...")
	symLinkOut, err := os.Readlink(imagePath)
	if len(strings.TrimSpace(symLinkOut)) > 0 {
		log.Info("The image is a symlink, so will be skipping the image dm-crypt volume creation")
		skipImageVolumeCreation = true
	} else {
		// check if image is encrypted
		log.Info("Image is not a symlink, so checking is image is encrypted...")
		isImageEncrypted, err := crypt.EncryptionHeaderExists(imagePath)
		if !isImageEncrypted {
			log.Info("Image is not encrypted, returning to the hook")
			return true
		}
		if err != nil {
			log.Errorf("Error while trying to check if the image is encrypted: %s", err.Error())
			return false
		}
		log.Info("Image is encrypted")
	}

	// defer the CloseTpmInstance() to take care of closing the Tpm connection
	// Todo: ISECL-3352 remove when TPM vm is managed by daemon start and stop

	defer CloseTpmInstance()

	//check if the key is cached by filtercriteria imageUUID
	var keyID string
	var flavorKeyInfo wlsclient.FlavorKey
	var tpmWrappedKey []byte

	keyID, tpmWrappedKey, err = getKeyFromCache(imageUUID)
	if err != nil {
		log.Error("Error checking if the key exists in the cache and retrieving the keyID")
		return false
	}

	// get host hardware UUID
	log.Debug("Retrieving host hardware UUID...")
	hardwareUUID,err := pinfo.HardwareUUID()
	if err != nil {
		log.Error("Unable to get the host hardware UUID")
		return false
	}
	log.Debugf("The host hardware UUID is :%s", hardwareUUID)

	//get flavor-key from workload service
	log.Infof("Retrieving image-flavor-key for image %s from WLS", imageUUID)
	flavorKeyInfo, err = wlsclient.GetImageFlavorKey(imageUUID, hardwareUUID, keyID)
	if err != nil {
		log.Errorf("Error retrieving the image flavor and key: %s", err.Error())
		return false
	}

	if flavorKeyInfo.ImageFlavor.Image.Meta.ID == "" {
		log.Infof("Flavor does not exist for the image %s", imageUUID)
		// check with Ryan
		return true
	}

	if (flavorKeyInfo.Image.Encryption.EncryptionRequired) {
		// if key not cached, cache the key
		keyURLSplit := strings.Split(flavorKeyInfo.Image.Encryption.KeyURL, "/")
		keyID = keyURLSplit[len(keyURLSplit)-2]
		// if the WLS response includes a key, cache the key on host
		if len(flavorKeyInfo.Key) > 0 {
			// get key from flavor and store it in the cache
			log.Info("The image decryption key not cached, caching the key...")
			cacheErr := cacheKeyInMemory(imageUUID, keyID, flavorKeyInfo.Key)
			if cacheErr != nil {
				log.Error("Error caching the key")
			}
			// get the key from WLS response
			tpmWrappedKey = flavorKeyInfo.Key
		}

		// unwrap key
		log.Info("Unwrapping the key...")
		key, unWrapErr := unwrapKey(tpmWrappedKey)
		if unWrapErr != nil {
			log.Error("Error unwrapping the key")
			return false
		}

		if !skipImageVolumeCreation {
			log.Info("Creating and mounting image dm-crypt volume")
			err = imageVolumeManager(imageUUID, imagePath, size, key)
			if err != nil {
				log.Error(err.Error())
				return false
			}
		}

		log.Info("Creating and mounting vm dm-crypt volume")
		err = vmVolumeManager(vmUUID, vmPath, size, key)
		if err != nil {
			log.Error(err.Error())
			return false
		}
	}

	//create VM manifest
	log.Info("Creating VM Manifest")
	manifest, err := vml.CreateVMManifest(vmUUID, hardwareUUID, imageUUID, true)
	if err != nil {
		log.Errorf("Error creating the VM manifest: %s", err.Error())
		return false
	}

	//create VM Trust Report
	log.Info("Creating VM Trust Report")	
	vmTrustReport, err := verifier.Verify(&manifest, &flavorKeyInfo.ImageFlavor)
	if err != nil {
		log.Debugf("Error creating VM Trust Report: %s", err.Error())
		return false
	}

	// compute the hash and sign
	log.Info("Signing VM Trust Report")
	signedVMTrustReport, err := signVMTrustReport(vmTrustReport.(*verifier.VMTrustReport))
	if err != nil {
		log.Infof("Could not sign VM Trust Report using TPM :%s", err.Error())
		return false
	}

	//post VM Trust Report on to workload service
	log.Info("Post VM Trust Report on WLS")
	report, _ := json.Marshal(*signedVMTrustReport)

	log.Debugf("Report: %s", string(report))
	err = wlsclient.PostVMReport(report)
	if err!= nil {
		log.Infof("Failed to post the VM Trust Report on to workload service: %s", err.Error())
		return false
	}

	// Updating image-vm count association
	log.Info("Associating VM with image in image-vm-count file")
	err = imagevmCountAssociation(imageUUID, imagePath)
	if err != nil {
		log.Errorf("Error while updating the image-vm count file. %s", err.Error())
		return false
	}
	log.Infof("VM %s started", vmUUID)
	return true
}

func vmVolumeManager(vmUUID string, vmPath string, size int, key []byte) error {

	// create vm volume
	var err error
	vmDeviceMapperPath := consts.DevMapperDirPath + vmUUID
	vmSparseFilePath := strings.Replace(vmPath, "disk", vmUUID + "_sparse", -1)

	log.Debugf("Creating VM dm-crypt volume in %s", vmDeviceMapperPath)
	err = vml.CreateVolume(vmSparseFilePath, vmDeviceMapperPath, key, size)
	if err != nil {
		return fmt.Errorf("error creating vm dm-crypt volume: %s", err.Error())
	}

	// mount the vm dmcrypt volume on to a mount path
	log.Debug("Mounting the vm volume on a mount path")
	vmDeviceMapperMountPath := consts.MountPath + vmUUID
	err = checkMountPathExistsAndMountVolume(vmDeviceMapperMountPath, vmDeviceMapperPath)
	if err != nil {
		return fmt.Errorf("error checking if mount path exists and mounting the volume: %s", err.Error())
	}

	// copy the files from vm path
	log.Debugf("Copying all the files from %s to vm mount path", vmPath)
	args := []string{vmPath, vmDeviceMapperMountPath}
	_, err = exec.ExecuteCommand("cp", args)
	if err != nil {
		return fmt.Errorf("error copying the vm %s change disk to mount path", vmUUID)
	}

	// remove the encrypted image file and create a symlink with the dm-crypt volume
	log.Debugf("Deleting change disk %s:", vmPath)
	err = os.RemoveAll(vmPath)
	if err != nil {
		return fmt.Errorf("error deleting the change disk: %s", vmPath)
	}

	log.Debug("Creating a symlink between the vm and the volume")
	// create symlink between the image and the dm-crypt volume	
	changeDiskFile := vmDeviceMapperMountPath + "/disk"
	err = createSymLinkAndChangeOwnership(changeDiskFile, vmPath, vmDeviceMapperMountPath)
	if err != nil {
		return fmt.Errorf("error creating a symlink and changing file ownership: %s", err.Error())
	}
	return nil
}

func imageVolumeManager(imageUUID string, imagePath string, size int, key []byte) error {
	// create image dm-crypt volume
	log.Debugf("Creating a dm-crypt volume for the image %s", imageUUID)
	var err error
	imageDeviceMapperPath := consts.DevMapperDirPath + imageUUID
	sparseFilePath := imagePath + "_sparseFile"
	err = vml.CreateVolume(sparseFilePath, imageDeviceMapperPath, key, size)
	if err != nil {
		return fmt.Errorf("error while creating image dm-crypt volume for image: %s", err.Error())
	}

	//check if the image device mapper is mount path exists, if not create it
	imageDeviceMapperMountPath := consts.MountPath + imageUUID
	err = checkMountPathExistsAndMountVolume(imageDeviceMapperMountPath, imageDeviceMapperPath)
	if err != nil {
		return fmt.Errorf("error checking if image mount path exists and mounting the volume: %s", err.Error())
	}

	// read image file contents
	log.Debug("Reading the encrypted image file...")
	encryptedImage, ioReadErr := ioutil.ReadFile(imagePath)
	if ioReadErr != nil {
		return errors.New("error while reading the image file")
	}

	//decrypt the image
	log.Info("Decrypting the image")		
	decryptedImage, err := vml.Decrypt(encryptedImage, key)
	if err != nil {
		return fmt.Errorf("error while decrypting the image: %s", err.Error())
	}
	log.Info("Image decrypted successfully")
	// write the decrypted data into a file in image mount path
	log.Debug("Writting decrypted data in to a file")
	decryptedImagePath := imageDeviceMapperMountPath + "/" + imageUUID
	ioWriteErr := ioutil.WriteFile(decryptedImagePath, decryptedImage, 0644)
	if ioWriteErr != nil {
		return errors.New("error writing the decrypted data to file")
	}

	log.Debug("Creating a symlink between the image and the volume")
	err = createSymLinkAndChangeOwnership(decryptedImagePath, imagePath, imageDeviceMapperMountPath)
	if err != nil {
		return fmt.Errorf("error creating a symlink and changing file ownership: %s", err.Error())
	}
	return nil
}

func createSymLinkAndChangeOwnership(targetFile, sourceFile, mountPath string) error {

	// get the qemu user info and change image and vm file owner to qemu
	log.Debug("Looking up qemu user information")
	userInfo, err := user.Lookup("qemu")
	if err != nil {
		return errors.New("error trying to look up qemu userID and groupID")
	}
	userID, _ := strconv.Atoi(userInfo.Uid)
	groupID, _ := strconv.Atoi(userInfo.Gid)

	// remove the encrypted image file and create a symlink with the dm-crypt volume
	log.Debugf("Deleting the enc image file from :%s", sourceFile)
	rmErr := os.RemoveAll(sourceFile)
	if rmErr != nil {
		return fmt.Errorf("error deleting the change disk: %s", sourceFile)
	}

	log.Debugf("Creating a symlink between %s and %s", sourceFile, targetFile)
	// create symlink between the image and the dm-crypt volume
	err = os.Symlink(targetFile, sourceFile)
	if err != nil {
		return fmt.Errorf("error while creating symbolic link: %s", err.Error())
	}

	// change the image symlink file ownership to qemu 
	log.Debug("Changing symlink ownership to qemu")
	err = os.Lchown(sourceFile, userID, groupID)
	if err != nil {
		return errors.New("error while trying to change symlink owner to qemu")
	}

	// change the image mount path directory ownership to qemu
	log.Debug("Changing the mount path ownership to qemu")
	err = osutil.ChownR(mountPath, userID, groupID)
	if err != nil {
		return errors.New("error trying to change mount path owner to qemu")
	}
	return nil
}

func signVMTrustReport(report *verifier.VMTrustReport) (*crypt.SignedData, error) {

	var signedreport crypt.SignedData

	jsonVMTrustReport, err := json.Marshal(*report)
	if err != nil {
		return nil, fmt.Errorf("error : could not marshal VM Trust Report - %s", err)
	}

	signedreport.Data = jsonVMTrustReport
	signedreport.Alg = crypt.GetHashingAlgorithmName(config.HashingAlgorithm)
	log.Debug("Getting Signing Key Certificate from disk")
	signedreport.Cert, err = config.GetSigningCertFromFile()
	if err != nil {
		return nil, err
	}
	log.Debug("Using TPM to create signature")
	signature, err := createSignatureWithTPM([]byte(signedreport.Data), config.HashingAlgorithm)
	if err != nil {
		return nil, err
	}
	signedreport.Signature = signature

	return &signedreport, nil
}

func createSignatureWithTPM(data []byte, alg crypto.Hash) ([]byte, error) {

	var signingKey tpm.CertifiedKey

	// Get the Signing Key that is stored on disk
	log.Debug("Getting the signing key from WA config path")
	signingKeyJson, err := config.GetSigningKeyFromFile()
	if err != nil {
		return nil, err
	}

	log.Debug("Unmarshalling the signing key file contents into signing key struct")
	err = json.Unmarshal(signingKeyJson, &signingKey)
	if err != nil {
		return nil, err
	}

	// Get the secret associated when the SigningKey was created. 
	log.Debug("Retrieving the signing key secret form WA configuration")
	keyAuth, err := base64.StdEncoding.DecodeString(config.Configuration.SigningKeySecret)
	if err != nil{
		return nil, fmt.Errorf("error retrieving the signing key secret from configuration")
	}

    // Before we compute the hash, we need to check the version of TPM as TPM 1.2 only supports SHA1
	t, err := GetTpmInstance()
	if err != nil {
		return nil, fmt.Errorf("error attempting to create signature - could not open TPM")
	}

	if t.Version() == tpm.V12 {
		// tpm 1.2 only supports SHA1, so override the algorithm that we get here
		alg = crypto.SHA1
	}

	log.Debug("Computing the hash of the report to be signed by the TPM")
	h, err := crypt.GetHashData(data, alg)
	if err != nil {
		return nil, err
	}

	log.Debug("Using TPM to sign the hash")
	signature, err := t.Sign(&signingKey, keyAuth, alg, h)
	if err != nil {
		return nil, err
	}
	log.Debug("Report signed by TPM successfully")
	return signature, nil
}

func unwrapKey(tpmWrappedKey []byte) ([]byte, error) {
	
	var certifiedKey tpm.CertifiedKey
	t, err := GetTpmInstance()

	if err != nil {
		return nil, fmt.Errorf("could not establish connection to TPM: %s", err)
	}

	log.Debug("Reading the binding key certificate")
	bindingKeyFilePath := consts.ConfigDirPath + consts.BindingKeyFileName
	bindingKeyCert, fileErr := ioutil.ReadFile(bindingKeyFilePath)
	if fileErr != nil {
		return nil, errors.New("error while reading the binding key certificate")
	}

	log.Debug("Unmarshalling the binding key certificate file contents to TPM CertifiedKey object")
	jsonErr := json.Unmarshal(bindingKeyCert, &certifiedKey)
	if jsonErr != nil {
		return nil, errors.New("error unmarshalling the binding key file contents to TPM CertifiedKey object")
	}

	log.Debug("Binding key deserialized")
	keyAuth,_ := base64.StdEncoding.DecodeString(config.Configuration.BindingKeySecret)
	key, unbindErr := t.Unbind(&certifiedKey, keyAuth, tpmWrappedKey)
	if unbindErr != nil {
		return nil, fmt.Errorf("error while unbinding the tpm wrapped key: %s", unbindErr.Error())
	}

	log.Debug("Unbinding TPM wrapped key was successful, return the key")
	return key, nil
}

// This method is used to check if the key for an image file is cached.
// If the key is cached, the method you return the key ID.
func getKeyFromCache(imageUUID string) (string, []byte, error) {
	// checking if key is cached is not implemented yet
	return "", nil, nil
}

// This method is used add the key to cache and map it with the image UUID
func cacheKeyInMemory(imageUUID, keyID string, key []byte) error {
	// method not implemented yet
	return nil
}

// checkMountPathExistsAndMountVolume method is used to check if te mount path exists, 
// if it does not exists, the method creates the mount path and mounts the device mapper.
func checkMountPathExistsAndMountVolume(mountPath, deviceMapperPath string) error{	
	log.Debugf("Mounting the device mapper: %s", deviceMapperPath)
	mkdirErr := os.MkdirAll(mountPath, 0655)
	if mkdirErr != nil {
		return errors.New("error while creating the mount point for the image device mapper")
	}

	mountErr := vml.Mount(deviceMapperPath, mountPath)
	if mountErr != nil {
		if !strings.Contains(mountErr.Error(), "device is already mounted") {
			return errors.New("error while mounting the image device mapper")
		}
	}
	return nil
}
