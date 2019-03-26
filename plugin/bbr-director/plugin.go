package main

import (
	"archive/tar"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	fmt "github.com/jhunt/go-ansi"
	"github.com/starkandwayne/shield/plugin"
)

const (
	DefaultBinDir = "/var/vcap/packages/bbr/bin"
)

func main() {
	bbr := BBRPlugin{
		Name:    "BBR Director Plugin",
		Author:  "Stark & Wayne",
		Version: "1.4.0",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
		Example: `
{
  "bindir"     : "/path/to/bbr-package/bin",
  "director"   : "192.168.50.6",
  "username"   : "vcap",
  "key"        : "-----BEGIN RSA PRIVATE KEY-----\n(cert contents)\n(... etc ...)\n-----END RSA PRIVATE KEY-----"
}
`,
		Defaults: `
{
  "bindir"   : "/var/vcap/packages/bbr/bin",
}
`,
		Fields: []plugin.Field{
			plugin.Field{
				Mode:     "target",
				Name:     "director",
				Type:     "string",
				Title:    "BOSH Director",
				Help:     "The hostname or IP address of your BOSH Director.",
				Required: true,
			},
			plugin.Field{
				Mode:     "target",
				Name:     "username",
				Type:     "string",
				Title:    "Username",
				Help:     "Username to SSH to the BOSH Director as (director backups only).",
				Required: true,
			},
			plugin.Field{
				Mode:     "target",
				Name:     "key",
				Type:     "pem-rsa-pk",
				Title:    "Private Key",
				Help:     "RSA Private Key for the System user.",
				Required: true,
			},
			plugin.Field{
				Mode:    "target",
				Name:    "bindir",
				Type:    "abspath",
				Title:   "BBR bin/ Path",
				Help:    "The absolute path to the bin/ directory that contains the `bbr` command.",
				Default: "/var/vcap/packages/bbr/bin",
			},
			plugin.Field{
				Mode:    "target",
				Name:    "tmpdir",
				Type:    "abspath",
				Title:   "Temporary Directory",
				Help:    "The absolute path to a temporary directory (like /tmp) in which to stage backup files.",
				Default: "",
			},
		},
	}

	fmt.Fprintf(os.Stderr, "bbr plugin starting up...\n")
	plugin.Run(bbr)
}

type BBRPlugin plugin.PluginInfo

type details struct {
	BinDir     string
	TempDir    string
	Username   string
	Key        string
	Director   string
	Deployment string
	CACert     string
}

func (p BBRPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p BBRPlugin) Validate(endpoint plugin.ShieldEndpoint) error {
	fail := false

	if s, err := endpoint.StringValueDefault("bindir", DefaultBinDir); err != nil {
		fmt.Printf("@R{\u2717 bindir           %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 bindir}           @C{%s}\n", s)
	}

	if s, err := endpoint.StringValueDefault("tmpdir", os.TempDir()); err != nil {
		fmt.Printf("@R{\u2717 tmpdir           %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 tmpdir}           @C{%s}\n", s)
	}

	if s, err := endpoint.StringValue("director"); err != nil {
		fmt.Printf("@R{\u2717 director         %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 director}         @C{%s}\n", s)
	}

	if s, err := endpoint.StringValue("username"); err != nil {
		fmt.Printf("@R{\u2717 username  %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 username}  @C{%s}\n", s)
	}

	if s, err := endpoint.StringValue("key"); err != nil {
		fmt.Printf("@R{\u2717 key       %s}\n", err)
		fail = true
	} else {
		/* FIXME: validate that it's an RSA formatted key */
		lines := strings.Split(s, "\n")
		fmt.Printf("@G{\u2713 key}       <redacted>@C{%s}\n", lines[0])
		if len(lines) > 1 {
			for _, line := range lines[1:] {
				fmt.Printf("                  @C{%s}\n", line)
			}
		}
		fmt.Printf("</redacted>\n")
	}

	if fail {
		return fmt.Errorf("bbr: invalid configuration")
	}
	return nil
}

func persist(dir, contents string) (string, error) {
	f, err := ioutil.TempFile("", "shield-bbr-*")
	if err != nil {
		return "", err
	}

	if _, err := f.WriteString(contents); err != nil {
		return "", err
	}

	if err := f.Close(); err != nil {
		return "", err
	}

	return f.Name(), nil
}

