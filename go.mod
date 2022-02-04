module intel/isecl/wlagent/v4

require (
	github.com/containers/ocicrypt v1.1.2
	github.com/fsnotify/fsnotify v1.4.9
	github.com/intel-secl/intel-secl/v4 v4.1.1
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.6.1
	google.golang.org/grpc v1.33.2
	gopkg.in/yaml.v2 v2.3.0
	intel/isecl/lib/common/v4 v4.1.1
	intel/isecl/lib/platform-info/v4 v4.1.1
	intel/isecl/lib/tpmprovider/v4 v4.1.1
	intel/isecl/lib/verifier/v4 v4.1.1
	intel/isecl/lib/vml/v4 v4.1.1
)

replace (
	github.com/intel-secl/intel-secl/v4 => github.com/intel-innersource/applications.security.isecl.intel-secl/v4 v4.1.1/develop
	intel/isecl/lib/common/v4 => github.com/intel-innersource/libraries.security.isecl.common/v4  v4.1.1/develop
	intel/isecl/lib/flavor/v4 => github.com/intel-innersource/libraries.security.isecl.flavor/v4 v4.1.1/develop
	intel/isecl/lib/platform-info/v4 => github.com/intel-innersource/libraries.security.isecl.platform-info/v4  v4.1.1/develop
	intel/isecl/lib/tpmprovider/v4 => github.com/intel-innersource/libraries.security.isecl.tpm-provider/v4  v4.1.1/develop
	intel/isecl/lib/verifier/v4 => github.com/intel-innersource/libraries.security.isecl.verifier/v4  v4.1.1/develop
	intel/isecl/lib/vml/v4 => github.com/intel-innersource/libraries.security.isecl.volume-management/v4  v4.1.1/develop
)
