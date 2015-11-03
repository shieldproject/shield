package main

import (
	"fmt"
	"plugin"
)

func main() {
	fs := FSPlugin{
		meta: plugin.PluginInfo{
			Name:    "Dummy Plugin",
			Author:  "Stark & Wane",
			Version: "1.0.0",
			Features: plugin.PluginFeatures{
				Target: "yes",
				Store:  "yes",
			},
		},
	}

	plugin.Run(fs)
}

type FSPlugin struct {
	meta plugin.PluginInfo
}

func (p FSPlugin) Meta() plugin.PluginInfo {
	return p.meta
}

func (p FSPlugin) Backup(endpoint plugin.ShieldEndpoint) (int, error) {
	data, err := endpoint.StringValue("data")
	if err != nil {
		return plugin.PLUGIN_FAILURE, err
	}

	return plugin.Exec(fmt.Sprintf("/bin/echo %s", data))
}

func (p FSPlugin) Restore(endpoint plugin.ShieldEndpoint) (int, error) {
	file, err := endpoint.StringValue("file")
	if err != nil {
		return plugin.PLUGIN_FAILURE, err
	}

	return plugin.Exec(fmt.Sprintf("/bin/sh -c \"/bin/cat > %s\"", file))
}

func (p FSPlugin) Store(endpoint plugin.ShieldEndpoint) (int, error) {
	file, err := endpoint.StringValue("file")
	if err != nil {
		return plugin.PLUGIN_FAILURE, err
	}

	return plugin.Exec(fmt.Sprintf("/bin/sh -c \"/bin/cat > %s\"", file))
}

func (p FSPlugin) Retrieve(endpoint plugin.ShieldEndpoint) (int, error) {
	file, err := endpoint.StringValue("file")
	if err != nil {
		return plugin.PLUGIN_FAILURE, err
	}

	return plugin.Exec(fmt.Sprintf("/bin/cat %s", file))
}
