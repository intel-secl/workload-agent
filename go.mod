module intel/isecl/wlagent/v2

require (
	github.com/Gurpartap/logrus-stack v0.0.0-20170710170904-89c00d8a28f4 // indirect
	github.com/facebookgo/stack v0.0.0-20160209184415-751773369052 // indirect
	github.com/fsnotify/fsnotify v1.4.7
	github.com/intel-secl/intel-secl/v3 v3.0.0
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.4.0
	gopkg.in/yaml.v2 v2.2.2
	intel/isecl/lib/clients/v2 v2.1.0
	intel/isecl/lib/common/v2 v2.1.0
	intel/isecl/lib/flavor/v2 v2.1.0
	intel/isecl/lib/platform-info/v2 v2.1.0
	intel/isecl/lib/tpmprovider/v2 v2.1.0
	intel/isecl/lib/verifier/v2 v2.1.0
	intel/isecl/lib/vml/v2 v2.1.0

)

replace intel/isecl/lib/tpmprovider/v2 => github.com/intel-secl/tpm-provider/v2 v2.1.0

replace intel/isecl/lib/vml/v2 => github.com/intel-secl/volume-management-library/v2 v2.1.0

replace intel/isecl/lib/common/v2 => github.com/intel-secl/common/v2 v2.1.0

replace intel/isecl/lib/flavor/v2 => github.com/intel-secl/flavor/v2 v2.1.0

replace intel/isecl/lib/verifier/v2 => github.com/intel-secl/verifier/v2 v2.1.0

replace intel/isecl/lib/platform-info/v2 => github.com/intel-secl/platform-info/v2 v2.1.0

replace intel/isecl/lib/clients/v2 => github.com/intel-secl/clients/v2 v2.1.0

replace github.com/intel-secl/intel-secl/v3 => gitlab.devtools.intel.com/sst/isecl/intel-secl.git/v3 v3.0/feature/go-hvs
