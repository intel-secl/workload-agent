package rpc

import (
	"intel/isecl/wlagent/filewatch"
	"intel/isecl/wlagent/wlavm"

	log "github.com/sirupsen/logrus"
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
func (vm *VirtualMachine) Start(args *DomainXML, reply *int) error {
	// pass in vm.Watcher to get the instance to the File System Watcher
	log.Info("vm start server calling WLA start")
	*reply = wlavm.Start(args.XML, vm.Watcher)
	return nil
}

// Stop forwards the RPC request to wlavm.Stop
func (vm *VirtualMachine) Stop(args *DomainXML, reply *int) error {
	// pass in vm.Watcher to get the instance to the File System Watcher
	*reply = wlavm.Stop(args.XML, vm.Watcher)
	return nil
}
