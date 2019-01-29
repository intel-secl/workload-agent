package main

import (
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/filewatch"
	wlrpc "intel/isecl/wlagent/rpc"
	"log"
	"net"
	"net/rpc"
)

func main() {
	fileWatcher, err := filewatch.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	// stop signaler
	stop := make(chan bool)
	defer fileWatcher.Close()
	go func() {
		for {
			fileWatcher.Watch()
		}
	}()
	go func() {
		for {
			// block and loop, daemon doesnt need to run on go routine
			l, err := net.Listen("unix", config.RPCSocketFilePath)
			if err != nil {
				return
			}
			r := rpc.NewServer()
			vm := &wlrpc.VirtualMachine{
				Watcher: fileWatcher,
			}
			err = r.Register(vm)
			if err != nil {
				return
			}
			r.Accept(l)
		}
	}()
	// block until stop channel receives
	<-stop
}
