package wlavm

import (
	"strings"

	log "github.com/sirupsen/logrus"
)

// ImageVMAssoc structure is used to call create and delete with input parameters
type ImageVMAssoc struct {
	ImageUUID string
	ImagePath string
}

// Create method is used to check if an entry exists with the image ID. If it does, increment the instance count,
// else create an entry with image instance association and append it.
func (IAssoc ImageVMAssoc) Create() error {
	imageUUIDFound := false
	log.Debug("Loading yaml file to instance image association structure.")
	err := LoadImageVMAssociation()
	if err != nil {
		log.Error("Failed to unmarshal.")
		return err
	}
	for _, item := range ImageVMAssociations {
		if strings.Contains(item.ImageID, IAssoc.ImageUUID) {
			log.Debug("Image ID already exist in file, increasing the count of instance by 1.")
			item.VMCount = item.VMCount + 1
			imageUUIDFound = true
			break
		}
	}

	if !imageUUIDFound {
		log.Debug("Image ID does not exist in file, adding an entry with the image ID ", IAssoc.ImageUUID)
		data := ImageVMAssociation{
			ImageID:   IAssoc.ImageUUID,
			ImagePath: IAssoc.ImagePath,
			VMCount:   1,
		}
		ImageVMAssociations = append(ImageVMAssociations, data)
	}
	err = SaveImageVMAssociation()
	if err != nil {
		log.Error("Failed to marshal.")
		return err
	}
	return nil
}

// Delete method is used to check if an entry exists with the image ID. If it does, decrement the instance count.
// Check if the instance count is zero, then delete the image entry from the file.
func (IAssoc ImageVMAssoc) Delete() (bool, string) {
	imagePath := ""
	isLastVM := false
	err := LoadImageVMAssociation()
	if err != nil {
		log.Error("Failed to unmarshal.", err)
	}
	for i, item := range ImageVMAssociations {
		imagePath = item.ImagePath
		if strings.Contains(item.ImageID, IAssoc.ImageUUID) {
			log.Debug("Image ID already exist in file, decreasing the count of instance by 1.")
			item.VMCount = item.VMCount - 1
			if item.VMCount == 0 {
				log.Debug("VM count is 0, hence deleting the entry with image id ", IAssoc.ImageUUID)
				ImageVMAssociations[i] = ImageVMAssociations[0]
				ImageVMAssociations = ImageVMAssociations[1:]
				isLastVM = true
				break
			}
		}
	}
	err = SaveImageVMAssociation()
	if err != nil {
		log.Error("Failed to marshal.", err)
	}
	return isLastVM, imagePath
}
