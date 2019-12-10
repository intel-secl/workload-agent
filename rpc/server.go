/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package rpc

import (
	"encoding/json"
	"fmt"
	cLog "intel/isecl/lib/common/log"
	"intel/isecl/lib/common/proc"
	"intel/isecl/lib/common/pkg/instance"
	flvr "intel/isecl/lib/flavor"
	"intel/isecl/wlagent/filewatch"
	"intel/isecl/wlagent/flavor"
	"intel/isecl/wlagent/util"
	wlsclient "intel/isecl/wlagent/clients"
	"intel/isecl/wlagent/wlavm"
	"github.com/pkg/errors"
)

var log = cLog.GetDefaultLogger()
var secLog = cLog.GetSecurityLogger()

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
	ImageID    string
}

// KeyInfo is a struct containing image ID and key ID as arguments to allow invocation over RPC
type KeyInfo struct {
	KeyID      string
	Key        []byte
	ImageID    string
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

func (e rpcError) Error() string {
	return fmt.Sprintf("%d: %s", e.StatusCode, e.Message)
}

// Start forwards the RPC request to wlavm.Start
func (vm *VirtualMachine) Start(args *DomainXML, reply *bool) error {
	// Passing the false parameter to ensure the start vm task is not added waitgroup if there is pending signal termination on rpc
	_, err := proc.AddTask(false)
	if err != nil{
		errors.Wrap(err, "rpc/server:Start() Could not add task for vm start")
	}
	defer proc.TaskDone()
	log.Trace("rpc/server:Start() Entering")
	defer log.Trace("rpc/server:Start() Leaving")

	// pass in vm.Watcher to get the instance to the File System Watcher
	*reply = wlavm.Start(args.XML, vm.Watcher)
	return nil
}

// Stop forwards the RPC request to wlavm.Stop
func (vm *VirtualMachine) Stop(args *DomainXML, reply *bool) error {
	// Passing the true parameter to ensure the stop vm task is added to waitgroup as this action needs to be completed 
	// even if there is pending signal termination on rpc
	_, err := proc.AddTask(true)
        if err != nil{
                errors.Wrap(err, "rpc/server:Stop() Could not add task for vm stop")
        }
        defer proc.TaskDone()

	log.Trace("rpc/server:Stop() Entering")
	defer log.Trace("rpc/server:Stop() Leaving")

	// pass in vm.Watcher to get the instance to the File System Watcher
	*reply = wlavm.Stop(args.XML, vm.Watcher)
	return nil
}

// CreateInstanceTrustReport forwards the RPC request to wlavm.CreateImageTrustReport
func (vm *VirtualMachine) CreateInstanceTrustReport(args *ManifestString, status *bool) error {
	// Passing the true parameter to ensure that CreateInstanceTrustReport task is added to the waitgroup, as this action needs to be completed
        // even if there is pending signal termination on rpc 
        _, err := proc.AddTask(true)
        if err != nil{
                errors.Wrap(err, "rpc/server:CreateInstanceTrustReport() Could not add task for CreateInstanceTrustReport")
        }
        defer proc.TaskDone()

	log.Trace("rpc/server:CreateInstanceTrustReport() Entering")
	defer log.Trace("rpc/server:CreateInstanceTrustReport() Leaving")

	var manifestJSON instance.Manifest
	var imageFlavor flvr.SignedImageFlavor
	err = json.Unmarshal([]byte(args.Manifest), &manifestJSON)
	if err != nil {
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

	f, _ := json.Marshal(flavor)
	if string(f) == "" {
		return &rpcError{Message: "rpc/server:CreateInstanceTrustReport() error while retrieving flavor", StatusCode: 1}
	}
	err = json.Unmarshal([]byte(f), &imageFlavor)
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
        if err != nil{
                errors.Wrap(err, "rpc/server:CreateInstanceTrustReport() Could not add task for FetchFlavor")
        }
        defer proc.TaskDone()

	log.Trace("rpc/server:FetchFlavor() Entering")
	defer log.Trace("rpc/server:FetchFlavor() Leaving")

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
        if err != nil{
                errors.Wrap(err, "rpc/server:CreateInstanceTrustReport() Could not add task for FetchKey")
        }
        defer proc.TaskDone()
	log.Trace("rpc/server:FetchKey() Entering")
	defer log.Trace("rpc/server:FetchKey() Leaving")

	wrappedKey, returnCode := flavor.RetrieveKey(args.KeyID, args.ImageID)
	key, err := util.UnwrapKey(wrappedKey)
	if err != nil {
		return &rpcError{Message: "rpc/server:FetchKey() error while unwrapping the key", StatusCode: 1}
	}
	var k = KeyInfo{
		KeyID:      args.KeyID,
		Key:        key,
		ReturnCode: returnCode,
	}

	*outKeyInfo = k
	return nil
}
