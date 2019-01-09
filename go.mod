module intel/isecl/wlagent

require (
	intel/isecl/lib/tpm v0.0.0
	intel/isecl/lib/vml v0.0.0
	intel/isecl/lib/common v0.0.0
	gopkg.in/yaml.v2 v2.2.2
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)

replace intel/isecl/lib/tpm => gitlab.devtools.intel.com/sst/isecl/lib/tpm v1.0/develop
replace intel/isecl/lib/vml => gitlab.devtools.intel.com/sst/isecl/lib/volume-management v0.0.0-20181206215408-38309deeadf7
replace intel/isecl/lib/common => gitlab.devtools.intel.com/sst/isecl/lib/common v1.0/features/setup
