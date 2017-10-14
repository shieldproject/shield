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
// Deployment
//
//    {
//						"bbr_username": "admin",																										 # the username of the director
//						"bbr_password": "c1oudc0w",																						   		 # the password of the director
//						"bbr_target": "192.168.50.6",																								 # the director ip
//						"bbr_deployment": "cf",																											 # the deployment name
//						"bbr_cacert": "-----BEGIN CERTIFICATE----- my cert -----END CERTIFICATE----- # a single line certificate of you bosh instance
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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver"
	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/plugin"
)

func main() {
	// Create an object representing this plugin, which is a type conforming to the Plugin interface
	bbr := BbrPlugin{
		// give it some authorship info
		Name:    "BOSH BBR Deployment Plugin",
		Author:  "Stark & Wayne",
		Version: "1.0.0",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
		Example: `
			// example to make online keys "sed ':a;N;$!ba;s/\n/\\n/g' state/ssh.key"
			Deployment
			   {
									"bbr_bindir"   : "/path/to/pg/bin",
									"bbr_username": "admin",
									"bbr_password": "c1oudc0w",
									"bbr_target": "192.168.50.6",
									"bbr_deployment": "cf",
									"bbr_cacert": "-----BEGIN CERTIFICATE----- my single line certs -----END CERTIFICATE-----"
			   }
	`,
		Defaults: `
	{
		"bbr_bindir": "/var/vcap/packages/bbr/bin",
	  "bbr_target": "192.168.50.6",
		"bbr_username": "admin"
	}
	`,
		Fields: []plugin.Field{
			plugin.Field{
				Mode:     "target",
				Name:     "bbr_target",
				Type:     "string",
				Title:    "BOSH Director Host",
				Help:     "The hostname or IP address of your BOSH director.",
				Required: true,
			},
			plugin.Field{
				Mode:     "target",
				Name:     "bbr_username",
				Type:     "string",
				Title:    "BOSH Director username",
				Help:     "Username to authenticate to the BOSH director.",
				Required: true,
			},
			plugin.Field{
				Mode:     "target",
				Name:     "bbr_password",
				Type:     "password",
				Title:    "BOSH Director Password",
				Help:     "Password to authenticate to the BOSH director.",
				Required: true,
			},
			plugin.Field{
				Mode:     "target",
				Name:     "bbr_cacert",
				Type:     "string",
				Title:    "BOSH director vm private ssh key",
				Help:     "CaCert for BOSH director, in a single line, use `sed ':a;N;$!ba;s/\n/\\n/g' ca.pem` to create one.",
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
	Bin        string // location of the bbr binary
	Username   string // used for deployment
	Password   string // used for deployment
	Target     string // used for deployment
	Deployment string // used for deployment
	CaCert     string // used for deployment
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
	s, err = endpoint.StringValue("bbr_deployment")
	if err != nil {
		ansi.Printf("@R{\u2717 bbr_deployment   %s}\n", err)
		fail = true
	} else {
		ansi.Printf("@G{\u2713 bbr_deployment}  @C{%s}\n", s)
	}
	s, err = endpoint.StringValue("bbr_target")
	if err != nil {
		ansi.Printf("@R{\u2717 bbr_target   %s}\n", err)
		fail = true
	} else {
		ansi.Printf("@G{\u2713 bbr_target}  @C{%s}\n", s)
	}
	s, err = endpoint.StringValue("bbr_username")
	if err != nil {
		ansi.Printf("@R{\u2717 bbr_username   %s}\n", err)
		fail = true
	} else {
		ansi.Printf("@G{\u2713 bbr_username}  @C{%s}\n", s)
	}
	s, err = endpoint.StringValue("bbr_password")
	if err != nil {
		ansi.Printf("@R{\u2717 bbr_password   %s}\n", err)
		fail = true
	} else {
		ansi.Printf("@G{\u2713 bbr_password}  @C{%s}\n", s)
	}
	s, err = endpoint.StringValue("bbr_cacert")
	if err != nil {
		ansi.Printf("@R{\u2717 bbr_cacert   %s}\n", err)
		fail = true
	} else {
		ansi.Printf("@G{\u2713 bbr_cacert}  @C{%s}\n", s)
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
	caCertPath, err := tmpfile(bbr.CaCert)
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

	// execut bbr cli
	cmd := exec.Command(fmt.Sprintf("%s/bbr", bbr.Bin), "deployment", "--target", bbr.Target, "--username", bbr.Username, "--password", bbr.Password, "--deployment", bbr.Deployment, "--ca-cert", caCertPath, "backup")
	plugin.DEBUG("Executing: `%s`", cmd)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}

	// //TODO: join string can't this be a onliner?
	s := []string{bbr.Deployment, "*"}
	joined := strings.Join(s, "")

	// create path so we can use it this path to zip the directory
	backups, err := filepath.Glob(path.Join(tmpdir, joined))
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

// Called when you want to restore data Examine the plugin.ShieldEndpoint passed in, and perform actions accordingly
func (p BbrPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	bbr, err := BConnectionInfo(endpoint)
	if err != nil {
		return err
	}
	caCertPath, err := tmpfile(bbr.CaCert)
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
	//TODO: join string can't this be a onliner?
	s := []string{bbr.Deployment, "*"}
	joined := strings.Join(s, "")
	backups, err := filepath.Glob(path.Join(tmpdir, joined))
	if err != nil {
		return err
	}
	// execut bbr cli
	cmd := exec.Command(fmt.Sprintf("%s/bbr", bbr.Bin), "deployment", "--target", bbr.Target, "--username", bbr.Username, "--password", bbr.Password, "--deployment", bbr.Deployment, "--ca-cert", caCertPath, "restore", "--artifact-path", backups[0])
	plugin.DEBUG("Executing: `%s`", cmd)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

// Called when you want to store backup data. Examine the plugin.ShieldEndpoint passed in, and perform actions accordingly
func (p BbrPlugin) Store(endpoint plugin.ShieldEndpoint) (string, int64, error) {
	return "", 0, plugin.UNIMPLEMENTED
}

// Called when you want to retreive backup data. Examine the plugin.ShieldEndpoint passed in, and perform actions accordingly
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

	username, err := endpoint.StringValue("bbr_username")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("BBRUSERNAME: '%s'", username)

	password, err := endpoint.StringValue("bbr_password")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("BBRPASSWORD: '%s'", password)

	target, err := endpoint.StringValueDefault("bbr_target", "")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("BBRTARGET: '%s'", target)

	deployment, err := endpoint.StringValueDefault("bbr_deployment", "")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("BBRDEPLOYMENT: '%s'", deployment)

	cacert, err := endpoint.StringValueDefault("bbr_cacert", "")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("BBRCACERT: '%s'", cacert)

	return &BbrConnectionInfo{
		Bin:        bin,
		Username:   username,
		Password:   password,
		Target:     target,
		Deployment: deployment,
		CaCert:     cacert,
	}, nil
}
