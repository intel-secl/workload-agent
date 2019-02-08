package setup

/**
** @author srege
**/

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
	err := common.RegisterKey(tpm.Binding)
	if err != nil {
		return errors.New("error registering binding key. " + err.Error())
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
