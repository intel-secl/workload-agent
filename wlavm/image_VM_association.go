/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package wlavm

import (
	cLog "intel/isecl/lib/common/v2/log"
	"intel/isecl/wlagent/v2/util"
	"sync"

	"github.com/pkg/errors"
)

var log = cLog.GetDefaultLogger()
var secLog = cLog.GetSecurityLogger()

// ImageVMAssociation with ID and path
type ImageVMAssociation struct {
	ImageUUID string
	ImagePath string
}

var fileMutex sync.Mutex

// Create method is used to check if an entry exists with the image ID. If it does, increment the instance count,
// else create an entry with image instance association and append it.
func (IAssoc ImageVMAssociation) Create() error {
	log.Trace("wlavm/image_VM_association:Create() Entering")
	defer log.Trace("wlavm/image_VM_association:Create() Leaving")
	fileMutex.Lock()
	defer fileMutex.Unlock()

	log.Debug("wlavm/image_VM_association:Create() Loading yaml file to vm image association structure.")
	err := util.LoadImageVMAssociation()
	if err != nil {
		return errors.Wrap(err, "wlavm/image_VM_association:Create() error occured while loading image VM association from a file")
	}
	ImageAttributes, Bool := util.ImageVMAssociations[IAssoc.ImageUUID]
	if Bool == true {
		log.Debug("wlavm/image_VM_association:Create() Image ID already exist in file, increasing the count of vm by 1.")
		ImageAttributes.VMCount = ImageAttributes.VMCount + 1
	} else {
		log.Debug("wlavm/image_VM_association:Create() Image ID does not exist in file, adding an entry with the image ID ", IAssoc.ImageUUID)
		util.ImageVMAssociations[IAssoc.ImageUUID] = &util.ImageVMAssociation{IAssoc.ImagePath, 1}
	}
	err = util.SaveImageVMAssociation()
	if err != nil {
		return errors.Wrap(err, "wlavm/image_VM_association:Create() error occured while saving image VM association to a file")
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

	fileMutex.Lock()
	defer fileMutex.Unlock()

	err := util.LoadImageVMAssociation()
	if err != nil {
		return isLastVM, imagePath, errors.Wrap(err, "wlavm/image_VM_association:Delete() error occured while loading image VM association from a file")
	}
	ImageAttributes, Bool := util.ImageVMAssociations[IAssoc.ImageUUID]
	imagePath = ImageAttributes.ImagePath
	if Bool == true && ImageAttributes.VMCount > 0 {
		log.Debug("wlavm/image_VM_association:Delete() Image ID already exist in file, decreasing the count of vm by 1.")
		ImageAttributes.VMCount = ImageAttributes.VMCount - 1
		if ImageAttributes.VMCount == 0 {
			isLastVM = true
		}
	}

	err = util.SaveImageVMAssociation()
	if err != nil {
		return isLastVM, imagePath, errors.Wrap(err, "wlavm/image_VM_association:Delete() error occured while saving image VM association to a file")
	}
	return isLastVM, imagePath, nil
}

func imagePathFromVMAssociationFile(imageUUID string) (string, error) {
	log.Trace("wlavm/image_VM_association:imagePathFromVMAssociationFile() Entering")
	defer log.Trace("wlavm/image_VM_association:imagePathFromVMAssociationFile() Leaving")

	log.Debug("wlavm/image_VM_association:imagePathFromVMAssociationFile() Checking if the image UUID exists in image-vm association file")
	log.Debug("wlavm/image_VM_association:imagePathFromVMAssociationFile() Loading yaml file to vm image association structure.")
	err := util.LoadImageVMAssociation()
	if err != nil {
		return "", errors.Wrap(err, "wlavm/image_VM_association:imagePathFromVMAssociationFile() error occured while loading image VM association from a file")
	}
	ImageAttributes, Bool := util.ImageVMAssociations[imageUUID]
	if Bool == true {
		return ImageAttributes.ImagePath, nil
	}
	return "", nil
}
