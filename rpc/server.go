package rpc

import (
	"errors"
	"intel/isecl/wlagent/pkg"
	"intel/isecl/wlagent/wlavm"
	"net"
	"net/rpc"
)

// StartVMArgs is an struct containing arguments to start a VM instance to allow invocation over RPC
type StartVMArgs struct {
	InstanceUUID string
	ImageUUID    string
	ImagePath    string
	InstancePath string
	DiskSize     string
}

// StartVMArgs is an struct containing arguments to stop a VM instance to allow invocation over RPC
type StopVMArgs struct {
	InstanceUUID string
	ImageUUID    string
	InstancePath string
	ImagePath    string
}

// VirtualMachine is type that defines the RPC functions for communicating with the Wlagent daemon Starting/Stopping a VM
type VirtualMachine int

// Start forwards the RPC request to wlavm.Start
func (*VirtualMachine) Start(args *StartVMArgs, reply *int) error {
	*reply = wlavm.Start(args.InstanceUUID, args.ImageUUID, args.ImagePath, args.InstancePath, args.DiskSize)
	return nil
}

// Stop forwards the RPC request to pkg.QemuStopIntercept
func (*VirtualMachine) Stop(args *StopVMArgs, reply *int) error {
	*reply = pkg.QemuStopIntercept(args.InstanceUUID, args.ImageUUID, args.InstancePath, args.ImagePath)
	return nil
}

// ListenAndAccept creates a listener for the RPC server, and begins Accepting connections
// This function blocks and loops, and will always return an error
// Typically this function will be invoked by a goroutine
func ListenAndAccept(socketType, socketAddr string) (err error) {
	l, err := net.Listen(socketType, socketAddr)
	if err != nil {
		return
	}
	r := rpc.NewServer()
	err = r.Register(new(VirtualMachine))
	if err != nil {
		return
	}
	r.Accept(l)
	return errors.New("rpc server stopped")
}
