module intel/isecl/wlagent/v3

require (
	github.com/Gurpartap/logrus-stack v0.0.0-20170710170904-89c00d8a28f4 // indirect
	github.com/facebookgo/stack v0.0.0-20160209184415-751773369052 // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/intel-secl/intel-secl/v3 v3.0.0
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.4.0
	gopkg.in/yaml.v2 v2.3.0
	intel/isecl/lib/clients/v3 v3.0.0
	intel/isecl/lib/common/v3 v3.0.0
	intel/isecl/lib/flavor/v3 v3.0.0
	intel/isecl/lib/platform-info/v3 v3.0.0
	intel/isecl/lib/tpmprovider/v3 v3.0.0
	intel/isecl/lib/verifier/v3 v3.0.0
	intel/isecl/lib/vml/v3 v3.0.0
)

replace intel/isecl/lib/tpmprovider/v3 => github.com/intel-secl/tpm-provider/v3 v3.0.0

replace intel/isecl/lib/vml/v3 => github.com/intel-secl/volume-management-library/v3 v3.0.0

replace intel/isecl/lib/common/v3 => github.com/intel-secl/common/v3 v3.0.0

replace intel/isecl/lib/flavor/v3 => github.com/intel-secl/flavor/v3 v3.0.0

replace intel/isecl/lib/verifier/v3 => github.com/intel-secl/verifier/v3 v3.0.0

replace intel/isecl/lib/platform-info/v3 => github.com/intel-secl/platform-info/v3 v3.0.0

replace intel/isecl/lib/clients/v3 => github.com/intel-secl/clients/v3 v3.0.0

replace github.com/intel-secl/intel-secl/v3 => gitlab.devtools.intel.com/sst/isecl/intel-secl.git/v3 v3.1/develop

replace github.com/vmware/govmomi => github.com/arijit8972/govmomi fix-tpm-attestation-output
