/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package main

import (
	"fmt"
	"intel/isecl/lib/clients"
	"intel/isecl/lib/clients/aas"
	"intel/isecl/lib/common/exec"
	cLog "intel/isecl/lib/common/log"
	"intel/isecl/lib/common/proc"
	csetup "intel/isecl/lib/common/setup"
	"intel/isecl/lib/common/validation"
	"intel/isecl/lib/tpm"
	"intel/isecl/wlagent/config"
	"intel/isecl/wlagent/consts"
	"intel/isecl/wlagent/filewatch"
	"intel/isecl/wlagent/flavor"
	wlrpc "intel/isecl/wlagent/rpc"
	"intel/isecl/wlagent/setup"
	"intel/isecl/wlagent/util"
	"net"
	"net/rpc"
	"os"
	"strings"
	"time"
)

var (
	Version           string = ""
	Time              string = ""
	Branch            string = ""
	rpcSocketFilePath string = consts.RunDirPath + consts.RPCSocketFileName
)

var log = cLog.GetDefaultLogger()
var secLog = cLog.GetSecurityLogger()

func printVersion() {
	fmt.Printf("Version %s\nBuild %s at %s\n", Version, Branch, Time)
}

func printUsage() {
	fmt.Printf("Usage:\n\n")
	fmt.Printf("    %s <command> [arguments]\n\n", os.Args[0])
	fmt.Printf("Available Commands:\n")
	fmt.Printf("    help|-help|--help  Show this help message\n")
	fmt.Printf("    setup [task]       Run setup task\n")
	fmt.Printf("    start              Start wlagent\n")
	fmt.Printf("    stop               Stop wlagent\n")
	fmt.Printf("    status             Reports the status of wlagent service\n")
	fmt.Printf("    uninstall          Uninstall wlagent\n")
	fmt.Printf("    uninstall --purge  Uninstalls workload agent and deletes the existing configuration directory\n")
	fmt.Printf("    version            Reports the version of the workload agent\n\n")
	fmt.Printf("Available Tasks for setup:\n")
	fmt.Printf("    SigningKey         Generate a TPM signing key\n")
	fmt.Printf("    BindingKey         Generate a TPM binding key\n")
	fmt.Printf("    RegisterSigningKey Register a signing key with the host verification service\n")
        fmt.Printf("                        - Environment variable BEARER_TOKEN=<token> for authenticating with Verification service\n")
	fmt.Printf("    RegisterBindingKey Register a binding key with the host verification service\n")
	fmt.Printf("                        - Environment variable BEARER_TOKEN=<token> for authenticating with Verification service\n")
}

