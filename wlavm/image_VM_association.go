/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package wlavm

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"intel/isecl/wlagent/util"
	"sync"
)

// ImageVMAssociation with ID and path
type ImageVMAssociation struct {
	ImageUUID string
	ImagePath string
}

var fileMutex sync.Mutex

// Create method is used to check if an entry exists with the image ID. If it does, increment the instance count,
// else create an entry with image instance association and append it.
func (IAssoc ImageVMAssociation) Create() error {
	log.Debug("Loading yaml file to vm image association structure.")
	fileMutex.Lock()
	defer fileMutex.Unlock()

	err := util.LoadImageVMAssociation()
	if err != nil {
		return fmt.Errorf("error occured while loading image VM association from a file. %s" + err.Error())
	}
	ImageAttributes, Bool := util.ImageVMAssociations[IAssoc.ImageUUID]
	if Bool == true {
		log.Debug("Image ID already exist in file, increasing the count of vm by 1.")
		ImageAttributes.VMCount = ImageAttributes.VMCount + 1
	} else {
		log.Debug("Image ID does not exist in file, adding an entry with the image ID ", IAssoc.ImageUUID)
		util.ImageVMAssociations[IAssoc.ImageUUID] = &util.ImageVMAssociation{IAssoc.ImagePath, 1}
	}
	err = util.SaveImageVMAssociation()
	if err != nil {
		return fmt.Errorf("error occured while saving image VM association to a file. %s" + err.Error())
	}
	return nil
}

// Delete method is used to check if an entry exists with the image ID. If it does, decrement the vm count.
// Check if the vm count is zero, then delete the image entry from the file.
func (IAssoc ImageVMAssociation) Delete() (bool, string, error) {
	imagePath := ""
	isLastVM := false

	fileMutex.Lock()
	defer fileMutex.Unlock()

	err := util.LoadImageVMAssociation()
	if err != nil {
		return isLastVM, imagePath, fmt.Errorf("error occured while loading image VM association from a file. %s" + err.Error())
	}
	ImageAttributes, Bool := util.ImageVMAssociations[IAssoc.ImageUUID]
	imagePath = ImageAttributes.ImagePath
	if Bool == true && ImageAttributes.VMCount > 0 {
		log.Debug("Image ID already exist in file, decreasing the count of vm by 1.")
		ImageAttributes.VMCount = ImageAttributes.VMCount - 1
		if ImageAttributes.VMCount == 0 {
			isLastVM = true
		}
	}

	err = util.SaveImageVMAssociation()
	if err != nil {
		return isLastVM, imagePath, fmt.Errorf("error occured while saving image VM association to a file. %s" + err.Error())
	}
	return isLastVM, imagePath, nil
}

func imagePathFromVMAssociationFile(imageUUID string) (string, error) {
	log.Debug("Checking if the image UUID exists in image-vm association file")
	log.Debug("Loading yaml file to vm image association structure.")
	err := util.LoadImageVMAssociation()
	if err != nil {
		return "", fmt.Errorf("error occured while loading image VM association from a file. %s" + err.Error())
	}
	ImageAttributes, Bool := util.ImageVMAssociations[imageUUID]
	if Bool == true {
		return ImageAttributes.ImagePath, nil
	}
	return "", nil
}
