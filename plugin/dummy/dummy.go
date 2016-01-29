package main

/*

This is a generic and not terribly helpful plugin. However, it shows the basics
of what is needed in a backup plugin, and how they execute.

*/

import (
	"fmt"
	"os"

	"github.com/starkandwayne/shield/plugin"
)

func main() {
	// Create an object representing this plugin, which is a type conforming to the Plugin interface
	dummy := DummyPlugin{
		// give it some authorship info
		meta: plugin.PluginInfo{
			Name:    "Dummy Plugin",
			Author:  "Stark & Wayne",
			Version: "1.0.0",
			Features: plugin.PluginFeatures{
				Target: "yes",
				Store:  "yes",
			},
		},
	}

	fmt.Fprintf(os.Stderr, "dummy plugin starting up...\n")
	// Run the plugin - the plugin framework handles all arg parsing, exit handling, error/debug formatting for you
	plugin.Run(dummy)
}

// Define my DummyPlugin type
type DummyPlugin struct {
	meta plugin.PluginInfo // needs a place to store metadata
}

// This function should be used to return the plugin's PluginInfo, however you decide to implement it
func (p DummyPlugin) Meta() plugin.PluginInfo {
	return p.meta
}

// Called when you want to back data up. Examine the ShieldEndpoint passed in, and perform actions accordingly
func (p DummyPlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	data, err := endpoint.StringValue("data")
	if err != nil {
		return err
	}

	return plugin.Exec(fmt.Sprintf("/bin/echo %s", data), plugin.STDOUT)
}

// Called when you want to restore data Examine the ShieldEndpoint passed in, and perform actions accordingly
func (p DummyPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	file, err := endpoint.StringValue("file")
	if err != nil {
		return err
	}

	return plugin.Exec(fmt.Sprintf("/bin/sh -c \"/bin/cat > %s\"", file), plugin.STDIN)
}

// Called when you want to store backup data. Examine the ShieldEndpoint passed in, and perform actions accordingly
func (p DummyPlugin) Store(endpoint plugin.ShieldEndpoint) (string, error) {
	directory, err := endpoint.StringValue("directory")
	if err != nil {
		return "", err
	}

	file := plugin.GenUUID()

	err = plugin.Exec(fmt.Sprintf("/bin/sh -c \"/bin/cat > %s/%s\"", directory, file), plugin.STDIN)
	return file, err
}

// Called when you want to retreive backup data. Examine the ShieldEndpoint passed in, and perform actions accordingly
func (p DummyPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	directory, err := endpoint.StringValue("directory")
	if err != nil {
		return err
	}

	return plugin.Exec(fmt.Sprintf("/bin/cat %s/%s", directory, file), plugin.STDOUT)
}

func (p DummyPlugin) Purge(endpoint plugin.ShieldEndpoint, key string) error {
	return plugin.UNIMPLEMENTED
}

//That's all there is to writing a plugin. If your plugin doesn't need to implement Store/Retrieve, or Backup/Restore,
// Define the functions, and have them return plugin.UNIMPLEMENTED
