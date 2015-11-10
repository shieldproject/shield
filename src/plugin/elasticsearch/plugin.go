package main

import (
	"fmt"
	"plugin"
)

func main() {
	p := ElasticSearchPlugin{
		Name:    "ElasticSearch Backup Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
	}

	plugin.Run(p)
}

type ElasticSearchPlugin plugin.PluginInfo

func (p ElasticSearchPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p ElasticSearchPlugin) Backup(endpoint plugin.ShieldEndpoint) (int, error) {
	return plugin.UNSUPPORTED_ACTION, fmt.Errorf("Not yet implemented")
}

func (p ElasticSearchPlugin) Restore(endpoint plugin.ShieldEndpoint) (int, error) {
	return plugin.UNSUPPORTED_ACTION, fmt.Errorf("Not yet implemented")
}

func (p ElasticSearchPlugin) Store(endpoint plugin.ShieldEndpoint) (string, int, error) {
	return "", plugin.UNSUPPORTED_ACTION, fmt.Errorf("The ElasticSearch plugin does not store data")
}

func (p ElasticSearchPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) (int, error) {
	return plugin.UNSUPPORTED_ACTION, fmt.Errorf("The ElasticSearch plugin does not store data, and does not know how to retrieve it")
}

func (p ElasticSearchPlugin) Purge(endpoint plugin.ShieldEndpoint, file string) (int, error) {
	return plugin.UNSUPPORTED_ACTION, fmt.Errorf("The ElasticSearch plugin does not store data, and does not know how to purge it")
}
