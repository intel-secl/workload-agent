module intel/isecl/wlagent

require (
	github.com/Gurpartap/logrus-stack v0.0.0-20170710170904-89c00d8a28f4 // indirect
	github.com/facebookgo/stack v0.0.0-20160209184415-751773369052 // indirect
	github.com/fsnotify/fsnotify v1.4.7
	github.com/sirupsen/logrus v1.4.0
	github.com/stretchr/testify v1.3.0
	golang.org/x/net v0.0.0-20190206173232-65e2d4e15006 // indirect

	gopkg.in/yaml.v2 v2.2.2
	intel/isecl/lib/common v0.0.0
	intel/isecl/lib/flavor v0.0.0
	intel/isecl/lib/mtwilson-client v0.0.0
	intel/isecl/lib/platform-info v0.0.0
	intel/isecl/lib/tpm v0.0.0
	intel/isecl/lib/verifier v0.0.0
	intel/isecl/lib/vml v0.0.0
)

replace intel/isecl/lib/tpm => gitlab.devtools.intel.com/sst/isecl/lib/tpm.git v0.0.0-20190202165337-322040ceed08

replace intel/isecl/lib/vml => gitlab.devtools.intel.com/sst/isecl/lib/volume-management.git v0.0.0-20190318085416-be922c5e335f

replace intel/isecl/lib/common => gitlab.devtools.intel.com/sst/isecl/lib/common.git v0.0.0-20190723202540-c602a6f38a48

replace intel/isecl/lib/flavor => gitlab.devtools.intel.com/sst/isecl/lib/flavor.git v0.0.0-20190724130420-55de0ba53dc1

replace intel/isecl/lib/verifier => gitlab.devtools.intel.com/sst/isecl/lib/verifier.git v0.0.0-20190724131213-64d659e7973d

replace intel/isecl/lib/platform-info => gitlab.devtools.intel.com/sst/isecl/lib/platform-info.git v0.0.0-20181206180455-b2908f06aa05

replace intel/isecl/lib/mtwilson-client => gitlab.devtools.intel.com/sst/isecl/lib/mtwilson-client.git v0.0.0-20190213202719-11876bdbab7c
