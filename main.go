package main

import (
	"fmt"
	"intel/isecl/wlagent/src/setuptasks/wlasetup"
	//"setuptasks/wlasetup"
	"os"
	"strings"
)



func printUsage() {
	fmt.Printf("Work Load Agent\n")
	fmt.Printf("===============\n\n")
	fmt.Printf("usage : %s <command> [<args>]\n\n" , os.Args[0])
	fmt.Printf("Following are the list of commands\n")
	fmt.Printf("\tsetup|vmstart|vmstop\n\n")
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
	case "setup":
		
		for name, task := range wlasetup.GetSetupTasks(args) {
			fmt.Println("Running setup task : " + name)
			if (! task.Installed()) {
				if err := task.Execute(); err != nil {
					fmt.Println(err)
				}

				fmt.Println("Need to execute task :" + name)
			} else {
				fmt.Println (name + "already installed .. skipping.. ")
			}
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
