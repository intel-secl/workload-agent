/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package rpc

import (
	"encoding/json"
	"fmt"
	wlsModel "github.com/intel-secl/intel-secl/v3/pkg/model/wls"
	cLog "intel/isecl/lib/common/v3/log"
	"intel/isecl/lib/common/v3/log/message"
	"intel/isecl/lib/common/v3/pkg/instance"
	"intel/isecl/lib/common/v3/proc"
	"intel/isecl/lib/common/v3/validation"
	wlsclient "intel/isecl/wlagent/v3/clients"
	"intel/isecl/wlagent/v3/filewatch"
	"intel/isecl/wlagent/v3/flavor"
	"intel/isecl/wlagent/v3/util"
	"intel/isecl/wlagent/v3/wlavm"
	"sync"

	"github.com/pkg/errors"
)

var log = cLog.GetDefaultLogger()
var secLog = cLog.GetSecurityLogger()

var wlaMtx sync.Mutex

// DomainXML is a struct containing domain XML as argument to allow invocation over RPC
type DomainXML struct {
	XML string
}

// ManifestString is a struct containing manifest as argument to allow invocation over RPC
type ManifestString struct {
	Manifest string
}

// FlavorInfo is a struct containing image id as argument to allow invocation over RPC
type FlavorInfo struct {
	ImageID string
}

// KeyInfo is a struct containing image ID and key ID as arguments to allow invocation over RPC
type KeyInfo struct {
	KeyID      string
	Key        []byte
	ReturnCode bool
}

// VirtualMachine is type that defines the RPC functions for communicating with the Wlagent daemon Starting/Stopping a VM
type VirtualMachine struct {
	Watcher *filewatch.Watcher
}

type rpcError struct {
	StatusCode int
	Message    string
}

//TransferURL ...
type TransferURL struct {
	URL string
}

// KeyOnly ...
type KeyOnly struct {
	KeyUrl string `json:"key_url"`
	Key    []byte `json:"key"`
}

func (e rpcError) Error() string {
	return fmt.Sprintf("%d: %s", e.StatusCode, e.Message)
}

// Start forwards the RPC request to wlavm.Start
func (vm *VirtualMachine) Start(args *DomainXML, reply *bool) error {
	// Passing the false parameter to ensure the start vm task is not added to waitgroup if there is pending signal termination on rpc
	_, err := proc.AddTask(false)
	if err != nil {
		return errors.Wrap(err, "rpc/server:Start() Could not add task for vm start")
	}
	defer proc.TaskDone()
	log.Trace("rpc/server:Start() Entering")
	defer log.Trace("rpc/server:Start() Leaving")

	if err = validation.ValidateXMLString(args.XML); err != nil {
		secLog.Errorf("rpc:server() Start: %s, Invalid domain XML format", message.InvalidInputBadParam)
		return nil
	}

	// pass in vm.Watcher to get the instance to the File System Watcher
	*reply = wlavm.Start(args.XML, vm.Watcher)
	return nil
}

// Prepare forwards the RPC request to wlavm.Prepare
func (vm *VirtualMachine) Prepare(args *DomainXML, reply *bool) error {
	// Passing the false parameter to ensure the prepare vm task is not added to waitgroup if there is pending signal termination on rpc
	_, err := proc.AddTask(false)
	if err != nil {
		return errors.Wrap(err, "rpc/server:Prepare() Could not add task for vm prepare")
	}
	defer proc.TaskDone()
	log.Trace("rpc/server:Prepare() Entering")
	defer log.Trace("rpc/server:Prepare() Leaving")

	wlaMtx.Lock()
	defer wlaMtx.Unlock()

	if err = validation.ValidateXMLString(args.XML); err != nil {
		secLog.Errorf("rpc:server() Prepare: %s, Invalid domain XML format", message.InvalidInputBadParam)
		return nil
	}

	// pass in vm.Watcher to get the instance to the File System Watcher
	*reply = wlavm.Prepare(args.XML, vm.Watcher)
	return nil
}

