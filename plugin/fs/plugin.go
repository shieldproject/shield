package main

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"strings"

	fmt "github.com/jhunt/go-ansi"

	"github.com/starkandwayne/shield/plugin"
)

func main() {
	p := FSPlugin{
		Name:    "Local Filesystem Plugin",
		Author:  "Stark & Wayne",
		Version: "1.0.0",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
		Example: `
{
  "base_dir" : "/path/to/backup"   # REQUIRED

  "include"  : "*.txt",            # UNIX glob of files to include in backup
  "exclude"  : "*.o"               # ... and another for what to exclude
}
`,
		Defaults: `
{
}
`,

		Fields: []plugin.Field{
			plugin.Field{
				Mode:     "target",
				Name:     "base_dir",
				Type:     "abspath",
				Title:    "Base Directory",
				Help:     "Absolute path of the directory to backup.",
				Example:  "/srv/www/htdocs",
				Required: true,
			},
			plugin.Field{
				Mode:  "target",
				Name:  "include",
				Type:  "string",
				Title: "Files to Include",
				Help:  "Only files that match this pattern will be included in the backup archive.  If not specified, all files will be included.",
			},
			plugin.Field{
				Mode:  "target",
				Name:  "exclude",
				Type:  "string",
				Title: "Files to Exclude",
				Help:  "Files that match this pattern will be excluded from the backup archive.  If not specified, no files will be excluded.",
			},
			plugin.Field{
				Mode:  "target",
				Name:  "strict",
				Type:  "bool",
				Title: "Strict Mode",
				Help:  "If files go missing while walking the directory, consider that a fatal error.",
			},
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
	Strict   bool
}

func (cfg *FSConfig) Match(path string) bool {
	if cfg.Exclude != "" {
		if ok, err := filepath.Match(cfg.Exclude, path); ok && err == nil {
			return false
		}
	}
	if cfg.Include != "" {
		if ok, err := filepath.Match(cfg.Include, path); ok && err == nil {
			return true
		}
		return false
	}
	return true
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

	base_dir, err := endpoint.StringValue("base_dir")
	if err != nil {
		return nil, err
	}

	strict, err := endpoint.BooleanValueDefault("strict", false)
	if err != nil {
		return nil, err
	}

	return &FSConfig{
		Include:  include,
		Exclude:  exclude,
		BasePath: base_dir,
		Strict:   strict,
	}, nil
}

func (p FSPlugin) Validate(endpoint plugin.ShieldEndpoint) error {
	var (
		s    string
		b    bool
		err  error
		fail bool
	)

	b, err = endpoint.BooleanValueDefault("strict", false)
	if err != nil {
		fmt.Printf("@R{\u2717 strict    %s}\n", err)
		fail = true
	} else if b {
		fmt.Printf("@G{\u2713 strict}    @C{yes} - files that go missing are considered an error\n")
	} else {
		fmt.Printf("@G{\u2713 strict}    @C{no} (default)\n")
	}

	s, err = endpoint.StringValue("base_dir")
	if err != nil {
		fmt.Printf("@R{\u2717 base_dir  %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 base_dir}  files in @C{%s} will be backed up\n", s)
	}

	s, err = endpoint.StringValueDefault("include", "")
	if err != nil {
		fmt.Printf("@R{\u2717 include   %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 include}   all files will be included\n")
	} else {
		fmt.Printf("@G{\u2713 include}   only files matching @C{%s} will be backed up\n", s)
	}

	s, err = endpoint.StringValueDefault("exclude", "")
	if err != nil {
		fmt.Printf("@R{\u2717 base_dir  %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 exclude}   no files will be excluded\n")
	} else {
		fmt.Printf("@G{\u2713 exclude}   files matching @C{%s} will be skipped\n", s)
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

	archive := tar.NewWriter(os.Stdout)
	n := 0
	walker := func(path string, info os.FileInfo, err error) error {
		baseRelative := strings.TrimPrefix(strings.Replace(path, cfg.BasePath, "", 1), "/")
		if baseRelative == "" { /* musta been cfg.BasePath or cfg.BasePath + '/' */
			return nil
		}

		fmt.Fprintf(os.Stderr, " - found '%s' ... ", path)
		if info == nil {
			if _, ok := err.(*os.PathError); !cfg.Strict && ok {
				fmt.Fprintf(os.Stderr, "no longer exists; skipping.\n")
				return nil
			} else {
				fmt.Fprintf(os.Stderr, "FAILED\n")
				return fmt.Errorf("failed to walk %s: %s", path, err)
			}
		}

		if !cfg.Match(info.Name()) {
			fmt.Fprintf(os.Stderr, "ignoring (per include/exclude)\n")
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		n += 1
		fmt.Fprintf(os.Stderr, "ok\n")

		link := ""
		if info.Mode()&os.ModeType == os.ModeSymlink {
			link, err = os.Readlink(path)
			if err != nil {
				return err
			}
		}
		header, err := tar.FileInfoHeader(info, link)
		if err != nil {
			return err
		}

		header.Name = baseRelative
		if err := archive.WriteHeader(header); err != nil {
			return err
		}

		if info.Mode().IsDir() || link != "" {
			return nil
		}

		if info.Mode().IsRegular() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			io.Copy(archive, f)
			return nil
		}

		return fmt.Errorf("unable to archive special file '%s'", path)
	}

	fmt.Fprintf(os.Stderr, "backing up files in '%s'...\n", cfg.BasePath)
	if err := filepath.Walk(cfg.BasePath, walker); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "done; found %d files / directories to archive...\n\n", n)

	return archive.Close()
}

func (p FSPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	cfg, err := getFSConfig(endpoint)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cfg.BasePath, 0777); err != nil {
		return err
	}

	n := 0
	archive := tar.NewReader(os.Stdin)
	for {
		header, err := archive.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		info := header.FileInfo()
		path := fmt.Sprintf("%s/%s", cfg.BasePath, header.Name)
		n += 1
		fmt.Fprintf(os.Stderr, " - restoring '%s'... ", path)
		if info.Mode().IsDir() {
			if err := os.MkdirAll(path, 0777); err != nil {
				fmt.Fprintf(os.Stderr, "FAILED (could not create directory)\n")
				return err
			}
			fmt.Fprintf(os.Stderr, "created directory\n")

		} else if info.Mode().IsRegular() {
			f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, info.Mode())
			if err != nil {
				fmt.Fprintf(os.Stderr, "FAILED (could not create new file)\n")
				return err
			}
			if _, err := io.Copy(f, archive); err != nil {
				fmt.Fprintf(os.Stderr, "FAILED (could not copy data to disk)\n")
				return err
			}
			fmt.Fprintf(os.Stderr, "created file\n")

		} else {
			fmt.Fprintf(os.Stderr, "FAILED (not a regular file or a directory)\n")
			return fmt.Errorf("unable to unpack special file '%s'", path)
		}

		/* put things back the way they were... */
		if err := os.Chtimes(path, header.AccessTime, header.ModTime); err != nil {
			fmt.Fprintf(os.Stderr, "FAILED (could not set atime / mtime / ctime)\n")
			return err
		}
		if err := os.Chown(path, header.Uid, header.Gid); err != nil {
			fmt.Fprintf(os.Stderr, "FAILED (could not set user ownership)\n")
			return err
		}
		if err := os.Chmod(path, info.Mode()); err != nil {
			fmt.Fprintf(os.Stderr, "FAILED (could not set group ownership)\n")
			return err
		}

		fmt.Fprintf(os.Stderr, "ok\n")
	}

	fmt.Fprintf(os.Stderr, "done; restored %d files / directories...\n\n", n)
	return nil
}

func (p FSPlugin) Store(endpoint plugin.ShieldEndpoint) (string, int64, error) {
	return "", 0, plugin.UNIMPLEMENTED
}

func (p FSPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p FSPlugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}
