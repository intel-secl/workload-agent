module intel/isecl/wlagent

require (
	github.com/sirupsen/logrus v1.3.0
	golang.org/x/net v0.0.0-20190206173232-65e2d4e15006 // indirect
	gopkg.in/xmlpath.v2 v2.0.0-20150820204837-860cbeca3ebc

	gopkg.in/yaml.v2 v2.2.2
	intel/isecl/lib/common v0.0.0
	intel/isecl/lib/flavor v0.0.0
	intel/isecl/lib/platform-info v0.0.0
	intel/isecl/lib/tpm v0.0.0
	intel/isecl/lib/verifier v0.0.0
	intel/isecl/lib/vml v0.0.0
)

replace intel/isecl/lib/tpm => gitlab.devtools.intel.com/sst/isecl/lib/tpm v0.0.0-20190202165337-322040ceed08

replace intel/isecl/lib/vml => gitlab.devtools.intel.com/sst/isecl/lib/volume-management v0.0.0-20190208083034-9644024d1523

replace intel/isecl/lib/common => gitlab.devtools.intel.com/sst/isecl/lib/common v0.0.0-20190215063009-c87aeabe69ec

replace intel/isecl/lib/flavor => gitlab.devtools.intel.com/sst/isecl/lib/flavor v0.0.0-20190212205826-02e0559b238d

replace intel/isecl/lib/verifier => gitlab.devtools.intel.com/sst/isecl/lib/verifier v0.0.0-20190219043745-4337d01a5a89

replace intel/isecl/lib/platform-info => gitlab.devtools.intel.com/sst/isecl/lib/platform-info v0.0.0-20181206180455-b2908f06aa05
