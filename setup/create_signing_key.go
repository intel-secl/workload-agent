package setup

import (
	csetup "intel/isecl/lib/common/setup"
	"intel/isecl/wlagent/common"
)

type SigningKey struct{}

func (sk SigningKey) Run(c csetup.Context) error {
	usage, err := common.NewCertifiedKey("SIGN")
	if err != nil {
		return err
	}
	common.Run(usage)
	return nil
}

func (sk SigningKey) Validate(c csetup.Context) error {
	usage, err := common.NewCertifiedKey("SIGN")
	if err != nil {
		return err
	}
	common.Validate(usage)
	return nil
}
