// +build linux

/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */

package wlavm

import (
	"crypto"
	"encoding/json"
	wlsModel "github.com/intel-secl/intel-secl/v3/pkg/model/wls"
	"github.com/pkg/errors"
	"intel/isecl/lib/common/v3/crypt"
	"intel/isecl/lib/common/v3/log/message"
	"intel/isecl/lib/common/v3/pkg/instance"
	pinfo "intel/isecl/lib/platform-info/v3/platforminfo"
	"intel/isecl/lib/tpmprovider/v3"
	"intel/isecl/lib/verifier/v3"
	"intel/isecl/lib/vml/v3"
	wlsclient "intel/isecl/wlagent/v3/clients"
	"intel/isecl/wlagent/v3/config"
	"intel/isecl/wlagent/v3/consts"
	"intel/isecl/wlagent/v3/filewatch"
	"intel/isecl/wlagent/v3/libvirt"
	"intel/isecl/wlagent/v3/util"
	"strings"
)

// Start method is used perform the VM confidentiality check before launching the VM
// Input Parameters: domainXML content string
// Return : Returns a boolean value to the main method.
// true if the vm is launched successfully, else returns false.
func Start(domainXMLContent string, filewatcher *filewatch.Watcher) bool {

	log.Trace("wlavm/start:Start() Entering")
	defer log.Trace("wlavm/start:Start() Leaving")
	var err error

	log.Info("wlavm/start:Start() Parsing domain XML to get image UUID, image path, VM UUID, VM path and disk size")
	d, err := libvirt.NewDomainParser(domainXMLContent, libvirt.Start)
	if err != nil {
		log.Error("wlavm/start:Start() Parsing error: ", err.Error())
		log.Tracef("%+v", err)
		return false
	}

	vmUUID := d.GetVMUUID()
	imageUUID := d.GetImageUUID()
	imagePath := d.GetImagePath()

	var flavorKeyInfo wlsModel.FlavorKey

	// check if the image is in the crypto path - if yes it was launched from an encrypted image
	// need to push VM instance trust report to WLS
	if strings.HasPrefix(imagePath, consts.MountPath) {
		// get host hardware UUID
		secLog.Infof("wlavm/start:Start() %s, Trying to get host hardware UUID", message.SU)
		hardwareUUID, err := pinfo.HardwareUUID()
		if err != nil {
			log.WithError(err).Error("wlavm/start:Start() Unable to get the host hardware UUID")
			return false
		}
		log.Debugf("wlavm/start:Start() The host hardware UUID is :%s", hardwareUUID)

		//get flavor-key from workload service
		// we will be hitting the WLS to retrieve the the flavor and key.
		// TODO: Investigate if it makes sense to cache the flavor locally as well with
		// an expiration time. Believe this was discussed and previously ruled out..
		// but still worth exploring for performance reasons as we want to minimize
		// making http client calls to external servers.
		log.Infof("wlavm/start:Start() Retrieving image-flavor-key for image %s from WLS", imageUUID)

		flavorKeyInfo, err = wlsclient.GetImageFlavorKey(imageUUID, hardwareUUID)
		if err != nil {
			secLog.WithError(err).Error("wlavm/start:Start() Error retrieving the image flavor and key")
			return false
		}

		if flavorKeyInfo.Flavor.Meta.ID == "" {
			log.Infof("wlavm/start:Start() Flavor does not exist for the image %s", imageUUID)
			return false
		}

		//create VM manifest
		log.Info("wlavm/start:Start() Creating VM Manifest")
		manifest, err := vml.CreateVMManifest(vmUUID, hardwareUUID, imageUUID, true)
		if err != nil {
			log.WithError(err).Error("wlavm/start:Start() Error creating the VM manifest")
			return false
		}

		//Create Image trust report
		status := CreateInstanceTrustReport(manifest, wlsModel.SignedImageFlavor{ImageFlavor: flavorKeyInfo.Flavor, Signature: flavorKeyInfo.Signature})
		if status == false {
			log.Error("wlavm/start:Start() Error while creating image trust report")
			return false
		}

		// Updating image-vm count association
		log.Info("wlavm/start:Start() Associating VM with image in image-vm-count file")
		iAssoc := ImageVMAssociation{imageUUID, imagePath}
		err = iAssoc.Create()
		if err != nil {
			log.WithError(err).Error("wlavm/start:Start() Error while updating the image-instance count file")
			return false
		}
	} else {
		log.Info("wlavm/start:Start() Image is not encrypted, returning to the hook")
	}

	log.Infof("wlavm/start:Start() VM %s started", vmUUID)
	return true
}

