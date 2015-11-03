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
				Target: true,
				Store:  false,
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
	return plugin.UNSUPPORTED_ACTION, fmt.Errorf("'backup' is not supported by the %s", p.meta.Name)
}

func (p FSPlugin) Restore(endpoint plugin.ShieldEndpoint) (int, error) {
	return plugin.UNSUPPORTED_ACTION, fmt.Errorf("'restore' is not supported by the %s", p.meta.Name)
}

func (p FSPlugin) Store(endpoint plugin.ShieldEndpoint) (int, error) {
	return plugin.UNSUPPORTED_ACTION, fmt.Errorf("'store' is not supported by the %s", p.meta.Name)
}

func (p FSPlugin) Retrieve(endpoint plugin.ShieldEndpoint) (int, error) {
	return plugin.UNSUPPORTED_ACTION, fmt.Errorf("'retrieve' is not supported by the %s", p.meta.Name)
}
