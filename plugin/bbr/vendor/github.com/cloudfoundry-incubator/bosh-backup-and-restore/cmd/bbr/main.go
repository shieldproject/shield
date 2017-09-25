package main

import (
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/urfave/cli"

	"time"

	"io/ioutil"

	"strings"

	"net/url"

	"os/signal"

	"bufio"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/mgutz/ansi"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/backup"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/standalone"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/writer"
	"github.com/pkg/errors"
)

var version string
var stdout *writer.PausableWriter = writer.NewPausableWriter(os.Stdout)
var stderr *writer.PausableWriter = writer.NewPausableWriter(os.Stderr)

func main() {
	cli.AppHelpTemplate = `NAME:
   {{.Name}}{{if .Usage}} - {{.Usage}}{{end}}

USAGE:
   bbr command [arguments...] [subcommand]{{if .Version}}{{if not .HideVersion}}

VERSION:
   {{.Version}}{{end}}{{end}}{{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if len .Authors}}

AUTHOR{{with $length := len .Authors}}{{if ne 1 $length}}S{{end}}{{end}}:
   {{range $index, $author := .Authors}}{{if $index}}
   {{end}}{{$author}}{{end}}{{end}}{{if .VisibleCommands}}

COMMANDS:{{range .VisibleCategories}}{{if .Name}}
   {{.Name}}:{{end}}{{range .VisibleCommands}}
   {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}

SUBCOMMANDS:
   backup
   backup-cleanup
   restore
   pre-backup-check{{if .Copyright}}

COPYRIGHT:
   {{.Copyright}}{{end}}
`

	app := cli.NewApp()

	app.Version = version
	app.Name = "bbr"
	app.Usage = "BOSH Backup and Restore"
	app.HideHelp = true

	app.Commands = []cli.Command{
		{
			Name:   "deployment",
			Usage:  "Backup BOSH deployments",
			Flags:  availableDeploymentFlags(),
			Before: validateDeploymentFlags,
			Subcommands: []cli.Command{
				{
					Name:    "pre-backup-check",
					Aliases: []string{"c"},
					Usage:   "Check a deployment can be backed up",
					Action:  deploymentPreBackupCheck,
				},
				{
					Name:    "backup",
					Aliases: []string{"b"},
					Usage:   "Backup a deployment",
					Action:  deploymentBackup,
					Flags: []cli.Flag{cli.BoolFlag{
						Name:  "with-manifest",
						Usage: "Download the deployment manifest",
					}},
				},
				{
					Name:    "restore",
					Aliases: []string{"r"},
					Usage:   "Restore a deployment from backup",
					Action:  deploymentRestore,
					Flags: []cli.Flag{cli.StringFlag{
						Name:  "artifact-path",
						Usage: "Path to the artifact to restore",
					}},
				},
				{
					Name:   "backup-cleanup",
					Usage:  "Cleanup a deployment after a backup was interrupted",
					Action: deploymentCleanup,
				},
			},
		},
		{
			Name:   "director",
			Usage:  "Backup BOSH director",
			Flags:  availableDirectorFlags(),
			Before: validateDirectorFlags,
			Subcommands: []cli.Command{
				{
					Name:    "pre-backup-check",
					Aliases: []string{"c"},
					Usage:   "Check a BOSH Director can be backed up",
					Action:  directorPreBackupCheck,
				},
				{
					Name:    "backup",
					Aliases: []string{"b"},
					Usage:   "Backup a BOSH Director",
					Action:  directorBackup,
				},
				{
					Name:    "restore",
					Aliases: []string{"r"},
					Usage:   "Restore a deployment from backup",
					Action:  directorRestore,
					Flags: []cli.Flag{cli.StringFlag{
						Name:  "artifact-path",
						Usage: "Path to the artifact to restore",
					}},
				},
				{
					Name:   "backup-cleanup",
					Usage:  "Cleanup a director after a backup was interrupted",
					Action: directorCleanup,
				},
			},
		},
		{
			Name:    "help",
			Aliases: []string{"h"},
			Usage:   "Shows a list of commands or help for one command",
			Action: func(c *cli.Context) error {
				cli.ShowAppHelp(c)
				return nil
			},
		},
		{
			Name:    "version",
			Aliases: []string{"v"},
			Usage:   "Shows the version",
			Action: func(c *cli.Context) error {
				cli.ShowVersion(c)
				return nil
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}

func trapSigint() {
	sigintChan := make(chan os.Signal, 1)
	signal.Notify(sigintChan, os.Interrupt)
	go func() {
		for range sigintChan {
			stdinReader := bufio.NewReader(os.Stdin)
			stdout.Pause()
			stderr.Pause()
			fmt.Fprintln(os.Stdout, "\nStopping a backup can leave the system in bad state. Are you sure you want to cancel? [yes/no]")
			input, err := stdinReader.ReadString('\n')
			if err != nil {
				fmt.Println("\nCouldn't read from Stdin, if you still want to stop the backup send SIGTERM.")
			} else if strings.ToLower(strings.TrimSpace(input)) == "yes" {
				os.Exit(1)
			}
			stdout.Resume()
			stderr.Resume()
		}
	}()
}

func deploymentPreBackupCheck(c *cli.Context) error {
	var deployment = c.Parent().String("deployment")

	backuper, err := makeDeploymentBackuper(c)
	if err != nil {
		return err
	}

	backupable, checkErr := backuper.CanBeBackedUp(deployment)

	if backupable {
		fmt.Printf("Deployment '%s' can be backed up.\n", deployment)
		return cli.NewExitError("", 0)
	} else {
		fmt.Printf("Deployment '%s' cannot be backed up.\n", deployment)
		writeStackTrace(checkErr.PrettyError(true))
		return cli.NewExitError(checkErr.Error(), 1)
	}
}

func directorPreBackupCheck(c *cli.Context) error {
	directorName := ExtractNameFromAddress(c.Parent().String("host"))

	backuper := makeDirectorBackuper(c)

	backupable, checkErr := backuper.CanBeBackedUp(directorName)

	if backupable {
		fmt.Printf("Director can be backed up.\n")
		return cli.NewExitError("", 0)
	} else {
		fmt.Printf("Director cannot be backed up.\n")
		writeStackTrace(checkErr.PrettyError(true))
		return cli.NewExitError(checkErr.Error(), 1)
	}
}

func deploymentBackup(c *cli.Context) error {
	trapSigint()

	backuper, err := makeDeploymentBackuper(c)
	if err != nil {
		return err
	}

	deployment := c.Parent().String("deployment")
	backupErr := backuper.Backup(deployment)

	errorCode, errorMessage, errorWithStackTrace := orchestrator.ProcessError(backupErr)
	if err := writeStackTrace(errorWithStackTrace); err != nil {
		return errors.Wrap(backupErr, err.Error())
	}

	return cli.NewExitError(errorMessage, errorCode)
}

func directorBackup(c *cli.Context) error {
	trapSigint()

	directorName := ExtractNameFromAddress(c.Parent().String("host"))

	backuper := makeDirectorBackuper(c)

	backupErr := backuper.Backup(directorName)

	errorCode, errorMessage, errorWithStackTrace := orchestrator.ProcessError(backupErr)
	if err := writeStackTrace(errorWithStackTrace); err != nil {
		return errors.Wrap(backupErr, err.Error())
	}

	return cli.NewExitError(errorMessage, errorCode)
}

func deploymentRestore(c *cli.Context) error {
	if err := validateFlags([]string{"artifact-path"}, c); err != nil {
		return err
	}

	deployment := c.Parent().String("deployment")
	artifactPath := c.String("artifact-path")

	restorer, err := makeDeploymentRestorer(c)
	if err != nil {
		return err
	}

	restoreErr := restorer.Restore(deployment, artifactPath)
	errorCode, errorMessage, errorWithStackTrace := orchestrator.ProcessError(restoreErr)
	if err := writeStackTrace(errorWithStackTrace); err != nil {
		return errors.Wrap(restoreErr, err.Error())
	}

	return cli.NewExitError(errorMessage, errorCode)
}

func directorRestore(c *cli.Context) error {
	if err := validateFlags([]string{"artifact-path"}, c); err != nil {
		return err
	}

	directorName := ExtractNameFromAddress(c.Parent().String("host"))
	artifactPath := c.String("artifact-path")

	restorer := makeDirectorRestorer(c)

	restoreErr := restorer.Restore(directorName, artifactPath)
	errorCode, errorMessage, errorWithStackTrace := orchestrator.ProcessError(restoreErr)
	if err := writeStackTrace(errorWithStackTrace); err != nil {
		return errors.Wrap(restoreErr, err.Error())
	}

	return cli.NewExitError(errorMessage, errorCode)
}

func deploymentCleanup(c *cli.Context) error {
	trapSigint()

	cleaner, err := makeDeploymentCleaner(c)
	if err != nil {
		return err
	}

	deployment := c.Parent().String("deployment")
	cleanupErr := cleaner.Cleanup(deployment)

	errorCode, errorMessage, errorWithStackTrace := orchestrator.ProcessError(cleanupErr)
	if err := writeStackTrace(errorWithStackTrace); err != nil {
		return errors.Wrap(cleanupErr, err.Error())
	}

	return cli.NewExitError(errorMessage, errorCode)
}

func directorCleanup(c *cli.Context) error {
	trapSigint()

	directorName := ExtractNameFromAddress(c.Parent().String("host"))

	cleaner := makeDirectorCleaner(c)

	cleanupErr := cleaner.Cleanup(directorName)

	errorCode, errorMessage, errorWithStackTrace := orchestrator.ProcessError(cleanupErr)
	if err := writeStackTrace(errorWithStackTrace); err != nil {
		return errors.Wrap(cleanupErr, err.Error())
	}

	return cli.NewExitError(errorMessage, errorCode)
}

func writeStackTrace(errorWithStackTrace string) error {
	if errorWithStackTrace != "" {
		err := ioutil.WriteFile(fmt.Sprintf("bbr-%s.err.log", time.Now().UTC().Format(time.RFC3339)), []byte(errorWithStackTrace), 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

func validateDeploymentFlags(c *cli.Context) error {
	return validateFlags([]string{"target", "username", "password", "deployment"}, c)
}

func validateDirectorFlags(c *cli.Context) error {
	return validateFlags([]string{"host", "username", "private-key-path"}, c)
}

func validateFlags(requiredFlags []string, c *cli.Context) error {
	if containsHelpFlag(c) {
		return nil
	}

	for _, flag := range requiredFlags {
		if c.String(flag) == "" {
			cli.ShowCommandHelp(c, c.Parent().Command.Name)
			return redCliError(errors.Errorf("--%v flag is required.", flag))
		}
	}
	return nil
}

func containsHelpFlag(c *cli.Context) bool {
	for _, arg := range c.Args() {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

func availableDeploymentFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:  "target, t",
			Value: "",
			Usage: "Target BOSH Director URL",
		},
		cli.StringFlag{
			Name:  "username, u",
			Value: "",
			Usage: "BOSH Director username",
		},
		cli.StringFlag{
			Name:   "password, p",
			Value:  "",
			EnvVar: "BOSH_CLIENT_SECRET",
			Usage:  "BOSH Director password",
		},
		cli.StringFlag{
			Name:  "deployment, d",
			Value: "",
			Usage: "Name of BOSH deployment",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug logs",
		},
		cli.StringFlag{
			Name:   "ca-cert",
			Value:  "",
			EnvVar: "CA_CERT",
			Usage:  "Custom CA certificate",
		},
	}
}

func availableDirectorFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:  "host",
			Value: "",
			Usage: "BOSH Director hostname, with an optional port. Port defaults to 22",
		},
		cli.StringFlag{
			Name:  "username, u",
			Value: "",
			Usage: "BOSH Director SSH username",
		},
		cli.StringFlag{
			Name:  "private-key-path, key",
			Value: "",
			Usage: "BOSH Director SSH private key",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug logs",
		},
	}
}

func ExtractNameFromAddress(address string) string {
	url, err := url.Parse(address)
	if err == nil && url.Hostname() != "" {
		address = url.Hostname()
	}
	return strings.Split(address, ":")[0]
}

func makeDeploymentCleaner(c *cli.Context) (*orchestrator.Cleaner, error) {
	logger := makeLogger(c)
	deploymentManager, err := newDeploymentManager(
		c.Parent().String("target"),
		c.Parent().String("username"),
		c.Parent().String("password"),
		c.Parent().String("ca-cert"),
		logger,
		c.Bool("with-manifest"),
	)

	if err != nil {
		return nil, redCliError(err)
	}

	return orchestrator.NewCleaner(logger, deploymentManager), nil
}

func makeDirectorCleaner(c *cli.Context) *orchestrator.Cleaner {
	logger := makeLogger(c)
	deploymentManager := standalone.NewDeploymentManager(logger,
		c.Parent().String("host"),
		c.Parent().String("username"),
		c.Parent().String("private-key-path"),
		instance.NewJobFinder(logger),
		ssh.NewConnection,
	)

	return orchestrator.NewCleaner(logger, deploymentManager)
}

func makeDeploymentBackuper(c *cli.Context) (*orchestrator.Backuper, error) {
	logger := makeLogger(c)
	deploymentManager, err := newDeploymentManager(
		c.Parent().String("target"),
		c.Parent().String("username"),
		c.Parent().String("password"),
		c.Parent().String("ca-cert"),
		logger,
		c.Bool("with-manifest"),
	)

	if err != nil {
		return nil, redCliError(err)
	}

	return orchestrator.NewBackuper(backup.BackupDirectoryManager{}, logger, deploymentManager, time.Now), nil
}

func makeDirectorBackuper(c *cli.Context) *orchestrator.Backuper {
	logger := makeLogger(c)
	deploymentManager := standalone.NewDeploymentManager(logger,
		c.Parent().String("host"),
		c.Parent().String("username"),
		c.Parent().String("private-key-path"),
		instance.NewJobFinder(logger),
		ssh.NewConnection,
	)

	return orchestrator.NewBackuper(backup.BackupDirectoryManager{}, logger, deploymentManager, time.Now)
}

func makeDeploymentRestorer(c *cli.Context) (*orchestrator.Restorer, error) {
	logger := makeLogger(c)
	deploymentManager, err := newDeploymentManager(
		c.Parent().String("target"),
		c.Parent().String("username"),
		c.Parent().String("password"),
		c.Parent().String("ca-cert"),
		logger,
		false,
	)

	if err != nil {
		return nil, redCliError(err)
	}

	return orchestrator.NewRestorer(backup.BackupDirectoryManager{}, logger, deploymentManager), nil
}

func makeDirectorRestorer(c *cli.Context) *orchestrator.Restorer {
	logger := makeLogger(c)
	deploymentManager := standalone.NewDeploymentManager(logger,
		c.Parent().String("host"),
		c.Parent().String("username"),
		c.Parent().String("private-key-path"),
		instance.NewJobFinder(logger),
		ssh.NewConnection,
	)
	return orchestrator.NewRestorer(backup.BackupDirectoryManager{}, logger, deploymentManager)
}

func newDeploymentManager(targetUrl, username, password, caCert string, logger boshlog.Logger, downloadManifest bool) (orchestrator.DeploymentManager, error) {
	boshClient, err := bosh.BuildClient(targetUrl, username, password, caCert, logger)
	if err != nil {
		return nil, redCliError(err)
	}

	return bosh.NewDeploymentManager(boshClient, logger, downloadManifest), nil
}

func makeLogger(c *cli.Context) boshlog.Logger {
	var debug = c.GlobalBool("debug")
	return makeBoshLogger(debug)
}

func redCliError(err error) *cli.ExitError {
	return cli.NewExitError(ansi.Color(err.Error(), "red"), 1)
}

func makeBoshLogger(debug bool) boshlog.Logger {
	if debug {
		return boshlog.NewWriterLogger(boshlog.LevelDebug, stdout, stderr)
	}
	return boshlog.NewWriterLogger(boshlog.LevelInfo, stdout, stderr)
}
