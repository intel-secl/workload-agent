package setup

import (
	"fmt"
	csetup "intel/isecl/lib/common/setup"
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/common"
	"log"
)

type SigningKey struct {
	T tpm.Tpm
}

func (sk SigningKey) Run(c csetup.Context) error {
	if sk.Validate(c) == nil {
		fmt.Println("Signing key already created, skipping ...")
		return nil
	}
	log.Println("Creating of signing key.")
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
	log.Println("Validation for signing key.")
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
