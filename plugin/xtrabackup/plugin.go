// The `xtrabackup` plugin for SHIELD implements backup + restore functionality
// for the cf-mysql-release.
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
// The endpoint configuration passed to this plugin is used to identify how to
// to connect to a MySQL instance co-located on the same machine.
//
// Your endpoint JSON should look something like this:
//
//    {
//        "mysql_user":           "username-for-mysql",
//        "mysql_password":       "password-for-above-user",
//        "mysql_databases":      <list_of_databases>,       # OPTIONAL
//        "mysql_datadir":        "/var/lib/mysql",          # OPTIONAL
//        "mysql_xtrabackup":     "/path/to/xtrabackup",     # OPTIONAL
//        "mysql_temp_targetdir": "/tmp/backups"             # OPTIONAL
//        "mysql_tar":            "tar"                      # OPTIONAL
//    }
//
// Default Configuration
//
//    {
//        "mysql_tar"           : "tar",
//        "mysql_datadir"       : "/var/lib/mysql",
//        "mysql_xtrabackup"    : "/var/vcap/packages/shield-mysql/bin/xtrabackup",
//        "mysql_temp_targetdir": "/tmp/backups"
//    }
//
// mysql_databases:
// This option specifies the list of databases to back up.
// It accepts a string argument or path to a file that contains the list of databases to back up.
// The list is of the form "databasename1[.table_name1] databasename2[.table_name2]".
// If this option is not specified, all databases containing MyISAM and InnoDB tables will be backed up.
//
// mysql_datadir:
// This option specifies MySQL's datadir.
//
// mysql_xtrabackup:
// This option specifies the absolute path to the `xtrabackup` tool.
//
// mysql_temp_targetdir:
// This option specifies the absolute path to a temporary directory used by
// the `xtrabackup` tool to backup the MySQL databases. It must be empty after
// each run of the plugin.
//
// mysql_tar:
// This option specifies the absolute path to the `tar` tool.
//
//
// BACKUP DETAILS
//
// The `xtrabackup` plugin backs up all data in the data directory. If the `databases` option is specified
// the plugin will only back up these databases.
//
// RESTORE DETAILS
//
// To restore, the `xtrabackup` plugin moves back the backed up data files to
// the MySQL data directory. The MySQL data directory needs to be empty before
// the restore operation.
//
// DEPENDENCIES
//
// This plugin relies on the `xtrabackup` and `tar` utilities. Please ensure
// that they are present on the system that will be running the
// backups + restores for MySQL.
package main

import (
	"fmt"
	"os"

	"github.com/starkandwayne/goutils/ansi"

	. "github.com/starkandwayne/shield/plugin"
)

var (
	DefaultTar           = "tar"
	DefaultDataDir       = "/var/lib/mysql"
	DefaultTempTargetDir = "/tmp/backups"
	DefaultXtrabackup    = "/var/vcap/packages/shield-mysql/bin/xtrabackup"
)

func main() {
	p := XtraBackupPlugin{
		Name:    "MySQL XtraBackup Plugin",
		Author:  "Swisscom",
		Version: "0.0.1",
		Features: PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
		Example: `
{
  "mysql_user":           "username-for-mysql",      # REQUIRED
  "mysql_password":       "password-for-above-user", # REQUIRED

  "mysql_databases":      "db1,db2",              # List of databases to limit
                                                  # backup and recovery to.

  "mysql_datadir":        "/var/lib/mysql",       # Path to the MySQL data directory
  "mysql_xtrabackup":     "/path/to/xtrabackup",  # Full path to the xtrabackup binary
  "mysql_temp_targetdir": "/tmp/backups"          # Temporary work directory
  "mysql_tar":            "tar"                   # Tar-compatible archival tool to use
}
`,
		Defaults: `
{
  "mysql_tar"           : "tar",
  "mysql_datadir"       : "/var/lib/mysql",
  "mysql_xtrabackup"    : "/var/vcap/packages/shield-mysql/bin/xtrabackup",
  "mysql_temp_targetdir": "/tmp/backups"
}
`,
	}

	Run(p)
}

type XtraBackupPlugin PluginInfo

type XtraBackupEndpoint struct {
	Databases string
	DataDir   string
	User      string
	Password  string
	Bin       string
	TargetDir string
	Tar       string
}

