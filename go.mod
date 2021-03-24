module intel/isecl/wlagent/v3

require (
	github.com/fsnotify/fsnotify v1.4.9
	github.com/intel-secl/intel-secl/v3 v3.6.0
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.6.1
	gopkg.in/yaml.v2 v2.3.0
	intel/isecl/lib/common/v3 v3.6.0
	intel/isecl/lib/platform-info/v3 v3.6.0
	intel/isecl/lib/tpmprovider/v3 v3.6.0
	intel/isecl/lib/verifier/v3 v3.6.0
	intel/isecl/lib/vml/v3 v3.6.0
)

replace (
	github.com/intel-secl/intel-secl/v3 => github.com/intel-secl/intel-secl/v3 v3.6.0
	github.com/vmware/govmomi => github.com/arijit8972/govmomi v0.22.2-0.20200607061538-3311e9e4cdb1
	intel/isecl/lib/common/v3 => github.com/intel-secl/common/v3 v3.6.0
	intel/isecl/lib/flavor/v3 => github.com/intel-secl/flavor/v3 v3.6.0
	intel/isecl/lib/platform-info/v3 => github.com/intel-secl/platform-info/v3 v3.6.0
	intel/isecl/lib/tpmprovider/v3 => github.com/intel-secl/tpm-provider/v3 v3.6.0
	intel/isecl/lib/verifier/v3 => github.com/intel-secl/verifier/v3 v3.6.0
	intel/isecl/lib/vml/v3 => github.com/intel-secl/volume-management-library/v3 v3.6.0
)
