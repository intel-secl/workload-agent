/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
// +build linux

package wlavm

import (
	"intel/isecl/lib/common/exec"
	"intel/isecl/lib/vml"
	"intel/isecl/lib/common/log/message"
	"intel/isecl/wlagent/consts"
	"intel/isecl/wlagent/filewatch"
	"intel/isecl/wlagent/libvirt"
	"os"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

var (
	isVmVolume bool
	isLastVm   bool
	imagePath  string
)

var (
	mtx sync.Mutex
)

// Stop is called from the libvirt hook. Everytime stop cycle is called
// in any of the VM lifecycle events, this method will be called.
// e.g. shutdown, reboot, stop etc.
// Input Parameters: domainXML content string
// Return : Returns a boolean value to the main method.
// true if the vm is launched sucessfully, else returns false.
func Stop(domainXMLContent string, filewatcher *filewatch.Watcher) bool {
	log.Trace("wlavm/stop:Stop() Entering")
	defer log.Trace("wlavm/stop:Stop() Leaving")
	log.Info("wlavm/stop:Stop() Parsing domain XML to get image UUID, VM UUID and VM path")

	d, err := libvirt.NewDomainParser(domainXMLContent, libvirt.Stop)
	if err != nil {
		log.Error("wlavm/stop.go:Stop() Parsing error")
		return false
	}

	// check if vm exists at given path
	log.Infof("Checking if VM exists in %s", d.GetVMPath())
	if _, err := os.Stat(d.GetVMPath()); os.IsNotExist(err) {
		log.Error("wlavm/stop.go:Stop() VM does not exist")
		return false
	}

	// check if the vm volume is encrypted
	log.Info("wlavm/stop:Stop() Checking if a dm-crypt volume for the image is created")
	isVmVolume, err := isVmVolumeEncrypted(d.GetVMUUID())
	if err != nil {
		log.Error("wlavm/stop.go:Stop() Error while checking if a dm-crypt volume is created for the VM and is active")
		log.Tracef("%+v", err)
		return false
	}
	// if vm volume is encrypted, close the volume
	if isVmVolume {
		var vmMountPath = consts.MountPath + d.GetVMUUID()
		// Unmount the image
		secLog.Infof("wlavm/stop:Stop() A dm-crypt volume for the image is created, deleting the vm volume %s", message.SU)
		vml.Unmount(vmMountPath)
		vml.DeleteVolume(d.GetVMUUID())
	}

	// check if this is the last vm associated with the image
	log.Info("wlavm/stop:Stop() Checking if this is the last vm using the image...")
	iAssoc := ImageVMAssociation{d.GetImageUUID(), ""}
	isLastVm, imagePath, err = iAssoc.Delete()
	if err != nil {
		log.WithError(err).Error("wlavm/stop:Stop() Error while image association deletion")
		log.Tracef("%+v", err)
		return false
	}
	// as the original image is deleted during the VM start process, there is no way
	// to check if original image is encrypted. Instead we check if sparse file of image
	// exists at given path, if it does that means the image was enrypted and volumes were created
	if _, err := os.Stat(imagePath + "_sparseFile"); os.IsNotExist(err) {
		log.Info("wlavm/stop:Stop() The base image is not ecrypted, returning to hook...")
		return true
	}

	// check if this is the last vm associated with the image
	if !isLastVm {
		log.Infof("wlavm/stop:Stop() Not deleting the image volume as this is not the last vm using the image, VM %s stopped", d.GetVMUUID())
		return true
	}

	log.Info("wlavm/stop:Stop() Unmounting and deleting the image volume as this is the last vm using the image")

	mtx.Lock()
	defer mtx.Unlock()
	var imageMountPath = consts.MountPath + d.GetImageUUID()
	secLog.Infof("wlavm/stop:Stop() Unmounting the image volume: %s, %s", imageMountPath, message.SU)

	// Unmount the image
	vml.Unmount(imageMountPath)
	secLog.Infof("wlavm/stop:Stop() Deleting the image volume: %s, %s", d.GetImageUUID(), message.SU)
	// Close the image volume
	vml.DeleteVolume(d.GetImageUUID())
	log.Infof("wlavm/stop:Stop() VM %s stopped", d.GetVMUUID())
	return true
}

func isVmVolumeEncrypted(vmUUID string) (bool, error) {
	log.Trace("wlavm/stop:isVmVolumeEncrypted() Entering")
	defer log.Trace("wlavm/stop:isVmVolumeEncrypted() Leaving")

	// check the status of the device mapper
	log.Debug("wlavm/stop:isVmVolumeEncrypted() Checking the status of the device mapper")
	log.Debugf("wlavm/stop:isVmVolumeEncrypted() Checking for volume with UUID:%s is encrypted", vmUUID)
	deviceMapperLocation := consts.DevMapperDirPath + vmUUID
	args := []string{"status", deviceMapperLocation}


	secLog.Infof("Checking for volume with UUID:%s is encrypted, %s", vmUUID, message.SU)
	cmdOutput, err := exec.ExecuteCommand("cryptsetup", args)

	if cmdOutput != "" && strings.Contains(cmdOutput, "inactive") {
		log.Debug("wlavm/stop:isVmVolumeEncrypted() The device mapper is inactive")
		return false, nil
	}

	if err != nil {
		return false, errors.Wrap(err, "wlavm/stop.go:isVmVolumeEncrypted() error occured while executing cryptsetup status command")
	}
	log.Debug("wlavm/stop:isVmVolumeEncrypted() The device mapper is encrypted and active")
	return true, nil
}
