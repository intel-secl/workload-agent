package setup

import (
	csetup "intel/isecl/lib/common/setup"
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/common"
)

type SigningKey struct {
	T tpm.Tpm
}

func (sk SigningKey) Run(c csetup.Context) error {
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
	usage, err := common.NewCertifiedKey("SIGN")
	if err != nil {
		return err
	}
	common.KeyValidation(usage)
	return nil
}
