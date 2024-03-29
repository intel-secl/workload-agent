/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package main

import (
	"fmt"
	keyproviderpb "github.com/containers/ocicrypt/utils/keyprovider"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"intel/isecl/lib/common/v4/exec"
	cLog "intel/isecl/lib/common/v4/log"
	"intel/isecl/lib/common/v4/log/message"
	"intel/isecl/lib/common/v4/proc"
	csetup "intel/isecl/lib/common/v4/setup"
	"intel/isecl/lib/common/v4/validation"
	"intel/isecl/lib/tpmprovider/v4"
	"intel/isecl/wlagent/v4/config"
	"intel/isecl/wlagent/v4/consts"
	"intel/isecl/wlagent/v4/filewatch"
	kpgrpc "intel/isecl/wlagent/v4/keyprovider-grpc"
	wlrpc "intel/isecl/wlagent/v4/rpc"
	"intel/isecl/wlagent/v4/setup"
	"intel/isecl/wlagent/v4/util"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	// Version holds the version number for the WLA binary
	Version string = ""
	// BuildDate holds the build date for the WLA binary
	BuildDate string = ""
	// GitHash holds the commit hash for the WLA binary
	GitHash           = ""
	rpcSocketFilePath = consts.RunDirPath + consts.RPCSocketFileName
	log, secLog       *logrus.Entry
)

func init() {
	log = cLog.GetDefaultLogger()
	secLog = cLog.GetSecurityLogger()
}

func getVersion() string {
	verStr := fmt.Sprintf("Service Name: %s\n", consts.ExplicitServiceName)
	verStr = verStr + fmt.Sprintf("Version: %s-%s\n", Version, GitHash)
	verStr = verStr + fmt.Sprintf("Build Date: %s\n", BuildDate)
	return verStr
}

