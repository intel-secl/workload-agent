/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
// +build linux

package wlavm

import (
	"crypto"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"intel/isecl/lib/common/crypt"
	"intel/isecl/lib/common/exec"
	osutil "intel/isecl/lib/common/os"
	"intel/isecl/lib/common/pkg/instance"
	flvr "intel/isecl/lib/flavor"
	pinfo "intel/isecl/lib/platform-info/platforminfo"
	"intel/isecl/lib/tpm"
	"intel/isecl/lib/verifier"
	"intel/isecl/lib/vml"
	wlsclient "intel/isecl/wlagent/clients"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/consts"
	"intel/isecl/wlagent/filewatch"
	"intel/isecl/wlagent/libvirt"
	"intel/isecl/wlagent/util"
	"io/ioutil"
	"os"
	"os/user"
	"strconv"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
)

var (
	imgVolumeMtx sync.Mutex
	vmVolumeMtx  sync.Mutex
)

// Start method is used perform the VM confidentiality check before lunching the VM
// Input Parameters: domainXML content string
// Return : Returns a boolean value to the main method.
// true if the vm is launched sucessfully, else returns false.
func Start(domainXMLContent string, filewatcher *filewatch.Watcher) bool {

	log.Info("VM start call intercepted")
	var skipImageVolumeCreation = false
	var err error
	var skipManifestAndReportCreation = false

	log.Info("Parsing domain XML to get image UUID, image path, VM UUID, VM path and disk size")
	d, err := libvirt.NewDomainParser(domainXMLContent, libvirt.Start)
	if err != nil {
		log.Error("Parsing error: ", err.Error())
		return false
	}

	vmUUID := d.GetVMUUID()
	vmPath := d.GetVMPath()
	imageUUID := d.GetImageUUID()
	imagePath := d.GetImagePath()
	size := d.GetDiskSize()

	// In prepare state, the image path returned is nil
	if len(imagePath) <= 0 {
		// skip manifest and report creation in prepare state and create them in start state
		skipManifestAndReportCreation = true
		imagePath, err = imagePathFromVMAssociationFile(imageUUID)
		if err != nil {
			log.Errorf("Error while retrieving image path from image-vm association file: %s", err.Error())
			return false
		} else if len(imagePath) <= 0 {
			// if image path does not exist in image-vm association file, return back to hook
			log.Infof("There are no VM's launched from %s encrypted image, returning to hook", imageUUID)
			return true
		}
	}

	// check if the image is a symlink, if it is avoid creating image dm-crypt volume
	log.Info("Checking if the image file is a symlink...")
	symLinkOut, err := os.Readlink(imagePath)
	imageFileStat, imageFileStatErr := os.Stat(symLinkOut)
	if len(strings.TrimSpace(symLinkOut)) > 0 && imageFileStat.Size() > 0 && imageFileStatErr == nil {
		log.Info("The image is a symlink and the file linked exists, so will be skipping the image dm-crypt volume creation")
		skipImageVolumeCreation = true
	} else if err != nil {
		// check if image is encrypted
		_, err = os.Stat(imagePath)
		if os.IsNotExist(err) {
			log.Errorf("Image does not exist in location %s", imagePath)
			return false
		}
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
		// defer the CloseTpmInstance() to take care of closing the Tpm connection
		// Todo: ISECL-3352 remove when TPM vm is managed by daemon start and stop

		defer util.CloseTpmInstance()
	}

	var flavorKeyInfo wlsclient.FlavorKey
	var tpmWrappedKey []byte

	// get host hardware UUID
	log.Debug("Retrieving host hardware UUID...")
	hardwareUUID, err := pinfo.HardwareUUID()
	if err != nil {
		log.Error("Unable to get the host hardware UUID")
		return false
	}
	log.Debugf("The host hardware UUID is :%s", hardwareUUID)

	//get flavor-key from workload service
	// we will be hitting the WLS to retrieve the the flavor and key.
	// TODO: Investigate if it makes sense to cache the flavor locally as well with
	// an expiration time. Believe this was discussed and previously ruled out..
	// but still worth exploring for performance reasons as we want to minimize
	// making http client calls to external servers.
	log.Infof("Retrieving image-flavor-key for image %s from WLS", imageUUID)
	flavorKeyInfo, err = wlsclient.GetImageFlavorKey(imageUUID, hardwareUUID)
	if err != nil {
		log.Errorf("Error retrieving the image flavor and key: %s", err.Error())
		return false
	}

	if flavorKeyInfo.Flavor.Meta.ID == "" {
		log.Infof("Flavor does not exist for the image %s", imageUUID)
		return false
	}

	if flavorKeyInfo.Flavor.EncryptionRequired {
		tpmWrappedKey = flavorKeyInfo.Key
		// unwrap key
		log.Info("Unwrapping the key...")
		key, unWrapErr := util.UnwrapKey(tpmWrappedKey)
		if unWrapErr != nil {
			log.Errorf("Error unwrapping the key. %s", unWrapErr.Error())
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

		vmSymLinkOut, _ := os.Readlink(vmPath)
		vmSymLinkStat, vmSymLinkStatErr := os.Stat(vmSymLinkOut)
		if len(strings.TrimSpace(vmSymLinkOut)) <= 0 || vmSymLinkStatErr != nil || vmSymLinkStat.Size() <= 0 {
			log.Info("Creating and mounting vm dm-crypt volume")
			err = vmVolumeManager(vmUUID, vmPath, size, key, filewatcher)
			if err != nil {
				log.Error(err.Error())
				return false
			}
		}
	}

	if skipManifestAndReportCreation {
		log.Debug("skipping manifest and report creation in prepare VM state")
		return true
	}
	//create VM manifest
	log.Info("Creating VM Manifest")
	manifest, err := vml.CreateVMManifest(vmUUID, hardwareUUID, imageUUID, true)
	if err != nil {
		log.Errorf("Error creating the VM manifest: %s", err.Error())
		return false
	}

	//Create Image trust report
	status := CreateInstanceTrustReport(manifest, flvr.SignedImageFlavor{flavorKeyInfo.Flavor, flavorKeyInfo.Signature})
	if status == false {
		log.Error("Error while creating image trust report")
		return false
	}

	// Updating image-vm count association
	log.Info("Associating VM with image in image-vm-count file")
	iAssoc := ImageVMAssociation{imageUUID, imagePath}
	err = iAssoc.Create()
	if err != nil {
		log.Errorf("Error while updating the image-instance count file: %s", err.Error())
		return false
	}

	log.Infof("VM %s started", vmUUID)
	return true
}

func vmVolumeManager(vmUUID string, vmPath string, size int, key []byte, filewatcher *filewatch.Watcher) error {

	// create vm volume
	var err error
	vmDeviceMapperPath := consts.DevMapperDirPath + vmUUID
	vmSparseFilePath := strings.Replace(vmPath, "disk", vmUUID+"_sparse", -1)
	// check if sparse file exists, if it does, skip copying the change disk file to mount point
	_, sparseFleStatErr := os.Stat(vmSparseFilePath)
	log.Debugf("Creating VM dm-crypt volume in %s", vmDeviceMapperPath)
	vmVolumeMtx.Lock()
	err = vml.CreateVolume(vmSparseFilePath, vmDeviceMapperPath, key, size)
	vmVolumeMtx.Unlock()
	if err != nil {
		return fmt.Errorf("error creating vm dm-crypt volume: %s", err.Error())
	}

	// mount the vm dmcrypt volume on to a mount path
	log.Debug("Mounting the vm volume on a mount path")
	// vmDeviceMapperMountPath := strings.Replace(vmPath, "disk", "", -1) + vmUUID
	var vmMountPath = consts.MountPath + vmUUID
	err = checkMountPathExistsAndMountVolume(vmMountPath, vmDeviceMapperPath, "disk")
	if err != nil {
		return fmt.Errorf("error checking if mount path exists and mounting the volume: %s", err.Error())
	}

	// is sparse file exists, the image is already decrypted, so returning back to VM start method
	if !os.IsNotExist(sparseFleStatErr) {
		log.Debug("Sparse file of the VM already exists, skipping copying the change disk to mount point...")
		return nil
	}

	// copy the files from vm path
	log.Debugf("Copying all the files from %s to vm mount path", vmPath)
	args := []string{vmPath, vmMountPath}
	_, err = exec.ExecuteCommand("cp", args)
	if err != nil {
		return fmt.Errorf("error copying the vm %s change disk to mount path. %s", vmUUID, err.Error())
	}

	// remove the encrypted image file and create a symlink with the dm-crypt volume
	log.Debugf("Deleting change disk %s:", vmPath)
	err = os.RemoveAll(vmPath)
	if err != nil {
		return fmt.Errorf("error deleting the change disk: %s, %s", vmPath, err.Error())
	}

	changeDiskFile := vmMountPath + "/disk"
	log.Debug("Creating a symlink between the vm and the volume")
	// create symlink between the image and the dm-crypt volume
	err = createSymLinkAndChangeOwnership(changeDiskFile, vmPath, vmMountPath)
	if err != nil {
		return fmt.Errorf("error creating a symlink and changing file ownership: %s", err.Error())
	}

	// trigger a file watcher event to delete VM mount path when disk.info file is deleted on VM delete
	vmDiskInfoFile := strings.Replace(vmPath, "disk", "disk.info", -1)
	// Watch the symlink for deletion, and remove the _sparseFile if image is deleted
	filewatcher.HandleEvent(vmDiskInfoFile, func(e fsnotify.Event) {
		if e.Op&fsnotify.Remove == fsnotify.Remove {
			log.Debugf("Removing vm mount path at: %s", vmMountPath)
			os.RemoveAll(vmMountPath)
		}
	})

	return nil
}

func imageVolumeManager(imageUUID string, imagePath string, size int, key []byte) error {
	// create image dm-crypt volume
	log.Debugf("Creating a dm-crypt volume for the image %s", imageUUID)
	var err error
	imageDeviceMapperPath := consts.DevMapperDirPath + imageUUID
	sparseFilePath := imagePath + "_sparseFile"
	// check if the sprse file already exists, if it does, skip image file decryption
	_, sparseFileStatErr := os.Stat(sparseFilePath)
	imgVolumeMtx.Lock()
	err = vml.CreateVolume(sparseFilePath, imageDeviceMapperPath, key, size)
	imgVolumeMtx.Unlock()
	if err != nil {
		if strings.Contains(err.Error(), "device mapper of the same already exists") {
			log.Debug("Device mapper of same name already exists. Skipping image volume creation..")
			return nil
		} else {
			return fmt.Errorf("error while creating image dm-crypt volume for image: %s", err.Error())
		}
	}

	//check if the image device mapper is mount path exists, if not create it
	imageDeviceMapperMountPath := consts.MountPath + imageUUID
	err = checkMountPathExistsAndMountVolume(imageDeviceMapperMountPath, imageDeviceMapperPath, imageUUID)
	if err != nil {
		return fmt.Errorf("error checking if image mount path exists and mounting the volume: %s", err.Error())
	}

	// is sparse file exists, the image is already decrypted, so returning back to VM start method
	if !os.IsNotExist(sparseFileStatErr) {
		log.Debug("Sparse file of the image already exists, skipping image decryption...")
		return nil
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
	ioWriteErr := ioutil.WriteFile(decryptedImagePath, decryptedImage, 0660)
	if ioWriteErr != nil {
		return errors.New("error writing the decrypted data to file")
	}

	log.Debug("Creating a symlink between the image and the volume")
	err = createSymLinkAndChangeOwnership(decryptedImagePath, imagePath, imageDeviceMapperMountPath)
	if err != nil {
		return fmt.Errorf("error creating a symlink and changing file ownership: %s", err.Error())
	}

	// Watch the symlink for deletion, and remove the _sparseFile if image is deleted
	// filewatcher.HandleEvent(imagePath, func(e fsnotify.Event) {
	// 	if e.Op&fsnotify.Remove == fsnotify.Remove {
	// 		os.Remove(sparseFilePath)
	// 	}
	// })
	return nil
}

func createSymLinkAndChangeOwnership(targetFile, sourceFile, mountPath string) error {

	// get the qemu user info and change image and vm file owner to qemu
	userID, groupID, err := userInfoLookUp("qemu")
	if err != nil {
		return err
	}

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

	// Giving read write access to the target file
	err = os.Chmod(targetFile, 0660)
	if err != nil {
		return fmt.Errorf("error while giving permissions to changed disk path: %s %s", targetFile, err.Error())
	}

	// change the image mount path directory ownership to qemu
	log.Debug("Changing the mount path ownership to qemu")
	err = osutil.ChownR(mountPath, userID, groupID)
	if err != nil {
		return errors.New("error trying to change mount path owner to qemu")
	}
	return nil
}

func CreateInstanceTrustReport(manifest instance.Manifest, flavor flvr.SignedImageFlavor) bool {
	//create VM trust report
	log.Info("Creating image trust report")
	instanceTrustReport, err := verifier.Verify(&manifest, &flavor, consts.FlavorSigningCertPath, config.Configuration.SkipFlavorSignatureVerification)
	if err != nil {
		log.Errorf("Error creating image trust report: %s", err.Error())
		return false
	}
	trustreport, _ := json.Marshal(instanceTrustReport)
	log.Info(string(trustreport))

	// compute the hash and sign
	log.Info("Signing image trust report")
	signedInstanceTrustReport, err := signInstanceTrustReport(instanceTrustReport.(*verifier.InstanceTrustReport))
	if err != nil {
		log.Errorf("Could not sign image trust report using TPM :%s", err.Error())
		return false
	}

	//post VM trust report on to workload service
	log.Info("Post image trust report on WLS")
	report, _ := json.Marshal(*signedInstanceTrustReport)
	log.Debugf("Report: %s", string(report))

	err = wlsclient.PostVMReport(report)
	if err != nil {
		log.Errorf("Failed to post the instance trust report on to workload service: %s", err.Error())
		return false
	}
	return true
}

//Using SHA256 signing algorithm as TPM2.0 supports SHA256
func signInstanceTrustReport(report *verifier.InstanceTrustReport) (*crypt.SignedData, error) {

	var signedreport crypt.SignedData

	jsonVMTrustReport, err := json.Marshal(*report)
	if err != nil {
		return nil, fmt.Errorf("error : could not marshal instance trust report - %s", err)
	}

	signedreport.Data = jsonVMTrustReport
	signedreport.Alg = crypt.GetHashingAlgorithmName(crypto.SHA256)
	log.Debug("Getting Signing Key Certificate from disk")
	signedreport.Cert, err = config.GetSigningCertFromFile()
	if err != nil {
		return nil, err
	}
	log.Debug("Using TPM to create signature")
	signature, err := createSignatureWithTPM([]byte(signedreport.Data), crypto.SHA256)
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
	keyAuth, err := hex.DecodeString(config.Configuration.SigningKeySecret)
	if err != nil {
		return nil, fmt.Errorf("error retrieving the signing key secret from configuration. %s", err.Error())
	}

	// Before we compute the hash, we need to check the version of TPM as TPM 1.2 only supports SHA1
	t, err := util.GetTpmInstance()
	if err != nil {
		return nil, fmt.Errorf("error attempting to create signature - could not open TPM. %s", err.Error())
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

// checkMountPathExistsAndMountVolume method is used to check if te mount path exists,
// if it does not exists, the method creates the mount path and mounts the device mapper.
func checkMountPathExistsAndMountVolume(mountPath, deviceMapperPath, emptyFileName string) error {
	log.Debugf("Mounting the device mapper: %s", deviceMapperPath)
	mkdirErr := os.MkdirAll(mountPath, 0655)
	if mkdirErr != nil {
		return errors.New("error while creating the mount point for the image device mapper")
	}

	// create an empty image and disk file so that symlinks are not broken after VM stop
	emptyFilePath := mountPath + "/" + emptyFileName
	log.Debugf("Creating an empty file in : %s", emptyFilePath)
	sampleFile, err := os.Create(emptyFilePath)
	if err != nil {
		return errors.New("Error creating a sample file")
	}
	defer sampleFile.Close()

	// get the qemu user info and change image and vm file owner to qemu
	userID, groupID, err := userInfoLookUp("qemu")
	if err != nil {
		return err
	}

	// change the image mount path directory ownership to qemu
	log.Debug("Changing the mount path ownership to qemu")
	err = osutil.ChownR(mountPath, userID, groupID)
	if err != nil {
		return errors.New("error trying to change mount path owner to qemu")
	}

	mountErr := vml.Mount(deviceMapperPath, mountPath)
	if mountErr != nil {
		if !strings.Contains(mountErr.Error(), "device is already mounted") {
			return errors.New("error while mounting the image device mapper")
		}
	}
	return nil
}

func userInfoLookUp(userName string) (int, int, error) {
	log.Debug("Looking up qemu user information")
	userInfo, err := user.Lookup(userName)
	if err != nil {
		return 0, 0, errors.New("error trying to look up qemu userID and groupID")
	}
	userID, _ := strconv.Atoi(userInfo.Uid)
	groupID, _ := strconv.Atoi(userInfo.Gid)

	return userID, groupID, nil
}
