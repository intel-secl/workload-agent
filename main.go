/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package main

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"intel/isecl/lib/common/v3/exec"
	cLog "intel/isecl/lib/common/v3/log"
	"intel/isecl/lib/common/v3/log/message"
	"intel/isecl/lib/common/v3/proc"
	csetup "intel/isecl/lib/common/v3/setup"
	"intel/isecl/lib/common/v3/validation"
	"intel/isecl/lib/tpmprovider/v3"
	"intel/isecl/wlagent/v3/config"
	"intel/isecl/wlagent/v3/consts"
	"intel/isecl/wlagent/v3/filewatch"
	"intel/isecl/wlagent/v3/flavor"
	wlrpc "intel/isecl/wlagent/v3/rpc"
	"intel/isecl/wlagent/v3/setup"
	"intel/isecl/wlagent/v3/util"
	"net"
	"net/rpc"
	"os"
	"strings"
	"time"
)

var (
	Version           = ""
	Time              = ""
	Branch            = ""
	rpcSocketFilePath = consts.RunDirPath + consts.RPCSocketFileName
	log, secLog       *logrus.Entry
)

func init() {
	config.LogConfiguration(config.Configuration.LogEnableStdout)
	log = cLog.GetDefaultLogger()
	secLog = cLog.GetSecurityLogger()
}

func printVersion() {
	fmt.Printf("Version %s\nBuild %s at %s\n", Version, Branch, Time)
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Printf("    %s <command> [arguments]\n\n", os.Args[0])
	fmt.Printf("Available Commands:\n")
	fmt.Printf("    help|-help|--help      Show this help message\n")
	fmt.Printf("    -v|--version           Print version/build information\n")
	fmt.Printf("    start                  Start wlagent\n")
	fmt.Printf("    stop                   Stop wlagent\n")
	fmt.Printf("    status                 Reports the status of wlagent service\n")
	fmt.Printf("    fetch-key-url <keyUrl>      Fetch a key from the keyUrl\n")
	fmt.Printf("    uninstall  [--purge]   Uninstall wlagent. --purge option needs to be applied to remove configuration and secureoverlay2 data files\n")
	fmt.Printf("    setup [task]           Run setup task\n")
	fmt.Printf("Available Tasks for setup:\n")
	fmt.Printf("    download_ca_cert       Download CMS root CA certificate\n")
	fmt.Printf("\t\t                           - Option [--force] overwrites any existing files, and always downloads new root CA cert\n")
	fmt.Printf("                           - Environment variable CMS_BASE_URL=<url> for CMS API url\n")
	fmt.Printf("                           - Environment variable CMS_TLS_CERT_SHA384=<CMS TLS cert sha384 hash> to ensure that WLS is talking to the right CMS instance\n")
	fmt.Printf("    SigningKey             Generate a TPM signing key\n")
	fmt.Printf("\t\t                           - Option [--force] overwrites any existing files, and always creates a new Signing key\n")
	fmt.Printf("    BindingKey             Generate a TPM binding key\n")
	fmt.Printf("\t\t                           - Option [--force] overwrites any existing files, and always creates a new Binding key\n")
	fmt.Printf("    RegisterSigningKey     Register a signing key with the host verification service\n")
	fmt.Printf("\t\t                           - Option [--force] Always registers the Signing key with Verification service\n")
	fmt.Printf("                           - Environment variable HVS_URL=<url> for registering the key with Verification service\n")
	fmt.Printf("                           - Environment variable BEARER_TOKEN=<token> for authenticating with Verification service\n")
	fmt.Printf("    RegisterBindingKey     Register a binding key with the host verification service\n")
	fmt.Printf("\t\t                           - Option [--force] Always registers the Binding key with Verification service\n")
	fmt.Printf("                           - Environment variable HVS_URL=<url> for registering the key with Verification service\n")
	fmt.Printf("                           - Environment variable BEARER_TOKEN=<token> for authenticating with Verification service\n")
	fmt.Printf("                           - Environment variable TRUSTAGENT_USERNAME=<TA user> for changing binding key file ownership to TA application user\n")
}