// main is the primary control loop for wlagent. support setup, vmstart, vmstop etc
func main() {

	config.LogConfiguration(false, true, false)

	log.Trace("main:main() Entering")
	defer log.Trace("main:main() Leaving")
	// Save log configurations
	var context csetup.Context
	inputValArr := []string{os.Args[0]}
	if valErr := validation.ValidateStrings(inputValArr); valErr != nil {
		fmt.Fprintln(os.Stderr, "Invalid string format")
		os.Exit(1)
	}

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

		err := config.SaveConfiguration(context)
		if err != nil {
			fmt.Fprintln(os.Stderr, "main:main() Unable to save configuration in config.yml ")
			log.WithError(err).Error("main:main() Unable to save configuration in config.yml")
			os.Exit(1)
		}
		config.LogConfiguration(false, true, false)
		// Workaround for tpm2-abrmd bug in RHEL 7.5
		t, err := tpm.Open()
		if err != nil {
			secLog.WithError(err).Error("main:main() Error while opening a connection to TPM.")
			os.Exit(1)
		}

		flags := args
		if len(args) > 1 {
			flags = args[2:]
		}
		
		if len(args) >= 2 &&
                        args[1] != "download_ca_cert" &&
                        args[1] != "SigningKey" &&
                        args[1] != "BindingKey" &&
                        args[1] != "RegisterSigningKey" &&
                        args[1] != "RegisterBindingKey" &&
                        args[1] != "all" {
                        fmt.Fprintln(os.Stderr, "Error: Unknown setup task ", args[1])
                        printUsage()
                        os.Exit(1)
                }

		// Run list of setup tasks one by one
		setupRunner := &csetup.Runner{
			Tasks: []csetup.Task{
				csetup.Download_Ca_Cert{
					Flags:                flags,
					CmsBaseURL:           config.Configuration.Cms.BaseURL,
					CaCertDirPath:        consts.TrustedCaCertsDir,
					TrustedTlsCertDigest: config.Configuration.CmsTlsCertDigest,
					ConsoleWriter:        os.Stdout,
				},
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
			log.WithError(err).Error("main:main() Error running setup")
			log.Tracef("%+v", err)
			fmt.Fprintf(os.Stderr, "Error running setup tasks. %s\n", err.Error())
			os.Exit(1)
		}

	case "runservice":
		config.LogConfiguration(false, true, true)
		runservice()

	case "start":
		start()

	case "stop":
		stop()

	case "status":
		fmt.Println("Workload Agent Status")
		stdout, stderr, _ := exec.RunCommandWithTimeout(consts.ServiceStatusCmd, 2)

		// When stopped, 'systemctl status workload-agent' will return '3' and print
		// the status message to stdout.  Other errors (ex 'systemctl status xyz') will return
		// an error code (ex. 4) and write to stderr.  Alwyas print stdout and print
		// stderr if present.
		fmt.Println(stdout)
		if stderr != "" {
			fmt.Println(stderr)
		}

	case "start-vm":
		if len(args[1:]) < 1 {
			log.Error("main:main() start-vm: Invalid number of parameters")
			os.Exit(1)
		}

		log.Info("main:main() start-vm: workload-agent start called")
		conn, err := net.Dial("unix", rpcSocketFilePath)
		if err != nil {
			log.Error("main:main() start-vm: Failed to dial wlagent.sock, is wlagent running?")
			os.Exit(1)
		}
		client := rpc.NewClient(conn)

		// validate domainXML input
		if err = validation.ValidateXMLString(args[1]); err != nil {
			secLog.Error("main:main() start-vm: Invalid domain XML format")
			os.Exit(1)
		}

		var args = wlrpc.DomainXML{
			XML: args[1],
		}
		var startState bool
		err = client.Call("VirtualMachine.Start", &args, &startState)
		if err != nil {
			log.Error("main:main() start-vm: Client call failed")
		}

		if !startState {
			os.Exit(1)
		} else {
			os.Exit(0)
		}

	case "stop-vm":
		if len(args[1:]) < 1 {
			log.Error("main:main() stop-vm: Invalid number of parameters")
			os.Exit(1)
		}
		log.Info("main/main() stop-vm: workload-agent stop called")
		conn, err := net.Dial("unix", rpcSocketFilePath)
		if err != nil {
			log.Error("main:main() stop-vm: Failed to dial wlagent.sock, is wlagent running?")
			os.Exit(1)
		}

		// validate domainXML input
		if err = validation.ValidateXMLString(args[1]); err != nil {
			secLog.Error("main:main() stop-vm: Invalid domain XML format")
			os.Exit(1)
		}

		client := rpc.NewClient(conn)
		var args = wlrpc.DomainXML{
			XML: args[1],
		}
		var stopState bool
		err = client.Call("VirtualMachine.Stop", &args, &stopState)
		if err != nil {
			log.Error("main:main() stop-vm: Client call failed")
			os.Exit(1)
		}

		if !stopState {
			os.Exit(1)
		} else {
			os.Exit(0)
		}

	case "create-instance-trust-report":
		if len(args[1:]) < 1 {
			log.Info("main:main() create-instance-trust-report Invalid number of parameters")
			os.Exit(1)
		}
		log.Info("main:main()  workload-agent create-instance-trust-report called")
		conn, err := net.Dial("unix", rpcSocketFilePath)
		if err != nil {
			log.WithError(err).Error("main:main() create-instance-trust-report: failed to dial wlagent.sock, is wlagent running?")
			os.Exit(1)
		}
		client := rpc.NewClient(conn)
		var args = wlrpc.ManifestString{
			Manifest: args[1],
		}
		var status bool
		err = client.Call("VirtualMachine.CreateInstanceTrustReport", &args, &status)
		if err != nil {
			log.WithError(err).Error("main:main() create-instance-trust-report: Error while creating trust report")
			os.Exit(1)
		}
		log.Info("main:main() create-instance-trust-report Successfully created trust report")
		os.Exit(0)

	case "fetch-flavor":
		if len(args[1:]) < 1 {
			log.Error("main:main() fetch-flavor: Invalid number of parameters")
			os.Exit(1)
		}

		conn, err := net.Dial("unix", rpcSocketFilePath)
		if err != nil {
			log.WithError(err).Error("main:main() fetch-flavor: failed to dial wlagent.sock, is wlagent running?")
			os.Exit(1)
		}

		// validate input
		if err = validation.ValidateUUIDv4(args[1]); err != nil {
			log.Error("main:main() fetch-flavor: Invalid imageUUID format")
			os.Exit(1)
		}

		client := rpc.NewClient(conn)
		var outFlavor flavor.OutFlavor
		var args = wlrpc.FlavorInfo{
			ImageID:    args[1],
		}

		err = client.Call("VirtualMachine.FetchFlavor", &args, &outFlavor)
		if err != nil {
			log.Error("main:main() fetch-flavor: Client call failed")
			log.Tracef("%+v", err)
		}
		if !outFlavor.ReturnCode {
			os.Exit(1)
		} else {
			fmt.Print(outFlavor.ImageFlavor)
			os.Exit(0)
		}

	case "uninstall":
		config.LogConfiguration(true, true, false)
		commandArgs := []string{consts.OptDirPath + "secure-docker-daemon"}
		_, err := exec.ExecuteCommand("ls", commandArgs)
		if err == nil {
			removeSecureDockerDaemon()
		}
		stop()
		removeservice()

		deleteFile(consts.WlagentSymLink)
		deleteFile(consts.OptDirPath)
		deleteFile(consts.LibvirtHookFilePath)
		deleteFile(consts.LogDirPath)
		deleteFile(consts.RunDirPath)
                deleteFile(consts.MountPath)
		if len(args) > 1 && strings.ToLower(args[1]) == "--purge" {
			deleteFile(consts.ConfigDirPath)
		}

	default:
		fmt.Printf("Unrecognized option : %s\n", arg)
		fallthrough

	case "help", "-help", "--help":
		printUsage()

	case "test-aas":
		aasClient := aas.NewJWTClient(config.Configuration.Aas.BaseURL)
		fmt.Println(aasClient)

		var err error
		aasClient.HTTPClient, err = clients.HTTPClientWithCADir(consts.TrustedCaCertsDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		aasClient.AddUser(config.Configuration.Wla.APIUsername, config.Configuration.Wla.APIPassword)
		err = aasClient.FetchAllTokens()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		jwtToken, err := aasClient.GetUserToken(config.Configuration.Wla.APIUsername)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		fmt.Println(string(jwtToken))
	}
}

func removeSecureDockerDaemon() {
	log.Trace("main/main:removeSecureDockerDaemon() Entering")
	defer log.Trace("main/main:removeSecureDockerDaemon() Leaving")

	commandArgs := []string{consts.OptDirPath + "secure-docker-daemon/uninstall-container-security-dependencies.sh"}
	_, err := exec.ExecuteCommand("/bin/bash", commandArgs)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func deleteFile(path string) {
	log.Trace("main/main:deleteFile() Entering")
	defer log.Trace("main/main:deleteFile() Leaving")
	fmt.Println("Deleting : ", path)
	// delete file
	var err = os.RemoveAll(path)
	if err != nil {
		log.Error(err)
		log.Tracef("%+v", err)
	}
}

func start() {
	log.Trace("main:start() Entering")
	defer log.Trace("main:start() Leaving")
	cmdOutput, _, err := exec.RunCommandWithTimeout(consts.ServiceStartCmd, 5)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not start Workload Agent Service")
		fmt.Fprintln(os.Stderr, "Error : ", err)
		os.Exit(1)
	}
	fmt.Println(cmdOutput)
	fmt.Println("Workload Agent Service Started...")
	log.Info("Workload Agent Service Started...")
}

func stop() {
	log.Trace("main:stop() Entering")
	defer log.Trace("main:stop() Leaving")

	cmdOutput, _, err := exec.RunCommandWithTimeout(consts.ServiceStopCmd, 12)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not stop Workload Agent Service")
		fmt.Fprintln(os.Stderr, "Error : ", err)
		os.Exit(1)
	}
	fmt.Println(cmdOutput)
	util.CloseTpmInstance()
	fmt.Println("Workload Agent Service Stopped...")
	log.Info("Workload Agent Service Stopped...")
}

