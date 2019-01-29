package main

import (
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/filewatch"
	"intel/isecl/wlagent/rpc"
	"log"
)

var FileWatcher *filewatch.Watcher

func main() {
	FileWatcher, err := filewatch.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	// stop signaler
	stop := make(chan bool)
	defer FileWatcher.Close()
	go FileWatcher.Watch()
	go func() {
		for {
			// block and loop, daemon doesnt need to run on go routine
			err := rpc.ListenAndAccept("unix", config.RPCSocketFilePath)
			log.Println(err)
		}
	}()
	// block until stop channel receives
	<-stop
}
