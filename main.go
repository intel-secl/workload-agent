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

type state bool

const (
	Stopped state = false
	Running state = true
)

func readPidFile() (int, error) {
	pidData, err := ioutil.ReadFile(pidFilePath)
	if err != nil {
		log.WithError(err).Debug("Failed to read wlagent.pid")
		return 0, err
	}
	pid, err := strconv.Atoi(string(pidData))
	if err != nil {
		log.WithError(err).WithField("pid", pidData).Debug("Failed to convert pid data string to int")
		return 0, err
	}
	return pid, nil
}

func status() state {
	pid, err := readPidFile()
	if err != nil {
		// failure reading pid file
		os.Remove(pidFilePath)
		return Stopped
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return Stopped
	}
	if err := p.Signal(syscall.Signal(0)); err != nil {
		return Stopped
	}
	return Running
}

func start() {
	if status() == Stopped {
		// exec wlagentd
		cmd := exec.Command(consts.BinDirPath + consts.DaemonFileName)
		err := cmd.Start()
		if err != nil {
			log.WithError(err).Fatal("Failed to start wlagentd")
		}
		file, err := os.Create(pidFilePath)
		if err != nil {
			log.WithError(err).Fatal("Failed to create wlagentd pid file")
		}
		file.WriteString(strconv.Itoa(cmd.Process.Pid))
		cmd.Process.Release()
	} else {
		fmt.Println("Workload Agent is already running")
	}
}

func stop() {
	if status() == Running {
		pid, err := readPidFile()
		if err != nil {
			log.WithError(err).Error("Could not read PID file")
			fmt.Println("Failed to stop Workload Agent")
			return
		}
		if err := syscall.Kill(pid, syscall.SIGQUIT); err != nil {
			log.WithError(err).Error("Failed to kill Workload Agent with signal SIGQUIT")
			fmt.Println("Failed to stop Workload Agent")
			return
		}
		fmt.Println("Workload Agent stopped")
	} else {
		fmt.Println("Workload Agent is already stopped")
	}
}
