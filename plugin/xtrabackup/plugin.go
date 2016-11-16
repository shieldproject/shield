// The `xtrabackup` plugin for SHIELD implements backup + restore functionality
// for the cf-mysql-release.
//
// PLUGIN FEATURES
//
// This plugin implements functionality suitable for use with the following SHIELD
// Job components:
//
//    Target: yes
//    Store:  no
//
// PLUGIN CONFIGURATION
//
// This option specifies the list of databases to back up.
// The option accepts a string argument or path to file that contains the list of databases to back up.
// The list is of the form "databasename1[.table_name1] databasename2[.table_name2]".
// If this option is not specified, all databases containing MyISAM and InnoDB tables will be backed up.
// The location of the source directory for the backup can be specified. Otherwise the location of the datadir
// for your MySQL server will be read from my.cnf.
//
// Your endpoint JSON should look something like this:
//
//    {
//        "databases":<list_of_databases> #OPTIONAL,
//        "mysql_user":"username-for-mysql",
//        "mysql_password":"password-for-above-user",
//		  "datadir":"backup-source-directory" #OPTIONAL
//    }
//
// BACKUP DETAILS
//
// The `xtrabackup` plugin backs up all data in the data directory. If the `databases` option is specified
// the plugin will only back up these databases.
//
// RESTORE DETAILS
//
// TODO
//
// DEPENDENCIES
//
// None.
package main

import (
	"fmt"
	"os"

	"github.com/starkandwayne/goutils/ansi"

	. "github.com/starkandwayne/shield/plugin"
)

func main() {
	p := XtraBackupPlugin{
		Name:    "MySQL XtraBackup Plugin",
		Author:  "-",
		Version: "0.0.1",
		Features: PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
	}

	Run(p)
}

type XtraBackupPlugin PluginInfo

type XtraBackupEndpoint struct {
	Databases string
	DataDir   string
	User      string
	Password  string
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

	s, err = endpoint.StringValueDefault("databases", "")
	if err != nil {
		ansi.Printf("@R{\u2717 databases  %s}\n", err)
		fail = true
	} else if s == "" {
		ansi.Printf("@G{\u2713 databases}  no databases\n")
	} else {
		ansi.Printf("@G{\u2713 databases}  @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("datadir", "")
	if err != nil {
		ansi.Printf("@R{\u2717 datadir  %s}\n", err)
		fail = true
	} else if s == "" {
		ansi.Printf("@G{\u2713 datadir}  no datadir\n")
	} else {
		ansi.Printf("@G{\u2713 datadir}  @C{%s}\n", s)
	}

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

	targetDir := "/tmp/backups/"

	dbs := ""
	if xtrabackup.Databases != "" {
		dbs = fmt.Sprintf("--databases=%s", dbs)
	}

	dd := ""
	if xtrabackup.DataDir != "" {
		dbs = fmt.Sprintf("--datadir=%s", dbs)
	}

	// create backup files
	cmdString := fmt.Sprintf("xtrabackup --backup --target-dir=%s %s %s --user=%s --password=%s", targetDir, dbs, dd, xtrabackup.User, xtrabackup.Password)
	opts := ExecOptions{
		Cmd:      cmdString,
		Stdout:   os.Stdout,
		ExpectRC: []int{0, 1},
	}

	DEBUG("Executing: `%s`", cmdString)
	if err = ExecWithOptions(opts); err != nil {
		return err
	}

	// create and return archive
	cmdString = fmt.Sprintf("tar -cf - %s", targetDir)
	if err = Exec(cmdString, STDOUT); err != nil {
		return err
	}

	// remove target directory
	return os.RemoveAll(targetDir)
}

func (p XtraBackupPlugin) Restore(endpoint ShieldEndpoint) error {
	xtrabackup, err := getXtraBackupEndpoint(endpoint)
	if err != nil {
		return err
	}

	backupDir := "/tmp/backups/"

	// create tmp folder and unpack archive
	cmdString := fmt.Sprintf("mkdir -p %s && tar -xf - -C %s", backupDir, backupDir)
	DEBUG("Executing: `%s`", cmdString)
	if err := Exec(cmdString, STDOUT); err != nil {
		return err
	}

	// xtrabackup --prepare --target-dir=tmpFolder
	cmdString = fmt.Sprintf("xtrabackup --prepare --target-dir=%s", backupDir)
	opts := ExecOptions{
		Cmd:      cmdString,
		Stdout:   os.Stdout,
		ExpectRC: []int{0, 1},
	}

	DEBUG("Executing: `%s`", cmdString)
	if err := ExecWithOptions(opts); err != nil {
		return err
	}

	// stop mysql server
	// remove data dir

	// xtrabackup --move-back --target-dir=/data/backups/
	cmdString = fmt.Sprintf("xtrabackup --move-back --target-dir=%s --user=%s --password=%s", backupDir, xtrabackup.User, xtrabackup.Password)
	opts = ExecOptions{
		Cmd:      cmdString,
		Stdout:   os.Stdout,
		ExpectRC: []int{0, 1},
	}

	DEBUG("Executing: `%s`", cmdString)
	if err := ExecWithOptions(opts); err != nil {
		return err
	}
	// chown -R mysql:mysql /var/lib/mysql
	// restart mysql server

	return nil
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
	databases, err := endpoint.StringValueDefault("databases", "")
	if err != nil {
		return XtraBackupEndpoint{}, err
	}

	dataDir, err := endpoint.StringValueDefault("datadir", "")
	if err != nil {
		return XtraBackupEndpoint{}, err
	}

	user, err := endpoint.StringValue("mysql_user")
	if err != nil {
		return XtraBackupEndpoint{}, err
	}

	password, err := endpoint.StringValue("mysql_password")
	if err != nil {
		return XtraBackupEndpoint{}, err
	}

	return XtraBackupEndpoint{
		Databases: databases,
		DataDir:   dataDir,
		User:      user,
		Password:  password,
	}, nil
}
