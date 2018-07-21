package main

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	fmt "github.com/jhunt/go-ansi"
	"github.com/mholt/archiver"
	"github.com/starkandwayne/shield/plugin"
)

func main() {
	bbr := BbrPlugin{
		Name:    "BOSH BBR Director Plugin",
		Author:  "Stark & Wayne",
		Version: "1.0.0",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
		Example: `
// example to make online keys "sed ':a;N;$!ba;s/\n/\\n/g' state/ssh.key"
{
  "bbr_bindir"      : "/path/to/pg/bin",
  "bbr_host"        : "192.168.50.6",
  "bbr_sshusername" : "jumpbox",
  "bbr_privatekey"  : "-----BEGIN RSA PRIVATE my single line certs -----END RSA PRIVATE KEY-----"
}
`,
		Defaults: `
{
  "bbr_bindir"      : "/var/vcap/packages/bbr/bin",
  "bbr_host"        : "192.168.50.6",
  "bbr_sshusername" : "jumpbox"
}
`,
		Fields: []plugin.Field{
			plugin.Field{
				Mode:     "target",
				Name:     "bbr_host",
				Type:     "string",
				Title:    "BOSH Director Host",
				Help:     "The hostname or IP address of your BOSH director.",
				Required: true,
			},
			plugin.Field{
				Mode:     "target",
				Name:     "bbr_sshusername",
				Type:     "string",
				Title:    "BOSH Director VM ssh username",
				Help:     "Username to authenticate to vm of the bosh director.",
				Required: true,
			},
			plugin.Field{
				Mode:     "target",
				Name:     "bbr_privatekey",
				Type:     "string",
				Title:    "BOSH director vm private ssh key",
				Help:     "Private ssh key for vm of the BOSH director, in a single line, use `sed ':a;N;$!ba;s/\n/\\n/g' ssh.key` to create one.",
				Required: true,
			},
			plugin.Field{
				Mode:    "target",
				Name:    "bbr_bindir",
				Type:    "abspath",
				Title:   "Path to BBR bin/ directory",
				Help:    "The absolute path to the bin/ directory that contains the `bbr` command.",
				Default: "/var/vcap/packages/bbr/bin",
			},
		},
	}

	fmt.Fprintf(os.Stderr, "bbr plugin starting up...\n")
	plugin.Run(bbr)
}

type BbrPlugin plugin.PluginInfo

type BbrConnectionInfo struct {
	Bin         string // location of the bbr binary
	Host        string // ip of the Director
	PrivateKey  string // used for director
	SSHUsername string // used for director
}

func (p BbrPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p BbrPlugin) Validate(endpoint plugin.ShieldEndpoint) error {
	var (
		s    string
		err  error
		fail bool
	)
	s, err = endpoint.StringValue("bbr_host")
	if err != nil {
		fmt.Printf("@R{\u2717 bbr_host   %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 bbr_host}  @C{%s}\n", s)
	}
	s, err = endpoint.StringValue("bbr_privatekey")
	if err != nil {
		fmt.Printf("@R{\u2717 bbr_privatekey   %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 bbr_privatekey}  @C{%s}\n", plugin.Redact(s))
	}
	s, err = endpoint.StringValue("bbr_sshusername")
	if err != nil {
		fmt.Printf("@R{\u2717 bbr_sshusername   %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 bbr_sshusername}  @C{%s}\n", plugin.Redact(s))
	}
	if fail {
		return fmt.Errorf("bbr: invalid configuration")
	}
	return nil
}

func tmpfile(body string) (path string, err error) {
	file, err := ioutil.TempFile("", "test")
	if err != nil {
		return "", err
	}

	plugin.DEBUG("writing: `%s`", body)
	if _, err := file.WriteString(body); err != nil {
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return file.Name(), nil
}

func (p BbrPlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	bbr, err := BConnectionInfo(endpoint)
	if err != nil {
		return err
	}
	privateKeyPath, err := tmpfile(bbr.PrivateKey)
	if err != nil {
		return err
	}
	tmpdir, err := ioutil.TempDir("", "bbr")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir)
	err = os.Chdir(tmpdir)
	if err != nil {
		return err
	}
	// TODO: are we going to do pre-backup-check and or clean if there is an unclean director or should this be a manuall step?
	cmd := exec.Command(fmt.Sprintf("%s/bbr", bbr.Bin), "director", "--host", bbr.Host, "--username", bbr.SSHUsername, "--private-key-path", privateKeyPath, "backup")
	plugin.DEBUG("Executing: `%s`", cmd)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}

	backups, err := filepath.Glob(path.Join(tmpdir, fmt.Sprintf("%s*", bbr.Host)))
	if err != nil {
		return err
	}
	plugin.DEBUG("PATHS: `%s`", backups)

	tmpfile, err := ioutil.TempFile("", ".zip")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name())
	err = archiver.Zip.Make(tmpfile.Name(), backups)
	if err != nil {
		return err
	}

	reader, err := os.Open(tmpfile.Name())
	if err != nil {
		return err
	}
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return err
	}
	return nil
}

func (p BbrPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	bbr, err := BConnectionInfo(endpoint)
	if err != nil {
		return err
	}
	privateKeyPath, err := tmpfile(bbr.PrivateKey)
	if err != nil {
		return err
	}
	tmpdir, err := ioutil.TempDir("", "bbr")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir)

	tmpfile, err := ioutil.TempFile("", ".zip")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name())
	_, err = io.Copy(tmpfile, os.Stdin)
	if err != nil {
		return err
	}
	err = archiver.Zip.Open(tmpfile.Name(), tmpdir)
	if err != nil {
		return err
	}
	backups, err := filepath.Glob(path.Join(tmpdir, fmt.Sprintf("%s*", bbr.Host)))
	plugin.DEBUG("PATHS: `%s`", backups)
	if err != nil {
		return err
	}

	cmd := exec.Command(fmt.Sprintf("%s/bbr", bbr.Bin), "director", "--host", bbr.Host, "--username", bbr.SSHUsername, "--private-key-path", privateKeyPath, "restore", "--artifact-path", backups[0])
	plugin.DEBUG("Executing: `%s`", cmd)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func (p BbrPlugin) Store(endpoint plugin.ShieldEndpoint) (string, int64, error) {
	return "", 0, plugin.UNIMPLEMENTED
}

func (p BbrPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p BbrPlugin) Purge(endpoint plugin.ShieldEndpoint, key string) error {
	return plugin.UNIMPLEMENTED
}


func BConnectionInfo(endpoint plugin.ShieldEndpoint) (*BbrConnectionInfo, error) {
	bin, err := endpoint.StringValueDefault("bbr_bindir", "/var/vcap/packages/bbr/bin")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("BBRBINDIR: '%s'", bin)

	host, err := endpoint.StringValue("bbr_host")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("BBRHOST: '%s'", host)

	privatekey, err := endpoint.StringValue("bbr_privatekey")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("BBRPRIVATEKEY: '%s'", privatekey)

	sshusername, err := endpoint.StringValue("bbr_sshusername")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("BBRSSHUSERNAME: '%s'", sshusername)

	return &BbrConnectionInfo{
		Bin:         bin,
		Host:        host,
		PrivateKey:  privatekey,
		SSHUsername: sshusername,
	}, nil
}
