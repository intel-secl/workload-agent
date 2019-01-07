package setup

import (
	csetup "intel/isecl/lib/common/setup"
	"intel/isecl/wlagent/common"
)

const secretKeyLength int = 20

type BindingKey struct{}

func (bk BindingKey) Run(c csetup.Context) error {
	usage, err := common.NewCertifiedKey("BIND")
	if err != nil {
		return err
	}
	common.Run(usage)
	return nil
}

func (bk BindingKey) Validate(c csetup.Context) error {
	usage, err := common.NewCertifiedKey("BIND")
	if err != nil {
		return err
	}
	common.Validate(usage)
	return nil
}
