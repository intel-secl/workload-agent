package setup

/**
** @author srege
**/

import (
	"errors"
	log "github.com/sirupsen/logrus"
	csetup "intel/isecl/lib/common/setup"
	"intel/isecl/wlagent/common"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/consts"
	"intel/isecl/wlagent/mtwilsonclient"
	"os"
)

const beginCert string = "-----BEGIN CERTIFICATE-----"
const endCert string = "-----END CERTIFICATE-----"

type RegisterBindingKey struct {
}

func (rb RegisterBindingKey) Run(c csetup.Context) error {
	if rb.Validate(c) == nil {
		log.Info("Binding key already registered. Skipping this setup task.")
		return nil
	}
	// save configuration from config.yml
	e := config.SaveConfiguration(c)
	if e != nil {
		log.Error(e.Error())
		return e
	}
	log.Info("Registering binding key with host verification service.")
	keyFilePath := consts.ConfigDirPath + consts.BindingKeyFileName

	httpRequestBody, err := common.CreateRequest(keyFilePath)
	if err != nil {
		return errors.New("error registering binding key. " + err.Error())
	}

	mc, err := mtwilsonclient.InitializeClient()
	if err != nil {
		return errors.New("error initializing HVS client")
	}
	registerKey, err := mc.HostKey().CertifyHostBindingKey(httpRequestBody)
	if err != nil {
		return errors.New("error while updating the KBS user with envelope public key. " + err.Error())
	}
	aikPem := beginCert + "\n" + registerKey.BindingKeyCertificate + "\n" + endCert + "\n"

	keyCertFilePath := consts.ConfigDirPath + consts.BindingKeyPemFileName
	_ = common.WriteKeyCertToDisk(keyCertFilePath, aikPem)
	return nil
}

// Validate checks whether or not the register binding key task was completed successfully
func (rb RegisterBindingKey) Validate(c csetup.Context) error {
	log.Info("Validation for registering binding key.")
	bindingKeyCertFilePath := consts.ConfigDirPath + consts.BindingKeyPemFileName
	_, err := os.Stat(bindingKeyCertFilePath)
	if os.IsNotExist(err) {
		return errors.New("Binding key certificate file does not exist")
	}
	return nil
}
