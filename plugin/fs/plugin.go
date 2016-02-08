// The `fs` plugin for SHIELD implements generic backup + restore
// functionality for filesystem based backups. It can be used against
// any server that has files that should be backed up. It's not safe
// to use if those files are held open and constantly written to
// by a service (like a database), since there is no coordination
// made with anything that may have those files open.
//
// PLUGIN FEATURES
//
// This plugin implements functionality suitable for use with the following
// SHIELD Job components:
//
//    Target: yes
//    Store:  no
//
// PLUGIN CONFIGURATION
//
// The endpoint configuration passed to this plugin is used to identify what
// files should be backed up from the local system. Your endpoint JSON
// should look something like this:
//
//    {
//        "include":"glob-of-files-to-include", // optional
//        "exclude":"glob-of-files-to-exclude", // optional
//        "base_dir":"base-directory-to-backup"
//    }
//
// BACKUP DETAILS
//
// The `fs` plugin uses `bsdtar` to back up all files located in `base_dir`
// which match the `include` pattern, but do not match the `exclude` pattern.
// If no exclude pattern is supplied, no files are filtered out. If no `include`
// pattern is supplied, all files found are included. Following `bsdtar`'s logic,
// excludes take priority over includes.
//
// RESTORE DETAILS
//
// The `fs` plugin restores the data backed up with `bsdtar` on top of `base_directory`.
// It does not clean up the directory first, so any files that exist on the FS, but are
// not in the restored archive will not be removed.
//
// DEPENDENCIES
//
// This plugin relies on the `bsdtar` utility. Please ensure that it is present on the
// system that will be running the backups + restores. If you are using shield-boshrelease,
// this is provided automatically for you as part of the `shield-agent` job template.
//
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
			Store:  "no",
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
