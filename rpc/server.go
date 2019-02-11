package rpc

import (
	"intel/isecl/wlagent/filewatch"
	"intel/isecl/wlagent/wlavm"

	log "github.com/sirupsen/logrus"
)

// StartVMArgs is an struct containing arguments to start a VM instance to allow invocation over RPC
type StartVMArgs struct {
	InstanceUUID string
	ImageUUID    string
	ImagePath    string
	InstancePath string
	DiskSize     string
}

// StopVMArgs is an struct containing arguments to stop a VM instance to allow invocation over RPC
type StopVMArgs struct {
	InstanceUUID string
	ImageUUID    string
	InstancePath string
}

// VirtualMachine is type that defines the RPC functions for communicating with the Wlagent daemon Starting/Stopping a VM
type VirtualMachine struct {
	Watcher *filewatch.Watcher
}

// Start forwards the RPC request to wlavm.Start
func (vm *VirtualMachine) Start(args *StartVMArgs, reply *int) error {
	// pass in vm.Watcher to get the instance to the File System Watcher
	log.Info("vm start server calling WLA start")
	*reply = wlavm.Start(args.InstanceUUID, args.ImageUUID, args.ImagePath, args.InstancePath, args.DiskSize, vm.Watcher)
	return nil
}

// Stop forwards the RPC request to wlavm.Stop
func (vm *VirtualMachine) Stop(args *StopVMArgs, reply *int) error {
	// pass in vm.Watcher to get the instance to the File System Watcher
	*reply = wlavm.Stop(args.InstanceUUID, args.ImageUUID, args.InstancePath, vm.Watcher)
	return nil
}
