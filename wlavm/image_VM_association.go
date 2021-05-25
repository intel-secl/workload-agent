/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package wlavm

import (
	"github.com/pkg/errors"
	cLog "intel/isecl/lib/common/v4/log"
	"intel/isecl/wlagent/v4/util"
)

var log = cLog.GetDefaultLogger()
var secLog = cLog.GetSecurityLogger()

// ImageVMAssociation with ID and path
type ImageVMAssociation struct {
	ImageUUID string
	ImagePath string
}

// Create method is used to check if an entry exists with the image ID. If it does, increment the instance count,
// else create an entry with image instance association and append it.
func (IAssoc ImageVMAssociation) Create() error {
	log.Trace("wlavm/image_VM_association:Create() Entering")
	defer log.Trace("wlavm/image_VM_association:Create() Leaving")

	log.Debug("wlavm/image_VM_association:Create() Loading yaml file to vm image association structure.")
	util.MapMtx.Lock()
	imageAttributes, ok := util.ImageVMAssociations[IAssoc.ImageUUID]
	if ok {
		log.Debug("wlavm/image_VM_association:Create() Image ID already exist in file, increasing the count of vm by 1.")
		imageAttributes.VMCount = imageAttributes.VMCount + 1
	} else {
		log.Debug("wlavm/image_VM_association:Create() Image ID does not exist in file, adding an entry with the image ID ", IAssoc.ImageUUID)
		util.ImageVMAssociations[IAssoc.ImageUUID] = &util.ImageVMAssociation{ImagePath: IAssoc.ImagePath, VMCount: 1}
	}
	util.MapMtx.Unlock()
	err := util.SaveImageVMAssociation()
	if err != nil {
		return errors.Wrap(err, "wlavm/image_VM_association:Create() error occurred while saving image VM association to a file")
	}
	return nil
}

// Delete method is used to check if an entry exists with the image ID. If it does, decrement the vm count.
// Check if the vm count is zero, then delete the image entry from the file.
func (IAssoc ImageVMAssociation) Delete() (bool, string, error) {
	log.Trace("wlavm/image_VM_association:Delete() Entering")
	defer log.Trace("wlavm/image_VM_association:Delete() Leaving")

	imagePath := ""
	isLastVM := false

	util.MapMtx.Lock()
	imageAttributes, ok := util.ImageVMAssociations[IAssoc.ImageUUID]
	if ok && imageAttributes.VMCount > 0 {
		log.Debug("wlavm/image_VM_association:Delete() Image ID already exist in file, decreasing the count of vm by 1.")
		imageAttributes.VMCount = imageAttributes.VMCount - 1
		if imageAttributes.VMCount == 0 {
			isLastVM = true
		}
	} else {
		util.MapMtx.Unlock()
		return true, "", errors.New("wlavm/image_VM_association:Delete() image VM association does not exist")
	}

	util.MapMtx.Unlock()
	imagePath = imageAttributes.ImagePath

	err := util.SaveImageVMAssociation()
	if err != nil {
		return isLastVM, imagePath, errors.Wrap(err, "wlavm/image_VM_association:Delete() error occurred while saving image VM association to a file")
	}
	return isLastVM, imagePath, nil
}

// DeleteEntry then delete the image entry from the file.
func (IAssoc ImageVMAssociation) DeleteEntry() error {
	log.Trace("wlavm/image_VM_association:DeleteEntry() Entering")
	defer log.Trace("wlavm/image_VM_association:DeleteEntry() Leaving")

	util.MapMtx.Lock()
	_, ok := util.ImageVMAssociations[IAssoc.ImageUUID]
	if ok {
		log.Debug("wlavm/image_VM_association:DeleteEntry() Deleting the image-vm entries from the file.")
		delete(util.ImageVMAssociations, IAssoc.ImageUUID)
	}
	util.MapMtx.Unlock()

	err := util.SaveImageVMAssociation()
	if err != nil {
		return errors.Wrap(err, "wlavm/image_VM_association:DeleteEntry() error occurred while saving image VM"+
			" association to a file")
	}

	return nil
}

func imagePathFromVMAssociationFile(imageUUID string) string {
	log.Trace("wlavm/image_VM_association:imagePathFromVMAssociationFile() Entering")
	defer log.Trace("wlavm/image_VM_association:imagePathFromVMAssociationFile() Leaving")

	if len(util.ImageVMAssociations) == 0 {
		err := util.LoadImageVMAssociation()
		if err != nil {
			log.Error("wlavm/image_VM_association:imagePathFromVMAssociationFile() Error loading ImageVMAssociations file: %s", err.Error())
			return ""
		}
	}

	log.Debug("wlavm/image_VM_association:imagePathFromVMAssociationFile() Loading yaml file to vm image association structure.")
	util.MapMtx.RLock()
	imageAttributes, ok := util.ImageVMAssociations[imageUUID]
	util.MapMtx.RUnlock()
	if ok {
		return imageAttributes.ImagePath
	}
	return ""
}
