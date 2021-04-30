/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package consts

// Env var names for setup
const (
	HvsUrlEnv            = "HVS_URL"
	WlsApiUrlEnv         = "WLS_API_URL"
	WlaUsernameEnv       = "WLA_SERVICE_USERNAME"
	WlaPasswordEnv       = "WLA_SERVICE_PASSWORD"
	CmsTlsCertDigestEnv  = "CMS_TLS_CERT_SHA384"
	AasUrl               = "AAS_API_URL"
	CmsBaseUrl           = "CMS_BASE_URL"
	BearerTokenEnv       = "BEARER_TOKEN"
	LogLevelEnvVar       = "LOG_LEVEL"
	LogEntryMaxlengthEnv = "LOG_ENTRY_MAXLENGTH"
	EnableConsoleLogEnv  = "WLA_ENABLE_CONSOLE_LOG"
)

const (
	ExplicitServiceName                = "Workload Agent"
	MinLogEntryMaxlength               = 100
	DefaultLogEntryMaxlength           = 300
	SkipFlavorSignatureVerificationEnv = "SKIP_FLAVOR_SIGNATURE_VERIFICATION"
	TAConfigDirEnvVar                  = "TRUSTAGENT_CONFIGURATION"
	TAConfigAikSecretCmd               = "tagent config aik.secret"
	TAAikPemFileName                   = "aik.pem"
	TAUserNameEnvVar                   = "TRUSTAGENT_USERNAME"
	BindingKeyFileName                 = "bindingkey.json"
	SigningKeyFileName                 = "signingkey.json"
	BindingKeyPemFileName              = "bindingkey.pem"
	SigningKeyPemFileName              = "signingkey.pem"
	ImageVmCountAssociationFileName    = "image_vm_association"
	DevMapperDirPath                   = "/dev/mapper/"
	MountPath                          = "/mnt/workload-agent/crypto/"
	LogDirPath                         = "/var/log/workload-agent/"
	SecurityLogFile                    = LogDirPath + "workload-agent-security.log"
	DefaultLogFile                     = LogDirPath + "workload-agent.log"
	ConfigFileName                     = "config.yml"
	ConfigDirPath                      = "/etc/workload-agent/"
	OptDirPath                         = "/opt/workload-agent/"
	RunDirPath                         = "/var/run/workload-agent/"
	LibvirtHookFilePath                = "/etc/libvirt/hooks/qemu"
	RPCSocketFileName                  = "wlagent.sock"
	WlagentSymLink                     = "/usr/local/bin/wlagent"
	ServiceStartCmd                    = "systemctl start wlagent"
	ServiceStopCmd                     = "systemctl stop wlagent"
	ServiceStatusCmd                   = "systemctl status wlagent"
	ServiceRemoveCmd                   = "systemctl disable wlagent"
	PemCertificateHeader               = "CERTIFICATE"
	TrustedCaCertsDir                  = ConfigDirPath + "certs/trustedca/"
	FlavorSigningCertDir               = ConfigDirPath + "certs/flavorsign/"
	DefaultTrustagentUser              = "tagent"
	DefaultTrustagentConfiguration     = "/opt/trustagent/configuration"
)

const (
	QemuImgUtilPath = "/usr/bin/qemu-img"
	// Fields for qemu-img info output
	QemuImgInfoBackingFileField = "backing file"
	QemuImgInfoVirtualSizeField = "virtual size"
	QemuImgInfoFileFormatField  = "file format"
	GetImgInfoCmd               = "info %s --force-share"
	CreateVmDiskCmd             = "create -f %s -o backing_file=%s,backing_fmt=%s %s"
	ResizeVmDiskCmd             = "resize %s %s"
)

// Task Names
const (
	SetupAllCommand            = "all"
	DownloadRootCACertCommand  = "download_ca_cert"
	RegisterSigningKeyCommand  = "RegisterSigningKey"
	RegisterBindingKeyCommand  = "RegisterBindingKey"
	UpdateServiceConfigCommand = "update_service_config"
	CreateBindingKey           = "BindingKey"
	CreateSigningKey           = "SigningKey"
)
