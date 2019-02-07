package wlavm

import (
	"strings"

	log "github.com/sirupsen/logrus"
)

// ImageVMAssociation structure is used to call create and delete with input parameters
type ImageVMAssociation struct {
	ImageUUID string
	ImagePath string
}

// Create method is used to check if an entry exists with the image ID. If it does, increment the instance count,
// else create an entry with image instance association and append it.
func (I ImageVMAssociation) Create() error {
	imageUUIDFound := false
	log.Debug("Loading yaml file to instance image association structure.")
	err := LoadImageInstanceAssociation()
	if err != nil {
		log.Error("Failed to unmarshal.")
		return err
	}
	for _, item := range ImageInstanceAssociations {
		if strings.Contains(item.ImageID, I.ImageUUID) {
			log.Debug("Image ID already exist in file, increasing the count of instance by 1.")
			item.InstanceCount = item.InstanceCount + 1
			imageUUIDFound = true
			break
		}
	}

	if !imageUUIDFound {
		log.Debug("Image ID does not exist in file, adding an entry with the image ID ", I.ImageUUID)
		data := ImageInstanceAssociation{
			ImageID:       I.ImageUUID,
			ImagePath:     I.ImagePath,
			InstanceCount: 1,
		}
		ImageInstanceAssociations = append(ImageInstanceAssociations, data)
	}
	err = SaveImageInstanceAssociation()
	if err != nil {
		log.Error("Failed to marshal.")
		return err
	}
	return nil
}

// Delete method is used to check if an entry exists with the image ID. If it does, decrement the instance count.
// Check if the instance count is zero, then delete the image entry from the file.
func (I ImageVMAssociation) Delete() (bool, string) {
	imagePath := ""
	isLastInstance := false
	err := LoadImageInstanceAssociation()
	if err != nil {
		log.Error("Failed to unmarshal.", err)
	}
	for i, item := range ImageInstanceAssociations {
		imagePath = item.ImagePath
		if strings.Contains(item.ImageID, I.ImageUUID) {
			log.Debug("Image ID already exist in file, decreasing the count of instance by 1.")
			item.InstanceCount = item.InstanceCount - 1
			if item.InstanceCount == 0 {
				log.Debug("Instance count is 0, hence deleting the entry with image id ", I.ImageUUID)
				ImageInstanceAssociations[i] = ImageInstanceAssociations[0]
				ImageInstanceAssociations = ImageInstanceAssociations[1:]
				isLastInstance = true
				break
			}
		}
	}
	err = SaveImageInstanceAssociation()
	if err != nil {
		log.Error("Failed to marshal.", err)
	}
	return isLastInstance, imagePath
}
