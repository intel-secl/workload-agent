package main

import (
	"fmt"
	csetup "intel/isecl/lib/common/setup"
	"log"
	"os"
	"strings"
	"time"
)

var (
	component string = "workload-agent"
	version   string = ""
	buildid   string = ""
	buildtype string = "dev"
)

func printVersion() {
	if version == "" {
		fmt.Printf("Version Infromation not set\n")
		fmt.Printf("Have to be set at build time using -ldflags -X options\n")
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
	fmt.Printf("\tsetup|vmstart|vmstop|--help|--version\n\n")
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
		setupRunner := &csetup.Runner{
			Tasks:    []csetup.Task{},
			AskInput: false,
		}
		err := setupRunner.RunTasks(args[1:]...)
		if err != nil {
			log.Fatal(err)
		}

	case "vmstart":

	case "vmstop":

	default:
		fmt.Printf("Unrecognized option : %s\n", arg)
		fallthrough

	case "help", "-help", "--help":
		printUsage()
	}
}
