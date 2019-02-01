package consts

// Define constants to be accessed in other packages
const (
	MTWILSON_API_URL                      = "MTWILSON_API_URL"
	MTWILSON_API_USERNAME                 = "MTWILSON_API_USERNAME"
	MTWILSON_API_PASSWORD                 = "MTWILSON_API_PASSWORD"
	MTWILSON_TLS_SHA256                   = "MTWILSON_TLS_SHA256"
	WLS_API_URL                           = "WLS_API_URL"
	WLS_API_USERNAME                      = "WLS_API_USERNAME"
	WLS_API_PASSWORD                      = "WLS_API_PASSWORD"
	WLS_TLS_SHA256                        = "WLS_TLS_SHA256"
	LOG_LEVEL                             = "LOG_LEVEL"
	AikSecretKeyName                      = "aik.secret"
	TrustAgentConfigDirEnv                = "TRUST_AGENT_CONFIGURATION"
	TAConfigAikSecretCmd                  = "tagent config aik.secret"
	BindingKeyFileName                    = "bindingkey.json"
	SigningKeyFileName                    = "signingkey.json"
	BindingKeyPemFileName                 = "bindingkey.pem"
	SigningKeyPemFileName                 = "signingkey.pem"
	ImageInstanceCountAssociationFileName = "image_instance_association"
	EnvFileName                           = "workloadagent.env"
	DevMapperDirPath                      = "/dev/mapper/"
	LogDirPath                            = "/var/log/workloadagent/"
	LogFileName                           = "workloadagent.log"
	ConfigFilePath                        = "/etc/workloadagent/config.yml"
	ConfigDirPath                         = "/etc/workloadagent/"
	OptDirPath                            = "/opt/workloadagent/"
	LibvirtHookFilePath                   = "/etc/libvirt/hooks/qemu"
)