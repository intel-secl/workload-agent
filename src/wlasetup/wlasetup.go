// This file should be common for all tasks. Things like getting a list of tasks,
// For individual tasks, organize into indivudual files

package wlasetup

import (
	"fmt"
	"log"
	"strings"
)

// SetupTask is an generic interface for a setup task.
// The Execute() method of the class executes the setup task
// The Installed() method check if the SetupTask is already completed
// The sequence of operation for a setup task should be to check if
// it has already installed and if not installed, then call the execute method
type SetupTask interface {
	Execute() error
	Installed() bool
}

// GetSetupTasks returns a map with SetupTasks in the module. These are all the struct/ class
// that implements the SetupTask interface.
// If there is a specific task(s) being requested, only the specific task(s) are returned
func GetSetupTasks(commandargs []string) map[string]SetupTask {

	//tasks = ParseSetupTasks(commandargs)
	if len(commandargs) < 1 || strings.ToLower(commandargs[0]) != "setup" {
		panic(fmt.Errorf("method GetSetupTasks need at least one parameter with command \"setup\". Arguments : %v", commandargs))
	}

	m := make(map[string]SetupTask)

	if len(commandargs) > 1 {
		// Todo - we should be able to find structs using reflection in this
		// package that implements the SetupTask Interface and add elements to the
		//  map. For now, we are just going to hardcode the setup tasks that we have

		// First argument is "setup" - the rest should be list of tasks
		for _, task := range commandargs[1:] {

			switch strings.ToLower(task) {
			case "signingkey":
				m["SigningKey"], _ = NewCertifiedKey("Signing")
			case "bindingkey":
				m["BindingKey"], _ = NewCertifiedKey("Binding")
			default:
				log.Printf("Unknown Setup Task in list : %s", task)
			}
		}

	} else {
		fmt.Println("No arguments passed in")
		// no specific tasks passed in. We will return a list of all tasks
		m["SigningKey"], _ = NewCertifiedKey("Signing")
		m["BindingKey"], _ = NewCertifiedKey("Binding")

	}

	//for key, obj := range m {
	//	fmt.Println("Key=" + key + "\tValue=" + reflect.TypeOf(obj).Name())
	//}

	return m
}

// ParseSetupTasks takes space seperated list of tasks along with any additional flags.
// Not used for now...
// TODO : to be implemented.
func ParseSetupTasks(commandargs ...[]string) []string {
	//TODO: This function for now takes a space seperated list of
	// setup arguments. We should parse this to check for the presence of --force
	//flags. This should be a common utility that is able to parse a list of
	// tasks as well as an associated flags
	if len(commandargs) > 1 {
		log.Println("Expecting a slice of string as argument.")
	}
	fmt.Println(commandargs)
	return commandargs[0]
}

// RunTasks - function to be implemented as part of the Common Installer module
func RunTasks(commandargs []string) {

}
