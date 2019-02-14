package libvirt

import (
	"fmt"
	"strconv"

	log "github.com/sirupsen/logrus"
	xmlpath "gopkg.in/xmlpath.v2"
)

// QemuIntercept is used to get the qemu intercept call
type QemuIntercept int

// const enumerates the Qemu intercept call. The values are Start, Stop or None
const (
	None QemuIntercept = iota
	// Start intercept call
	Start
	//Stop intercept call
	Stop
)

// DomainParser is used to set the XML node and the qemu intercept call
type DomainParser struct {
	XML               *xmlpath.Node
	QemuInterceptCall QemuIntercept
}

// Domain struct includes all the attributes that will be parsed from domain XML
type Domain struct {
	VMUUID    string
	VMPath    string
	ImageUUID string
	ImagePath string
	Size      int
}

// NewDomainParser is used to get the parsed values from domainXML
func NewDomainParser(domain *DomainParser) (*Domain, error) {
	log.Info("New domain parsed called")
	var parsedValues *Domain
	var vmUUID, vmPath, imageUUID, imagePath, diskSize string
	var size int
	var err error

	if vmUUID, err = getItemFromDomainXML(domain.XML, "/domain/uuid", "vmUUID"); err != nil {
		return parsedValues, err
	}

	if vmPath, err = getItemFromDomainXML(domain.XML, "/domain/devices/disk/source/@file", "vmPath"); err != nil {
		return parsedValues, err
	}

	if imageUUID, err = getItemFromDomainXML(domain.XML, "/domain/metadata//node()[@type='image']/@uuid", "imageUUID"); err != nil {
		return parsedValues, err
	}

	if domain.QemuInterceptCall == Start {
		if imagePath, err = getItemFromDomainXML(domain.XML, "/domain/devices/disk/backingStore/source/@file", "imagePath"); err != nil {
			if imagePath, err = getItemFromDomainXML(domain.XML, "/domain/devices/disk/backingStore/source/@dev", "imagePath"); err != nil {
				return parsedValues, err
			}
		}

		if diskSize, err = getItemFromDomainXML(domain.XML, "/domain/metadata//disk", "diskSize"); err != nil {
			return parsedValues, err
		}
		size, _ = strconv.Atoi(diskSize)
	}

	parsedValues = &Domain{
		VMUUID:    vmUUID,
		VMPath:    vmPath,
		ImageUUID: imageUUID,
		ImagePath: imagePath,
		Size:      size,
	}

	return parsedValues, nil
}

func getItemFromDomainXML(domainXML *xmlpath.Node, xmlPath string, item string) (string, error) {

	// parse the item in xml path from domainXMl
	parseItem := xmlpath.MustCompile(xmlPath)
	itemValue, ok := parseItem.String(domainXML)
	if !ok {
		return "", fmt.Errorf("Error while getting %s from domainXMl", item)
	}
	return itemValue, nil
}
