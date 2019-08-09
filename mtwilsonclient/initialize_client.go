package mtwilsonclient

import (
	"encoding/hex"
	"errors"
	mtwilson "intel/isecl/lib/mtwilson-client"
	config "intel/isecl/wlagent/config"
)

func InitializeClient() (*mtwilson.Client, error) {
	var mc *mtwilson.Client
	var certificateDigest [48]byte
	certDigestBytes, err := hex.DecodeString(config.Configuration.Mtwilson.TLSSha384)

	if err != nil || len(certDigestBytes) != 48 {
		return mc, errors.New("error converting certificate digest to hex. " + err.Error())
	}
	copy(certificateDigest[:], certDigestBytes)
	mc = &mtwilson.Client{
		BaseURL:    config.Configuration.Mtwilson.APIURL,
		Username:   config.Configuration.Mtwilson.APIUsername,
		Password:   config.Configuration.Mtwilson.APIPassword,
		CertSha384: &certificateDigest,
	}
	return mc, err
}
