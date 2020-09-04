/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package consts

import "crypto"

// Define constants to be accessed in other packages
const (
	HVS_URL                                      = "HVS_URL"
	WLS_API_URL                                  = "WLS_API_URL"
	WLA_USERNAME                                 = "WLA_SERVICE_USERNAME"
	WLA_PASSWORD                                 = "WLA_SERVICE_PASSWORD"
	LogLevelEnvVar                               = "LOG_LEVEL"
	LogEntryMaxlengthEnv                         = "LOG_ENTRY_MAXLENGTH"
	DefaultLogEntryMaxlength                     = 300
	SkipFlavorSignatureVerification              = "SKIP_FLAVOR_SIGNATURE_VERIFICATION"
	AikSecretKeyName                             = "aik.secret"
	TAConfigDirEnvVar                            = "TRUSTAGENT_CONFIGURATION"
	TAConfigAikSecretCmd                         = "tagent config aik.secret"
	TAAikPemFileName                             = "aik.pem"
	TAUserNameEnvVar                             = "TRUSTAGENT_USERNAME"
	BindingKeyFileName                           = "bindingkey.json"
	SigningKeyFileName                           = "signingkey.json"
	BindingKeyPemFileName                        = "bindingkey.pem"
	SigningKeyPemFileName                        = "signingkey.pem"
	ImageVmCountAssociationFileName              = "image_vm_association"
	EnvFileName                                  = "workload-agent.env"
	DevMapperDirPath                             = "/dev/mapper/"
	MountPath                                    = "/mnt/workload-agent/crypto/"
	LogDirPath                                   = "/var/log/workload-agent/"
	SecurityLogFile                              = LogDirPath + "workload-agent-security.log"
	DefaultLogFile                               = LogDirPath + "workload-agent.log"
	DaemonLogFile                                = LogDirPath + "daemon.log"
	ConfigFileName                               = "config.yml"
	ConfigDirPath                                = "/etc/workload-agent/"
	FlavorSigningCertPath                        = ConfigDirPath + "flavor-signing-cert.pem" //Manually copy Flavor Signing Certificate from WPM to WLA
	OptDirPath                                   = "/opt/workload-agent/"
	BinDirPath                                   = "/opt/workload-agent/bin/"
	RunDirPath                                   = "/var/run/workload-agent/"
	SecureOverlayLayersPath                      = "/var/lib/docker/secureoverlay2/"
	LibvirtHookFilePath                          = "/etc/libvirt/hooks/qemu"
	RPCSocketFileName                            = "wlagent.sock"
	WlagentSymLink                               = "/usr/local/bin/wlagent"
	ServiceStartCmd                              = "systemctl start wlagent"
	ServiceStopCmd                               = "systemctl stop wlagent"
	ServiceStatusCmd                             = "systemctl status wlagent"
	ServiceRemoveCmd                             = "systemctl disable wlagent"
	PemCertificateHeader                         = "CERTIFICATE"
	HashingAlgorithm                 crypto.Hash = crypto.SHA384
	WLABinFilePath                               = "/usr/local/bin/wlagent"
	TrustedCaCertsDir                            = ConfigDirPath + "certs/trustedca/"
	FlavorSigningCertDir                         = ConfigDirPath + "certs/flavorsign/"
	TrustedJWTSigningCertsDir                    = ConfigDirPath + "certs/trustedjwt/"
	CmsTlsCertDigestEnv                          = "CMS_TLS_CERT_SHA384"
	AAS_URL                                      = "AAS_API_URL"
	CMS_BASE_URL                                 = "CMS_BASE_URL"
	BEARER_TOKEN_ENV                             = "BEARER_TOKEN"
	DEFAULT_TRUSTAGENT_USER                      = "tagent"
	DEFAULT_TRUSTAGENT_CONFIGURATION             = "/opt/trustagent/configuration"
)
