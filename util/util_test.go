package util

import (
	"testing"
	"github.com/stretchr/testify/assert"
	xmlpath "gopkg.in/xmlpath.v2"
	"io/ioutil"
	"bytes"
)
func TestGetItemFromDomainXML(t *testing.T) {
	domainXMLFile := "../test/domain.xml"

	domainXMLFileContent, _ := ioutil.ReadFile(domainXMLFile)
	domainXML, err := xmlpath.Parse(bytes.NewReader(domainXMLFileContent))
	assert.NoError(t, err)

	// get instance UUID from domain XML
	instanceUUID, err := GetInstanceUUID(domainXML)
	assert.NoError(t, err)
	assert.Equal(t, instanceUUID, "412ea302-1759-440b-894a-bfef290d7a63")
	
	// get instance path from domain XML
	instancePath, err := GetInstancePath(domainXML)
	assert.NoError(t, err)
	assert.Equal(t, instancePath, "/var/lib/nova/instances/412ea302-1759-440b-894a-bfef290d7a63/disk") 
	
	// get image UUID from domain XML
	imageUUID, err := GetImageUUID(domainXML)
	assert.NoError(t, err)
	assert.Equal(t, imageUUID, "31ab5921-24fd-498c-8c9e-b20f61004fc0")

	// get image path from domain XML
	imagePath, err := GetImagePath(domainXML)
	assert.NoError(t, err)
	assert.Equal(t, imagePath, "/var/lib/nova/instances/_base/dbee5739d526f9b742b8c7d4d829097965f4f718")

	// get disk size from domain XML
	diskSize, err := GetDiskSize(domainXML)
	assert.NoError(t, err)
	assert.Equal(t, diskSize, "1")

}

func TestIsImageEncryptedTrue(t *testing.T) {
	encImagePath := "../test/cirros-x86.qcow2_enc"
	isImageEncrypted, err := IsImageEncrypted(encImagePath)
	assert.NoError(t, err)
	assert.True(t, isImageEncrypted)
}

func TestIsImageEncryptedFalse(t *testing.T) {
	encImagePath := "../test/cirros-x86.qcow2"
	isImageEncrypted, err := IsImageEncrypted(encImagePath)
	assert.NoError(t, err)
	assert.False(t, isImageEncrypted)
}
