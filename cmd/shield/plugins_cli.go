package main

import (
	"github.com/spf13/cobra"
	//"github.com/spf13/viper"
)

var (

	//== Applicable actions for Plugins

	listPluginCmd = &cobra.Command{
		Use:   "plugins",
		Short: "Lists all the plugins",
		Long:  "This is a Long Jaun (TBD)",
	}

	showPluginCmd = &cobra.Command{
		Use:   "plugin",
		Short: "Shows information about the specified plugin",
		Long:  "This is a Long Jaun (TBD)",
	}
)

func init() {
	listPluginCmd.Run = debug
	showPluginCmd.Run = debug

	listCmd.AddCommand(listPluginCmd)
	showCmd.AddCommand(showPluginCmd)
}
