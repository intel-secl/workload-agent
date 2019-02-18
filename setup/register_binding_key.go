package setup

/**
** @author srege
**/

import (
	"errors"
	"fmt"
	csetup "intel/isecl/lib/common/setup"
	"intel/isecl/wlagent/common"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/consts"
	"intel/isecl/wlagent/mtwilsonclient"
	"os"

	log "github.com/sirupsen/logrus"
)

type RegisterBindingKey struct {
}

func (rb RegisterBindingKey) Run(c csetup.Context) error {

	if config.Configuration.ConfigComplete == false {
		return fmt.Errorf("configuration is not complete - setup tasks can be completed only after configuration")
	}

	if rb.Validate(c) == nil {
		log.Info("Binding key already registered. Skipping this setup task.")
		return nil
	}

	log.Info("Registering binding key with host verification service.")
	bindingKey, err := config.GetBindingKeyFromFile()
	if err != nil {
		return errors.New("error reading binding key from  file. " + err.Error())
	}

	httpRequestBody, err := common.CreateRequest(bindingKey)
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

	err = common.WriteKeyCertToDisk(consts.ConfigDirPath+consts.BindingKeyPemFileName, registerKey.BindingKeyCertificate)
	if err != nil {
		return errors.New("error writing binding key certificate to file.")
	}
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
