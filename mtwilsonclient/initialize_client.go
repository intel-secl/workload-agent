package mtwilsonclient

import (
	"encoding/hex"
	"errors"
	mtwilson "intel/isecl/lib/mtwilson-client"
	config "intel/isecl/wlagent/config"
)

func InitializeClient() (*mtwilson.Client, error) {
	var mc *mtwilson.Client
	var certificateDigest [32]byte
	certDigestBytes, err := hex.DecodeString(config.Configuration.Mtwilson.TLSSha256)

	if err != nil || len(certDigestBytes) != 32 {
		return mc, errors.New("error converting certificate digest to hex. " + err.Error())
	}
	copy(certificateDigest[:], certDigestBytes)
	mc = &mtwilson.Client{
		BaseURL:    config.Configuration.Mtwilson.APIURL,
		Username:   config.Configuration.Mtwilson.APIUsername,
		Password:   config.Configuration.Mtwilson.APIPassword,
		CertSha256: &certificateDigest,
	}
	return mc, err
}
