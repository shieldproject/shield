package main

import (
	"fmt"
	"plugin"
)

func main() {
	p := PostgresPlugin{
		Name:    "PostgreSQL Backup Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
	}

	plugin.Run(p)
}

type PostgresPlugin plugin.PluginInfo

func (p PostgresPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p PostgresPlugin) Backup(endpoint plugin.ShieldEndpoint) (int, error) {
	return plugin.UNSUPPORTED_ACTION, fmt.Errorf("Not yet implemented")
}

func (p PostgresPlugin) Restore(endpoint plugin.ShieldEndpoint) (int, error) {
	return plugin.UNSUPPORTED_ACTION, fmt.Errorf("Not yet implemented")
}

func (p PostgresPlugin) Store(endpoint plugin.ShieldEndpoint) (string, int, error) {
	return "", plugin.UNSUPPORTED_ACTION, fmt.Errorf("The PostgresSQL plugin does not store data")
}

func (p PostgresPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) (int, error) {
	return plugin.UNSUPPORTED_ACTION, fmt.Errorf("The PostgresSQL plugin does not store data, and does not know how to retrieve it")
}

func (p PostgresPlugin) Purge(endpoint plugin.ShieldEndpoint, file string) (int, error) {
	return plugin.UNSUPPORTED_ACTION, fmt.Errorf("The PostgresSQL plugin does not store data, and does not know how to purge it")
}