func CreateInstanceTrustReport(manifest instance.Manifest, flavor wlsModel.SignedImageFlavor) bool {
	log.Trace("wlavm/start:CreateInstanceTrustReport() Entering")
	defer log.Trace("wlavm/start:CreateInstanceTrustReport() Leaving")

	//create VM trust report
	log.Info("wlavm/start:CreateInstanceTrustReport() Creating image trust report")
	instanceTrustReport, err := verifier.Verify(&manifest, &flavor, consts.FlavorSigningCertDir, consts.TrustedCaCertsDir, config.Configuration.SkipFlavorSignatureVerification)
	if err != nil {
		log.WithError(err).Error("wlavm/start:CreateInstanceTrustReport() Error creating image trust report")
		log.Tracef("%+v", err)
		return false
	}
	trustreport, _ := json.Marshal(instanceTrustReport)
	log.Infof("wlavm/start:CreateInstanceTrustReport() trustreport: %s", string(trustreport))

	// compute the hash and sign
	log.Info("wlavm/start:CreateInstanceTrustReport() Signing image trust report")
	signedInstanceTrustReport, err := signInstanceTrustReport(instanceTrustReport.(*verifier.InstanceTrustReport))
	if err != nil {
		log.WithError(err).Error("wlavm/start:CreateInstanceTrustReport() Could not sign image trust report using TPM")
		log.Tracef("%+v", err)
		return false
	}

	//post VM trust report on to workload service
	log.Info("wlavm/start:CreateInstanceTrustReport() Post image trust report on WLS")
	report, _ := json.Marshal(*signedInstanceTrustReport)
	log.Debugf("wlavm/start:CreateInstanceTrustReport() Report: %s", string(report))

	err = wlsclient.PostVMReport(report)
	if err != nil {
		secLog.WithError(err).Error("wlavm/start:CreateInstanceTrustReport() Failed to post the instance trust report on to workload service")
		return false
	}
	return true
}

//Using SHA256 signing algorithm as TPM2.0 supports SHA256
func signInstanceTrustReport(report *verifier.InstanceTrustReport) (*crypt.SignedData, error) {
	log.Trace("wlavm/start:signInstanceTrustReport() Entering")
	defer log.Trace("wlavm/start:signInstanceTrustReport() Leaving")

	var signedreport crypt.SignedData

	jsonVMTrustReport, err := json.Marshal(*report)
	if err != nil {
		return nil, errors.Wrap(err, "wlavm/start:signInstanceTrustReport() could not marshal instance trust report")
	}

	signedreport.Data = jsonVMTrustReport
	signedreport.Alg = crypt.GetHashingAlgorithmName(crypto.SHA256)
	log.Debug("wlavm/start:signInstanceTrustReport() Getting Signing Key Certificate from disk")
	signedreport.Cert, err = config.GetSigningCertFromFile()
	if err != nil {
		return nil, err
	}
	log.Debug("wlavm/start:signInstanceTrustReport() Using TPM to create signature")
	signature, err := createSignatureWithTPM(signedreport.Data, crypto.SHA256)
	if err != nil {
		return nil, err
	}
	signedreport.Signature = signature

	return &signedreport, nil
}

func createSignatureWithTPM(data []byte, alg crypto.Hash) ([]byte, error) {
	log.Trace("wlavm/start:createSignatureWithTPM() Entering")
	defer log.Trace("wlavm/start:createSignatureWithTPM() Leaving")

	var signingKey tpmprovider.CertifiedKey

	// Get the Signing Key that is stored on disk
	log.Debug("wlavm/start:createSignatureWithTPM() Getting the signing key from WA config path")
	signingKeyJson, err := config.GetSigningKeyFromFile()
	if err != nil {
		return nil, err
	}

	log.Debug("wlavm/start:createSignatureWithTPM() Unmarshalling the signing key file contents into signing key struct")
	err = json.Unmarshal(signingKeyJson, &signingKey)
	if err != nil {
		return nil, err
	}

	// // Get the secret associated when the SigningKey was created.
	// log.Debug("wlavm/start:createSignatureWithTPM() Retrieving the signing key secret form WA configuration")
	// keyAuth, err := hex.DecodeString(config.Configuration.SigningKeySecret)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "wlavm/start:createSignatureWithTPM() Error retrieving the signing key secret from configuration")
	// }

	// Before we compute the hash, we need to check the version of TPM as TPM 1.2 only supports SHA1
	t, err := util.GetTpmInstance()
	if err != nil {
		return nil, errors.Wrap(err, "wlavm/start:createSignatureWithTPM() Error attempting to create signature - could not open TPM")
	}
	defer t.Close()
	log.Debug("wlavm/start:createSignatureWithTPM() Computing the hash of the report to be signed by the TPM")
	h, err := crypt.GetHashData(data, alg)
	if err != nil {
		return nil, errors.Wrap(err, "wlavm/start:createSignatureWithTPM() Error while getting hash of instance report")
	}

	secLog.Infof("wlavm/start:createSignatureWithTPM() %s, Using TPM to sign the hash", message.SU)
	signature, err := t.Sign(&signingKey, config.Configuration.SigningKeySecret, h)
	if err != nil {
		return nil, errors.Wrap(err, "wlavm/start:createSignatureWithTPM() Error while creating tpm signature")
	}
	log.Debug("wlavm/start:createSignatureWithTPM() Report signed by TPM successfully")
	return signature, nil
}
