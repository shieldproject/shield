package main

/*

This is a generic and not terribly helpful plugin. However, it shows the basics
of what is needed in a backup plugin, and how they execute.

*/

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/pivotal-cf/bosh-backup-and-restore/backup"
	"github.com/pivotal-cf/bosh-backup-and-restore/bosh"
	"github.com/pivotal-cf/bosh-backup-and-restore/instance"
	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
	"github.com/pivotal-cf/bosh-backup-and-restore/ssh"
	"github.com/pivotal-cf/bosh-backup-and-restore/standalone"
	"github.com/starkandwayne/goutils/ansi"
	. "github.com/starkandwayne/shield/plugin"

	"github.com/mholt/archiver"
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
	{
	  "bbr_host"       : "username",   # REQUIRED
	  "bbr_sshusername"   : "password",   # REQUIRED
	  "bbr_privatekey" : "mykey",      # REQUIRED
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

	// s, err = endpoint.StringValue("bbr_type")
	// if err != nil {
	// 	ansi.Printf("@R{\u2717 bbr_type   %s}\n", err)
	// 	fail = true
	// } else {
	// 	if matched, _ := regexp.MatchString("deployment|director", s); matched {
	// 		ansi.Printf("@G{\u2713 bbr_type}  @C{%s}\n", s)
	// 	} else {
	// 		ansi.Printf("@R{\u2717 bbr_type   %s (Expected `director` or `deployment`)}\n", err)
	// 		return fmt.Errorf("bbr: invalid type: exepected 'director', 'deployment'")
	// 	}
	// }

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
	case directorType:
		s, err = endpoint.StringValue("bbr_host")
		if err != nil {
			ansi.Printf("@R{\u2717 bbr_host   %s}\n", err)
			fail = true
		} else {
			ansi.Printf("@G{\u2713 bbr_host}  @C{%s}\n", s)
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
	return orchestrator.NewResttargetUrlorer(backup.BackupDirectoryManager{}, logger, deploymentManager), nil
}

func makeDeploymentBackuper(bbr *BbrPlugin, logger boshlog.Logger) (*orchestrator.Backuper, error) {
	caCertPath, err := tmpfile(bbr.CaCert)
	if err != nil {
		return nil, err
	}
	boshClient, err := bosh.BuildClient(bbr.Target, bbr.Username, password, caCert, logger)
	if err != nil {
		return nil, err
	}

	return bosh.NewDeploymentManager(boshClient, logger, downloadManifest), nil
	deploymentManager, err := newDeploymentManager(
		bbr.Target,
		bbr.Username,
		bbr.Password,
		caCertPath,
		logger,
		false,
	)
	if err != nil {
		return nil, err
	}

	return orchestrator.NewBackuper(backup.BackupDirectoryManager{}, logger, deploymentManager, time.Now), nil
}

// // Called when you want to back data up. Examine the ShieldEndpoint passed in, and perform actions accordingly
func (p BbrPlugin) Backup(endpoint ShieldEndpoint) error {
	bbr, err := BbrConnectionInfo(endpoint)
	logger := boshlog.NewWriterLogger(boshlog.LevelInfo, os.Stderr, os.Stderr)

	if err != nil {
		return err
	}

	if bbr.Type == "director" {
		// backup director
		backuper, err := makeDirectorBackuper(bbr, logger)
		if err != nil {
			return err
		}
		tmp_dir, err := ioutil.TempDir("", "bbr")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmp_dir)
		err = os.Chdir(tmp_dir)
		if err != nil {
			return err
		}
		DEBUG("BACKUP 1")
		// backup with bbr
		err = orchestrator.ConvertErrors(backuper.Backup("director"))
		if err != nil {
			return err
		}
		DEBUG("BACKUP 2")
		// create path
		backups, err := filepath.Glob(path.Join(tmp_dir, "director*"))
		if err != nil {
			return err
		}
		DEBUG("PATHS: `%s`", backups)
		// Write our backuped directory to zip
		tmp_file, err := ioutil.TempFile("", ".zip")
		if err != nil {
			return err
		}
		defer os.Remove(tmp_file.Name())
		err = archiver.Zip.Make(tmp_file.Name(), backups)
		if err != nil {
			return err
		}
		// open zip for reader
		reader, err := os.Open(tmp_file.Name())
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

	tmp_dir, err := ioutil.TempDir("", "bbr")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp_dir)
	// Write our backuped directory to zip
	tmp_file, err := ioutil.TempFile("", ".zip")
	if err != nil {
		return err
	}
	defer os.Remove(tmp_file.Name())
	_, err = io.Copy(tmp_file, os.Stdin)
	if err != nil {
		return err
	}
	err = archiver.Zip.Open(tmp_file.Name(), tmp_dir)
	if err != nil {
		return err
	}
	backups, err := filepath.Glob(path.Join(tmp_dir, "director*"))
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
