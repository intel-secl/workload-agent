package wlavm

import (
	"fmt"
	"intel/isecl/wlagent/util"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

// ImageVMAssociation with ID and path
type ImageVMAssociation struct {
	ImageUUID string
	ImagePath string
}

// Create method is used to check if an entry exists with the image ID. If it does, increment the vm count,
// else create an entry with image vm association and append it.
func (IAssoc ImageVMAssocociation) Create() error {
	imageUUIDFound := false
	log.Debug("Loading yaml file to vm image association structure.")
	err := util.LoadImageVMAssociation()
	if err != nil {
		return fmt.Errorf("error occured while loading image VM association from a file. %s" + err.Error())
	}
	for i, item := range util.ImageVMAssociations {
		if strings.Contains(item.ImageID, IAssoc.ImageUUID) {
			log.Debug("Image ID already exist in file, increasing the count of vm by 1.")
			util.ImageVMAssociations[i].VMCount = item.VMCount + 1
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
		return fmt.Errorf("error occured while saving image VM association to a file. %s" + err.Error())
	}
	return nil
}

// Delete method is used to check if an entry exists with the image ID. If it does, decrement the vm count.
// Check if the vm count is zero, then delete the image entry from the file.
func (IAssoc ImageVMAssocociation) Delete() (bool, string, error) {
	imagePath := ""
	isLastVM := false

	fileMutex.Lock()
	defer fileMutex.Unlock()

	err := util.LoadImageVMAssociation()
	if err != nil {
		return isLastVM, imagePath, fmt.Errorf("error occured while loading image VM association from a file. %s" + err.Error())
	}
	for i, item := range util.ImageVMAssociations {
		imagePath = item.ImagePath
		if strings.Contains(item.ImageID, IAssoc.ImageUUID) {
			log.Debug("Image ID already exist in file, decreasing the count of vm by 1.")
			util.ImageVMAssociations[i].VMCount = item.VMCount - 1
			if util.ImageVMAssociations[i].VMCount == 0 {
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
		return isLastVM, imagePath, fmt.Errorf("error occured while saving image VM association to a file. %s" + err.Error())
	}
	return isLastVM, imagePath, nil
}
