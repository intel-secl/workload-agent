// +build linux

/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */

package wlavm

import (
	"fmt"
	wlsModel "github.com/intel-secl/intel-secl/v3/pkg/model/wls"
	"intel/isecl/lib/common/v3/crypt"
	"intel/isecl/lib/common/v3/exec"
	"intel/isecl/lib/common/v3/log/message"
	osutil "intel/isecl/lib/common/v3/os"
	pinfo "intel/isecl/lib/platform-info/v3/platforminfo"
	"intel/isecl/lib/vml/v3"
	wlsclient "intel/isecl/wlagent/v3/clients"
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

// Prepare method is used perform the VM confidentiality check before launching the VM
// Input Parameters: domainXML content string
// Return : Returns a boolean value to the main method.
// true if the vm is launched successfully, else returns false.
func Prepare(domainXMLContent string, filewatcher *filewatch.Watcher) bool {

	log.Trace("wlavm/prepare:Prepare() Entering")
	defer log.Trace("wlavm/prepare:Prepare() Leaving")
	var skipImageVolumeCreation = false
	var err error
	var isImageEncrypted bool

	log.Info("wlavm/prepare:Prepare() Parsing domain XML to get image UUID, image path, VM UUID, VM path and disk size")
	d, err := libvirt.NewDomainParser(domainXMLContent, libvirt.Prepare)
	if err != nil {
		log.Error("wlavm/prepare:Prepare() Parsing error: ", err.Error())
		log.Tracef("%+v", err)
		return false
	}

	vmUUID := d.GetVMUUID()
	vmPath := d.GetVMPath()
	imageUUID := d.GetImageUUID()
	imagePath := d.GetImagePath()
	size := d.GetDiskSize()
	var vmVirtualSize string
	var vmVirtualFormat string
	var vmBackFileFormat string
	var key []byte
	isImageDecrypted := false
	isVmShutoff := false
	mustRecreateVMDisk := false
	decryptedImagePath := ""

	// Step 1 - Check if the VM is in shutoff state - detect the symlink from VM Disk to Volume
	// if not, this is a fresh launch
	// Step 2 - Deduce the backing image path from qemu-img info on the VM Disk - this gives us path to encrypted disk - imagePath
	// Step 3 - Check if the backing image is encrypted - if yes, go to step 3.1, if not go to step 4
	// Step 3.1 - Decrypt the backing image
	// Step 3.2 - if mustRecreateVMDisk is set, then go to step 3.3
	// Step 3.3 - detect the format and virtual file size of the VM disk - virtualFormat, virtualSize
	// Step 3.4 - Recreate the VM Disk - qemu-img create pointing to the decrypted backingFile
	// Step 3.5 - Resize the VM disk - qemu-img resize
	// Step 4 - Create the VM disk volume
	// Step 5 - Move to Start stage

	// Step 1 - Check if the VM is in shutoff state
	// consider the possibility of this being a VM Startup from Shutoff state
	// check if the VM Image volume exists already
	vmSymLinkOut, vmSymlinkReadErr := os.Readlink(vmPath)
	if vmSymlinkReadErr != nil {
		mustRecreateVMDisk = true
		// discover backing file path via qemu-img info on VM disk file
		getBackingFileOutput, err := exec.ExecuteCommand(consts.QemuImgUtilPath,
			strings.Fields(fmt.Sprintf(consts.GetImgInfoCmd, vmPath)))
		if err != nil {
			log.Errorf("wlavm/prepare:Prepare() Error discovering backing file path: %s", err.Error())
			return false
		}

		// set the image path and continue with prepare stage
		for _, line := range strings.Split(getBackingFileOutput, "\n") {
			lineSplit := strings.Split(strings.TrimSpace(line), ": ")
			if len(lineSplit) > 1 {
				switch lineSplit[0] {
				case consts.QemuImgInfoBackingFileField:
					imagePath = lineSplit[1]
					log.Debugf("wlavm/prepare:Prepare() Backing file path for VM : %s", imagePath)
				}
			}
		}
	}

	// in case of reboot from VM shutoff no image path is available from the domain xml
	// fetch from the association file
	if imagePath == "" {
		imagePath = imagePathFromVMAssociationFile(imageUUID)
		if imagePath == "" {
			log.Errorf("wlavm/prepare:Prepare() Error while retrieving image path from image-vm association "+
				"file for image %s", imageUUID)
			return false
		} else {
			isVmShutoff = true
		}
	}

	// Step 2 - check if the image is decrypted
	isImageEncrypted, err = crypt.EncryptionHeaderExists(imagePath)
	if err != nil {
		log.Errorf("wlavm/prepare:Prepare() Error while trying to check if the image is encrypted: %s", err.Error())
		log.Tracef("%+v", err)
		return false
	}

	if isImageEncrypted {
		log.Info("wlavm/prepare:Prepare() Checking if the image file has already been decrypted")
		decryptedImagePath = consts.MountPath + imageUUID + "/" + imageUUID
		imageFileStat, imageFileStatErr := os.Stat(decryptedImagePath)
		if imageFileStatErr == nil && imageFileStat.Size() > 0 {
			isImageDecrypted, err = crypt.EncryptionHeaderExists(imagePath)
			if err == nil && !isImageDecrypted {
				log.Info("wlavm/prepare:Prepare() The image is already decrypted, " +
					"so will be skipping the image dm-crypt volume creation")
				skipImageVolumeCreation = true
			}
		}
	} else if !isVmShutoff {
		log.Info("wlavm/prepare:Prepare() Image is not encrypted, returning to the hook")
		return true
	}

	// Step 3 - if encrypted pull flavor and key
	if isImageEncrypted {
		var flavorKeyInfo wlsModel.FlavorKey
		var tpmWrappedKey []byte

		// get host hardware UUID
		secLog.Infof("wlavm/prepare:Prepare() %s, Trying to get host hardware UUID", message.SU)
		hardwareUUID, err := pinfo.HardwareUUID()
		if err != nil {
			log.WithError(err).Error("wlavm/prepare:Prepare() Unable to get the host hardware UUID")
			return false
		}
		log.Debugf("wlavm/prepare:Prepare() The host hardware UUID is :%s", hardwareUUID)

		//get flavor-key from workload service
		// we will be hitting the WLS to retrieve the the flavor and key.
		log.Infof("wlavm/prepare:Prepare() Retrieving image-flavor-key for image %s from WLS", imageUUID)

		flavorKeyInfo, err = wlsclient.GetImageFlavorKey(imageUUID, hardwareUUID)
		if err != nil {
			secLog.WithError(err).Error("wlavm/prepare:Prepare() Error retrieving the image flavor and key")
			return false
		}

		if flavorKeyInfo.Flavor.Meta.ID == "" {
			log.Infof("wlavm/prepare:Prepare() Flavor does not exist for the image %s", imageUUID)
			return false
		}

		if flavorKeyInfo.Flavor.EncryptionRequired {
			if len(flavorKeyInfo.Key) == 0 {
				log.Error("wlavm/prepare:Prepare() Flavor Key is empty")
				return false
			}
			tpmWrappedKey = flavorKeyInfo.Key
			// unwrap key
			log.Info("wlavm/prepare:Prepare() Unwrapping the key...")
			TpmMtx.Lock()
			key, unWrapErr := util.UnwrapKey(tpmWrappedKey)
			TpmMtx.Unlock()
			if unWrapErr != nil {
				secLog.WithError(err).Error("wlavm/prepare:Prepare() Error unwrapping the key")
				return false
			}

			// decrypt and mount the VM image
			if !skipImageVolumeCreation {
				log.Info("wlavm/prepare:Prepare() Creating and mounting image dm-crypt volume")
				err = imageVolumeManager(imageUUID, imagePath, size, key)
				if err != nil {
					log.WithError(err).Error("wlavm/prepare:Prepare() Error while creating and mounting image dm-crypt volume ")
					return false
				}

				// discover via qemu-img info on decrypted image file
				qemuImgInfoOutput, err := exec.ExecuteCommand(consts.QemuImgUtilPath, strings.Fields(fmt.Sprintf(consts.GetImgInfoCmd, decryptedImagePath)))
				if err != nil {
					log.Errorf("wlavm/prepare:Prepare() Error discovering backing file path: %s", err.Error())
					return false
				}

				// set the image path and continue with prepare stage
				for _, line := range strings.Split(qemuImgInfoOutput, "\n") {
					lineSplit := strings.Split(strings.TrimSpace(line), ": ")
					if len(lineSplit) > 1 {
						switch lineSplit[0] {
						case consts.QemuImgInfoFileFormatField:
							vmBackFileFormat = lineSplit[1]
							log.Debugf("wlavm/prepare:Prepare() Backing File format: %s", vmBackFileFormat)
						}
					}
				}
			}
		}
	}

	// check if the VM Image has been decrypted and remounted
	vmSymLinkOut, vmSymlinkReadErr = os.Readlink(vmPath)
	if vmSymlinkReadErr != nil {
		log.Debugf("wlavm/prepare:Prepare() VM Disk Path %s is not a symlink - %s", vmPath, vmSymlinkReadErr.Error())
	}
	vmSymLinkStat, vmSymLinkStatErr := os.Stat(vmSymLinkOut)
	if vmSymlinkReadErr != nil || vmSymLinkStatErr != nil || vmSymLinkStat.Size() == 0 {
		if mustRecreateVMDisk {
			// since we need to recreate the VM disk file - discover info via qemu-img
			queryVmDisk, err := exec.ExecuteCommand(consts.QemuImgUtilPath,
				strings.Fields(fmt.Sprintf(consts.GetImgInfoCmd, vmPath)))
			if err != nil {
				log.Errorf("wlavm/prepare:Prepare() Error discovering backing file path: %s", err.Error())
				return false
			}

			// set the image properties
			for _, line := range strings.Split(queryVmDisk, "\n") {
				lineSplit := strings.Split(strings.TrimSpace(line), ": ")
				if len(lineSplit) > 1 {
					switch lineSplit[0] {
					case consts.QemuImgInfoBackingFileField:
						imagePath = lineSplit[1]
						log.Debugf("wlavm/prepare:Prepare() Original Backing file path for VM : %s", imagePath)
						log.Debugf("wlavm/prepare:Prepare() Decrypted Image Backing file path : %s", decryptedImagePath)
					case consts.QemuImgInfoVirtualSizeField:
						vmVirtualSize = strings.Split(strings.Split(line, "(")[1], " ")[0]
						log.Debugf("wlavm/prepare:Prepare() VM Virtual Disk Size: %s", vmVirtualSize)
					case consts.QemuImgInfoFileFormatField:
						vmVirtualFormat = lineSplit[1]
						log.Debugf("wlavm/prepare:Prepare() VM Virtual Disk Format: %s", vmVirtualFormat)
					}
				}
			}

			recreateVMDiskOutput, err := exec.ExecuteCommand(consts.QemuImgUtilPath, strings.Fields(
				fmt.Sprintf(consts.CreateVmDiskCmd, vmVirtualFormat, decryptedImagePath, vmBackFileFormat, vmPath)))
			if err != nil {
				log.Errorf("wlavm/prepare:Prepare() Error recreating VM disk file: %s", err.Error())
				return false
			}
			log.Debugf("wlavm/prepare:Prepare() Reformatting VM disk: %s", recreateVMDiskOutput)

			// resize the disk file per the Nova flavor
			resizeDiskFileOutput, err := exec.ExecuteCommand(consts.QemuImgUtilPath, strings.Fields(
				fmt.Sprintf(consts.ResizeVmDiskCmd, vmPath, vmVirtualSize)))
			if err != nil {
				log.Errorf("wlavm/prepare:Prepare() Error resizing VM disk: %s", err.Error())
				return false
			}
			log.Debugf("wlavm/prepare:Prepare() Resizing VM disk output: %s", resizeDiskFileOutput)
		}
	}

	log.Info("wlavm/prepare:Prepare() Creating and mounting vm dm-crypt volume")
	err = vmVolumeManager(vmUUID, vmPath, size, key, filewatcher)
	if err != nil {
		log.WithError(err).Error("wlavm/prepare:Prepare() Error while creating and mounting vm dm-crypt volume ")
		return false
	}

	log.Infof("wlavm/prepare:Prepare() VM %s prepared", vmUUID)
	return true
}

func vmVolumeManager(vmUUID string, vmPath string, size int, key []byte, filewatcher *filewatch.Watcher) error {
	log.Trace("wlavm/prepare:vmVolumeManager() Entering")
	defer log.Trace("wlavm/prepare:vmVolumeManager() Leaving")

	// create vm volume
	var err error
	vmDeviceMapperPath := consts.DevMapperDirPath + vmUUID
	vmSparseFilePath := strings.Replace(vmPath, "disk", vmUUID+"_sparse", -1)
	// check if sparse file exists, if it does, skip copying the change disk file to mount point
	_, sparseFleStatErr := os.Stat(vmSparseFilePath)
	vmVolumeMtx.Lock()
	secLog.Infof("wlavm/prepare:vmVolumeManager() %s, Creating VM dm-crypt volume in %s", message.SU, vmDeviceMapperPath)
	err = vml.CreateVolume(vmSparseFilePath, vmDeviceMapperPath, key, size)
	vmVolumeMtx.Unlock()
	if err != nil {
		return errors.Wrap(err, "wlavm/prepare:vmVolumeManager() error creating vm dm-crypt volume")
	}

	// mount the vm dmcrypt volume on to a mount path
	log.Debug("wlavm/prepare:vmVolumeManager() Mounting the vm volume on a mount path")
	var vmMountPath = consts.MountPath + vmUUID
	err = checkMountPathExistsAndMountVolume(vmMountPath, vmDeviceMapperPath, "disk")
	if err != nil {
		return errors.Wrap(err, "wlavm/prepare:vmVolumeManager() error checking if mount path exists and mounting the volume")
	}

	// is sparse file exists, the image is already decrypted, so returning back to VM start method
	if !os.IsNotExist(sparseFleStatErr) {
		log.Debug("wlavm/prepare:vmVolumeManager() Sparse file of the VM already exists, skipping copying the change disk to mount point...")
		return nil
	}

	// copy the files from vm path
	args := []string{vmPath, vmMountPath}
	secLog.Infof("wlavm/prepare:vmVolumeManager() %s, Copying all the files from %s to vm mount path", message.SU, vmPath)
	_, err = exec.ExecuteCommand("cp", args)
	if err != nil {
		return errors.Wrapf(err, "wlavm/prepare:vmVolumeManager() error copying the vm path %s change disk to mount path. %s", vmPath, vmMountPath)
	}

	// remove the encrypted image file and create a symlink with the dm-crypt volume
	secLog.Infof("wlavm/prepare:vmVolumeManager() %s, Deleting change disk: %s", message.SU, vmPath)
	err = os.RemoveAll(vmPath)
	if err != nil {
		return errors.Wrapf(err, "wlavm/prepare:vmVolumeManager() error deleting the change disk: %s", vmPath)
	}

	changeDiskFile := vmMountPath + "/disk"
	log.Debug("wlavm/prepare:vmVolumeManager() Creating a symlink between the vm and the volume")
	// create symlink between the image and the dm-crypt volume
	err = createSymLinkAndChangeOwnership(changeDiskFile, vmPath, vmMountPath)
	if err != nil {
		return errors.Wrap(err, "wlavm/prepare:vmVolumeManager() error creating a symlink and changing file ownership")
	}

	// trigger a file watcher event to delete VM mount path when disk.info file is deleted on VM delete
	vmDiskInfoFile := strings.Replace(vmPath, "disk", "disk.info", -1)
	// Watch the symlink for deletion, and remove the _sparseFile if image is deleted
	err = filewatcher.HandleEvent(vmDiskInfoFile, func(e fsnotify.Event) {
		if e.Op&fsnotify.Remove == fsnotify.Remove {
			secLog.Infof("wlavm/prepare:vmVolumeManager() %s, Removing vm mount path at: %s", message.SU, vmMountPath)
			err = os.RemoveAll(vmMountPath)
			if err != nil {
				log.Errorf("wlavm/prepare:vmVolumeManager() Failed to remove mount path")
			}
		}
	})
	if err != nil {
		log.Errorf("wlavm/prepare:vmVolumeManager() Failed to handle event: %v", err)
	}

	return nil
}

func imageVolumeManager(imageUUID string, imagePath string, size int, key []byte) error {
	log.Trace("wlavm/prepare:imageVolumeManager() Entering")
	defer log.Trace("wlavm/prepare:imageVolumeManager() Leaving")

	// create image dm-crypt volume
	var err error
	imageDeviceMapperPath := consts.DevMapperDirPath + imageUUID
	sparseFilePath := imagePath + "_sparseFile"
	// check if the sparse file already exists, if it does, skip image file decryption
	_, sparseFileStatErr := os.Stat(sparseFilePath)
	imgVolumeMtx.Lock()
	secLog.Infof("wlavm/prepare:imageVolumeManager() %s, Creating a dm-crypt volume for the image %s", message.SU, imageUUID)
	err = vml.CreateVolume(sparseFilePath, imageDeviceMapperPath, key, size)
	imgVolumeMtx.Unlock()
	if err != nil {
		if strings.Contains(err.Error(), "device mapper of the same already exists") {
			log.Debug("wlavm/prepare:imageVolumeManager() Device mapper of same name already exists. Skipping image volume creation..")
			return nil
		} else {
			return errors.Wrap(err, "wlavm/prepare:imageVolumeManager() error while creating image dm-crypt volume for image")
		}
	}

	//check if the image device mapper is mount path exists, if not create it
	imageDeviceMapperMountPath := consts.MountPath + imageUUID
	err = checkMountPathExistsAndMountVolume(imageDeviceMapperMountPath, imageDeviceMapperPath, imageUUID)
	if err != nil {
		return errors.Wrap(err, "wlavm/prepare:imageVolumeManager() error checking if image mount path exists and mounting the volume")
	}

	// is sparse file exists, the image is already decrypted, so returning back to VM start method
	if !os.IsNotExist(sparseFileStatErr) {
		log.Debug("wlavm/prepare:imageVolumeManager() Sparse file of the image already exists, skipping image decryption...")
		return nil
	}

	// read image file contents
	log.Debug("Reading the encrypted image file...")
	encryptedImage, ioReadErr := ioutil.ReadFile(imagePath)
	if ioReadErr != nil {
		return errors.New("wlavm/prepare:imageVolumeManager() error while reading the image file")
	}

	//decrypt the image
	log.Info("wlavm/prepare:imageVolumeManager() Decrypting the image")
	decryptedImage, err := vml.Decrypt(encryptedImage, key)
	if err != nil {
		return errors.Wrap(err, "wlavm/prepare:imageVolumeManager() error while decrypting the image")
	}
	log.Info("wlavm/prepare:imageVolumeManager() Image decrypted successfully")
	// write the decrypted data into a file in image mount path
	decryptedImagePath := imageDeviceMapperMountPath + "/" + imageUUID
	secLog.Infof("wlavm/prepare:imageVolumeManager() %s, Writing decrypted data in to a file: %s", message.SU, decryptedImagePath)
	ioWriteErr := ioutil.WriteFile(decryptedImagePath, decryptedImage, 0664)
	if ioWriteErr != nil {
		return errors.New("wlavm/prepare:imageVolumeManager() error writing the decrypted data to file")
	}

	// get the qemu user info and change image and vm file owner to qemu
	userID, groupID, err := userInfoLookUp("qemu")
	if err != nil {
		return err
	}

	// Stop using symlinks for decrypted images are these seem to be manipulated in the backend by nova for VM launch
	// that arrive simultaneously. Between detecting the symlink and the actual launch, the symlink seems to be replaced
	// with the actual encrypted file by nova - this is probably something that should be expected as nova-compute
	// behavior

	// Giving read write access to the decrypted image file
	secLog.Infof("wlavm/prepare:imageVolumeManager() %s, Changing permissions to changed disk path: %s", message.SU, decryptedImagePath)
	err = os.Chmod(decryptedImagePath, 0664)
	if err != nil {
		return errors.Wrapf(err, "wlavm/prepare:imageVolumeManager() error while giving permissions to changed disk path: %s", decryptedImagePath)
	}

	// change the image mount path directory ownership to qemu
	secLog.Infof("wlavm/prepare:imageVolumeManager() %s, Changing the mount path ownership to qemu", message.SU)
	err = osutil.ChownR(imageDeviceMapperMountPath, userID, groupID)

	if err != nil {
		return errors.New("wlavm/prepare:imageVolumeManager() error trying to change mount path owner to qemu")
	}

	return nil
}

func createSymLinkAndChangeOwnership(targetFile, sourceFile, mountPath string) error {
	log.Trace("wlavm/prepare:createSymLinkAndChangeOwnership() Entering")
	defer log.Trace("wlavm/prepare:createSymLinkAndChangeOwnership() Leaving")

	// get the qemu user info and change image and vm file owner to qemu
	userID, groupID, err := userInfoLookUp("qemu")
	if err != nil {
		return err
	}

	// remove the encrypted image file and create a symlink with the dm-crypt volume
	secLog.Infof("wlavm/prepare:createSymLinkAndChangeOwnership() %s, Deleting the enc image file from :%s", message.SU, sourceFile)
	rmErr := os.RemoveAll(sourceFile)
	if rmErr != nil {
		return errors.Wrapf(err, "wlavm/prepare:createSymLinkAndChangeOwnership() error deleting the change disk: %s", sourceFile)
	}

	// create symlink between the image and the dm-crypt volume
	secLog.Infof("wlavm/prepare:imageVolumeManager() %s, Creating a symlink between %s and %s", message.SU, sourceFile, targetFile)
	err = os.Symlink(targetFile, sourceFile)
	if err != nil {
		return errors.Wrapf(err, "wlavm/prepare:createSymLinkAndChangeOwnership() error while creating symbolic link")
	}

	// change the image symlink file ownership to qemu
	secLog.Infof("wlavm/prepare:createSymLinkAndChangeOwnership() %s, Changing symlink ownership to qemu", message.SU)
	err = os.Lchown(sourceFile, userID, groupID)
	if err != nil {
		return errors.New("wlavm/prepare:createSymLinkAndChangeOwnership() error while trying to change symlink owner to qemu")
	}

	// Giving read write access to the target file
	secLog.Infof("wlavm/prepare:createSymLinkAndChangeOwnership() %s, Changing permissions to changed disk path: %s", message.SU, targetFile)
	err = os.Chmod(targetFile, 0664)
	if err != nil {
		return errors.Wrapf(err, "wlavm/prepare:createSymLinkAndChangeOwnership() error while giving permissions to changed disk path: %s", targetFile)
	}

	// change the image mount path directory ownership to qemu
	secLog.Infof("wlavm/prepare:createSymLinkAndChangeOwnership() %s, Changing the mount path ownership to qemu", message.SU)
	err = osutil.ChownR(mountPath, userID, groupID)

	if err != nil {
		return errors.New("wlavm/prepare:createSymLinkAndChangeOwnership() error trying to change mount path owner to qemu")
	}
	return nil
}

// checkMountPathExistsAndMountVolume method is used to check if te mount path exists,
// if it does not exists, the method creates the mount path and mounts the device mapper.
func checkMountPathExistsAndMountVolume(mountPath, deviceMapperPath, emptyFileName string) error {
	log.Trace("wlavm/prepare:checkMountPathExistsAndMountVolume() Entering")
	defer log.Trace("wlavm/prepare:checkMountPathExistsAndMountVolume() Leaving")

	secLog.Infof("wlavm/prepare:checkMountPathExistsAndMountVolume() %s, Mounting the device mapper: %s", message.SU, deviceMapperPath)
	mkdirErr := os.MkdirAll(mountPath, 0655)
	if mkdirErr != nil {
		return errors.New("wlavm/prepare:checkMountPathExistsAndMountVolume() error while creating the mount point for the image device mapper")
	}

	// create an empty image and disk file so that symlinks are not broken after VM stop
	emptyFilePath := mountPath + "/" + emptyFileName
	secLog.Infof("wlavm/prepare:checkMountPathExistsAndMountVolume() %s, Creating an empty file in : %s", message.SU, emptyFilePath)
	// creating a file with 664 permissions so that nova and qemu can also access it
	sampleFile, err := os.OpenFile(emptyFilePath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0664)
	if err != nil {
		return errors.New("wlavm/prepare:checkMountPathExistsAndMountVolume() error creating a sample file")
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
	secLog.Infof("wlavm/prepare:checkMountPathExistsAndMountVolume() %s, Changing the mount path ownership to qemu", message.SU)
	err = osutil.ChownR(mountPath, userID, groupID)
	if err != nil {
		return errors.New("wlavm/prepare:checkMountPathExistsAndMountVolume() error trying to change mount path owner to qemu")
	}

	secLog.Infof("wlavm/prepare:checkMountPathExistsAndMountVolume() %s, Mounting the image device mapper %s on %s", message.SU, deviceMapperPath, mountPath)
	mountErr := vml.Mount(deviceMapperPath, mountPath)
	if mountErr != nil {
		if !strings.Contains(mountErr.Error(), "device is already mounted") {
			return errors.New("wlavm/prepare:checkMountPathExistsAndMountVolume() error while mounting the image device mapper")
		}
	}
	return nil
}

func userInfoLookUp(userName string) (int, int, error) {
	log.Trace("wlavm/prepare:userInfoLookUp() Entering")
	defer log.Trace("wlavm/prepare:userInfoLookUp() Leaving")

	userInfo, err := user.Lookup(userName)
	if err != nil {
		return 0, 0, errors.Wrap(err, "wlavm/prepare:userInfoLookUp() error trying to look up qemu userID and groupID")
	}
	userID, _ := strconv.Atoi(userInfo.Uid)
	groupID, _ := strconv.Atoi(userInfo.Gid)

	return userID, groupID, nil
}