// main is the primary control loop for wlagent. support setup, vmstart, vmstop etc
func main() {

	log.Trace("main:main() Entering")
	defer log.Trace("main:main() Leaving")
	var context csetup.Context
	inputValArr := []string{os.Args[0]}
	if valErr := validation.ValidateStrings(inputValArr); valErr != nil {
		fmt.Fprintln(os.Stderr, "Invalid string format")
		os.Exit(1)
	}

	args := os.Args[1:]
	if len(args) <= 0 {
		fmt.Println("Command not found. Usage below")
		secLog.Errorf("Command not found, %s", message.InvalidInputProtocolViolation)
		printUsage()
		return
	}
	switch arg := strings.ToLower(args[0]); arg {
	case "--version", "-v":
		config.LogConfiguration(false)
		printVersion()

	case "setup":
		// Everytime, we run setup, need to make sure that the configuration is complete
		// So lets run the Configurer as a separate runner. We could have made a single runner
		// with the first task as the Configurer. However, the logic in the common setup task
		// runner runs only the tasks passed in the argument if there are 1 or more tasks.
		// This means that with current logic, if there are no specific tasks passed in the
		// argument, we will only run the Configurer but the intention was to run all of them

		// TODO : The right way to address this is to pass the arguments from the commandline
		// to a function in the workload agent setup package and have it build a slice of tasks
		// to run.
		config.LogConfiguration(false)
		err := config.SaveConfiguration(context)
		if err != nil {
			fmt.Fprintln(os.Stderr, "main:main() Unable to save configuration in config.yml ")
			log.WithError(err).Error("main:main() Unable to save configuration in config.yml")
			os.Exit(1)
		}

		flags := args
		if len(args) > 1 {
			flags = args[2:]
		} else {
			fmt.Fprintln(os.Stderr, "Error: setup task not mentioned")
			printUsage()
			os.Exit(1)
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

		secLog.Infof("%s, Opening tpm connection", message.SU)
		// Workaround for tpm2-abrmd bug in RHEL 7.5
		tpmFactory, err := tpmprovider.NewTpmFactory()
		if err != nil {
			fmt.Println("Error while creating the tpm factory.")
			os.Exit(1)
		}

		t, err := tpmFactory.NewTpmProvider()
		if err != nil {
			fmt.Println("Error while opening a connection to TPM.")
			os.Exit(1)
		}
		defer t.Close()

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
					T:     t,
					Flags: flags,
				},
				setup.BindingKey{
					T:     t,
					Flags: flags,
				},
				setup.RegisterBindingKey{
					Flags: flags,
				},
				setup.RegisterSigningKey{
					Flags: flags,
				},
			},
			AskInput: false,
		}
		tasklist := []string{}
		if args[1] != "all" {
			tasklist = args[1:]
		}
		config.LogConfiguration(false)
		err = setupRunner.RunTasks(tasklist...)
		if err != nil {
			log.WithError(err).Error("main:main() Error running setup")
			log.Tracef("%+v", err)
			fmt.Fprintf(os.Stderr, "Error running setup tasks...\n")
			t.Close()
			os.Exit(1)
		}

	case "runservice":
		runservice()

	case "start":
		start()

	case "stop":
		stop()

	case "status":
		fmt.Println("Workload Agent Status")
		stdout, stderr, _ := exec.RunCommandWithTimeout(consts.ServiceStatusCmd, 2)

		// When stopped, 'systemctl status wlagent' will return '3' and print
		// the status message to stdout.  Other errors (ex 'systemctl status xyz') will return
		// an error code (ex. 4) and write to stderr. Always print stdout and print
		// stderr if present.
		fmt.Println(stdout)
		if stderr != "" {
			fmt.Println(stderr)
		}

	case "start-vm":
		if len(args[1:]) < 1 {
			log.Errorf("main:main() start-vm: Invalid number of parameters %s", message.InvalidInputProtocolViolation)
			os.Exit(1)
		}

		secLog.Info("main:main() start-vm: wlagent start-vm called")
		conn, err := net.Dial("unix", rpcSocketFilePath)
		if err != nil {
			secLog.Errorf("main:main() start-vm: Failed to dial wlagent.sock, %s", message.BadConnection)
			os.Exit(1)
		}
		client := rpc.NewClient(conn)
		defer client.Close()

		// validate domainXML input
		if err = validation.ValidateXMLString(args[1]); err != nil {
			secLog.Errorf("main:main() start-vm: %s, Invalid domain XML format", message.InvalidInputBadParam)
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
			secLog.Errorf("main:main() stop-vm: Invalid number of parameters, %s", message.InvalidInputProtocolViolation)
			os.Exit(1)
		}
		secLog.Info("main/main() stop-vm: wlagent stop-vm called")
		conn, err := net.Dial("unix", rpcSocketFilePath)
		if err != nil {
			secLog.Errorf("main:main() stop-vm: Failed to dial wlagent.sock, %s", message.BadConnection)
			os.Exit(1)
		}

		// validate domainXML input
		if err = validation.ValidateXMLString(args[1]); err != nil {
			secLog.Errorf("main:main() stop-vm: %s, Invalid domain XML format", message.InvalidInputBadParam)
			os.Exit(1)
		}

		client := rpc.NewClient(conn)
		defer client.Close()
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
			secLog.Infof("main:main() create-instance-trust-report, Invalid number of parameters, %s", message.InvalidInputProtocolViolation)
			os.Exit(1)
		}
		secLog.Info("main:main()  wlagent create-instance-trust-report called")
		conn, err := net.Dial("unix", rpcSocketFilePath)
		if err != nil {
			log.WithError(err).Errorf("main:main() create-instance-trust-report: failed to dial wlagent.sock, %s", message.BadConnection)
			os.Exit(1)
		}
		client := rpc.NewClient(conn)
		defer client.Close()
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
			secLog.Errorf("main:main() fetch-flavor: Invalid number of parameters, %s", message.InvalidInputProtocolViolation)
			os.Exit(1)
		}

		conn, err := net.Dial("unix", rpcSocketFilePath)
		if err != nil {
			secLog.WithError(err).Errorf("main:main() fetch-flavor: failed to dial wlagent.sock, %s", message.BadConnection)
			os.Exit(1)
		}

		// validate input
		if err = validation.ValidateUUIDv4(args[1]); err != nil {
			secLog.Errorf("main:main() fetch-flavor: %s, Invalid Image UUID format", message.InvalidInputBadParam)
			os.Exit(1)
		}

		client := rpc.NewClient(conn)
		defer client.Close()
		var outFlavor flavor.OutFlavor
		var args = wlrpc.FlavorInfo{
			ImageID: args[1],
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

	case "fetch-key-url":
		if len(args[1:]) < 1 {
			secLog.Errorf("main:main() fetch-key-url: Invalid number of parameters, %s", message.InvalidInputProtocolViolation)
			os.Exit(1)
		}

		conn, err := net.Dial("unix", rpcSocketFilePath)
		if err != nil {
			secLog.WithError(err).Errorf("main:main() fetch-key-url: failed to dial wlagent.sock, %s", message.BadConnection)
			os.Exit(1)
		}

		client := rpc.NewClient(conn)
		defer client.Close()
		var keyOut wlrpc.KeyOnly
		var args = wlrpc.TransferURL{
			URL: args[1],
		}

		err = client.Call("VirtualMachine.FetchKeyWithURL", &args, &keyOut)
		if err != nil {
			log.Error("main:main() fetch-key-url: Client call failed")
			log.Tracef("%+v", err)
			os.Exit(1)
		}

		retKey, err := json.Marshal(keyOut)
		if err != nil {
			log.Error("main:main() fetch-key-url while marshalling key")
		}
		fmt.Println(string(retKey))
		os.Exit(0)

	case "uninstall":
		config.LogConfiguration(false)

		_, err := os.Stat(consts.OptDirPath + "secure-docker-daemon")
		if err == nil {
			removeSecureDockerDaemon()

			// restart docker daemon
			commandArgs := []string{"start", "docker"}
			_, err = exec.ExecuteCommand("systemctl", commandArgs)
			if err != nil {
				fmt.Print("Error starting docker daemon post-uninstall. Refer dockerd logs for more information.")
			}
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
		config.LogConfiguration(false)
		fmt.Printf("Unrecognized option : %s\n", arg)
		secLog.Errorf("%s Command not found", message.InvalidInputProtocolViolation)
		fallthrough

	case "help", "-help", "--help":
		config.LogConfiguration(false)
		printUsage()
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
		fmt.Fprintf(os.Stderr, "Error deleting file :%s", path)
	}
}

