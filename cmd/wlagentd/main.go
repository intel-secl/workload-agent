package main

import (
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/consts"
	"intel/isecl/wlagent/filewatch"
	wlrpc "intel/isecl/wlagent/rpc"
	"net"
	"net/rpc"

	log "github.com/sirupsen/logrus"
)

func main() {
	// Save log configurations
	config.LogConfiguration(consts.LogDirPath + consts.DaemonLogFileName)

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
		RPCSocketFilePath := consts.RunDirPath + consts.RPCSocketFileName
		// block and loop, daemon doesnt need to run on go routine
		l, err := net.Listen("unix", RPCSocketFilePath)
		if err != nil {
			log.Error(err)
			return
		}
		defer l.Close()
		for {
			r := rpc.NewServer()
			vm := &wlrpc.VirtualMachine{
				Watcher: fileWatcher,
			}
			err = r.Register(vm)
			if err != nil {
				log.Error(err)
				return
			}
			r.Accept(l)
		}
	}()
	// block until stop channel receives
	<-stop
}
