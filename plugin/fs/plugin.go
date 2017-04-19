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
//    Store:  yes
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
//        "bsdtar":"path-to-bsdtar",            // optional
//        "base_dir":"base-directory-to-backup"
//    }
//
// Default Configuration
//
//    {
//        "bsdtar": "/var/vcap/packages/bsdtar/bin/bsdtar"
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
	"io"
	"os"
	"time"

	"github.com/starkandwayne/goutils/ansi"

	"github.com/starkandwayne/shield/plugin"
)

var (
	DefaultBsdTar = "/var/vcap/packages/bsdtar/bin/bsdtar"
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
		Example: `
{
  "base_dir" : "/path/to/backup"   # REQUIRED

  "include"  : "*.txt",            # UNIX glob of files to include in backup
  "exclude"  : "*.o",              # ... and another for what to exclude

  "bsdtar"   : "/usr/bin/bsdtar"   # where is the BSD tar utility?
                                   # (GNU tar is insufficient)
}
`,
		Defaults: `
{
  "bsdtar" : "/var/vcap/packages/bsdtar/bin/bsdtar"
}
`,
	}

	plugin.DEBUG("fs plugin starting up...")
	plugin.Run(p)
}

type FSPlugin plugin.PluginInfo

type FSConfig struct {
	Include  string
	Exclude  string
	BasePath string
	BsdTar   string
}

func (p FSPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func getFSConfig(endpoint plugin.ShieldEndpoint) (*FSConfig, error) {
	include, err := endpoint.StringValueDefault("include", "")
	if err != nil {
		return nil, err
	}

	exclude, err := endpoint.StringValueDefault("exclude", "")
	if err != nil {
		return nil, err
	}

	bsdtar, err := endpoint.StringValueDefault("bsdtar", DefaultBsdTar)
	if err != nil {
		return nil, err
	}

	base_dir, err := endpoint.StringValue("base_dir")
	if err != nil {
		return nil, err
	}

	return &FSConfig{
		Include:  include,
		Exclude:  exclude,
		BasePath: base_dir,
		BsdTar:   bsdtar,
	}, nil
}

func (p FSPlugin) Validate(endpoint plugin.ShieldEndpoint) error {
	var (
		s    string
		err  error
		fail bool
	)

	s, err = endpoint.StringValue("base_dir")
	if err != nil {
		ansi.Printf("@R{\u2717 base_dir  %s}\n", err)
		fail = true
	} else {
		ansi.Printf("@G{\u2713 base_dir}  files in @C{%s} will be backed up\n", s)
	}

	s, err = endpoint.StringValueDefault("include", "")
	if err != nil {
		ansi.Printf("@R{\u2717 include   %s}\n", err)
		fail = true
	} else if s == "" {
		ansi.Printf("@G{\u2713 include}   all files will be included\n")
	} else {
		ansi.Printf("@G{\u2713 include}   only files matching @C{%s} will be backed up\n", s)
	}

	s, err = endpoint.StringValueDefault("exclude", "")
	if err != nil {
		ansi.Printf("@R{\u2717 base_dir  %s}\n", err)
		fail = true
	} else if s == "" {
		ansi.Printf("@G{\u2713 exclude}   no files will be excluded\n")
	} else {
		ansi.Printf("@G{\u2713 exclude}   files matching @C{%s} will be skipped\n", s)
	}

	s, err = endpoint.StringValueDefault("bsdtar", "")
	if err != nil {
		ansi.Printf("@R{\u2717 bsdtar    %s}\n", err)
		fail = true
	} else if s == "" {
		ansi.Printf("@G{\u2713 bsdtar}    using default @C{%s}\n", DefaultBsdTar)
	} else {
		ansi.Printf("@G{\u2713 bsdtar}    @C{%s}\n", s)
	}

	if fail {
		return fmt.Errorf("fs: invalid configuration")
	}
	return nil
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
		flags = fmt.Sprintf("%s --exclude '%s'", flags, cfg.Exclude)
	}
	cmd := fmt.Sprintf("%s -c -C %s -f - %s .", cfg.BsdTar, cfg.BasePath, flags)
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

	os.MkdirAll(cfg.BasePath, 0777)
	cmd := fmt.Sprintf("%s -x -C %s -f -", cfg.BsdTar, cfg.BasePath)
	plugin.DEBUG("Executing `%s`", cmd)
	err = plugin.Exec(cmd, plugin.STDIN)
	if err != nil {
		return err
	}

	return nil
}

func (p FSPlugin) Store(endpoint plugin.ShieldEndpoint) (string, error) {
	cfg, err := getFSConfig(endpoint)
	if err != nil {
		return "", err
	}

	t := time.Now()
	year, mon, day := t.Date()
	hour, min, sec := t.Clock()
	uuid := plugin.GenUUID()

	dir := fmt.Sprintf("%04d/%02d/%02d", year, mon, day)
	file := fmt.Sprintf("%04d-%02d-%02d-%02d%02d%02d-%s", year, mon, day, hour, min, sec, uuid)

	err = os.MkdirAll(fmt.Sprintf("%s/%s", cfg.BasePath, dir), 0777) // umask will lower...
	if err != nil {
		return "", err
	}

	f, err := os.Create(fmt.Sprintf("%s/%s/%s", cfg.BasePath, dir, file))
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err = io.Copy(f, os.Stdin); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s", dir, file), nil
}

func (p FSPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	cfg, err := getFSConfig(endpoint)
	if err != nil {
		return err
	}

	f, err := os.Open(fmt.Sprintf("%s/%s", cfg.BasePath, file))
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = io.Copy(os.Stdout, f); err != nil {
		return err
	}

	return nil
}

func (p FSPlugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	cfg, err := getFSConfig(endpoint)
	if err != nil {
		return err
	}

	err = os.Remove(fmt.Sprintf("%s/%s", cfg.BasePath, file))
	if err != nil {
		return err
	}

	return nil
}
