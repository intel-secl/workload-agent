/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package rpc

import (
	"fmt"
	cLog "intel/isecl/lib/common/v4/log"
	"intel/isecl/lib/common/v4/log/message"
	"intel/isecl/lib/common/v4/proc"
	"intel/isecl/lib/common/v4/validation"
	"intel/isecl/wlagent/v4/filewatch"
	"intel/isecl/wlagent/v4/wlavm"
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