func (p XtraBackupPlugin) Meta() PluginInfo {
	return PluginInfo(p)
}

func (p XtraBackupPlugin) Validate(endpoint ShieldEndpoint) error {
	var (
		s    string
		err  error
		fail bool
	)

	s, err = endpoint.StringValue("mysql_user")
	if err != nil {
		ansi.Printf("@R{\u2717 mysql_user          %s}\n", err)
		fail = true
	} else {
		ansi.Printf("@G{\u2713 mysql_user}          @C{%s}\n", s)
	}

	s, err = endpoint.StringValue("mysql_password")
	if err != nil {
		ansi.Printf("@R{\u2717 mysql_password      %s}\n", err)
		fail = true
	} else {
		ansi.Printf("@G{\u2713 mysql_password}      @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("mysql_databases", "")
	if err != nil {
		ansi.Printf("@R{\u2717 mysql_databases  %s}\n", err)
		fail = true
	} else if s == "" {
		ansi.Printf("@G{\u2713 mysql_databases}  no databases\n")
	} else {
		ansi.Printf("@G{\u2713 mysql_databases}  @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("mysql_datadir", DefaultDataDir)
	if err != nil {
		ansi.Printf("@R{\u2717 mysql_datadir  %s}\n", err)
		fail = true
	} else if s == "" {
		ansi.Printf("@R{\u2717 mysql_datadir}  no datadir\n")
		fail = true
	} else {
		ansi.Printf("@G{\u2713 mysql_datadir}  @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("mysql_xtrabackup", DefaultXtrabackup)
	if err != nil {
		ansi.Printf("@R{\u2717 mysql_xtrabackup  %s}\n", err)
		fail = true
	} else if s == "" {
		ansi.Printf("@R{\u2717 mysql_xtrabackup}  xtrabackup command not specified\n")
		fail = true
	} else {
		ansi.Printf("@G{\u2713 mysql_xtrabackup}  @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("mysql_temp_targetdir", DefaultTempTargetDir)
	if err != nil {
		ansi.Printf("@R{\u2717 mysql_temp_targetdir  %s}\n", err)
		fail = true
	} else if s == "" {
		ansi.Printf("@R{\u2717 mysql_temp_targetdir}  no temporary target dir\n")
		fail = true
	} else {
		ansi.Printf("@G{\u2713 mysql_temp_targetdir}  @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("mysql_tar", DefaultTar)
	if err != nil {
		ansi.Printf("@R{\u2717 mysql_tar  %s}\n", err)
		fail = true
	} else if s == "" {
		ansi.Printf("@R{\u2717 mysql_tar}  tar command not specified\n")
		fail = true
	} else {
		ansi.Printf("@G{\u2713 mysql_tar}  @C{%s}\n", s)
	}

	if fail {
		return fmt.Errorf("xtrabackup: invalid configuration")
	}
	return nil
}

func (p XtraBackupPlugin) Backup(endpoint ShieldEndpoint) error {
	xtrabackup, err := getXtraBackupEndpoint(endpoint)
	if err != nil {
		return err
	}

	targetDir := xtrabackup.TargetDir

	dbs := ""
	if xtrabackup.Databases != "" {
		dbs = fmt.Sprintf(`--databases="%s"`, xtrabackup.Databases)
	}

	// create backup files
	cmdString := fmt.Sprintf("%s --backup --target-dir=%s --datadir=%s %s --user=%s --password=%s", xtrabackup.Bin, targetDir, xtrabackup.DataDir, dbs, xtrabackup.User, xtrabackup.Password)
	opts := ExecOptions{
		Cmd:      cmdString,
		Stdout:   os.Stdout,
		ExpectRC: []int{0},
	}

	DEBUG("Executing: `%s`", cmdString)
	if err = ExecWithOptions(opts); err != nil {
		return err
	}

	// create and return archive
	cmdString = fmt.Sprintf("%s -cf - -C %s .", xtrabackup.Tar, targetDir)
	if err = Exec(cmdString, STDOUT); err != nil {
		return err
	}

	// remove temporary target directory
	return os.RemoveAll(targetDir)
}

func (p XtraBackupPlugin) Restore(endpoint ShieldEndpoint) error {
	xtrabackup, err := getXtraBackupEndpoint(endpoint)
	if err != nil {
		return err
	}

	backupDir := xtrabackup.TargetDir

	// create tmp folder
	cmdString := fmt.Sprintf("mkdir -p %s", backupDir)
	opts := ExecOptions{
		Cmd:      cmdString,
		Stdout:   os.Stdout,
		ExpectRC: []int{0},
	}
	DEBUG("Executing: `%s`", cmdString)
	if err := ExecWithOptions(opts); err != nil {
		return err
	}

	// unpack archive
	cmdString = fmt.Sprintf("%s -xf - -C %s", xtrabackup.Tar, backupDir)
	DEBUG("Executing: `%s`", cmdString)
	if err := Exec(cmdString, STDIN); err != nil {
		return err
	}

	cmdString = fmt.Sprintf("%s --prepare --target-dir=%s", xtrabackup.Bin, backupDir)
	opts = ExecOptions{
		Cmd:      cmdString,
		Stdout:   os.Stdout,
		ExpectRC: []int{0},
	}
	DEBUG("Executing: `%s`", cmdString)
	if err := ExecWithOptions(opts); err != nil {
		return err
	}

	cmdString = fmt.Sprintf("%s --move-back --target-dir=%s --datadir=%s", xtrabackup.Bin, backupDir, xtrabackup.DataDir)
	opts = ExecOptions{
		Cmd:      cmdString,
		Stdout:   os.Stdout,
		ExpectRC: []int{0},
	}
	DEBUG("Executing: `%s`", cmdString)
	if err := ExecWithOptions(opts); err != nil {
		return err
	}

	// remove temporary target directory
	return os.RemoveAll(xtrabackup.TargetDir)
}

func (p XtraBackupPlugin) Store(endpoint ShieldEndpoint) (string, error) {
	return "", UNIMPLEMENTED
}

func (p XtraBackupPlugin) Retrieve(endpoint ShieldEndpoint, file string) error {
	return UNIMPLEMENTED
}

func (p XtraBackupPlugin) Purge(endpoint ShieldEndpoint, file string) error {
	return UNIMPLEMENTED
}

func getXtraBackupEndpoint(endpoint ShieldEndpoint) (XtraBackupEndpoint, error) {
	user, err := endpoint.StringValue("mysql_user")
	if err != nil {
		return XtraBackupEndpoint{}, err
	}
	DEBUG("MYSQL_USER: '%s'", user)

	password, err := endpoint.StringValue("mysql_password")
	if err != nil {
		return XtraBackupEndpoint{}, err
	}
	DEBUG("MYSQL_PWD: '%s'", password)

	databases, err := endpoint.StringValueDefault("mysql_databases", "")
	if err != nil {
		return XtraBackupEndpoint{}, err
	}
	DEBUG("MYSQL_DATABASES: '%s'", databases)

	dataDir, err := endpoint.StringValueDefault("mysql_datadir", DefaultDataDir)
	if err != nil {
		return XtraBackupEndpoint{}, err
	}
	DEBUG("MYSQL_DATADIR: '%s'", dataDir)

	targetDir, err := endpoint.StringValueDefault("mysql_temp_targetdir", DefaultTempTargetDir)
	if err != nil {
		return XtraBackupEndpoint{}, err
	}
	DEBUG("MYSQL_TEMP_TARGETDIR: '%s'", targetDir)

	xtrabackupBin, err := endpoint.StringValueDefault("mysql_xtrabackup", DefaultXtrabackup)
	if err != nil {
		return XtraBackupEndpoint{}, err
	}
	DEBUG("MYSQL_XTRABACKUP: '%s'", xtrabackupBin)

	tar, err := endpoint.StringValueDefault("mysql_tar", DefaultTar)
	if err != nil {
		return XtraBackupEndpoint{}, err
	}
	DEBUG("MYSQL_TAR: '%s'", tar)

	return XtraBackupEndpoint{
		User:      user,
		Password:  password,
		Databases: databases,
		DataDir:   dataDir,
		TargetDir: targetDir,
		Bin:       xtrabackupBin,
		Tar:       tar,
	}, nil
}
