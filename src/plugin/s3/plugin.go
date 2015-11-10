package main

import (
	"plugin"
)

func main() {
	p := S3Plugin{
		Name:    "S3 Backup + Storage Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "yes",
		},
	}

	plugin.Run(p)
}

type S3Plugin plugin.PluginInfo

func (p S3Plugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p S3Plugin) Backup(endpoint plugin.ShieldEndpoint) error {
	return plugin.UNIMPLEMENTED
}

func (p S3Plugin) Restore(endpoint plugin.ShieldEndpoint) error {
	return plugin.UNIMPLEMENTED
}

func (p S3Plugin) Store(endpoint plugin.ShieldEndpoint) (string, error) {
	return "", plugin.UNIMPLEMENTED
}

func (p S3Plugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p S3Plugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}
