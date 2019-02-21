package setup

import (
	"fmt"
	csetup "intel/isecl/lib/common/setup"
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/common"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/consts"
	"os"
	"os/user"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type BindingKey struct {
	T tpm.Tpm
}

func (bk BindingKey) Run(c csetup.Context) error {
	if config.Configuration.ConfigComplete == false {
		return fmt.Errorf("configuration is not complete - setup tasks can be completed only after configuration")
	}
	if bk.Validate(c) == nil {
		log.Info("Binding key already created, skipping ...")
		return nil
	}
	log.Info("Creating binding key.")

	err := common.GenerateKey(tpm.Binding, bk.T)
	if err != nil {
		return err
	}

	return bk.setBindingKeyFileOwner()
}

func (bk BindingKey) Validate(c csetup.Context) error {
	log.Info("Validation for binding key.")

	err := common.ValidateKey(tpm.Binding)
	if err != nil {
		return err
	}

	return nil
}

// setBindingKeyFileOwner sets the owner of the binding key file to the trustagent user
// This is necessary for the TrustAgent to add the binding key to the manifest.
func (bk BindingKey) setBindingKeyFileOwner() (err error) {

	var usr *user.User
	err = nil
	// get the user id from the configuration variable that we have set
	if config.Configuration.TrustAgent.User == "" {
		return fmt.Errorf("trust agent user name cannot be empty in configuration")
	}

	if usr, err = user.Lookup(config.Configuration.TrustAgent.User); err != nil {
		return fmt.Errorf("could not lookup up user id of trust agent user : %s", config.Configuration.TrustAgent.User)
	}

	uid, _ := strconv.Atoi(usr.Uid)
	gid, _ := strconv.Atoi(usr.Gid)
	// no need to check errors for the above two call since had just looked up the user
	// using the user.Lookup call
	err = os.Chown(consts.ConfigDirPath+consts.BindingKeyFileName, uid, gid)

	return err
}
