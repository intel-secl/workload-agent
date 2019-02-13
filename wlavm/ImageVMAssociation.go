package wlavm

import (
	"intel/isecl/wlagent/util"
	"strings"

	log "github.com/sirupsen/logrus"
)

// ImageVMAssocociation structure is used to call create and delete with input parameters
type ImageVMAssocociation struct {
	ImageUUID string
	ImagePath string
}

// Create method is used to check if an entry exists with the image ID. If it does, increment the instance count,
// else create an entry with image instance association and append it.
func (IAssoc ImageVMAssocociation) Create() error {
	imageUUIDFound := false
	log.Debug("Loading yaml file to instance image association structure.")
	err := util.LoadImageVMAssociation()
	if err != nil {
		log.Error("Failed to unmarshal.")
		return err
	}
	for i, item := range util.ImageVMAssociations {
		if strings.Contains(item.ImageID, IAssoc.ImageUUID) {
			log.Debug("Image ID already exist in file, increasing the count of instance by 1.")
			item.VMCount = item.VMCount + 1
			util.ImageVMAssociations[i] = util.ImageVMAssociation{
				ImageID:   item.ImageID,
				ImagePath: item.ImagePath,
				VMCount:   item.VMCount,
			}
			imageUUIDFound = true
			break
		}
	}

	if !imageUUIDFound {
		log.Debug("Image ID does not exist in file, adding an entry with the image ID ", IAssoc.ImageUUID)
		data := util.ImageVMAssociation{
			ImageID:   IAssoc.ImageUUID,
			ImagePath: IAssoc.ImagePath,
			VMCount:   1,
		}
		util.ImageVMAssociations = append(util.ImageVMAssociations, data)
	}
	err = util.SaveImageVMAssociation()
	if err != nil {
		log.Error("Failed to marshal.")
		return err
	}
	return nil
}

// Delete method is used to check if an entry exists with the image ID. If it does, decrement the instance count.
// Check if the instance count is zero, then delete the image entry from the file.
func (IAssoc ImageVMAssocociation) Delete() (bool, string) {
	imagePath := ""
	isLastVM := false
	err := util.LoadImageVMAssociation()
	if err != nil {
		log.Error("Failed to unmarshal.", err)
	}
	for i, item := range util.ImageVMAssociations {
		imagePath = item.ImagePath
		if strings.Contains(item.ImageID, IAssoc.ImageUUID) {
			log.Debug("Image ID already exist in file, decreasing the count of instance by 1.")
			item.VMCount = item.VMCount - 1
			util.ImageVMAssociations[i] = util.ImageVMAssociation{
				ImageID:   item.ImageID,
				ImagePath: item.ImagePath,
				VMCount:   item.VMCount,
			}
			if item.VMCount == 0 {
				log.Debug("VM count is 0, hence deleting the entry with image id ", IAssoc.ImageUUID)
				util.ImageVMAssociations[i] = util.ImageVMAssociations[0]
				util.ImageVMAssociations = util.ImageVMAssociations[1:]
				isLastVM = true
				break
			}
		}
	}
	err = util.SaveImageVMAssociation()
	if err != nil {
		log.Error("Failed to marshal.", err)
	}
	return isLastVM, imagePath
}
