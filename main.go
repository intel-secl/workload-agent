package main

import (
	"fmt"
	csetup "intel/isecl/lib/common/setup"
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/consts"
	wlrpc "intel/isecl/wlagent/rpc"
	"intel/isecl/wlagent/setup"
	"io/ioutil"
	"net"
	"net/rpc"

	// "intel/isecl/wlagent/wlavm"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	component         string = "workload-agent"
	version           string = ""
	buildid           string = ""
	buildtype         string = "dev"
	rpcSocketFilePath string = consts.RunDirPath + consts.RPCSocketFileName
	pidFilePath              = consts.RunDirPath + consts.PIDFileName
)

func printVersion() {
	if version == "" {
		fmt.Println("Version information not set")
		fmt.Println("Have to be set at build time using -ldflags -X options")
		return
	}
	if buildid == "" {
		buildid = time.Now().Format("2006-01-02 15:04")
	}
	fmt.Printf("%s Version : %s\nBuild : %s-%s\n", component, version, buildid, buildtype)

}

func printUsage() {
	fmt.Printf("Work Load Agent\n")
	fmt.Printf("===============\n\n")
	fmt.Printf("usage : %s <command> [<args>]\n\n", os.Args[0])
	fmt.Printf("Following are the list of commands\n")
	fmt.Printf("\tsetup|start-vm|stop-vm|--help|--version\n\n")
	fmt.Printf("setup command is used to run setup tasks\n")
	fmt.Printf("\tusage : %s setup [<tasklist>]\n", os.Args[0])
	fmt.Printf("\t\t<tasklist>-space seperated list of tasks\n")
	fmt.Printf("\t\t\t-Supported tasks - SigningKey BindingKey\n")
	fmt.Printf("\tExample :-\n")
	fmt.Printf("\t\t%s setup\n", os.Args[0])
	fmt.Printf("\t\t%s setup SigningKey\n", os.Args[0])
}

// main is the primary control loop for wlagent. support setup, vmstart, vmstop etc
func main() {
	// Save log configurations
	config.LogConfiguration(consts.LogDirPath + consts.LogFileName)

	args := os.Args[1:]
	if len(args) <= 0 {
		fmt.Println("Command not found. Usage below")
		printUsage()
		return
	}

	switch arg := strings.ToLower(args[0]); arg {
	case "--version", "-v", "version":
		printVersion()

	case "setup":
		// Check if nosetup environment variable is true, if yes then skip the setup tasks
		if nosetup, err := strconv.ParseBool(os.Getenv("WORKLOAD_AGENT_NOSETUP")); err != nil && nosetup == false {
			// Workaround for tpm2-abrmd bug in RHEL 7.5
			t, err := tpm.Open()
			if err != nil {
				log.Error("Error while opening a connection to TPM.")
				os.Exit(1)
			}

			// Run list of setup tasks one by one
			setupRunner := &csetup.Runner{
				Tasks: []csetup.Task{
					setup.SigningKey{
						T: t,
					},
					setup.BindingKey{
						T: t,
					},
					setup.RegisterBindingKey{},
					setup.RegisterSigningKey{},
				},
				AskInput: false,
			}
			defer t.Close()
			err = setupRunner.RunTasks(args[1:]...)
			if err != nil {
				fmt.Println("Error running setup: ", err)
				os.Exit(1)
			}
		} else {
			fmt.Println("WORKLOAD_AGENT_NOSETUP is set, skipping setup")
			os.Exit(0)
		}

	case "start":
		if len(args[1:]) < 1 {
			log.Info("Invalid number of parameters")
			os.Exit(1)
		}

		log.Info("workload-agent start called")
		conn, err := net.Dial("unix", rpcSocketFilePath)
		if err != nil {
			log.Fatal("start-vm: failed to dial wlagent.sock, is wlagent running?")
		}
		client := rpc.NewClient(conn)
		var returnCode int
		var args = wlrpc.DomainXML{
			XML: args[1],
		}
		err = client.Call("VirtualMachine.Start", &args, &returnCode)
		if err != nil {
			log.Error("client call failed")
		}

		if returnCode == 1 {
			os.Exit(1)
		} else {
			os.Exit(0)
		}
		log.Info("Return code from VM start :", returnCode)
	case "stop-vm":
		if len(args[1:]) < 1 {
			log.Info("Invalid number of parameters")
			os.Exit(1)
		}
		log.Info("workload-agent stop called")
		conn, err := net.Dial("unix", rpcSocketFilePath)
		if err != nil {
			log.Fatal("stop-vm: failed to dial wlagent.sock, is wlagent running?")
			os.Exit(1)
		}
		client := rpc.NewClient(conn)
		var returnCode int
		var args = wlrpc.DomainXML{
			XML: args[1],
		}
		client.Call("VirtualMachine.Stop", &args, &returnCode)
		if returnCode == 1 {
			os.Exit(1)
		} else {
			os.Exit(0)
		}
		fmt.Println("Return code from VM stop :", returnCode)

	case "uninstall":
		deleteFile(consts.WLABinFilePath)
		deleteFile(consts.OptDirPath)
		deleteFile(consts.LibvirtHookFilePath)
		deleteFile(consts.ConfigDirPath)
		deleteFile(consts.LogDirPath)
		deleteFile(consts.RunDirPath)

	default:
		fmt.Printf("Unrecognized option : %s\n", arg)
		fallthrough

	case "help", "-help", "--help":
		printUsage()
	}
}

func deleteFile(path string) {
	log.Info("Deleting file: ", path)
	// delete file
	var err = os.RemoveAll(path)
	if err != nil {
		log.Error(err)
	}
}
