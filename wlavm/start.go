/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
// +build linux

package wlavm

import (
	"crypto"
	"encoding/json"

	wlsModel "github.com/intel-secl/intel-secl/v3/pkg/model/wls"
	"intel/isecl/lib/common/v3/crypt"
	"intel/isecl/lib/common/v3/exec"
	"intel/isecl/lib/common/v3/log/message"
	osutil "intel/isecl/lib/common/v3/os"
	"intel/isecl/lib/common/v3/pkg/instance"
	pinfo "intel/isecl/lib/platform-info/v3/platforminfo"
	"intel/isecl/lib/tpmprovider/v3"
	"intel/isecl/lib/verifier/v3"
	"intel/isecl/lib/vml/v3"
	wlsclient "intel/isecl/wlagent/v3/clients"
	"intel/isecl/wlagent/v3/config"
	"intel/isecl/wlagent/v3/consts"
	"intel/isecl/wlagent/v3/filewatch"
	"intel/isecl/wlagent/v3/libvirt"
	"intel/isecl/wlagent/v3/util"
	"io/ioutil"
	"os"
	"os/user"
	"strconv"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

var (
	imgVolumeMtx sync.Mutex
	vmVolumeMtx  sync.Mutex
	TpmMtx       sync.Mutex
)

// Start method is used perform the VM confidentiality check before lunching the VM
// Input Parameters: domainXML content string
// Return : Returns a boolean value to the main method.
// true if the vm is launched sucessfully, else returns false.
func Start(domainXMLContent string, filewatcher *filewatch.Watcher) bool {

	log.Trace("wlavm/start:Start() Entering")
	defer log.Trace("wlavm/start:Start() Leaving")
	var skipImageVolumeCreation = false
	var err error
	var skipManifestAndReportCreation = false
	var isImageEncrypted bool

	log.Info("wlavm/start:Start() Parsing domain XML to get image UUID, image path, VM UUID, VM path and disk size")
	d, err := libvirt.NewDomainParser(domainXMLContent, libvirt.Start)
	if err != nil {
		log.Error("wlavm/start.go:Start() Parsing error: ", err.Error())
		log.Tracef("%+v", err)
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
			log.Errorf("wlavm/start.go:Start() Error while retrieving image path from image-vm association file: %s", err.Error())
			log.Tracef("%+v", err)
			return false
		} else if len(imagePath) <= 0 {
			// if image path does not exist in image-vm association file, return back to hook
			log.Infof("wlavm/start:Start() There are no VM's launched from %s encrypted image, returning to hook", imageUUID)
			return true
		}
	}

	// check if the image is a symlink, if it is avoid creating image dm-crypt volume
	log.Info("wlavm/start:Start() Checking if the image file is a symlink...")
	symLinkOut, err := os.Readlink(imagePath)
	imageFileStat, imageFileStatErr := os.Stat(symLinkOut)
	if len(strings.TrimSpace(symLinkOut)) > 0 && imageFileStat.Size() > 0 && imageFileStatErr == nil {
		log.Info("wlavm/start:Start() The image is a symlink and the file linked exists, so will be skipping the image dm-crypt volume creation")
		skipImageVolumeCreation = true
	} else if err != nil {
		// check if image is encrypted
		_, err = os.Stat(imagePath)
		if os.IsNotExist(err) {
			log.Errorf("wlavm/start.go:Start() Image does not exist in location %s", imagePath)
			log.Tracef("%+v", err)
			return false
		}
		log.Info("wlavm/start:Start() Image is not a symlink, so checking is image is encrypted...")
		isImageEncrypted, err = crypt.EncryptionHeaderExists(imagePath)
		if err != nil {
			log.Errorf("wlavm/start.go:Start() Error while trying to check if the image is encrypted: %s", err.Error())
			log.Tracef("%+v", err)
			return false
		}
		log.Info("wlavm/start:Start() Image encryption status : ", isImageEncrypted)
		// if image is not encrypted, return true to libvirt hook
		if !isImageEncrypted {
			log.Info("wlavm/start:Start() Image is not encrypted, returning to the hook")
			return true
		}
	}

	var flavorKeyInfo wlsModel.FlavorKey
	var tpmWrappedKey []byte

	// get host hardware UUID
	secLog.Infof("wlavm/start.go:Start() %s, Trying to get host hardware UUID", message.SU)
	hardwareUUID, err := pinfo.HardwareUUID()
	if err != nil {
		log.WithError(err).Error("wlavm/start.go:Start() Unable to get the host hardware UUID")
		return false
	}
	log.Debugf("wlavm/start:Start() The host hardware UUID is :%s", hardwareUUID)

	//get flavor-key from workload service
	// we will be hitting the WLS to retrieve the the flavor and key.
	// TODO: Investigate if it makes sense to cache the flavor locally as well with
	// an expiration time. Believe this was discussed and previously ruled out..
	// but still worth exploring for performance reasons as we want to minimize
	// making http client calls to external servers.
	log.Infof("wlavm/start:Start() Retrieving image-flavor-key for image %s from WLS", imageUUID)

	flavorKeyInfo, err = wlsclient.GetImageFlavorKey(imageUUID, hardwareUUID)
	if err != nil {
		secLog.WithError(err).Error("wlavm/start.go:Start() Error retrieving the image flavor and key")
		return false
	}

	if flavorKeyInfo.Flavor.Meta.ID == "" {
		log.Infof("wlavm/start:Start() Flavor does not exist for the image %s", imageUUID)
		return false
	}

	if flavorKeyInfo.Flavor.EncryptionRequired {
		if len(flavorKeyInfo.Key) == 0 {
			log.Error("wlavm/start.go:Start() Flavor Key is empty")
			return false
		}
		tpmWrappedKey = flavorKeyInfo.Key
		// unwrap key
		log.Info("wlavm/start:Start() Unwrapping the key...")
		TpmMtx.Lock()
		key, unWrapErr := util.UnwrapKey(tpmWrappedKey)
		TpmMtx.Unlock()
		if unWrapErr != nil {
			secLog.WithError(err).Error("wlavm/start.go:Start() Error unwrapping the key")
			return false
		}

		if !skipImageVolumeCreation {
			log.Info("wlavm/start:Start() Creating and mounting image dm-crypt volume")
			err = imageVolumeManager(imageUUID, imagePath, size, key)
			if err != nil {
				log.WithError(err).Error("wlavm/start.go:Start() Error while creating and mounting image dm-crypt volume ")
				return false
			}
		}

		vmSymLinkOut, _ := os.Readlink(vmPath)
		vmSymLinkStat, vmSymLinkStatErr := os.Stat(vmSymLinkOut)
		if len(strings.TrimSpace(vmSymLinkOut)) <= 0 || vmSymLinkStatErr != nil || vmSymLinkStat.Size() <= 0 {
			log.Info("wlavm/start:Start() Creating and mounting vm dm-crypt volume")
			err = vmVolumeManager(vmUUID, vmPath, size, key, filewatcher)
			if err != nil {
				log.WithError(err).Error("wlavm/start.go:Start() Error while creating and mounting vm dm-crypt volume ")
				return false
			}
		}
	}

	if skipManifestAndReportCreation {
		log.Debug("wlavm/start:Start() Skipping manifest and report creation in prepare VM state")
		return true
	}
	//create VM manifest
	log.Info("wlavm/start:Start() Creating VM Manifest")
	manifest, err := vml.CreateVMManifest(vmUUID, hardwareUUID, imageUUID, true)
	if err != nil {
		log.WithError(err).Error("wlavm/start.go:Start() Error creating the VM manifest")
		return false
	}

	//Create Image trust report
	status := CreateInstanceTrustReport(manifest, wlsModel.SignedImageFlavor{flavorKeyInfo.Flavor, flavorKeyInfo.Signature})
	if status == false {
		log.Error("wlavm/start.go:Start() Error while creating image trust report")
		return false
	}

	// Updating image-vm count association
	log.Info("wlavm/start:Start() Associating VM with image in image-vm-count file")
	iAssoc := ImageVMAssociation{imageUUID, imagePath}
	err = iAssoc.Create()
	if err != nil {
		log.WithError(err).Error("wlavm/start.go:Start() Error while updating the image-instance count file")
		return false
	}

	log.Infof("wlavm/start:Start() VM %s started", vmUUID)
	return true
}

