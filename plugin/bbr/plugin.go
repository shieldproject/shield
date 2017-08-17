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
//					"bbr_type": "director",																													# set the type to director
//					"bbr_host": "192.168.50.6",																											# the ip of the director
//					"bbr_sshusername": "jumpbox",																 										# the ssh username for the director
//					"bbr_privatekey": "-----BEGIN RSA PRIVATE mycert -----END RSA PRIVATE KEY-----" # a single line private key of the director
//    }
//
// Deployment
//
//    {
//						"bbr_type": "deployment",																									 	 # set the type to deployment
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
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/backup"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/standalone"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/mholt/archiver"
	"github.com/starkandwayne/goutils/ansi"
	. "github.com/starkandwayne/shield/plugin"
)

func main() {
	// Create an object representing this plugin, which is a type conforming to the Plugin interface
	bbr := BbrPlugin{
		// give it some authorship info
		meta: PluginInfo{
			Name:    "Bbr Plugin",
			Author:  "Stark & Wayne",
			Version: "1.0.0",
			Features: PluginFeatures{
				Target: "yes",
				Store:  "no",
			},
			Example: `
			Director
			   {
								"bbr_type": "director",
								"bbr_host": "192.168.50.6",
								"bbr_sshusername": "jumpbox",
								"bbr_privatekey": "-----BEGIN RSA PRIVATE my single line certs -----END RSA PRIVATE KEY-----"
			   }

			Deployment
			   {
									"bbr_type": "deployment",
									"bbr_username": "admin",
									"bbr_password": "c1oudc0w",
									"bbr_target": "192.168.50.6",
									"bbr_deployment": "cf",
									"bbr_cacert": "-----BEGIN CERTIFICATE----- my single line certs -----END CERTIFICATE-----"
			   }
	`,
			Defaults: `
	{
	  "bbr_host"  : "192.168.50.6",
	  "bbr_sshusername": "jumpbox"
		"bbr_dusername": "admin"
	}
	`,
		},
	}

	fmt.Fprintf(os.Stderr, "bbr plugin starting up...\n")
	// Run the plugin - the plugin framework handles all arg parsing, exit handling, error/debug formatting for you
	Run(bbr)
}

// Define my BbrPlugin type
type BbrPlugin struct {
	meta        PluginInfo // needs a place to store metadata
	Type        string     // select director or deployment
	Host        string     // SSH username of the Director
	PrivateKey  string     // used for director
	SSHUsername string     // used for director
	Username    string     // used for deployment
	Password    string     // used for deployment
	Target      string     // used for deployment
	Deployment  string     // used for deployment
	CaCert      string     // used for deployment
}

const (
	directorType   = "director"
	deploymentType = "deployment"
)

// This function should be used to return the plugin's PluginInfo, however you decide to implement it
func (p BbrPlugin) Meta() PluginInfo {
	return p.meta
}

// Called to validate endpoints from the command line
func (p BbrPlugin) Validate(endpoint ShieldEndpoint) error {
	var (
		s    string
		err  error
		fail bool
	)

	s, err = endpoint.StringValue("bbr_type")
	if err != nil {
		ansi.Printf("@R{\u2717 bbr_type   %s}\n", err)
		return fmt.Errorf("bbr: invalid type: exepected 'director', 'deployment'")
	}
	switch s, _ = endpoint.StringValue("bbr_type"); s {
	case deploymentType:
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
	case directorType:
		s, err = endpoint.StringValue("bbr_host")
		if err != nil {
			ansi.Printf("@R{\u2717 bbr_host   %s}\n", err)
			fail = true
		} else {
			ansi.Printf("@G{\u2713 bbr_host}  @C{%s}\n", s)
		}
		s, err = endpoint.StringValue("bbr_privatekey")
		if err != nil {
			ansi.Printf("@R{\u2717 bbr_privatekey   %s}\n", err)
			fail = true
		} else {
			ansi.Printf("@G{\u2713 bbr_privatekey}  @C{%s}\n", s)
		}
		s, err = endpoint.StringValue("bbr_sshusername")
		if err != nil {
			ansi.Printf("@R{\u2717 bbr_sshusername   %s}\n", err)
			fail = true
		} else {
			ansi.Printf("@G{\u2713 bbr_sshusername}  @C{%s}\n", s)
		}
	default:
		ansi.Printf("@R{\u2717 bbr_type   %s (Expected `director` or `deployment`)}\n", err)
		fail = true
	}

	if fail {
		return fmt.Errorf("bbr: invalid configuration")
	}
	return nil
}

