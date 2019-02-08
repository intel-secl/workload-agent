package setup

import (
	"errors"
	log "github.com/sirupsen/logrus"
	csetup "intel/isecl/lib/common/setup"
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/common"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/consts"
	"os"
)

type RegisterSigningKey struct {
}

func (rs RegisterSigningKey) Run(c csetup.Context) error {
	if rs.Validate(c) == nil {
		log.Info("Signing key already registered. Skipping this setup task.")
		return nil
	}
	// save configuration from config.yml
	e := config.SaveConfiguration(c)
	if e != nil {
		log.Error(e.Error())
		return e
	}
	log.Info("Registering signing key with host verification service.")
	err := common.RegisterKey(tpm.Signing)
	if err != nil {
		return err
	}

	return nil
}

// Validate checks whether or not the Register Signing Key task was completed successfully
func (rs RegisterSigningKey) Validate(c csetup.Context) error {
	log.Info("Validation for registering signing key.")
	signingKeyCertPath := consts.ConfigDirPath + consts.SigningKeyPemFileName
	_, err := os.Stat(signingKeyCertPath)
	if os.IsNotExist(err) {
		return errors.New("Signing key certificate file does not exist")
	}
	return nil
}
