package libvirt

import (
	"io/ioutil"
	"testing"
	"github.com/stretchr/testify/assert"
	xmlpath "gopkg.in/xmlpath.v2"
	"bytes"
)

func TestGetItemFromDomainXML(t *testing.T) {
	domainXMLFile := "../test/domain.xml"
	var parser *DomainParser
	domainXMLFileContent, _ := ioutil.ReadFile(domainXMLFile)
	domainXML, err := xmlpath.Parse(bytes.NewReader(domainXMLFileContent))
	assert.NoError(t, err)

	parser = &DomainParser{
		XML : domainXML,      
		QemuInterceptCall : Start,
	}
	
	parsedValue, err := NewDomainParser(parser)
	assert.NoError(t, err)
	// get vm UUID from domain XML
	vmUUID := parsedValue.VMUUID
	assert.Equal(t, vmUUID, "412ea302-1759-440b-894a-bfef290d7a63")
	// get vm path from domain XML
	vmPath := parsedValue.VMPath
	assert.Equal(t, vmPath, "/var/lib/nova/instances/412ea302-1759-440b-894a-bfef290d7a63/disk")
	// get image UUID from domain XML
	imageUUID := parsedValue.ImageUUID
	assert.Equal(t, imageUUID, "31ab5921-24fd-498c-8c9e-b20f61004fc0")
	// get image path from domain XML
	imagePath := parsedValue.ImagePath
	assert.Equal(t, imagePath, "/var/lib/nova/instances/_base/dbee5739d526f9b742b8c7d4d829097965f4f718")
	// get disk size from domain XML
	size := parsedValue.Size

	assert.Equal(t, size, 1)
}
