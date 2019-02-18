package setup

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	csetup "intel/isecl/lib/common/setup"
	"intel/isecl/wlagent/common"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/consts"
	"intel/isecl/wlagent/mtwilsonclient"
	"os"
)

type RegisterSigningKey struct {
}

func (rs RegisterSigningKey) Run(c csetup.Context) error {

	if config.Configuration.ConfigComplete == false {
		return fmt.Errorf("configuration is not complete - setup tasks can be completed only after configuration")
	}
	
	if rs.Validate(c) == nil {
		log.Info("Signing key already registered. Skipping this setup task.")
		return nil
	}

	log.Info("Registering signing key with host verification service.")
	signingKey, err := config.GetSigningKeyFromFile()
	if err != nil {
		return errors.New("error reading signing key from  file. " + err.Error())
	}

	httpRequestBody, err := common.CreateRequest(signingKey)
	if err != nil {
		return errors.New("error registering signing key. " + err.Error())
	}

	mc, err := mtwilsonclient.InitializeClient()
	if err != nil {
		return errors.New("error initializing HVS client")
	}
	registerKey, err := mc.HostKey().CertifyHostSigningKey(httpRequestBody)
	if err != nil {
		return errors.New("error while updating the KBS user with envelope public key. " + err.Error())
	}

	err = common.WriteKeyCertToDisk(consts.ConfigDirPath+consts.SigningKeyPemFileName, registerKey.SigningKeyCertificate)
	if err != nil {
		return errors.New("error writing signing key certificate to file.")
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