// Stop forwards the RPC request to wlavm.Stop
func (vm *VirtualMachine) Stop(args *DomainXML, reply *bool) error {
	// Passing the true parameter to ensure the stop vm task is added to waitgroup as this action needs to be completed
	// even if there is pending signal termination on rpc
	_, err := proc.AddTask(true)
	if err != nil {
		return errors.Wrap(err, "rpc/server:Stop() Could not add task for vm stop")
	}
	defer proc.TaskDone()

	log.Trace("rpc/server:Stop() Entering")
	defer log.Trace("rpc/server:Stop() Leaving")

	wlaMtx.Lock()
	defer wlaMtx.Unlock()

	if err = validation.ValidateXMLString(args.XML); err != nil {
		secLog.Errorf("rpc:server() Stop: %s, Invalid domain XML format", message.InvalidInputBadParam)
		return nil
	}

	// pass in vm.Watcher to get the instance to the File System Watcher
	*reply = wlavm.Stop(args.XML, vm.Watcher)
	return nil
}

// CreateInstanceTrustReport forwards the RPC request to wlavm.CreateImageTrustReport
func (vm *VirtualMachine) CreateInstanceTrustReport(args *ManifestString, status *bool) error {
	// Passing the true parameter to ensure that CreateInstanceTrustReport task is added to the waitgroup, as this action needs to be completed
	// even if there is pending signal termination on rpc
	_, err := proc.AddTask(true)
	if err != nil {
		return errors.Wrap(err, "rpc/server:CreateInstanceTrustReport() Could not add task for CreateInstanceTrustReport")
	}
	defer proc.TaskDone()

	log.Trace("rpc/server:CreateInstanceTrustReport() Entering")
	defer log.Trace("rpc/server:CreateInstanceTrustReport() Leaving")

	var manifestJSON instance.Manifest
	var imageFlavor wlsModel.SignedImageFlavor

	err = json.Unmarshal([]byte(args.Manifest), &manifestJSON)
	if err != nil {
		secLog.Error(message.InvalidInputBadEncoding)
		return &rpcError{Message: "rpc/server:CreateInstanceTrustReport() error while unmarshalling manifest", StatusCode: 1}
	}
	imageID := manifestJSON.InstanceInfo.ImageID
	flavor, err := wlsclient.GetImageFlavor(imageID, "CONTAINER_IMAGE")
	if err != nil {
		return &rpcError{Message: "rpc/server:CreateInstanceTrustReport() Error while retrieving the image flavor", StatusCode: 1}
	}

	if flavor.ImageFlavor.Meta.ID == "" {
		log.Infof("rpc/server:CreateInstanceTrustReport() Flavor does not exist for the image: %s", imageID)
		return nil
	}

	f, err := json.Marshal(flavor)
	if err != nil {
		return &rpcError{Message: "rpc/server:CreateInstanceTrustReport() error while marshalling flavor", StatusCode: 1}
	}

	if string(f) == "" {
		return &rpcError{Message: "rpc/server:CreateInstanceTrustReport() error while retrieving flavor", StatusCode: 1}
	}
	err = json.Unmarshal(f, &imageFlavor)
	if err != nil {
		return &rpcError{Message: "rpc/server:CreateInstanceTrustReport() error while unmarshalling flavor", StatusCode: 1}
	}
	//adding integrity enforced value from flavor to that of manifest
	manifestJSON.ImageIntegrityEnforced = imageFlavor.ImageFlavor.IntegrityEnforced
	reportCreated := wlavm.CreateInstanceTrustReport(manifestJSON, imageFlavor)
	if !reportCreated {
		return &rpcError{Message: "rpc/server:CreateInstanceTrustReport() error while creating trust report", StatusCode: 1}
	}
	return nil
}

