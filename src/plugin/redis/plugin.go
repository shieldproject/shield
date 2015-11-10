package main

import (
	"fmt"
	"plugin"
)

func main() {
	p := RedisPlugin{
		Name:    "Redis Backup Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
	}

	plugin.Run(p)
}

type RedisPlugin plugin.PluginInfo

func (p RedisPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p RedisPlugin) Backup(endpoint plugin.ShieldEndpoint) (int, error) {
	return plugin.UNSUPPORTED_ACTION, fmt.Errorf("Not yet implemented")
}

func (p RedisPlugin) Restore(endpoint plugin.ShieldEndpoint) (int, error) {
	return plugin.UNSUPPORTED_ACTION, fmt.Errorf("Not yet implemented")
}

func (p RedisPlugin) Store(endpoint plugin.ShieldEndpoint) (string, int, error) {
	return "", plugin.UNSUPPORTED_ACTION, fmt.Errorf("The Redis plugin does not store data")
}

func (p RedisPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) (int, error) {
	return plugin.UNSUPPORTED_ACTION, fmt.Errorf("The Redis plugin does not store data, and does not know how to retrieve it")
}

func (p RedisPlugin) Purge(endpoint plugin.ShieldEndpoint, file string) (int, error) {
	return plugin.UNSUPPORTED_ACTION, fmt.Errorf("The Redis plugin does not store data, and does not know how to purge it")
}
