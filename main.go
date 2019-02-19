package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	csetup "intel/isecl/lib/common/setup"
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/consts"
	wlrpc "intel/isecl/wlagent/rpc"
	"intel/isecl/wlagent/setup"
	"intel/isecl/wlagent/wlavm"
	"os"
	"strings"
)

var (
	Version string = ""
	Time    string = ""
	Branch  string = ""
)

func printVersion() {
	fmt.Printf("Version %s\nBuild : %s at %s\n", Version, Branch, Time)
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
		
	case "start":
		if len(args[1:]) < 1 {
			log.Info("Invalid number of parameters")
			os.Exit(1)
		}

		if startState := wlavm.Start(args[1]); !startState {
			os.Exit(1)
		}
		os.Exit(0)

	case "stop":
		if len(args[1:]) < 1 {
			log.Info("Invalid number of parameters")
			os.Exit(1)
		}

		if stopState := wlavm.Stop(args[1]); !stopState {
			os.Exit(1)
		}
		os.Exit(0)

	case "uninstall":
		deleteFile(consts.WlagentSymLink)
		deleteFile(consts.OptDirPath)
		deleteFile(consts.LibvirtHookFilePath)
		deleteFile(consts.LogDirPath)
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
