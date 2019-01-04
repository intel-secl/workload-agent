module intel/isecl/wlagent

require (
	intel/isecl/lib/tpm v0.0.0
	intel/isecl/lib/common v0.0.0
)

replace intel/isecl/lib/tpm => gitlab.devtools.intel.com/sst/isecl/lib/tpm v0.0.0-20181212192313-a74fef1b8042

replace intel/isecl/lib/common => gitlab.devtools.intel.com/sst/isecl/lib/common v1.0/features/setup
