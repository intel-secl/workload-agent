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
	log.Info("Creating of binding key.")
	usage, err := common.NewCertifiedKey("BIND")
	if err != nil {
		return err
	}

	err = common.KeyGeneration(usage, bk.T)
	if err != nil {
		return err
	}
	return nil
}

func (bk BindingKey) Validate(c csetup.Context) error {
	log.Info("Validation for binding key.")
	usage, err := common.NewCertifiedKey("BIND")
	if err != nil {
		return err
	}

	err = common.KeyValidation(usage)
	if err != nil {
		return err
	}

	return nil
}
