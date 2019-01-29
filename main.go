package main

import (
	"fmt"
	csetup "intel/isecl/lib/common/setup"
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/config"
	wlrpc "intel/isecl/wlagent/rpc"
	"intel/isecl/wlagent/setup"
	"io/ioutil"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
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
		start()
	case "stop":
		stop()

	case "start-vm":
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
		conn, err := net.Dial("unix", config.RPCSocketFilePath)
		if err != nil {
			log.Fatal("start-vm: failed to dial wlagent.sock, is wlagent running?")
		}
		client := rpc.NewClient(conn)
		var returnCode int
		var args = wlrpc.StartVMArgs{
			InstanceUUID: args[1],
			ImageUUID:    args[2],
			ImagePath:    args[3],
			InstancePath: args[4],
			DiskSize:     args[5],
		}
		client.Call("VirtualMachine.Start", &args, &returnCode)
		if returnCode == 1 {
			os.Exit(1)
		} else {
			os.Exit(0)
		}
		fmt.Println("Return code from VM start :", returnCode)
	case "stop-vm":
		conn, err := net.Dial("unix", config.RPCSocketFilePath)
		if err != nil {
			log.Fatal("stop-vm: failed to dial wlagent.sock, is wlagent running?")
		}
		client := rpc.NewClient(conn)
		var returnCode int
		var args = wlrpc.StopVMArgs{
			InstanceUUID: args[1],
			ImageUUID:    args[2],
			ImagePath:    args[3],
			InstancePath: args[4],
		}
		client.Call("VirtualMachine.Stop", &args, &returnCode)

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

type state bool

const (
	Stopped state = false
	Running state = true
)

func readPidFile() (int, error) {
	pidData, err := ioutil.ReadFile(config.PIDFilePath)
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
		os.Remove(config.PIDFilePath)
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
		cmd := exec.Command(config.DaemonFilePath)
		err := cmd.Start()
		if err != nil {
			log.WithError(err).Fatal("Failed to start wlagentd")
		}
		file, err := os.Create(config.PIDFilePath)
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
			log.WithError(err).Error("Failed to kill server with signal SIGQUIT")
			fmt.Println("Failed to stop Workload Agent")
			return
		}
		fmt.Println("Workloa Agent stopped")
	} else {
		fmt.Println("Workload Agent is already stopped")
	}
}