// FetchFlavor forwards the RPC request to flavor.Fetch
func (vm *VirtualMachine) FetchFlavor(args *FlavorInfo, outFlavor *flavor.OutFlavor) error {
	// Passing the false parameter to ensure that FetchFlavor task is not added to the wait group if there is pending signal termination on rpc
	_, err := proc.AddTask(false)
	if err != nil {
		return errors.Wrap(err, "rpc/server:FetchFlavor() Could not add task for FetchFlavor")
	}
	defer proc.TaskDone()

	log.Trace("rpc/server:FetchFlavor() Entering")
	defer log.Trace("rpc/server:FetchFlavor() Leaving")

	// validate input
	if err = validation.ValidateUUIDv4(args.ImageID); err != nil {
		secLog.Errorf("rpc/server:FetchFlavor() %s, Invalid Image UUID format", message.InvalidInputBadParam)
		return nil
	}

	imageFlavor, returnCode := flavor.Fetch(args.ImageID)
	var o = flavor.OutFlavor{
		ReturnCode:  returnCode,
		ImageFlavor: imageFlavor,
	}

	*outFlavor = o
	return nil
}

// FetchKey forwards the RPC request to flavor.RetrieveKey
func (vm *VirtualMachine) FetchKey(args *KeyInfo, outKeyInfo *KeyInfo) error {
	// Passing the true parameter to ensure that FetchKey is added to the waitgroup  as this action needs to be completed
	// even if there is pending signal termination on rpc

	_, err := proc.AddTask(true)
	if err != nil {
		return errors.Wrap(err, "rpc/server:FetchKey() Could not add task for FetchKey")
	}
	defer proc.TaskDone()
	log.Trace("rpc/server:FetchKey() Entering")
	defer log.Trace("rpc/server:FetchKey() Leaving")

	if err = validation.ValidateUUIDv4(args.KeyID); err != nil {
		secLog.Errorf("rpc/server:FetchKey() %s, Invalid Key UUID format", message.InvalidInputBadParam)
		return nil
	}

	wrappedKey, returnCode := flavor.RetrieveKey(args.KeyID)
	log.Debugf("rpc/server:FetchKey() returnCode: %v", returnCode)
	if !returnCode {
		return &rpcError{Message: "rpc/server:FetchKey() Error while retrieving the key", StatusCode: 1}
	}
	key, err := util.UnwrapKey(wrappedKey)
	if err != nil {
		log.Errorf("rpc/server:FetchKey() Error while unwrapping the key %+v", err)
		return &rpcError{Message: "rpc/server:FetchKey() Error while unwrapping the key", StatusCode: 1}
	}

	var k = KeyInfo{
		KeyID:      args.KeyID,
		Key:        key,
		ReturnCode: returnCode,
	}

	*outKeyInfo = k
	return nil
}

// FetchKeyWithURL forwards the RPC request to flavor.RetrieveKeywithURL
func (vm *VirtualMachine) FetchKeyWithURL(args *TransferURL, outKey *KeyOnly) error {
	//func (vm *VirtualMachine) FetchKeyWithURL(args *TransferURL, outKey *[]byte) error {
	// Passing the true parameter to ensure that FetchKey is added to the waitgroup  as this action needs to be completed
	// even if there is pending signal termination on rpc

	_, err := proc.AddTask(true)
	if err != nil {
		return errors.Wrap(err, "rpc/server:FetchKeyWithURL() Could not add task for FetchKeyWithURL")
	}
	defer proc.TaskDone()
	log.Trace("rpc/server:FetchKeyWithURL() Entering")
	defer log.Trace("rpc/server:FetchKeyWithURL() Leaving")

	wrappedKey, returnCode := flavor.RetrieveKeyWithURL(args.URL)
	if !returnCode {
		return &rpcError{Message: "rpc/server:FetchKeyWithURL() error while unwrapping the key", StatusCode: 1}
	}
	key, err := util.UnwrapKey(wrappedKey)
	if err != nil {
		return &rpcError{Message: "rpc/server:FetchKeyWithURL() error while unwrapping the key", StatusCode: 1}
	}
	outKey.KeyUrl = args.URL
	outKey.Key = key
	return nil
}
