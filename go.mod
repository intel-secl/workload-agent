module intel/isecl/wlagent/v4

require (
	github.com/fsnotify/fsnotify v1.4.9
	github.com/intel-secl/intel-secl/v4 v4.0.3
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.6.1
	gopkg.in/yaml.v2 v2.3.0
	intel/isecl/lib/common/v4 v4.0.3
	intel/isecl/lib/platform-info/v4 v4.0.3
	intel/isecl/lib/tpmprovider/v4 v4.0.3
	intel/isecl/lib/verifier/v4 v4.0.3
	intel/isecl/lib/vml/v4 v4.0.3
)

replace (
	intel/isecl/lib/common/v4 => github.com/intel-secl/common/v4 v4.0.3
	intel/isecl/lib/flavor/v4 => github.com/intel-secl/flavor/v4 v4.0.3
	intel/isecl/lib/platform-info/v4 => github.com/intel-secl/platform-info/v4 v4.0.3
	intel/isecl/lib/tpmprovider/v4 => github.com/intel-secl/tpm-provider/v4 v4.0.3
	intel/isecl/lib/verifier/v4 => github.com/intel-secl/verifier/v4 v4.0.3
	intel/isecl/lib/vml/v4 => github.com/intel-secl/volume-management-library/v4 v4.0.3
	github.com/intel-secl/intel-secl/v4 => github.com/intel-secl/intel-secl/v4 v4.0.3
)
