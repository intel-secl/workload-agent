package main

import (
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/rpc"
)

func main() {
	for {
		// block and loop, daemon doesnt need to run on go routine
		rpc.ListenAndAccept("unix", config.RPCSocketFilePath)
	}
}
