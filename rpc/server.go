package rpc

import (
	"intel/isecl/wlagent/filewatch"
	"intel/isecl/wlagent/wlavm"
)

// DomainXML is an struct containing domain XML as argument to allow invocation over RPC
type DomainXML struct {
	XML string
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
