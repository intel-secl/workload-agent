package main

import (
	"fmt"
	"intel/isecl/lib/common/exec"
	csetup "intel/isecl/lib/common/setup"
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/consts"
	wlrpc "intel/isecl/wlagent/rpc"
	"intel/isecl/wlagent/setup"
	"intel/isecl/wlagent/filewatch"
	"net"
	"net/rpc"

	log "github.com/sirupsen/logrus"

	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

var (
	Version string = ""
	Time    string = ""
	Branch  string = ""
	rpcSocketFilePath string = consts.RunDirPath + consts.RPCSocketFileName
)

func printVersion() {
	fmt.Printf("Version %s\nBuild %s at %s\n", Version, Branch, Time)
}

func printUsage() {
	fmt.Printf("Work Load Agent\n")
	fmt.Printf("===============\n\n")
	fmt.Printf("usage : %s <command> [<args>]\n\n", os.Args[0])
	fmt.Printf("Following are the list of commands\n")
	fmt.Printf("\tsetup|start|stop|status|uninstall [--purge]|--help|--version\n\n")
	fmt.Printf("\tusage : %s setup [<tasklist>]\n", os.Args[0])
	fmt.Printf("\t\t<tasklist>-space seperated list of tasks\n")
	fmt.Printf("\t\t\t-Supported tasks - SigningKey BindingKey RegisterSigningKey RegisterBindingKey\n")
	fmt.Printf("\tExample :-\n")
	fmt.Printf("\t\t%s setup\n", os.Args[0])
	fmt.Printf("\t\t%s setup SigningKey\n", os.Args[0])
}

// main is the primary control loop for wlagent. support setup, vmstart, vmstop etc
func main() {
	// Save log configurations
	config.LogConfiguration()

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
		// Everytime, we run setup, need to make sure that the configuration is complete
		// So lets run the Configurer as a seperate runner. We could have made a single runner
		// with the first task as the Configurer. However, the logic in the common setup task
		// runner runs only the tasks passed in the argument if there are 1 or more tasks.
		// This means that with current logic, if there are no specific tasks passed in the
		// argument, we will only run the confugurer but the intention was to run all of them
		
		// TODO : The right way to address this is to pass the arguments from the commandline
		// to a functon in the workload agent setup package and have it build a slice of tasks
		// to run.
		installRunner := &csetup.Runner{
			Tasks: []csetup.Task{
				setup.Configurer{},
			},
			AskInput: false,
		}
		err := installRunner.RunTasks("Configurer")
		if err != nil {
			fmt.Println("Error running setup: ", err)
			os.Exit(1)
		}

		// Workaround for tpm2-abrmd bug in RHEL 7.5
		t, err := tpm.Open()
		if err != nil {
			fmt.Println("Error while opening a connection to TPM.")
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

	case "runservice":
		runservice()

	case "start":
		start()

	case "stop":
		stop()

	case "status":
		if cmdOutput, _, err := exec.RunCommandWithTimeout(consts.ServiceStatusCmd, 2); err == nil {
			fmt.Println("Workload Agent Status")
			fmt.Println(cmdOutput)
		}

	case "start-vm":
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
		var args = wlrpc.DomainXML{
			XML: args[1],
		}
		var startState bool
		err = client.Call("VirtualMachine.Start", &args, &startState)
		if err != nil {
			log.Error("client call failed")
		}

		if !startState {
			os.Exit(1)
		}
		os.Exit(0)

	case "stop":
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
		var args = wlrpc.DomainXML{
			XML: args[1],
		}
		var stopState bool
		err = client.Call("VirtualMachine.Stop", &args, &stopState)
		if err != nil {
			log.Error("client call failed")
		}

		if stopState := wlavm.Stop(args[1]); !stopState {
			os.Exit(1)
		}
		os.Exit(0)
		fmt.Println("Return code from VM stop :", returnCode)

	case "uninstall":
		stop()
		removeservice()

		deleteFile(consts.WlagentSymLink)
		deleteFile(consts.OptDirPath)
		deleteFile(consts.LibvirtHookFilePath)
		deleteFile(consts.LogDirPath)
		deleteFile(consts.RunDirPath)
		if len(args) > 1 && strings.ToLower(args[1]) == "--purge" {
			deleteFile(consts.ConfigDirPath)
		}

	default:
		fmt.Printf("Unrecognized option : %s\n", arg)
		fallthrough

	case "help", "-help", "--help":
		printUsage()
	}
}

func deleteFile(path string) {
	log.Info("Deleting : ", path)
	// delete file
	var err = os.RemoveAll(path)
	if err != nil {
		log.Error(err)
	}
}


func start() {
	cmdOutput, _, err := exec.RunCommandWithTimeout(consts.ServiceStartCmd, 5)
	if err != nil {
		fmt.Println("Could not start Workload Agent Service")
		fmt.Println("Error : ", err)
		os.Exit(1)
	}
	fmt.Println(cmdOutput)
	fmt.Println("Workload Agent Service Started...")
}

func stop() {
	cmdOutput, _, err := exec.RunCommandWithTimeout(consts.ServiceStopCmd, 5)
	if err != nil {
		fmt.Println("Could not stop Workload Agent Service")
		fmt.Println("Error : ", err)
		os.Exit(1)
	}
	fmt.Println(cmdOutput)
	fmt.Println("Workload Agent Service Stopped...")
}

func removeservice() {
	_, _, err := exec.RunCommandWithTimeout(consts.ServiceRemoveCmd, 5)
	if err != nil {
		fmt.Println("Could not remove Workload Agent Service")
		fmt.Println("Error : ", err)
	}
	fmt.Println("Workload Agent Service Removed...")
}

func runservice() {
	// Save log configurations
	//TODO : daemon log configuration - does it need to be passed in?
	config.LogConfiguration(consts.LogDirPath + consts.DaemonLogFileName)

	fileWatcher, err := filewatch.NewWatcher()
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
	// stop signaler
	stop := make(chan bool)
	defer fileWatcher.Close()
	go func() {
		for {
			fileWatcher.Watch()
		}
	}()
    if _, err := os.Stat(consts.RunDirPath); os.IsNotExist(err) {
		if err := os.MkdirAll(consts.RunDirPath, 0600); err != nil {
			log.WithError(err).Fatalf("Could not create directory: %s, err: %s", consts.RunDirPath, err)
		}
	}
	go func() {
		for {
			RPCSocketFilePath := consts.RunDirPath + consts.RPCSocketFileName
			// When the socket is closed, the file handle on the socket file isn't handled.
			// This code is added to manually remove any stale socket file before the connection
			// is reopened; prevent error: bind address already in use
			os.Remove(RPCSocketFilePath)
			// block and loop, daemon doesnt need to run on go routine
			l, err := net.Listen("unix", RPCSocketFilePath)
			if err != nil {
				log.Error(err)
				return
			}
			r := rpc.NewServer()
				vm := &wlrpc.VirtualMachine{
				Watcher : fileWatcher,
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
