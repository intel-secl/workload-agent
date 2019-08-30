/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package rpc

import (
	"encoding/json"
	"intel/isecl/lib/common/pkg/instance"
	flvr "intel/isecl/lib/flavor"
	"intel/isecl/wlagent/filewatch"
	"intel/isecl/wlagent/flavor"
	"intel/isecl/wlagent/key"
	"intel/isecl/wlagent/keycache"
	"intel/isecl/wlagent/wlavm"
	"errors"
)

// DomainXML is a struct containing domain XML as argument to allow invocation over RPC
type DomainXML struct {
	XML string
}

// ManifestString is a struct containing manifest as argument to allow invocation over RPC
type ManifestString struct {
	Manifest string
}

// FlavorInfo is a struct containing image id and flavor part as arguments to allow invocation over RPC
type FlavorInfo struct {
	ImageID    string
	FlavorPart string
}

// KeyInfo is a struct containing image ID and key ID as arguments to allow invocation over RPC
type KeyInfo struct {
	ImageID    string
	KeyID      string
	Key        []byte
	ReturnCode bool
}

// VirtualMachine is type that defines the RPC functions for communicating with the Wlagent daemon Starting/Stopping a VM
type VirtualMachine struct {
	Watcher *filewatch.Watcher
}

// Start forwards the RPC request to wlavm.Start
func (vm *VirtualMachine) Start(args *DomainXML, reply *bool) error {
	// pass in vm.Watcher to get the instance to the File System Watcher
	*reply = wlavm.Start(args.XML, vm.Watcher)
	return nil
}

// Stop forwards the RPC request to wlavm.Stop
func (vm *VirtualMachine) Stop(args *DomainXML, reply *bool) error {
	// pass in vm.Watcher to get the instance to the File System Watcher
	*reply = wlavm.Stop(args.XML, vm.Watcher)
	return nil
}

// CreateInstanceTrustReport forwards the RPC request to wlavm.CreateImageTrustReport
func (vm *VirtualMachine) CreateInstanceTrustReport(args *ManifestString, status *bool) error {
	var manifestJSON instance.Manifest
	var imageFlavor flvr.SignedImageFlavor
	json.Unmarshal([]byte(args.Manifest), &manifestJSON)
	imageID := manifestJSON.InstanceInfo.ImageID
	flavor, ok := flavor.Fetch(imageID, "CONTAINER_IMAGE")
	if (flavor == "" && !ok) {
		return errors.New("Error while retrieving flavor")
	}
	json.Unmarshal([]byte(flavor), &imageFlavor)
	//adding integrity enforced value from flavor to that of manifest
	manifestJSON.ImageIntegrityEnforced = imageFlavor.ImageFlavor.IntegrityEnforced
	reportCreated := wlavm.CreateInstanceTrustReport(manifestJSON, imageFlavor)
	if !reportCreated {
		return errors.New("Error while creating trust report")
	}
	return nil
}

// FetchFlavor forwards the RPC request to flavor.Fetch
func (vm *VirtualMachine) FetchFlavor(args *FlavorInfo, outFlavor *flavor.OutFlavor) error {

	imageFlavor, returnCode := flavor.Fetch(args.ImageID, args.FlavorPart)
	var o = flavor.OutFlavor{
		ReturnCode:  returnCode,
		ImageFlavor: imageFlavor,
	}

	*outFlavor = o
	return nil
}

// CacheKey forwards the RPC request to key.Cache
func (vm *VirtualMachine) CacheKey(args *KeyInfo, reply *bool) error {

	*reply = key.Cache(args.ImageID, args.KeyID)
	return nil
}

func (vm *VirtualMachine) GetKeyFromKeyCache (args *KeyInfo, outKeyInfo *KeyInfo) (error) {
        key, returnCode := keycache.Get(args.KeyID)
        var k = KeyInfo{
          KeyID:   args.KeyID,
          Key:     key.Bytes,
          ReturnCode: returnCode,
        }
        *outKeyInfo = k
        return nil
}
