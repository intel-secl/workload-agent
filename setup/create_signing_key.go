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
	log.Info("Creating of signing key.")
	usage, err := common.NewCertifiedKey("SIGN")
	if err != nil {
		return err
	}

	err = common.KeyGeneration(usage, sk.T)
	if err != nil {
		return err
	}
	return nil
}

func (sk SigningKey) Validate(c csetup.Context) error {
	log.Info("Validation for signing key.")
	usage, err := common.NewCertifiedKey("SIGN")
	if err != nil {
		return err
	}

	err = common.KeyValidation(usage)
	if err != nil {
		return err
	}

	return nil
}
