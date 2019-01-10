module intel/isecl/wlagent

require (
	intel/isecl/lib/common v0.0.0
	intel/isecl/lib/flavor v0.0.0
	intel/isecl/lib/platform-info v0.0.0
	intel/isecl/lib/tpm v0.0.0
	intel/isecl/lib/verifier v0.0.0
	intel/isecl/lib/vml v0.0.0
)

replace intel/isecl/lib/tpm => gitlab.devtools.intel.com/sst/isecl/lib/tpm v0.0.0-20190108014617-bb939ece6600

replace intel/isecl/lib/common => gitlab.devtools.intel.com/sst/isecl/lib/common v0.0.0-20190102215830-f9c9e2da84e8

replace intel/isecl/lib/flavor => gitlab.devtools.intel.com/sst/isecl/lib/flavor v0.0.0-20181206181952-1ec1e81fed41

replace intel/isecl/lib/verifier => gitlab.devtools.intel.com/sst/isecl/lib/verifier v0.0.0-20181221212114-b1d5e4114406

replace intel/isecl/lib/vml => gitlab.devtools.intel.com/sst/isecl/lib/volume-management v0.0.0-20181206215408-38309deeadf7

replace intel/isecl/lib/platform-info => gitlab.devtools.intel.com/sst/isecl/lib/platform-info v0.0.0-20181206180455-b2908f06aa05
