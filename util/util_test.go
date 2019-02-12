package util

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

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
