module intel/isecl/wlagent/v4

require (
	github.com/fsnotify/fsnotify v1.4.9
	github.com/intel-secl/intel-secl/v4 v4.0.1
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.6.1
	gopkg.in/yaml.v2 v2.3.0
	intel/isecl/lib/common/v4 v4.0.1
	intel/isecl/lib/platform-info/v4 v4.0.1
	intel/isecl/lib/tpmprovider/v4 v4.0.1
	intel/isecl/lib/verifier/v4 v4.0.1
	intel/isecl/lib/vml/v4 v4.0.1
)

replace (
	intel/isecl/lib/common/v4 => gitlab.devtools.intel.com/sst/isecl/lib/common.git/v4 v4.0.1/develop
	intel/isecl/lib/flavor/v4 => gitlab.devtools.intel.com/sst/isecl/lib/flavor.git/v4 v4.0.1/develop
	intel/isecl/lib/platform-info/v4 => gitlab.devtools.intel.com/sst/isecl/lib/platform-info.git/v4 v4.0.1/develop
	intel/isecl/lib/tpmprovider/v4 => gitlab.devtools.intel.com/sst/isecl/lib/tpm-provider.git/v4 v4.0.1/develop
	intel/isecl/lib/verifier/v4 =>  gitlab.devtools.intel.com/sst/isecl/lib/verifier.git/v4 v4.0.1/develop
	intel/isecl/lib/vml/v4 => gitlab.devtools.intel.com/sst/isecl/lib/volume-management.git/v4 v4.0.1/develop
	github.com/intel-secl/intel-secl/v4 => gitlab.devtools.intel.com/sst/isecl/intel-secl.git/v4 v4.0.1/develop
)
