package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pborman/uuid"
	"github.com/spf13/cobra"

	. "github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

var (

	//== Applicable actions for Stores

	listStoreCmd = &cobra.Command{
		Use:   "stores",
		Short: "List all the Stores",
	}

	showStoreCmd = &cobra.Command{
		Use:   "store",
		Short: "Show all the Stores",
	}

	deleteStoreCmd = &cobra.Command{
		Use:   "store",
		Short: "Delete all the Stores",
	}

	editStoreCmd = &cobra.Command{
		Use:   "store",
		Short: "Edit all the Stores",
	}
)

func init() {
	// Set options for the subcommands
	listStoreCmd.Flags().StringVarP(&pluginFilter, "plugin", "p", "", "Filter by plugin name")
	listStoreCmd.Flags().BoolVar(&unusedFilter, "unused", false, "Show only unused stores")
	listStoreCmd.Flags().BoolVar(&usedFilter, "used", false, "Show only used stores")

	// Hookup functions to the subcommands
	listStoreCmd.Run = processListStoresRequest
	showStoreCmd.Run = processShowStoreRequest
	editStoreCmd.Run = processEditStoreRequest
	deleteStoreCmd.Run = processDeleteStoreRequest

	// Add the subcommands to the base actions
	listCmd.AddCommand(listStoreCmd)
	showCmd.AddCommand(showStoreCmd)
	editCmd.AddCommand(editStoreCmd)
	deleteCmd.AddCommand(deleteStoreCmd)
}

type ListStoreOptions struct {
	Unused bool
	Used   bool
	Plugin string
	UUID   string
}

func ListStores(opts ListStoreOptions) error {
	stores, err := GetStores(StoreFilter{
		Plugin: opts.Plugin,
		Unused: MaybeBools(opts.Unused, opts.Used),
	})
	if err != nil {
		return fmt.Errorf("\nERROR: Could not fetch list of stores: %s\n", err)
	}
	t := tui.NewTable("UUID", "Name", "Description", "Plugin", "Endpoint")
	for _, store := range stores {
		if len(opts.UUID) > 0 && opts.UUID == store.UUID {
			t.Row(store.UUID, store.Name, store.Summary, store.Plugin, store.Endpoint)
			break
		} else if len(opts.UUID) > 0 && opts.UUID != store.UUID {
			continue
		}
		t.Row(store.UUID, store.Name, store.Summary, store.Plugin, store.Endpoint)
	}
	t.Output(os.Stdout)
	return nil
}

func processListStoresRequest(cmd *cobra.Command, args []string) {
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "\nERROR: Unexpected arguments following command: %v\n", args)
		//FIXME  show help
		os.Exit(1)
	}

	stores, err := GetStores(StoreFilter{
		Plugin: pluginFilter,
		Unused: MaybeString(parseTristateOptions(cmd, "unused", "used")),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not fetch list of stores:\n", err)
	}

	t := tui.NewTable("UUID", "Name", "Description", "Plugin", "Endpoint")
	for _, store := range stores {
		t.Row(store.UUID, store.Name, store.Summary, store.Plugin, store.Endpoint)
	}
	t.Output(os.Stdout)
}

func processCreateStoreRequest(cmd *cobra.Command, args []string) {

	// Validate Request
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "\nERROR: Unexpected arguments following command: %v\n", args)
		//FIXME  show help
		os.Exit(1)
	}

	// Invoke editor
	content := invokeEditor(`{
	"name":     "",
	"summary":  "",
	"plugin":   "",
	"endpoint": "{\"\":\"\"}",
}`)

	fmt.Println("Got the following content:\n\n", content)

	data, err := CreateStore(content)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not fetch list of stores:\n", err)
		os.Exit(1)
	}

	// Print
	output, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render list of stores:\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output[:]))

	return
}

func processShowStoreRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	store, err := GetStore(uuid.Parse(args[0]))
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not show store:\n", err)
		os.Exit(1)
	}

	t := tui.NewReport()
	t.Add("UUID", store.UUID)
	t.Add("Name", store.Name)
	t.Add("Summary", store.Summary)
	t.Break()

	t.Add("Plugin", store.Plugin)
	t.Add("Endpoint", store.Endpoint)
	t.Output(os.Stdout)
}

func processEditStoreRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	requested_UUID := uuid.Parse(args[0])

	original_data, err := GetStore(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not show store:\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(original_data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render store:\n", err)
	}

	fmt.Println("Got the following original store:\n\n", string(data))

	// Invoke editor
	content := invokeEditor(string(data))

	fmt.Println("Got the following edited store:\n\n", content)

	update_data, err := UpdateStore(requested_UUID, content)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not update stores:\n", err)
		os.Exit(1)
	}
	// Print
	output, err := json.MarshalIndent(update_data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render store:\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output[:]))

	return
}

func processDeleteStoreRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	requested_UUID := uuid.Parse(args[0])

	err := DeleteStore(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not delete store:\n", err)
		os.Exit(1)
	}

	// Print
	fmt.Println(requested_UUID, " Deleted")

	return
}