func start() {
	log.Trace("main:start() Entering")
	defer log.Trace("main:start() Leaving")

	fmt.Fprintln(os.Stdout, `Forwarding to "systemctl start wlagent"`)
	_, _, err := exec.RunCommandWithTimeout(consts.ServiceStartCmd, 5)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not start Workload Agent Service")
		fmt.Fprintln(os.Stderr, "Error : ", err)
		os.Exit(1)
	}
}

func stop() {
	log.Trace("main:stop() Entering")
	defer log.Trace("main:stop() Leaving")
	fmt.Fprintln(os.Stdout, `Forwarding to "systemctl stop wlagent"`)

	_, _, err := exec.RunCommandWithTimeout(consts.ServiceStopCmd, 12)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not stop Workload Agent Service")
		fmt.Fprintln(os.Stderr, "Error : ", err)
		os.Exit(1)
	}
	util.CloseTpmInstance()
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
	secLog.Info("Service Removed")
}

func runservice() {
	log.Trace("main:runservice() Entering")
	defer log.Trace("main:runservice() Leaving")
	// Save log configurations
	//TODO : daemon log configuration - does it need to be passed in?

	//check if the wlagent run directory path is already created
	if _, err := os.Stat(consts.RunDirPath); os.IsNotExist(err) {
		if err := os.MkdirAll(consts.RunDirPath, 0600); err != nil {
			log.WithError(err).Fatalf("main:runservice() could not create directory: %s, err: %s", consts.RunDirPath, err)
		}
	}

	loadIVAMapErr := util.LoadImageVMAssociation()
	if loadIVAMapErr != nil {
		log.WithError(loadIVAMapErr).Fatal("main:runservice() error loading ImageVMAssociation map")
	}

	// open a connection to TPM
	_, err := util.GetTpmInstance()
	if err != nil {
		log.WithError(err).Error("main:runservice() Could not open a new connection to the TPM")
		secLog.Info(message.AppRuntimeErr)
		os.Exit(1)
	}

	fileWatcher, err := filewatch.NewWatcher()
	if err != nil {
		log.WithError(err).Error("main:runservice() Could not create File Watcher")
		secLog.Info(message.AppRuntimeErr)
		os.Exit(1)
	}
	defer fileWatcher.Close()
	// Passing the false parameter to ensure that fileWatcher task is not added to the wait group if there is pending signal termination
	_, err = proc.AddTask(false)
	if err != nil {
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
	if err != nil {
		log.WithError(err).Fatal("main:runservice() could not add the task for rpc service")
	}
	go func() {
		defer proc.TaskDone()
		for {
			RPCSocketFilePath := consts.RunDirPath + consts.RPCSocketFileName
			// When the socket is closed, the file handle on the socket file isn't handled.
			// This code is added to manually remove any stale socket file before the connection
			// is reopened; prevent error: bind address already in use
			// ensure that the socket file exists before removal
			if _, err = os.Stat(RPCSocketFilePath); err == nil {
				err = os.Remove(RPCSocketFilePath)
				if err != nil {
					log.WithError(err).Error("main:runservice() Failed to remove socket file")
				}
			}
			// block and loop, daemon doesnt need to run on go routine
			l, err := net.Listen("unix", RPCSocketFilePath)
			if err != nil {
				log.Error(err)
				secLog.Info(message.AppRuntimeErr)
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
				secLog.Info(message.AppRuntimeErr)
				return
			}
			r.Accept(l)
		}
	}()
	secLog.Info(message.ServiceStart)

	// block until stop channel receives
	err = proc.WaitForQuitAndCleanup(10 * time.Second)
	if err != nil {
		log.WithError(err).Error("main:runservice() Error while clean up")
	}
	secLog.Info(message.ServiceStop)
}