func (p BBRPlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	var cmd *exec.Cmd

	bbr, err := getDetails(endpoint)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Setting up temporary workspace directory...\n")
	workspace, err := ioutil.TempDir(bbr.TempDir, "shield-bbr-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workspace)
	fmt.Fprintf(os.Stderr, "  workspace is '%s'\n", workspace)

	fmt.Fprintf(os.Stderr, "Changing working directory to workspace...\n")
	if err := os.Chdir(workspace); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Writing SSH Private Key to disk...\n")
	key, err := persist(bbr.TempDir, bbr.Key)
	if err != nil {
		return err
	}
	defer os.Remove(key)
	fmt.Fprintf(os.Stderr, "  wrote to '%s'\n", key)

	cmd = exec.Command(fmt.Sprintf("%s/bbr", bbr.BinDir),
		"director",
		"--host", bbr.Director,
		"--username", bbr.Username,
		"--private-key-path", key,
		"backup")

	fmt.Fprintf(os.Stderr, "\nRunning BRR CLI...\n")
	fmt.Fprintf(os.Stderr, "  %s/bbr director \\\n", bbr.BinDir)
	fmt.Fprintf(os.Stderr, "    --host %s \\\n", bbr.Director)
	fmt.Fprintf(os.Stderr, "    --username %s \\\n", bbr.Username)
	fmt.Fprintf(os.Stderr, "    --private-key-path %s \\\n", key)
	fmt.Fprintf(os.Stderr, "    backup\n\n\n")

	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	fmt.Fprintf(os.Stderr, "----------------------------------------------------------\n")
	err = cmd.Run()
	fmt.Fprintf(os.Stderr, "----------------------------------------------------------\n\n\n")
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Combining BBR output files into an uncompressed tarball...\n")
	archive := tar.NewWriter(os.Stdout)
	err = filepath.Walk(workspace, func(path string, info os.FileInfo, err error) error {
		fmt.Fprintf(os.Stderr, "  - analyzing %s ... ", path)
		rel, err := filepath.Rel(workspace, path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{FAILED}\n")
			return err
		}

		if !strings.HasPrefix(rel, bbr.Deployment) {
			fmt.Fprintf(os.Stderr, "skipping\n")
			return nil
		}

		fmt.Fprintf(os.Stderr, "INCLUDING\n")
		h, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		h.Name = rel
		archive.WriteHeader(h)

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		io.Copy(archive, f)
		f.Close()

		return nil
	})
	if err != nil {
		return err
	}

	archive.Close()
	return nil
}

func (p BBRPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	var cmd *exec.Cmd

	bbr, err := getDetails(endpoint)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Setting up temporary workspace directory...\n")
	workspace, err := ioutil.TempDir(bbr.TempDir, "shield-bbr-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workspace)
	fmt.Fprintf(os.Stderr, "  workspace is '%s'\n", workspace)

	fmt.Fprintf(os.Stderr, "Changing working directory to workspace...\n")
	if err := os.Chdir(workspace); err != nil {
		return err
	}

	archive := tar.NewReader(os.Stdin)
	for {
		h, err := archive.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		f, err := os.Create(h.Name)
		if err != nil {
			return err
		}
		io.Copy(f, archive)
		f.Close()
	}

	artifacts, err := filepath.Glob("*")
	if err != nil {
		return err
	}
	if len(artifacts) == 0 {
		return fmt.Errorf("Found no BBR artifacts in backup archive!")
	}
	if len(artifacts) > 1 {
		return fmt.Errorf("Found too many BBR artifacts (%d) in backup archive!", len(artifacts))
	}

	fmt.Fprintf(os.Stderr, "Writing SSH Private Key to disk...\n")
	key, err := persist(bbr.TempDir, bbr.Key)
	if err != nil {
		return err
	}
	defer os.Remove(key)
	fmt.Fprintf(os.Stderr, "  wrote to '%s'\n", key)

	cmd = exec.Command(fmt.Sprintf("%s/bbr", bbr.BinDir),
		"director",
		"--host", bbr.Director,
		"--username", bbr.Username,
		"--private-key-path", key,
		"restore",
		"--artifact-path", artifacts[0])

	fmt.Fprintf(os.Stderr, "\nRunning BRR CLI...\n")
	fmt.Fprintf(os.Stderr, "  %s/bbr director \\\n", bbr.BinDir)
	fmt.Fprintf(os.Stderr, "    --host %s \\\n", bbr.Director)
	fmt.Fprintf(os.Stderr, "    --username %s \\\n", bbr.Username)
	fmt.Fprintf(os.Stderr, "    --private-key-path %s \\\n", key)
	fmt.Fprintf(os.Stderr, "    restore \\\n")
	fmt.Fprintf(os.Stderr, "    --artifact-path %s \\\n\n\n", artifacts[0])

	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	fmt.Fprintf(os.Stderr, "----------------------------------------------------------\n")
	err = cmd.Run()
	fmt.Fprintf(os.Stderr, "----------------------------------------------------------\n\n\n")
	return err
}

func (p BBRPlugin) Store(endpoint plugin.ShieldEndpoint) (string, int64, error) {
	return "", 0, plugin.UNIMPLEMENTED
}

func (p BBRPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p BBRPlugin) Purge(endpoint plugin.ShieldEndpoint, key string) error {
	return plugin.UNIMPLEMENTED
}

func getDetails(endpoint plugin.ShieldEndpoint) (*details, error) {
	bin, err := endpoint.StringValueDefault("bindir", DefaultBinDir)
	if err != nil {
		return nil, err
	}

	tmp, err := endpoint.StringValueDefault("tmpdir", os.TempDir())
	if err != nil {
		return nil, err
	}

	director, err := endpoint.StringValue("director")
	if err != nil {
		return nil, err
	}

	username, err := endpoint.StringValue("username")
	if err != nil {
		return nil, err
	}

	key, err := endpoint.StringValue("key")
	if err != nil {
		return nil, err
	}

	return &details{
		BinDir:   bin,
		TempDir:  tmp,
		Director: director,
		Username: username,
		Key:      key,
	}, nil
}