func vmVolumeManager(vmUUID string, vmPath string, size int, key []byte, filewatcher *filewatch.Watcher) error {
	log.Trace("wlavm/start:vmVolumeManager() Entering")
	defer log.Trace("wlavm/start:vmVolumeManager() Leaving")

	// create vm volume
	var err error
	vmDeviceMapperPath := consts.DevMapperDirPath + vmUUID
	vmSparseFilePath := strings.Replace(vmPath, "disk", vmUUID+"_sparse", -1)
	// check if sparse file exists, if it does, skip copying the change disk file to mount point
	_, sparseFleStatErr := os.Stat(vmSparseFilePath)
	vmVolumeMtx.Lock()
	secLog.Infof("wlavm/start:vmVolumeManager() %s, Creating VM dm-crypt volume in %s", message.SU, vmDeviceMapperPath)
	err = vml.CreateVolume(vmSparseFilePath, vmDeviceMapperPath, key, size)
	vmVolumeMtx.Unlock()
	if err != nil {
		return errors.Wrap(err, "wlavm/start.go:vmVolumeManager() error creating vm dm-crypt volume")
	}

	// mount the vm dmcrypt volume on to a mount path
	log.Debug("wlavm/start:vmVolumeManager() Mounting the vm volume on a mount path")
	// vmDeviceMapperMountPath := strings.Replace(vmPath, "disk", "", -1) + vmUUID
	var vmMountPath = consts.MountPath + vmUUID
	err = checkMountPathExistsAndMountVolume(vmMountPath, vmDeviceMapperPath, "disk")
	if err != nil {
		return errors.Wrap(err, "wlavm/start.go:vmVolumeManager() error checking if mount path exists and mounting the volume")
	}

	// is sparse file exists, the image is already decrypted, so returning back to VM start method
	if !os.IsNotExist(sparseFleStatErr) {
		log.Debug("wlavm/start:vmVolumeManager() Sparse file of the VM already exists, skipping copying the change disk to mount point...")
		return nil
	}

	// copy the files from vm path
	args := []string{vmPath, vmMountPath}
	secLog.Infof("wlavm/start:vmVolumeManager() %s, Copying all the files from %s to vm mount path", message.SU, vmPath)
	_, err = exec.ExecuteCommand("cp", args)
	if err != nil {
		return errors.Wrapf(err, "wlavm/start.go:vmVolumeManager() error copying the vm path %s change disk to mount path. %s", vmPath, vmMountPath)
	}

	// remove the encrypted image file and create a symlink with the dm-crypt volume
	secLog.Infof("wlavm/start:vmVolumeManager() %s, Deleting change disk: %s", message.SU, vmPath)
	err = os.RemoveAll(vmPath)
	if err != nil {
		return errors.Wrapf(err, "wlavm/start.go:vmVolumeManager() error deleting the change disk: %s", vmPath)
	}

	changeDiskFile := vmMountPath + "/disk"
	log.Debug("wlavm/start:vmVolumeManager() Creating a symlink between the vm and the volume")
	// create symlink between the image and the dm-crypt volume
	err = createSymLinkAndChangeOwnership(changeDiskFile, vmPath, vmMountPath)
	if err != nil {
		return errors.Wrap(err, "wlavm/start.go:vmVolumeManager() error creating a symlink and changing file ownership")
	}

	// trigger a file watcher event to delete VM mount path when disk.info file is deleted on VM delete
	vmDiskInfoFile := strings.Replace(vmPath, "disk", "disk.info", -1)
	// Watch the symlink for deletion, and remove the _sparseFile if image is deleted
	err = filewatcher.HandleEvent(vmDiskInfoFile, func(e fsnotify.Event) {
		if e.Op&fsnotify.Remove == fsnotify.Remove {
			secLog.Infof("wlavm/start:vmVolumeManager() %s, Removing vm mount path at: %s", message.SU, vmMountPath)
			err = os.RemoveAll(vmMountPath)
			if err != nil {
				log.Errorf("wlavm/start:vmVolumeManager() Failed to remove mount path")
			}
		}
	})
	if err != nil {
		log.Errorf("wlavm/start:vmVolumeManager() Failed to handle event: %v", err)
	}

	return nil
}

