module intel/isecl/wlagent

require (
	github.com/Gurpartap/logrus-stack v0.0.0-20170710170904-89c00d8a28f4 // indirect
	github.com/facebookgo/stack v0.0.0-20160209184415-751773369052 // indirect
	github.com/fsnotify/fsnotify v1.4.7
	github.com/sirupsen/logrus v1.4.0
	github.com/stretchr/testify v1.3.0
	golang.org/x/net v0.0.0-20190206173232-65e2d4e15006 // indirect

	gopkg.in/yaml.v2 v2.2.2
	intel/isecl/lib/clients v0.0.0
	intel/isecl/lib/common v1.0.0-Beta
	intel/isecl/lib/flavor v0.0.0
	intel/isecl/lib/mtwilson-client v0.0.0
	intel/isecl/lib/platform-info v0.0.0
	intel/isecl/lib/tpm v0.0.0
	intel/isecl/lib/verifier v0.0.0
	intel/isecl/lib/vml v0.0.0
)

replace intel/isecl/lib/tpm => gitlab.devtools.intel.com/sst/isecl/lib/tpm.git v0.0.0-20190917062532-b9c85dd61886

replace intel/isecl/lib/vml => gitlab.devtools.intel.com/sst/isecl/lib/volume-management.git v0.0.0-20190915022206-560299d2b8e9

replace intel/isecl/lib/common => gitlab.devtools.intel.com/sst/isecl/lib/common.git v0.0.0-20190916071549-04625ad42e3a

replace intel/isecl/lib/flavor => gitlab.devtools.intel.com/sst/isecl/lib/flavor.git v0.0.0-20190915015315-7d9923b58ff3

replace intel/isecl/lib/verifier => gitlab.devtools.intel.com/sst/isecl/lib/verifier.git v0.0.0-20190917062400-ec57b0a4bedc

replace intel/isecl/lib/platform-info => gitlab.devtools.intel.com/sst/isecl/lib/platform-info.git v0.0.0-20190918083246-1f72bff4f238

replace intel/isecl/lib/mtwilson-client => gitlab.devtools.intel.com/sst/isecl/lib/mtwilson-client.git v0.0.0-20190916120658-1d41319bea5a

replace intel/isecl/lib/clients => gitlab.devtools.intel.com/sst/isecl/lib/clients.git v0.0.0-20190915023034-59e47e67cfd6
