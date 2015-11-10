package main

import (
	"github.com/starkandwayne/shield/plugin"
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

func (p ElasticSearchPlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	return plugin.UNIMPLEMENTED
}

func (p ElasticSearchPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	return plugin.UNIMPLEMENTED
}

func (p ElasticSearchPlugin) Store(endpoint plugin.ShieldEndpoint) (string, error) {
	return "", plugin.UNIMPLEMENTED
}

func (p ElasticSearchPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p ElasticSearchPlugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}
