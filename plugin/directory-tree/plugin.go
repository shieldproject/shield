package main

import (
	"fmt"
	"github.com/starkandwayne/shield/plugin"
)

func main() {
	p := DirectoryTreePlugin{
		Name:    "DirectoryTree Plugin",
		Author:  "Stark & Wayne",
		Version: "1.0.0",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "yes",
		},
	}

	plugin.DEBUG("directory-tree plugin starting up...")
	plugin.Run(p)
}

type DirectoryTreePlugin plugin.PluginInfo

type DirectoryTreeConfig struct {
	Include  string
	Exclude  string
	BasePath string
}

func (p DirectoryTreePlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func getBlobStoreConfig(endpoint plugin.ShieldEndpoint) (*DirectoryTreeConfig, error) {
	include, _ := endpoint.StringValue("include")
	exclude, _ := endpoint.StringValue("exclude")
	base_dir, err := endpoint.StringValue("base_dir")
	if err != nil {
		return nil, err
	}

	return &DirectoryTreeConfig{
		Include:  include,
		Exclude:  exclude,
		BasePath: base_dir,
	}, nil
}

func (p DirectoryTreePlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	cfg, err := getBlobStoreConfig(endpoint)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("tar -c -C %s --include '%s' --exclude '%s' .", cfg.BasePath, cfg.Include, cfg.Exclude)
	plugin.DEBUG("Executing `%s`", cmd)
	err = plugin.Exec(cmd, plugin.STDOUT)
	if err != nil {
		return err
	}

	return nil
}

func (p DirectoryTreePlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	cfg, err := getBlobStoreConfig(endpoint)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("tar -x -C %s .", cfg.BasePath)
	plugin.DEBUG("Executing `%s`", cmd)
	err = plugin.Exec(cmd, plugin.STDIN)
	if err != nil {
		return err
	}

	return nil
}

func (p DirectoryTreePlugin) Store(endpoint plugin.ShieldEndpoint) (string, error) {
	return "", plugin.UNIMPLEMENTED
}

func (p DirectoryTreePlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p DirectoryTreePlugin) Purge(endpoint plugin.ShieldEndpoint, key string) error {
	return plugin.UNIMPLEMENTED
}
