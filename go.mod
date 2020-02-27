module intel/isecl/wlagent

require (
	github.com/Gurpartap/logrus-stack v0.0.0-20170710170904-89c00d8a28f4 // indirect
	github.com/facebookgo/stack v0.0.0-20160209184415-751773369052 // indirect
	github.com/fsnotify/fsnotify v1.4.7
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.4.0

	gopkg.in/yaml.v2 v2.2.2
	intel/isecl/lib/clients v0.0.0
	intel/isecl/lib/common v1.0.0-Beta
	intel/isecl/lib/flavor v0.0.0
	intel/isecl/lib/platform-info v0.0.0
	intel/isecl/lib/tpmprovider v0.0.0
	intel/isecl/lib/verifier v0.0.0
	intel/isecl/lib/vml v0.0.0
)

replace intel/isecl/lib/tpmprovider => gitlab.devtools.intel.com/sst/isecl/lib/tpm-provider.git v2.1/develop

replace intel/isecl/lib/vml => github.com/intel-secl/volume-management-library v2.0.0

replace intel/isecl/lib/common => github.com/intel-secl/common v2.0.0

replace intel/isecl/lib/flavor => github.com/intel-secl/flavor v2.0.0

replace intel/isecl/lib/verifier => github.com/intel-secl/verifier v2.0.0

replace intel/isecl/lib/platform-info => github.com/intel-secl/platform-info v2.0.0

replace intel/isecl/lib/clients => github.com/intel-secl/clients v2.0.0