// jhunt proposes to get rid of this function and just put the code in our makeDirectorBackuper func
// because of the issues with the defer. otherwise we are going to make it to complex
func tmpfile(body string) (path string, err error) {
	// write privatekey from string
	file, err := ioutil.TempFile("", "test")
	if err != nil {
		return "", err
	}
	// defer os.Remove(file.Name()) // clean up
	DEBUG("writing: `%s`", body)
	if _, err := file.WriteString(body); err != nil {
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return file.Name(), nil
}

// backup the director
func makeDirectorBackuper(bbr *BbrPlugin, logger boshlog.Logger) (*orchestrator.Backuper, error) {
	privateKeyPath, err := tmpfile(bbr.PrivateKey)
	if err != nil {
		return nil, err
	}
	deploymentManager := standalone.NewDeploymentManager(logger,
		bbr.Host,
		bbr.SSHUsername,
		privateKeyPath,
		instance.NewJobFinder(logger),
		ssh.NewConnection,
	)

	return orchestrator.NewBackuper(backup.BackupDirectoryManager{}, logger, deploymentManager, time.Now), nil
}

func makeDirectorRestorer(bbr *BbrPlugin, logger boshlog.Logger) (*orchestrator.Restorer, error) {
	privateKeyPath, err := tmpfile(bbr.PrivateKey)
	if err != nil {
		return nil, err
	}
	deploymentManager := standalone.NewDeploymentManager(logger,
		bbr.Host,
		bbr.Username,
		privateKeyPath,
		instance.NewJobFinder(logger),
		ssh.NewConnection,
	)
	return orchestrator.NewRestorer(backup.BackupDirectoryManager{}, logger, deploymentManager), nil
}

func makeDeploymentBackuper(bbr *BbrPlugin, logger boshlog.Logger) (*orchestrator.Backuper, error) {
	caCertPath, err := tmpfile(bbr.CaCert)
	if err != nil {
		return nil, err
	}
	boshClient, err := bosh.BuildClient(bbr.Target, bbr.Username, bbr.Password, caCertPath, logger)
	if err != nil {
		return nil, err
	}

	deploymentManager := bosh.NewDeploymentManager(boshClient, logger, false)

	return orchestrator.NewBackuper(backup.BackupDirectoryManager{}, logger, deploymentManager, time.Now), nil
}

func makeDeploymentRestorer(bbr *BbrPlugin, logger boshlog.Logger) (*orchestrator.Restorer, error) {
	caCertPath, err := tmpfile(bbr.CaCert)
	if err != nil {
		return nil, err
	}
	boshClient, err := bosh.BuildClient(bbr.Target, bbr.Username, bbr.Password, caCertPath, logger)
	if err != nil {
		return nil, err
	}

	deploymentManager := bosh.NewDeploymentManager(boshClient, logger, false)

	return orchestrator.NewRestorer(backup.BackupDirectoryManager{}, logger, deploymentManager), nil
}

// // Called when you want to back data up. Examine the ShieldEndpoint passed in, and perform actions accordingly
func (p BbrPlugin) Backup(endpoint ShieldEndpoint) error {
	bbr, err := BbrConnectionInfo(endpoint)
	logger := boshlog.NewWriterLogger(boshlog.LevelInfo, os.Stderr, os.Stderr)

	if err != nil {
		return err
	}

	switch bbr.Type {
	case directorType:
		// backup director
		backuper, err := makeDirectorBackuper(bbr, logger)
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
		// backup with bbr
		err = orchestrator.ConvertErrors(backuper.Backup("director"))
		if err != nil {
			return err
		}
		// create path so we can use it this path to zip the directory
		backups, err := filepath.Glob(path.Join(tmpdir, "director*"))
		if err != nil {
			return err
		}
		DEBUG("PATHS: `%s`", backups)
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

	case deploymentType:
		// backup director
		backuper, err := makeDeploymentBackuper(bbr, logger)
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
		//TODO: join string can't this be a onliner?
		s := []string{bbr.Deployment, "*"}
		joined := strings.Join(s, "")
		// backup with bbr
		err = orchestrator.ConvertErrors(backuper.Backup(bbr.Deployment))
		if err != nil {
			return err
		}
		// create path so we can use it this path to zip the directory
		backups, err := filepath.Glob(path.Join(tmpdir, joined))
		if err != nil {
			return err
		}
		DEBUG("PATHS: `%s`", backups)
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

	}
	return nil
}

// Called when you want to restore data Examine the ShieldEndpoint passed in, and perform actions accordingly
func (p BbrPlugin) Restore(endpoint ShieldEndpoint) error {
	bbr, err := BbrConnectionInfo(endpoint)
	logger := boshlog.NewWriterLogger(boshlog.LevelInfo, os.Stderr, os.Stderr)
	if err != nil {
		return err
	}

	switch bbr.Type {
	case directorType:
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
		backups, err := filepath.Glob(path.Join(tmpdir, "director*"))
		if err != nil {
			return err
		}
		restorer, err := makeDirectorRestorer(bbr, logger)
		if err != nil {
			return err
		}
		err = orchestrator.ConvertErrors(restorer.Restore("director", backups[0]))
		if err != nil {
			return err
		}
		return nil

	case deploymentType:
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
		restorer, err := makeDeploymentRestorer(bbr, logger)
		if err != nil {
			return err
		}
		err = orchestrator.ConvertErrors(restorer.Restore(bbr.Deployment, backups[0]))
		if err != nil {
			return err
		}
		return nil
	}

	return nil
}

// Called when you want to store backup data. Examine the ShieldEndpoint passed in, and perform actions accordingly
func (p BbrPlugin) Store(endpoint ShieldEndpoint) (string, error) {
	return "", UNIMPLEMENTED
}

// Called when you want to retreive backup data. Examine the ShieldEndpoint passed in, and perform actions accordingly
func (p BbrPlugin) Retrieve(endpoint ShieldEndpoint, file string) error {
	return UNIMPLEMENTED
}

func (p BbrPlugin) Purge(endpoint ShieldEndpoint, key string) error {
	return UNIMPLEMENTED
}

//That's all there is to writing a plugin. If your plugin doesn't need to implement Store/Retrieve, or Backup/Restore,
// Define the functions, and have them return plugin.UNIMPLEMENTED

func BbrConnectionInfo(endpoint ShieldEndpoint) (*BbrPlugin, error) {
	bbrType, err := endpoint.StringValue("bbr_type")
	if err != nil {
		return nil, err
	}
	DEBUG("BBRTYPE: '%s'", bbrType)

	switch bbrType {
	case directorType:
		host, err := endpoint.StringValue("bbr_host")
		if err != nil {
			return nil, err
		}
		DEBUG("BBRHOST: '%s'", host)

		privatekey, err := endpoint.StringValue("bbr_privatekey")
		if err != nil {
			return nil, err
		}
		DEBUG("BBRPRIVATEKEY: '%s'", privatekey)

		sshusername, err := endpoint.StringValue("bbr_sshusername")
		if err != nil {
			return nil, err
		}
		DEBUG("BBRSSHUSERNAME: '%s'", sshusername)

		return &BbrPlugin{
			Type:        bbrType,
			Host:        host,
			PrivateKey:  privatekey,
			SSHUsername: sshusername,
		}, nil
	case deploymentType:
		username, err := endpoint.StringValue("bbr_username")
		if err != nil {
			return nil, err
		}
		DEBUG("BBRUSERNAME: '%s'", username)

		password, err := endpoint.StringValue("bbr_password")
		if err != nil {
			return nil, err
		}
		DEBUG("BBRPASSWORD: '%s'", password)

		target, err := endpoint.StringValueDefault("bbr_target", "")
		if err != nil {
			return nil, err
		}
		DEBUG("BBRTARGET: '%s'", target)

		deployment, err := endpoint.StringValueDefault("bbr_deployment", "")
		if err != nil {
			return nil, err
		}
		DEBUG("BBRDEPLOYMENT: '%s'", deployment)

		cacert, err := endpoint.StringValueDefault("bbr_cacert", "")
		if err != nil {
			return nil, err
		}
		DEBUG("BBRCACERT: '%s'", cacert)

		return &BbrPlugin{
			Type:       bbrType,
			Username:   username,
			Password:   password,
			Target:     target,
			Deployment: deployment,
			CaCert:     cacert,
		}, nil
	}
	return &BbrPlugin{}, nil
}
