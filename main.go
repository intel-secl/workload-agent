package main

import (
	"fmt"
	csetup "intel/isecl/lib/common/setup"
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/pkg"
	"intel/isecl/wlagent/setup"
	"log"
	"os"
	"strconv"
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
		// Save configurations that are provided during setup to config yaml file
		err := config.SaveSetupConfiguration()
		if err != nil {
			log.Fatal("Failed to save setup configurations.")
		}

		// Save log rotation configurations
		config.LogConfiguration()

		// Check if nosetup environment variable is true, if yes then skip the setup tasks
		if nosetup, err := strconv.ParseBool(os.Getenv("WORKLOAD_AGENT_NOSETUP")); err != nil && nosetup == false {
			// Workaround for tpm2-abrmd bug in RHEL 7.5
			t, err := tpm.Open()
			if err != nil {
				log.Fatal("Error while opening a connection to TPM.")
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
			os.Exit(1)
		}

	case "start":
		if len(args[1:]) < 5 {
			fmt.Println("Invalid number of parameters")
		}
		// log to logrus
		// fmt.Println("VM start called in main method")
		// fmt.Println("image path: ", args[3])
		// fmt.Println("image UUID: ", args[2])
		// fmt.Println("instance path: ", args[4])
		// fmt.Println("instance UUID: ", args[1])
		// fmt.Println("disksize: ", args[5])
		returnCode := wlavm.Start(args[1], args[2], args[3], args[4], args[5])
		if returnCode == 1 {
			os.Exit(1)
		} else {
			os.Exit(0)
		}
		fmt.Println("Return code from VM start :", returnCode)
	case "stop":
		pkg.QemuStopIntercept(strings.TrimSpace(args[1]), strings.TrimSpace(args[2]),
			strings.TrimSpace(args[3]), strings.TrimSpace(args[4]))

	case "uninstall":
		// use constants from config
		deleteFile("/usr/local/bin/wlagent")
		deleteFile("/opt/workloadagent/")
		deleteFile("/etc/libvirt/hooks/qemu")
		deleteFile("/etc/workloadagent/")
		deleteFile("/var/log/workloadagent/")

	default:
		fmt.Printf("Unrecognized option : %s\n", arg)
		fallthrough

	case "help", "-help", "--help":
		printUsage()
	}
}

func deleteFile(path string) {
	log.Println("Deleting file: ", path)
	// delete file
	var err = os.RemoveAll(path)
	if err != nil {
		log.Fatal(err)
	}
}
