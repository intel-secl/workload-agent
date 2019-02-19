package libvirt

import (
	"fmt"
	"strconv"

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

// DomainParser is used to set the XML node, qemu intercept call and all the values
// that will be parsed from Domain XMl
type DomainParser struct {
	xml               *xmlpath.Node
	qemuInterceptCall QemuIntercept
	vmUUID            string
	vmPath            string
	imageUUID         string
	imagePath         string
	size              int
}

// NewDomainParser method is used to get the DOmainParser struct values
func NewDomainParser(domainXML *xmlpath.Node, qemuInterceptCall QemuIntercept) (*DomainParser, error) {
	var d DomainParser
	var err error
	d.xml = domainXML
	d.qemuInterceptCall = qemuInterceptCall

	if d.vmUUID, err = d.getItemFromDomainXML("/domain/uuid", "vmUUID"); err != nil {
		return nil, err
	}

	if d.vmPath, err = d.getItemFromDomainXML("/domain/devices/disk/source/@file", "vmPath"); err != nil {
		return nil, err
	}

	if d.imageUUID, err = d.getItemFromDomainXML("/domain/metadata//node()[@type='image']/@uuid", "imageUUID"); err != nil {
		return nil, err
	}

	if d.qemuInterceptCall == Start {
		if d.imagePath, err = d.getItemFromDomainXML("/domain/devices/disk/backingStore/source/@file", "imagePath"); err != nil {
			if d.imagePath, err = d.getItemFromDomainXML("/domain/devices/disk/backingStore/source/@dev", "imagePath"); err != nil {
				return nil, err
			}
		}

		var diskSize string
		if diskSize, err = d.getItemFromDomainXML("/domain/metadata//disk", "diskSize"); err != nil {
			return nil, err
		}
		d.size, _ = strconv.Atoi(diskSize)
	}

	return &d, nil
}

// getItemFromDomainXML method is used to get an item from domain XML given the xPath value
func (d *DomainParser) getItemFromDomainXML(xmlPath string, item string) (string, error) {

	// parse the item in xml path from domainXMl
	parseItem := xmlpath.MustCompile(xmlPath)
	itemValue, ok := parseItem.String(d.xml)
	if !ok {
		return "", fmt.Errorf("Error while getting %s from domainXMl", item)
	}
	return itemValue, nil
}

// GetVMUUID method is used to get the vm UUID value from the domain XML
func (d *DomainParser) GetVMUUID() string {
	return d.vmUUID
}

// GetVMPath method is used to get the vm path value from the domain XML
func (d *DomainParser) GetVMPath() string {
	return d.vmPath
}

// GetImageUUID method is used to get the image UUID value from the domain XML
func (d *DomainParser) GetImageUUID() string {
	return d.imageUUID
}

// GetImagePath method is used to get the image path value from the domain XML
func (d *DomainParser) GetImagePath() string {
	return d.imagePath
}

// GetDiskSize method is used to get the disk size value from the domain XML
func (d *DomainParser) GetDiskSize() int {
	return d.size
}
