package setup

import (
	csetup "intel/isecl/lib/common/setup"
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/common"

	log "github.com/sirupsen/logrus"
)

type SigningKey struct {
	T tpm.Tpm
}

func (sk SigningKey) Run(c csetup.Context) error {
	if sk.Validate(c) == nil {
		log.Info("Signing key already created, skipping ...")
		return nil
	}
	log.Info("Creating signing key.")

	err := common.GenerateKey(tpm.Signing, sk.T)
	if err != nil {
		return err
	}
	return nil
}

func (sk SigningKey) Validate(c csetup.Context) error {
	log.Info("Validation for signing key.")

	err := common.ValidateKey(tpm.Signing)
	if err != nil {
		return err
	}

	return nil
}
