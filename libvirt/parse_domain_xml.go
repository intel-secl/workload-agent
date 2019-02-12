package libvirt

import (
	"fmt"

	xmlpath "gopkg.in/xmlpath.v2"
)

func getItemFromDomainXML(domainXML *xmlpath.Node, xmlPath string, item string) (string, error) {

	// parse the item in xml path from domainXMl
	parseItem := xmlpath.MustCompile(xmlPath)
	itemValue, ok := parseItem.String(domainXML)
	if !ok {
		return "", fmt.Errorf("Error while getting %s from domainXMl", item)
	}
	return itemValue, nil
}

// GetVmUUID method is used to get the vm UUID value from the domain XML
func GetVMUUID(domainXML *xmlpath.Node) (string, error) {
	return getItemFromDomainXML(domainXML, "/domain/uuid", "vmUUID")
}

// GetVmPath method is used to get the vm path value from the domain XML
func GetVMPath(domainXML *xmlpath.Node) (string, error) {
	return getItemFromDomainXML(domainXML, "/domain/devices/disk/source/@file", "vmPath")
}

// GetImageUUID method is used to get the image UUID value from the domain XML
func GetImageUUID(domainXML *xmlpath.Node) (string, error) {
	return getItemFromDomainXML(domainXML, "/domain/metadata//node()[@type='image']/@uuid", "imageUUID")
}

// GetImagePath method is used to get the image path value from the domain XML
func GetImagePath(domainXML *xmlpath.Node) (string, error) {
	imagePath, err := getItemFromDomainXML(domainXML, "/domain/devices/disk/backingStore/source/@file", "imagePath")
	if err != nil {
		return getItemFromDomainXML(domainXML, "/domain/devices/disk/backingStore/source/@dev", "imagePath")
	}
	return imagePath, nil
}

// GetDiskSize method is used to get the disk size value from the domain XML
func GetDiskSize(domainXML *xmlpath.Node) (string, error) {
	return getItemFromDomainXML(domainXML, "/domain/metadata//disk", "diskSize")
}
