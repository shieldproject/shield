package main

import (
	"fmt"

	"github.com/starkandwayne/shield/plugin"
)

func main() {
	p := FSPlugin{
		Name:    "FS Plugin",
		Author:  "Stark & Wayne",
		Version: "1.0.0",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "yes",
		},
	}

	plugin.DEBUG("fs plugin starting up...")
	plugin.Run(p)
}

type FSPlugin plugin.PluginInfo

type FSConfig struct {
	Include  string
	Exclude  string
	BasePath string
}

func (p FSPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func getFSConfig(endpoint plugin.ShieldEndpoint) (*FSConfig, error) {
	include, _ := endpoint.StringValue("include")
	exclude, _ := endpoint.StringValue("exclude")
	base_dir, err := endpoint.StringValue("base_dir")
	if err != nil {
		return nil, err
	}

	return &FSConfig{
		Include:  include,
		Exclude:  exclude,
		BasePath: base_dir,
	}, nil
}

func (p FSPlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	cfg, err := getFSConfig(endpoint)
	if err != nil {
		return err
	}

	//FIXME: drop include and exclude if they were not specified
	var flags string
	if cfg.Include != "" {
		flags = fmt.Sprintf("%s --include '%s'", flags, cfg.Include)
	}
	if cfg.Exclude != "" {
		flags = fmt.Sprintf("%s --include '%s'", flags, cfg.Exclude)
	}
	cmd := fmt.Sprintf("/var/vcap/packages/bsdtar/bin/bsdtar -c -C %s %s .", cfg.BasePath, flags)
	plugin.DEBUG("Executing `%s`", cmd)
	err = plugin.Exec(cmd, plugin.STDOUT)
	if err != nil {
		return err
	}

	return nil
}

func (p FSPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	cfg, err := getFSConfig(endpoint)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("/var/vcap/packages/bsdtar/bin/bsdtar -x -C %s", cfg.BasePath)
	plugin.DEBUG("Executing `%s`", cmd)
	err = plugin.Exec(cmd, plugin.STDIN)
	if err != nil {
		return err
	}

	return nil
}

func (p FSPlugin) Store(endpoint plugin.ShieldEndpoint) (string, error) {
	return "", plugin.UNIMPLEMENTED
}

func (p FSPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p FSPlugin) Purge(endpoint plugin.ShieldEndpoint, key string) error {
	return plugin.UNIMPLEMENTED
}