func removeservice() {
	log.Trace("main:removeservice() Entering")
	defer log.Trace("main:removeservice() Leaving")

	_, _, err := exec.RunCommandWithTimeout(consts.ServiceRemoveCmd, 12)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not remove Workload Agent Service")
		fmt.Fprintln(os.Stderr, "Error : ", err)
	}
	fmt.Println("Workload Agent Service Removed...")
}

func runservice() {
	log.Trace("main:runservice() Entering")
	defer log.Trace("main:runservice() Leaving")
	// Save log configurations
	//TODO : daemon log configuration - does it need to be passed in?

	// open a connection to TPM
	_, err := util.GetNewTpmInstance()
	if err != nil {
		log.WithError(err).Error("main:runservice() Could not open a new connection to the TPM")
		os.Exit(1)
	}

	fileWatcher, err := filewatch.NewWatcher()
	if err != nil {
		log.WithError(err).Error("main:runservice() Could not create File Watcher")
		os.Exit(1)
	}
	defer fileWatcher.Close()
	// Passing the false parameter to ensure that fileWatcher task is not added to the wait group if there is pending signal termination
	_, err = proc.AddTask(false)
	if err != nil{
		log.WithError(err).Fatal("main:runservice() could not add the task for filewatcher")
	}
	go func() {
		defer proc.TaskDone()
		for {
			fileWatcher.Watch()
		}
	}()

	if _, err = os.Stat(consts.RunDirPath); os.IsNotExist(err) {
		if err := os.MkdirAll(consts.RunDirPath, 0600); err != nil {
			log.WithError(err).Fatalf("main:runservice() Could not create directory: %s, err: %s", consts.RunDirPath, err)
		}
	}

	// Passing the false parameter to ensure that fileWatcher task is not added to the wait group if there is pending signal termination
	_, err = proc.AddTask(false)
	if err != nil{
                log.WithError(err).Fatal("main:runservice() could not add the task for rpc service")
        }
	go func() {
		defer proc.TaskDone()
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
				Watcher: fileWatcher,
			}

			err = r.Register(vm)
			if err != nil {
				log.WithError(err).Error("main:runservice() Unable to Register vm wathcer")
				log.Tracef("%+v", err)
				return
			}
			r.Accept(l)
		}
	}()
	// block until stop channel receives
	proc.WaitForQuitAndCleanup(10 * time.Second)
}
