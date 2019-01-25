package setup

import (
	csetup "intel/isecl/lib/common/setup"
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/common"

	log "github.com/sirupsen/logrus"
)

type BindingKey struct {
	T tpm.Tpm
}

func (bk BindingKey) Run(c csetup.Context) error {
	if bk.Validate(c) == nil {
		log.Info("Binding key already created, skipping ...")
		return nil
	}
	log.Info("Creating binding key.")

	err := common.GenerateKey(tpm.Binding, bk.T)
	if err != nil {
		return err
	}
	return nil
}

func (bk BindingKey) Validate(c csetup.Context) error {
	log.Info("Validation for binding key.")

	err := common.ValidateKey(tpm.Binding)
	if err != nil {
		return err
	}

	return nil
}
