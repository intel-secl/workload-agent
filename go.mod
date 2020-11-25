module intel/isecl/wlagent/v3

require (
	github.com/Gurpartap/logrus-stack v0.0.0-20170710170904-89c00d8a28f4 // indirect
	github.com/facebookgo/stack v0.0.0-20160209184415-751773369052 // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/intel-secl/intel-secl/v3 v3.3.0
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.4.0
	gopkg.in/yaml.v2 v2.3.0
	intel/isecl/lib/clients/v3 v3.3.0
	intel/isecl/lib/common/v3 v3.3.0
	intel/isecl/lib/flavor/v3 v3.3.0
	intel/isecl/lib/platform-info/v3 v3.3.0
	intel/isecl/lib/tpmprovider/v3 v3.3.0
	intel/isecl/lib/verifier/v3 v3.3.0
	intel/isecl/lib/vml/v3 v3.3.0
)

replace intel/isecl/lib/tpmprovider/v3 => gitlab.devtools.intel.com/sst/isecl/lib/tpm-provider.git/v3 v3.3/develop

replace intel/isecl/lib/vml/v3 => gitlab.devtools.intel.com/sst/isecl/lib/volume-management.git/v3 v3.3/develop

replace intel/isecl/lib/common/v3 => gitlab.devtools.intel.com/sst/isecl/lib/common.git/v3 v3.3/develop

replace intel/isecl/lib/flavor/v3 => gitlab.devtools.intel.com/sst/isecl/lib/flavor.git/v3 v3.3/develop

replace intel/isecl/lib/verifier/v3 => gitlab.devtools.intel.com/sst/isecl/lib/verifier.git/v3 v3.3/develop

replace intel/isecl/lib/platform-info/v3 => gitlab.devtools.intel.com/sst/isecl/lib/platform-info.git/v3 v3.3/develop

replace intel/isecl/lib/clients/v3 => gitlab.devtools.intel.com/sst/isecl/lib/clients.git/v3 v3.3/develop

replace github.com/intel-secl/intel-secl/v3 => gitlab.devtools.intel.com/sst/isecl/intel-secl.git/v3 v3.3/develop

replace github.com/vmware/govmomi => github.com/arijit8972/govmomi fix-tpm-attestation-output
