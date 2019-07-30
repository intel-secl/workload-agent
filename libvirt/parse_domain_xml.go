package libvirt

import (
	"encoding/xml"
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

// Domain is used to represent root of domain xml
type Domain struct {
	XMLName            xml.Name `xml:"domain"`
	UUID               string   `xml:"uuid"`
	Root               Root     `xml:"metadata>instance>root"`
	Disk               int      `xml:"metadata>instance>flavor>disk"`
	Source             Source   `xml:"devices>disk>source"`
	BackingStoreSource Source   `xml:"devices>disk>backingStore>source"`
}

// Root is used to represent root tag under metadata
type Root struct {
	XMLName xml.Name `xml:"root"`
	UUID    string   `xml:"uuid,attr"`
}

// Source is used to represent source tags under devices
type Source struct {
	XMLName xml.Name `xml:"source"`
	File    string   `xml:"file,attr"`
	Dev     string   `xml:"dev,attr"`
}

// DomainParser is used to set the XML content, qemu intercept call and all the values
// that will be parsed from Domain XMl
type DomainParser struct {
	xml               string
	qemuInterceptCall QemuIntercept
	vmUUID            string
	vmPath            string
	imageUUID         string
	imagePath         string
	size              int
}

// NewDomainParser method is used to get the DomainParser struct values
func NewDomainParser(domainXML string, qemuInterceptCall QemuIntercept) (*DomainParser, error) {
	var d DomainParser
	var domain Domain
	var err error
	d.xml = domainXML
	d.qemuInterceptCall = qemuInterceptCall

	err = xml.Unmarshal([]byte(domainXML), &domain)
	if err != nil {
		return nil, err
	}

	d.vmUUID = domain.UUID

	d.vmPath = domain.Source.File

	d.imageUUID = domain.Root.UUID

	if d.qemuInterceptCall == Start {
		d.imagePath = domain.BackingStoreSource.File
		if d.imagePath == "" {
			d.imagePath = domain.BackingStoreSource.Dev
		}

		d.size = domain.Disk
	}

	return &d, nil
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