func printVersion() {
	fmt.Printf(getVersion())
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
	fmt.Printf("    update_service_config  Updates service configuration\n")
	fmt.Printf("\t\t                           - Option [--force] overwrites existing server config")
	fmt.Printf("                           - Environment variable WLS_API_URL=<url> Workload Service URL\n")
	fmt.Printf("                           - Environment variable WLA_SERVICE_USERNAME WLA Service Username\n")
	fmt.Printf("                           - Environment variable WLA_SERVICE_PASSWORD WLA Service Password\n")
	fmt.Printf("                           - Environment variable SKIP_FLAVOR_SIGNATURE_VERIFICATION=<true/false> Skip flavor signature verification if set to true\n")
	fmt.Printf("                           - Environment variable LOG_ENTRY_MAXLENGTH=Maximum length of each entry in a log\n")
	fmt.Printf("                           - Environment variable WLA_ENABLE_CONSOLE_LOG=<true/false> Workload Agent Enable standard output\n")
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
		config.LogConfiguration(config.Configuration.LogEnableStdout)
		var flags []string
		if len(args) > 1 {
			flags = args[2:]
		} else {
			fmt.Fprintln(os.Stderr, "Error: setup task not mentioned")
			printUsage()
			os.Exit(1)
		}

		taskName := args[1]

		switch taskName {
		case consts.SetupAllCommand, consts.DownloadRootCACertCommand,
			consts.RegisterSigningKeyCommand, consts.RegisterBindingKeyCommand,
			consts.UpdateServiceConfigCommand, consts.CreateBindingKey,
			consts.CreateSigningKey:
			err := config.SaveConfiguration(context, taskName)
			if err != nil {
				fmt.Fprintln(os.Stderr, "main:main() Unable to save configuration in config.yml")
				log.WithError(err).Error("main:main() Unable to save configuration in config.yml")
				os.Exit(1)
			}

		default:
			fmt.Fprintln(os.Stderr, "Error: Unknown setup task ", args[1])
			printUsage()
			os.Exit(1)
		}

		secLog.Infof("%s, Opening tpm connection", message.SU)

		tpmFactory, err := tpmprovider.NewTpmFactory()
		if err != nil {
			fmt.Println("Error while creating the tpm factory.")
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
					T:     tpmFactory,
					Flags: flags,
				},
				setup.BindingKey{
					T:     tpmFactory,
					Flags: flags,
				},
				setup.RegisterBindingKey{
					Flags: flags,
				},
				setup.RegisterSigningKey{
					Flags: flags,
				},
				setup.Update_Service_Config{
					Flags: flags,
				},
			},
			AskInput: false,
		}
		tasklist := []string{}
		if taskName != consts.SetupAllCommand {
			tasklist = args[1:]
		}
		err = setupRunner.RunTasks(tasklist...)
		if err != nil {
			log.WithError(err).Error("main:main() Error running setup")
			log.Tracef("%+v", err)
			fmt.Fprintf(os.Stderr, "Error running setup tasks...\n")
			os.Exit(1)
		}

	case "runservice":
		config.LogConfiguration(config.Configuration.LogEnableStdout)
		runservice()

	case "rungrpcservice":
		config.LogConfiguration(config.Configuration.LogEnableStdout)
		runGRPCService()

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
		config.LogConfiguration(config.Configuration.LogEnableStdout)
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
		defer conn.Close()

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

	case "prepare-vm":
		config.LogConfiguration(config.Configuration.LogEnableStdout)
		if len(args[1:]) < 1 {
			log.Errorf("main:main() prepare-vm: Invalid number of parameters %s", message.InvalidInputProtocolViolation)
			os.Exit(1)
		}

		secLog.Info("main:main() prepare-vm: wlagent prepare-vm called")
		conn, err := net.Dial("unix", rpcSocketFilePath)
		if err != nil {
			secLog.Errorf("main:main() prepare-vm: Failed to dial wlagent.sock, %s", message.BadConnection)
			os.Exit(1)
		}
		defer conn.Close()

		client := rpc.NewClient(conn)
		defer client.Close()

		// validate domainXML input
		if err = validation.ValidateXMLString(args[1]); err != nil {
			secLog.Errorf("main:main() prepare-vm: %s, Invalid domain XML format", message.InvalidInputBadParam)
			os.Exit(1)
		}

		var args = wlrpc.DomainXML{
			XML: args[1],
		}
		var prepareState bool
		err = client.Call("VirtualMachine.Prepare", &args, &prepareState)
		if err != nil {
			log.Error("main:main() prepare-vm: Client call failed")
		}

		if !prepareState {
			os.Exit(1)
		} else {
			os.Exit(0)
		}

	case "stop-vm":
		config.LogConfiguration(config.Configuration.LogEnableStdout)
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
		defer conn.Close()

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

	case "uninstall":
		config.LogConfiguration(false)

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
		printUsage()
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
	vtpmInstance, err := util.GetTpmInstance()
	if err != nil {
		log.WithError(err).Error("main:runservice() Could not open a new connection to the TPM")
		secLog.Info(message.AppRuntimeErr)
		os.Exit(1)
	}
	vtpmInstance.Close()

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
		l, err := net.Listen("unix", RPCSocketFilePath)
		if err != nil {
			log.Error(err)
			secLog.Error(message.AppRuntimeErr + " Failed to initialize up WLA Unix socket")
			return
		}
		defer l.Close()

		r := rpc.NewServer()
		vm := &wlrpc.VirtualMachine{
			Watcher: fileWatcher,
		}

		err = r.Register(vm)
		if err != nil {
			log.WithError(err).Error("main:runservice() Unable to Register vm watcher")
			log.Tracef("%+v", err)
			secLog.Info(message.AppRuntimeErr)
			return
		}
		for {
			log.Trace("main:runservice() Listen and Serve enter")
			// block and loop, daemon doesnt need to run on go routine
			r.Accept(l)
			log.Trace("main:runservice() Listen and Serve exit")
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

func runGRPCService() {

	RPCSocketFilePath := consts.RunDirPath + consts.RPCSocketFileName
	// When the socket is closed, the file handle on the socket file isn't handled.
	// This code is added to manually remove any stale socket file before the connection
	// is reopened; prevent error: bind address already in use
	// ensure that the socket file exists before removal
	if _, err := os.Stat(RPCSocketFilePath); err == nil {
		err = os.Remove(RPCSocketFilePath)
		if err != nil {
			log.WithError(err).Error("main:runGRPCService() Failed to remove socket file")
			os.Exit(1)
		}
	}

	if _, err := os.Stat(consts.RunDirPath); os.IsNotExist(err) {
		if err := os.MkdirAll(consts.RunDirPath, 0600); err != nil {
			log.WithError(err).Fatalf("main:runGRPCService() Could not create directory: %s, err: %s", consts.RunDirPath, err)
		}
	}

	l, err := net.Listen("unix", RPCSocketFilePath)
	if err != nil {
		log.Error(err)
		secLog.Error(message.AppRuntimeErr + " Failed to initialize up WLA Unix socket")
		return
	}
	defer l.Close()

	s := grpc.NewServer()
	if s == nil {
		log.Error("Error creating grpc server")
		os.Exit(1)
	}
	keyproviderpb.RegisterKeyProviderServiceServer(s, &kpgrpc.GRPCServer{})

	stop := make(chan os.Signal)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	_, err = proc.AddTask(false)
	if err != nil {
		log.WithError(err).Fatal("main:runGRPCService() could not add the task for rpc service")
	}
	// dispatch grpc go routine
	go func() {
		defer proc.TaskDone()
		if err := s.Serve(l); err != nil {
			log.WithError(err).Fatal("main:runGRPCService() Failed to start GRPC server")
			stop <- syscall.SIGTERM
		}
	}()

	log.Info(message.ServiceStart)
	// block until stop channel receives
	err = proc.WaitForQuitAndCleanup(10 * time.Second)
	if err != nil {
		log.WithError(err).Error("main:runGRPCService() Error while clean up")
	}
	s.GracefulStop()

	log.Info(message.ServiceStop)

}
