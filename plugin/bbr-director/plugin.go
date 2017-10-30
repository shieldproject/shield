// The `bbr` plugin for SHIELD
// backup/restore plugin for all your bbr enabled deployments and directors.
//
// PLUGIN FEATURES
//
// This plugin implements functionality suitable for use with the following
// SHIELD Job components:
//
//   Target: yes
//   Store:  no
//
// PLUGIN CONFIGURATION
//
// The endpoint configuration passed to this plugin is used to identify
// which bosh director instance to back up, and how to connect to it. Your
// endpoint JSON should look something like this:
// Director
//    {
//					"bbr_host": "192.168.50.6",																											# the ip of the director
//					"bbr_sshusername": "jumpbox",																 										# the ssh username for the director
//					"bbr_privatekey": "-----BEGIN RSA PRIVATE mycert -----END RSA PRIVATE KEY-----" # a single line private key of the director
//    }
//
// BACKUP DETAILS
//
// The `bbr` plugin lets you backup your bbr enabled deployment or bosh director
//
// RESTORE DETAILS
//
// The `bbr` plugin will also restore your complete deployment or bosh director.
//
// DEPENDENCIES
//
//
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
	// Create an object representing this plugin, which is a type conforming to the Plugin interface
	bbr := BbrPlugin{
		// give it some authorship info
		Name:    "BOSH BBR Director Plugin",
		Author:  "Stark & Wayne",
		Version: "1.0.0",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
		Example: `
			// example to make online keys "sed ':a;N;$!ba;s/\n/\\n/g' state/ssh.key"
			Director
			   {
								"bbr_bindir"   : "/path/to/pg/bin",
								"bbr_host": "192.168.50.6",
								"bbr_sshusername": "jumpbox",
								"bbr_privatekey": "-----BEGIN RSA PRIVATE my single line certs -----END RSA PRIVATE KEY-----"
			   }
	`,
		Defaults: `
	{
		"bbr_bindir": "/var/vcap/packages/bbr/bin",
	  "bbr_host": "192.168.50.6",
	  "bbr_sshusername": "jumpbox"
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
	// Run the plugin - the plugin framework handles all arg parsing, exit handling, error/debug formatting for you
	plugin.Run(bbr)
}

type BbrPlugin plugin.PluginInfo

type BbrConnectionInfo struct {
	Bin         string // location of the bbr binary
	Host        string // ip of the Director
	PrivateKey  string // used for director
	SSHUsername string // used for director
}

// This function should be used to return the plugin's PluginInfo, however you decide to implement it
func (p BbrPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

// Called to validate endpoints from the command line
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
		fmt.Printf("@G{\u2713 bbr_privatekey}  @C{%s}\n", s)
	}
	s, err = endpoint.StringValue("bbr_sshusername")
	if err != nil {
		fmt.Printf("@R{\u2717 bbr_sshusername   %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 bbr_sshusername}  @C{%s}\n", s)
	}
	if fail {
		return fmt.Errorf("bbr: invalid configuration")
	}
	return nil
}

func tmpfile(body string) (path string, err error) {
	// write privatekey from string
	file, err := ioutil.TempFile("", "test")
	if err != nil {
		return "", err
	}
	// defer os.Remove(file.Name()) // clean up
	plugin.DEBUG("writing: `%s`", body)
	if _, err := file.WriteString(body); err != nil {
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return file.Name(), nil
}

// // Called when you want to back data up. Examine the ShieldEndpoint passed in, and perform actions accordingly
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
	// execut bbr cli
	cmd := exec.Command(fmt.Sprintf("%s/bbr", bbr.Bin), "director", "--host", bbr.Host, "--username", bbr.SSHUsername, "--private-key-path", privateKeyPath, "backup")
	plugin.DEBUG("Executing: `%s`", cmd)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}
	// create path so we can use it this path to zip the directory
	backups, err := filepath.Glob(path.Join(tmpdir, fmt.Sprintf("%s*", bbr.Host)))
	if err != nil {
		return err
	}
	plugin.DEBUG("PATHS: `%s`", backups)
	// Write our backuped directory to zip
	tmpfile, err := ioutil.TempFile("", ".zip")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name())
	err = archiver.Zip.Make(tmpfile.Name(), backups)
	if err != nil {
		return err
	}
	// open zip for reader
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

// Called when you want to restore data Examine the ShieldEndpoint passed in, and perform actions accordingly
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
	// Write our backuped directory to zip
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
	// execut bbr cli
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

// Called when you want to store backup data. Examine the ShieldEndpoint passed in, and perform actions accordingly
func (p BbrPlugin) Store(endpoint plugin.ShieldEndpoint) (string, int64, error) {
	return "", 0, plugin.UNIMPLEMENTED
}

// Called when you want to retreive backup data. Examine the ShieldEndpoint passed in, and perform actions accordingly
func (p BbrPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p BbrPlugin) Purge(endpoint plugin.ShieldEndpoint, key string) error {
	return plugin.UNIMPLEMENTED
}

//That's all there is to writing a plugin. If your plugin doesn't need to implement Store/Retrieve, or Backup/Restore,
// Define the functions, and have them return plugin.UNIMPLEMENTED

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
