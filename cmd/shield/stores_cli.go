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

	editStoreCmd = &cobra.Command{
		Use:   "store",
		Short: "Edit all the Stores",
	}
)

func init() {
	// Hookup functions to the subcommands
	editStoreCmd.Run = processEditStoreRequest

	// Add the subcommands to the base actions
	editCmd.AddCommand(editStoreCmd)
}

type ListStoreOptions struct {
	Unused bool
	Used   bool
	Plugin string
	UUID   string
}

func ListStores(opts ListStoreOptions) error {
	//FIXME: (un)?used flags not working; --plugin works.
	stores, err := GetStores(StoreFilter{
		Plugin: opts.Plugin,
		Unused: MaybeBools(opts.Unused, opts.Used),
	})
	if err != nil {
		return fmt.Errorf("ERROR: Could not fetch list of stores: %s", err)
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

func DeleteStoreByUUID(u string) error {
	err := DeleteStore(uuid.Parse(u))
	if err != nil {
		return fmt.Errorf("ERROR: Cannot delete store '%s': %s", u, err)
	}
	fmt.Fprintf(os.Stdout, "Deleted store '%s'\n", u)
	return nil
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