func imageVolumeManager(imageUUID string, imagePath string, size int, key []byte) error {
	log.Trace("wlavm/start:imageVolumeManager() Entering")
	defer log.Trace("wlavm/start:imageVolumeManager() Leaving")

	// create image dm-crypt volume
	var err error
	imageDeviceMapperPath := consts.DevMapperDirPath + imageUUID
	sparseFilePath := imagePath + "_sparseFile"
	// check if the sprse file already exists, if it does, skip image file decryption
	_, sparseFileStatErr := os.Stat(sparseFilePath)
	imgVolumeMtx.Lock()
	secLog.Infof("wlavm/start:imageVolumeManager() %s, Creating a dm-crypt volume for the image %s", message.SU, imageUUID)
	err = vml.CreateVolume(sparseFilePath, imageDeviceMapperPath, key, size)
	imgVolumeMtx.Unlock()
	if err != nil {
		if strings.Contains(err.Error(), "device mapper of the same already exists") {
			log.Debug("wlavm/start:imageVolumeManager() Device mapper of same name already exists. Skipping image volume creation..")
			return nil
		} else {
			return errors.Wrap(err, "wlavm/start.go:imageVolumeManager() error while creating image dm-crypt volume for image")
		}
	}

	//check if the image device mapper is mount path exists, if not create it
	imageDeviceMapperMountPath := consts.MountPath + imageUUID
	err = checkMountPathExistsAndMountVolume(imageDeviceMapperMountPath, imageDeviceMapperPath, imageUUID)
	if err != nil {
		return errors.Wrap(err, "wlavm/start.go:imageVolumeManager() error checking if image mount path exists and mounting the volume")
	}

	// is sparse file exists, the image is already decrypted, so returning back to VM start method
	if !os.IsNotExist(sparseFileStatErr) {
		log.Debug("wlavm/start:imageVolumeManager() Sparse file of the image already exists, skipping image decryption...")
		return nil
	}

	// read image file contents
	log.Debug("Reading the encrypted image file...")
	encryptedImage, ioReadErr := ioutil.ReadFile(imagePath)
	if ioReadErr != nil {
		return errors.New("wlavm/start.go:imageVolumeManager() error while reading the image file")
	}

	//decrypt the image
	log.Info("wlavm/start:imageVolumeManager() Decrypting the image")
	decryptedImage, err := vml.Decrypt(encryptedImage, key)
	if err != nil {
		return errors.Wrap(err, "wlavm/start.go:imageVolumeManager() error while decrypting the image")
	}
	log.Info("wlavm/start:imageVolumeManager() Image decrypted successfully")
	// write the decrypted data into a file in image mount path
	decryptedImagePath := imageDeviceMapperMountPath + "/" + imageUUID
	secLog.Infof("wlavm/start:imageVolumeManager() %s, Writing decrypted data in to a file: %s", message.SU, decryptedImagePath)
	ioWriteErr := ioutil.WriteFile(decryptedImagePath, decryptedImage, 0660)
	if ioWriteErr != nil {
		return errors.New("wlavm/start.go:imageVolumeManager() error writing the decrypted data to file")
	}

	log.Debug("wlavm/start:imageVolumeManager() Creating a symlink between the image and the volume")
	err = createSymLinkAndChangeOwnership(decryptedImagePath, imagePath, imageDeviceMapperMountPath)
	if err != nil {
		return errors.Wrap(err, "wlavm/start.go:imageVolumeManager() error creating a symlink and changing file ownership")
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
	log.Trace("wlavm/start:Start() Entering")
	defer log.Trace("wlavm/start:Start() Leaving")

	// get the qemu user info and change image and vm file owner to qemu
	userID, groupID, err := userInfoLookUp("qemu")
	if err != nil {
		return err
	}

	// remove the encrypted image file and create a symlink with the dm-crypt volume
	secLog.Infof("wlavm/start:imageVolumeManager() %s, Deleting the enc image file from :%s", message.SU, sourceFile)
	rmErr := os.RemoveAll(sourceFile)
	if rmErr != nil {
		return errors.Wrapf(err, "wlavm/start.go:createSymLinkAndChangeOwnership() error deleting the change disk: %s", sourceFile)
	}

	// create symlink between the image and the dm-crypt volume
	secLog.Infof("wlavm/start:imageVolumeManager() %s, Creating a symlink between %s and %s", message.SU, sourceFile, targetFile)
	err = os.Symlink(targetFile, sourceFile)
	if err != nil {
		return errors.Wrapf(err, "wlavm/start.go:createSymLinkAndChangeOwnership() error while creating symbolic link")
	}

	// change the image symlink file ownership to qemu
	secLog.Infof("wlavm/start.go:createSymLinkAndChangeOwnership() %s, Changing symlink ownership to qemu", message.SU)
	err = os.Lchown(sourceFile, userID, groupID)
	if err != nil {
		return errors.New("wlavm/start.go:createSymLinkAndChangeOwnership() error while trying to change symlink owner to qemu")
	}

	// Giving read write access to the target file
	secLog.Infof("wlavm/start.go:createSymLinkAndChangeOwnership() %s, Changing permissions to changed disk path: %s", message.SU, targetFile)
	err = os.Chmod(targetFile, 0660)
	if err != nil {
		return errors.Wrapf(err, "wlavm/start.go:createSymLinkAndChangeOwnership() error while giving permissions to changed disk path: %s", targetFile)
	}

	// change the image mount path directory ownership to qemu
	secLog.Infof("wlavm/start.go:createSymLinkAndChangeOwnership() %s, Changing the mount path ownership to qemu", message.SU)
	err = osutil.ChownR(mountPath, userID, groupID)

	if err != nil {
		return errors.New("wlavm/start.go:createSymLinkAndChangeOwnership() error trying to change mount path owner to qemu")
	}
	return nil
}

func CreateInstanceTrustReport(manifest instance.Manifest, flavor wlsModel.SignedImageFlavor) bool {
	log.Trace("wlavm/start:CreateInstanceTrustReport() Entering")
	defer log.Trace("wlavm/start:CreateInstanceTrustReport() Leaving")

	//create VM trust report
	log.Info("wlavm/start:CreateInstanceTrustReport() Creating image trust report")
	instanceTrustReport, err := verifier.Verify(&manifest, &flavor, consts.FlavorSigningCertDir, consts.TrustedCaCertsDir, config.Configuration.SkipFlavorSignatureVerification)
	if err != nil {
		log.WithError(err).Error("wlavm/start.go:CreateInstanceTrustReport() Error creating image trust report")
		log.Tracef("%+v", err)
		return false
	}
	trustreport, _ := json.Marshal(instanceTrustReport)
	log.Infof("wlavm/start:CreateInstanceTrustReport() trustreport: %s", string(trustreport))

	// compute the hash and sign
	log.Info("wlavm/start:CreateInstanceTrustReport() Signing image trust report")
	signedInstanceTrustReport, err := signInstanceTrustReport(instanceTrustReport.(*verifier.InstanceTrustReport))
	if err != nil {
		log.WithError(err).Error("wlavm/start.go:CreateInstanceTrustReport() Could not sign image trust report using TPM")
		log.Tracef("%+v", err)
		return false
	}

	//post VM trust report on to workload service
	log.Info("wlavm/start:CreateInstanceTrustReport() Post image trust report on WLS")
	report, _ := json.Marshal(*signedInstanceTrustReport)
	log.Debugf("wlavm/start:CreateInstanceTrustReport() Report: %s", string(report))

	err = wlsclient.PostVMReport(report)
	if err != nil {
		secLog.WithError(err).Error("wlavm/start.go:CreateInstanceTrustReport() Failed to post the instance trust report on to workload service")
		return false
	}
	return true
}

//Using SHA256 signing algorithm as TPM2.0 supports SHA256
func signInstanceTrustReport(report *verifier.InstanceTrustReport) (*crypt.SignedData, error) {
	log.Trace("wlavm/start:signInstanceTrustReport() Entering")
	defer log.Trace("wlavm/start:signInstanceTrustReport() Leaving")

	var signedreport crypt.SignedData

	jsonVMTrustReport, err := json.Marshal(*report)
	if err != nil {
		return nil, errors.Wrap(err, "wlavm/start.go:signInstanceTrustReport() could not marshal instance trust report")
	}

	signedreport.Data = jsonVMTrustReport
	signedreport.Alg = crypt.GetHashingAlgorithmName(crypto.SHA256)
	log.Debug("wlavm/start:signInstanceTrustReport() Getting Signing Key Certificate from disk")
	signedreport.Cert, err = config.GetSigningCertFromFile()
	if err != nil {
		return nil, err
	}
	log.Debug("wlavm/start:signInstanceTrustReport() Using TPM to create signature")
	signature, err := createSignatureWithTPM([]byte(signedreport.Data), crypto.SHA256)
	if err != nil {
		return nil, err
	}
	signedreport.Signature = signature

	return &signedreport, nil
}

func createSignatureWithTPM(data []byte, alg crypto.Hash) ([]byte, error) {
	log.Trace("wlavm/start:createSignatureWithTPM() Entering")
	defer log.Trace("wlavm/start:createSignatureWithTPM() Leaving")

	var signingKey tpmprovider.CertifiedKey

	// Get the Signing Key that is stored on disk
	log.Debug("wlavm/start:createSignatureWithTPM() Getting the signing key from WA config path")
	signingKeyJson, err := config.GetSigningKeyFromFile()
	if err != nil {
		return nil, err
	}

	log.Debug("wlavm/start:createSignatureWithTPM() Unmarshalling the signing key file contents into signing key struct")
	err = json.Unmarshal(signingKeyJson, &signingKey)
	if err != nil {
		return nil, err
	}

	// // Get the secret associated when the SigningKey was created.
	// log.Debug("wlavm/start:createSignatureWithTPM() Retrieving the signing key secret form WA configuration")
	// keyAuth, err := hex.DecodeString(config.Configuration.SigningKeySecret)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "wlavm/start.go:createSignatureWithTPM() Error retrieving the signing key secret from configuration")
	// }

	// Before we compute the hash, we need to check the version of TPM as TPM 1.2 only supports SHA1
	TpmMtx.Lock()
	defer TpmMtx.Unlock()
	t, err := util.GetTpmInstance()
	if err != nil {
		return nil, errors.Wrap(err, "wlavm/start.go:createSignatureWithTPM() Error attempting to create signature - could not open TPM")
	}

	log.Debug("wlavm/start:createSignatureWithTPM() Computing the hash of the report to be signed by the TPM")
	h, err := crypt.GetHashData(data, alg)
	if err != nil {
		return nil, errors.Wrap(err, "wlavm/start:createSignatureWithTPM() Error while getting hash of instance report")
	}

	secLog.Infof("wlavm/start:createSignatureWithTPM() %s, Using TPM to sign the hash", message.SU)
	signature, err := t.Sign(&signingKey, config.Configuration.SigningKeySecret, h)
	if err != nil {
		return nil, errors.Wrap(err, "wlavm/start:createSignatureWithTPM() Error while creating tpm signature")
	}
	log.Debug("wlavm/start:createSignatureWithTPM() Report signed by TPM successfully")
	return signature, nil
}

// checkMountPathExistsAndMountVolume method is used to check if te mount path exists,
// if it does not exists, the method creates the mount path and mounts the device mapper.
func checkMountPathExistsAndMountVolume(mountPath, deviceMapperPath, emptyFileName string) error {
	log.Trace("wlavm/start:checkMountPathExistsAndMountVolume() Entering")
	defer log.Trace("wlavm/start:checkMountPathExistsAndMountVolume() Leaving")

	secLog.Infof("wlavm/start:checkMountPathExistsAndMountVolume() %s, Mounting the device mapper: %s", message.SU, deviceMapperPath)
	mkdirErr := os.MkdirAll(mountPath, 0655)
	if mkdirErr != nil {
		return errors.New("wlavm/start.go:checkMountPathExistsAndMountVolume() error while creating the mount point for the image device mapper")
	}

	// create an empty image and disk file so that symlinks are not broken after VM stop
	emptyFilePath := mountPath + "/" + emptyFileName
	secLog.Infof("wlavm/start:checkMountPathExistsAndMountVolume() %s, Creating an empty file in : %s", message.SU, emptyFilePath)
	sampleFile, err := os.OpenFile(emptyFilePath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0640)
	if err != nil {
		return errors.New("wlavm/start.go:checkMountPathExistsAndMountVolume() error creating a sample file")
	}
	defer func() {
		derr := sampleFile.Close()
		if derr != nil {
			log.WithError(derr).Error("Error closing file")
		}
	}()

	// get the qemu user info and change image and vm file owner to qemu
	userID, groupID, err := userInfoLookUp("qemu")
	if err != nil {
		return err
	}

	// change the image mount path directory ownership to qemu
	secLog.Infof("wlavm/start.go:checkMountPathExistsAndMountVolume() %s, Changing the mount path ownership to qemu", message.SU)
	err = osutil.ChownR(mountPath, userID, groupID)
	if err != nil {
		return errors.New("wlavm/start.go:checkMountPathExistsAndMountVolume() error trying to change mount path owner to qemu")
	}

	secLog.Infof("wlavm/start.go:checkMountPathExistsAndMountVolume() %s, Mounting the image device mapper %s on %s", message.SU, deviceMapperPath, mountPath)
	mountErr := vml.Mount(deviceMapperPath, mountPath)
	if mountErr != nil {
		if !strings.Contains(mountErr.Error(), "device is already mounted") {
			return errors.New("wlavm/start.go:checkMountPathExistsAndMountVolume() error while mounting the image device mapper")
		}
	}
	return nil
}

func userInfoLookUp(userName string) (int, int, error) {
	log.Trace("wlavm/start:userInfoLookUp() Entering")
	defer log.Trace("wlavm/start:userInfoLookUp() Leaving")

	userInfo, err := user.Lookup(userName)
	if err != nil {
		return 0, 0, errors.New("wlavm/start.go:checkMountPathExistsAndMountVolume() error trying to look up qemu userID and groupID")
	}
	userID, _ := strconv.Atoi(userInfo.Uid)
	groupID, _ := strconv.Atoi(userInfo.Gid)

	return userID, groupID, nil
}
