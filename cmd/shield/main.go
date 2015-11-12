// Jamie: This contains the go source code that will become shield.

package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"strings"
)

var (
	//== Root Command for Shield

	ShieldCmd = &cobra.Command{
		Use:   "shield",
		Short: "TBD::Shield shield",
		Long:  "TBD::This is the Sheild command",
	}

	//== Base Verbs

	listCmd    = &cobra.Command{Use: "list", Short: "List all the {{children}}"}
	showCmd    = &cobra.Command{Use: "show", Short: "Show details for the specified {{children}}"}
	deleteCmd  = &cobra.Command{Use: "delete", Short: "Delete the specified {{children}}"}
	updateCmd  = &cobra.Command{Use: "update", Short: "Update the specified {{children}}"}
	editCmd    = &cobra.Command{Use: "edit", Short: "Edit the specified {{children}}"}
	pauseCmd   = &cobra.Command{Use: "pause", Short: "Pause the specified {{children}}"}
	unpauseCmd = &cobra.Command{Use: "unpause", Short: "Continue the paused specified {{children}}"}
	pausedCmd  = &cobra.Command{Use: "paused", Short: "Check if the specified {{children}} is paused"}
	runCmd     = &cobra.Command{Use: "run", Short: "Run the specified {{children}}"}
	cancelCmd  = &cobra.Command{Use: "cancel", Short: "Cancel the specified running {{children}}"}
	restoreCmd = &cobra.Command{Use: "restore", Short: "Restore the specified {{children}}"}
)

//--------------------------

func main() {
	viper.SetConfigType("yaml") // To support lnguyen development

	addSubCommandWithHelp(ShieldCmd, listCmd, showCmd, deleteCmd, updateCmd, editCmd, pauseCmd, unpauseCmd, pausedCmd, runCmd, cancelCmd, restoreCmd)
	ShieldCmd.Execute()
}

func debug(cmd *cobra.Command, args []string) {

	// Trace back through the cmd chain to assemble the full command
	var cmd_list = make([]string, 0)
	ptr := cmd
	for {
		cmd_list = append([]string{ptr.Use}, cmd_list...)
		if ptr.Parent() != nil {
			ptr = ptr.Parent()
		} else {
			break
		}
	}

	fmt.Print("Command: ")
	fmt.Print(strings.Join(cmd_list, " "))
	fmt.Printf(" Argv [%s]\n", args)
}

func addSubCommandWithHelp(tgtCmd *cobra.Command, subCmds ...*cobra.Command) {
	tgtCmd.AddCommand(subCmds...)

	for _, subCmd := range subCmds {
		var children = make([]string, 0)
		var sentence string

		for _, childCmd := range subCmd.Commands() {
			// TODO: if subCommand children have further children, assume compound command and add it
			children = append(children, childCmd.Use)
		}

		if len(children) > 0 {
			if len(children) == 1 {
				sentence = children[0]
			} else {
				sentence = strings.Join(children[0:(len(children)-1)], ", ") + " or " + children[len(children)-1]
			}
			subCmd.Short = strings.Replace(subCmd.Short, "{{children}}", sentence, -1)
		}
	}
}
