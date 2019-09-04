package main

import (
	"os"
	"path/filepath"
	"syscall"

	fmt "github.com/jhunt/go-ansi"

	"github.com/shieldproject/shield/plugin"
)

var (
	DefaultTar           = "tar"
	DefaultDataDir       = "/var/lib/mysql"
	DefaultTempTargetDir = "/tmp/backups"
	DefaultXtrabackup    = "/var/vcap/packages/shield-mysql/bin/xtrabackup"
	DefaultBackupType    = "fullLegacy"
)

func main() {
	p := XtraBackupPlugin{
		Name:    "MySQL XtraBackup Plugin",
		Author:  "Swisscom",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
		Example: `
{
  "mysql_user":            "username-for-mysql",      # REQUIRED
  "mysql_password":        "password-for-above-user", # REQUIRED

  "mysql_databases":       "db1,db2",              # List of databases to limit
                                                  # backup and recovery to.

  "mysql_datadir":         "/var/lib/mysql",                         # Path to the MySQL data directory
  "mysql_socket":          "/var/vcap/sys/run/mysql/mysqld.sock",    # Path to the MySQL socket
  "mysql_xtrabackup":      "/path/to/xtrabackup",                    # Full path to the xtrabackup binary
  "mysql_temp_targetdir":  "/tmp/backups",                           # Temporary work directory
  "mysql_tar":             "tar",                                    # Tar-compatible archival tool to use
  "mysql_xtrabackup_type": "fullLegacy"                              # Backup full by default
}
`,
		Defaults: `
{
  "mysql_tar"           : "tar",
  "mysql_datadir"       : "/var/lib/mysql",
  "mysql_xtrabackup"    : "/var/vcap/packages/shield-mysql/bin/xtrabackup",
  "mysql_temp_targetdir": "/tmp/backups",
  "mysql_xtrabackup_type": "fullLegacy"
}
`,
		Fields: []plugin.Field{
			plugin.Field{
				Mode:     "target",
				Name:     "mysql_user",
				Type:     "string",
				Title:    "MySQL Username",
				Help:     "The username to use for performing the backup against MySQL.",
				Required: true,
			},
			plugin.Field{
				Mode:     "target",
				Name:     "mysql_password",
				Type:     "password",
				Title:    "MySQL Password",
				Help:     "The password to authenticate to MySQL with.",
				Required: true,
			},
			plugin.Field{
				Mode:    "target",
				Name:    "mysql_databases",
				Type:    "wslist",
				Title:   "Databases to Backup",
				Help:    "A list of databases (and optional tables) to restrict the backup to.  By default, all tables, in all databases will be backed up.",
				Example: "`database1`, or `db.users db.sessions`",
			},
			plugin.Field{
				Mode:    "target",
				Name:    "mysql_datadir",
				Type:    "abspath",
				Title:   "MySQL Data Directory",
				Help:    "Specifies absolute path to MySQL's data directory.",
				Default: "/var/lib/mysql",
			},
			plugin.Field{
				Mode:  "target",
				Name:  "mysql_socket",
				Type:  "abspath",
				Title: "MySQL Socket",
				Help:  "Specifies absolute path to MySQL's socket.",
			},
			plugin.Field{
				Mode:  "target",
				Name:  "mysql_xtrabackup",
				Type:  "abspath",
				Title: "Path to `xtrabackup` Utility",
				Help:  "By default, the plugin will search the local `$PATH` to find the `xtrabackup` utility.",
			},
			plugin.Field{
				Mode:  "target",
				Name:  "mysql_tar",
				Type:  "abspath",
				Title: "Path to the `tar` Utility",
				Help:  "By default, the plugin will search the local `$PATH` to find the `tar` utility.",
			},
			plugin.Field{
				Mode:    "target",
				Name:    "mysql_temp_targetdir",
				Type:    "abspath",
				Title:   "Local Temporary Directory",
				Help:    "Path to a temporary directory that `xtrabackup` will use for its own purposes.",
				Default: "/tmp/backups",
			},
			plugin.Field{
				Mode:    "target",
				Name:    "mysql_xtrabackup_type",
				Type:    "string",
				Title:   "Mysql Xtrabackup type",
				Help:    "Mysql Xtrabackup type full by default or differential backup",
				Default: "fullLegacy",
			},
		},
	}

	plugin.Run(p)
}

type XtraBackupPlugin plugin.PluginInfo

type XtraBackupEndpoint struct {
	Databases string
	DataDir   string
	Socket    string
	User      string
	Password  string
	Bin       string
	TargetDir string
	Tar       string
	Type      string
}

func (p XtraBackupPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p XtraBackupPlugin) Validate(endpoint plugin.ShieldEndpoint) error {
	var (
		s    string
		err  error
		fail bool
	)

	s, err = endpoint.StringValue("mysql_user")
	if err != nil {
		fmt.Printf("@R{\u2717 mysql_user          %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 mysql_user}          @C{%s}\n", plugin.Redact(s))
	}

	s, err = endpoint.StringValue("mysql_password")
	if err != nil {
		fmt.Printf("@R{\u2717 mysql_password      %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 mysql_password}      @C{%s}\n", plugin.Redact(s))
	}

	s, err = endpoint.StringValueDefault("mysql_databases", "")
	if err != nil {
		fmt.Printf("@R{\u2717 mysql_databases  %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 mysql_databases}  no databases\n")
	} else {
		fmt.Printf("@G{\u2713 mysql_databases}  @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("mysql_socket", "")
	if err != nil {
		fmt.Printf("@R{\u2717 mysql_socket  %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 mysql_socket}  no socket\n")
	} else {
		fmt.Printf("@G{\u2713 mysql_socket}  @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("mysql_datadir", DefaultDataDir)
	if err != nil {
		fmt.Printf("@R{\u2717 mysql_datadir  %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@R{\u2717 mysql_datadir}  no datadir\n")
		fail = true
	} else {
		fmt.Printf("@G{\u2713 mysql_datadir}  @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("mysql_xtrabackup", DefaultXtrabackup)
	if err != nil {
		fmt.Printf("@R{\u2717 mysql_xtrabackup  %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@R{\u2717 mysql_xtrabackup}  xtrabackup command not specified\n")
		fail = true
	} else {
		fmt.Printf("@G{\u2713 mysql_xtrabackup}  @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("mysql_temp_targetdir", DefaultTempTargetDir)
	if err != nil {
		fmt.Printf("@R{\u2717 mysql_temp_targetdir  %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@R{\u2717 mysql_temp_targetdir}  no temporary target dir\n")
		fail = true
	} else {
		fmt.Printf("@G{\u2713 mysql_temp_targetdir}  @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("mysql_tar", DefaultTar)
	if err != nil {
		fmt.Printf("@R{\u2717 mysql_tar  %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@R{\u2717 mysql_tar}  tar command not specified\n")
		fail = true
	} else {
		fmt.Printf("@G{\u2713 mysql_tar}  @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("mysql_xtrabackup_type", DefaultBackupType)
	if s != "full" && s != "differential" && s != "fullLegacy" {
		fmt.Printf("@R{\u2717 mysql_xtrabackup_type is incorrect, set \"full\" or \"differential\" or don't set it}\n")
		fail = true
	} else {
		fmt.Printf("@G{\u2713 mysql_xtrabackup_type}  @C{%s}\n", s)
	}

	if fail {
		return fmt.Errorf("xtrabackup: invalid configuration")
	}
	return nil
}

func (p XtraBackupPlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	xtrabackup, err := getXtraBackupEndpoint(endpoint)
	if err != nil {
		return err
	}

	targetDir := xtrabackup.TargetDir
	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		if xtrabackup.Type == "full" || xtrabackup.Type == "fullLegacy" {
			err = os.RemoveAll(targetDir)
		}
	} else if xtrabackup.Type == "differential" {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Run full backup before running differential} \n")
		return err
	}

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		if errMkdir := os.MkdirAll(targetDir, 0755); errMkdir != nil {
			fmt.Fprintf(os.Stderr, "@R{\u2717 Cannot create target directory} \n")
			return errMkdir
		}
	}

	socket := ""
	if xtrabackup.Socket != "" {
		socket = fmt.Sprintf(`--socket="%s"`, xtrabackup.Socket)
	}
	dbs := ""
	if xtrabackup.Databases != "" {
		dbs = fmt.Sprintf(`--databases="%s"`, xtrabackup.Databases)
	}

	// removing previous differential as it is not used
	defer func() {
		if xtrabackup.Type == "differential" {
			os.RemoveAll(fmt.Sprintf("%s/diff/", targetDir))
		} else if xtrabackup.Type == "fullLegacy" {
			os.RemoveAll(fmt.Sprintf("%s/", targetDir))
		}
	}()

	// create backup files
	var cmdString string
	tarDir := targetDir

	// default values if "fullLegacy" for retro compatibility on --target-dir passed value

	if xtrabackup.Type == "full" || xtrabackup.Type == "fullLegacy" {
		if xtrabackup.Type == "full" {
			tarDir = fmt.Sprintf("%s/base", targetDir)
		}
		cmdString = fmt.Sprintf("%s --backup --target-dir=%s --datadir=%s %s %s --user=%s --password=%s", xtrabackup.Bin, tarDir, xtrabackup.DataDir, socket, dbs, xtrabackup.User, xtrabackup.Password)
	} else if xtrabackup.Type == "differential" {
		tarDir = fmt.Sprintf("%s/diff", targetDir)
		cmdString = fmt.Sprintf("%s --backup --target-dir=%s --incremental-basedir=%s/base --datadir=%s %s %s --user=%s --password=%s", xtrabackup.Bin, tarDir, targetDir, xtrabackup.DataDir, socket, dbs, xtrabackup.User, xtrabackup.Password)
	}

	opts := plugin.ExecOptions{
		Cmd:      cmdString,
		Stdout:   os.Stdout,
		ExpectRC: []int{0},
	}

	plugin.DEBUG("Executing: `%s`", cmdString)
	if err = plugin.ExecWithOptions(opts); err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Creating backup files failed}\n")
		return err
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Created backup files}\n")

	// create and return archive
	cmdString = fmt.Sprintf("%s -cf - -C %s/ .", xtrabackup.Tar, tarDir)
	if err = plugin.Exec(cmdString, plugin.STDOUT); err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Creating archive failed}\n")
		return err
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Created archive}\n")

	// do not remove temporary target directory for differential backup
	// i.e: return os.RemoveAll(targetDir)
	return nil
}

func (p XtraBackupPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	xtrabackup, err := getXtraBackupEndpoint(endpoint)
	if err != nil {
		return err
	}
	// mysql must be stopped
	cmdString := "bash -c \" ps -efw | grep -F mysqld | grep -vE 'grep|mysqld_' &> /dev/null \""
	if err = plugin.Exec(cmdString, plugin.STDOUT); err == nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 MySQL must be stopped} Stop it and restart restore\n")
		return err
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 MySQL is stopped}\n")
	// targetdir must not exist

	restoreDir := xtrabackup.TargetDir

	var restoreFullDir string

	if restoreFullDir = restoreDir; xtrabackup.Type == "full" || xtrabackup.Type == "differential" {
		restoreFullDir = fmt.Sprintf("%s/base", restoreDir)
	}

	restoreDiffDir := fmt.Sprintf("%s/diff", restoreDir)

	if _, err := os.Stat(restoreDir); !os.IsNotExist(err) {
		if xtrabackup.Type == "full" {
			err = os.RemoveAll(restoreDir)
		} else if xtrabackup.Type == "differential" {
			if _, errDiff := os.Stat(restoreFullDir); os.IsNotExist(errDiff) {
				fmt.Fprintf(os.Stderr, "@R{\u2717 Run full restore before running differential} \n")
				return errDiff
			}
		}
	}

	defer func() {
		//in case of full backup we let the base directory present
		if xtrabackup.Type == "fullLegacy" || xtrabackup.Type == "differential" {
			os.RemoveAll(restoreDir)
		}
	}()

	// datadir exist
	dataDir := xtrabackup.DataDir
	fi, err := os.Lstat(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 mysql_datadir not exist} %s \n", dataDir)
		return err
	}
	if !fi.IsDir() {
		fmt.Fprintf(os.Stderr, "@R{\u2717 mysql_datadir must be a directory} %s \n", dataDir)
		return err
	}
	myuid := fi.Sys().(*syscall.Stat_t).Uid
	mygid := fi.Sys().(*syscall.Stat_t).Gid

	files, err := filepath.Glob(fmt.Sprintf("%s/*", dataDir))
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 unable to read the directory} %s \n", dataDir)
		return err
	}

	for _, f := range files {
		err = os.RemoveAll(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{\u2717 unable to delete} %s \n", f)
			return err
		}
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Checked datadir directory} %s \n", dataDir)

	tarDir := restoreFullDir

	if xtrabackup.Type == "differential" {
		tarDir = restoreDiffDir
	}

	// create archive folder to extract in
	if _, err := os.Stat(tarDir); os.IsNotExist(err) {
		if errMkdir := os.MkdirAll(tarDir, 0755); errMkdir != nil {
			fmt.Fprintf(os.Stderr, "@R{\u2717 Cannot create specific restore directory} \n")
			return errMkdir
		}
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Created temporary specific restore directory} %s \n", tarDir)

	// unpack archive
	cmdString = fmt.Sprintf("%s -xf - -C %s", xtrabackup.Tar, tarDir)
	opts := plugin.ExecOptions{
		Cmd:      cmdString,
		Stdout:   os.Stdout,
		ExpectRC: []int{0},
	}
	plugin.DEBUG("Executing: `%s`", cmdString)
	if err = plugin.Exec(cmdString, plugin.STDIN); err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Unpacking backup file failed} \n")
		return err
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Unpacked backup file} \n")
	//Set targetdir before prepare and move-back
	if xtrabackup.Type == "full" || xtrabackup.Type == "fullLegacy" {
		//prepare backup base for apply-log-only
		fmt.Fprintf(os.Stderr, "@G{\u2713 Using target dir : %d}\n", xtrabackup.TargetDir)
		cmdString = fmt.Sprintf("%s --prepare --apply-log-only --target-dir=%s", xtrabackup.Bin, restoreFullDir)
		opts = plugin.ExecOptions{
			Cmd:      cmdString,
			Stdout:   os.Stdout,
			ExpectRC: []int{0},
		}

		plugin.DEBUG("Executing: `%s`", cmdString)
		if err = plugin.ExecWithOptions(opts); err != nil {
			fmt.Fprintf(os.Stderr, "@R{\u2717 The Xtrabackup Prepare apply-log-only base operation failed}\n")
			return err
		}

		plugin.DEBUG(fmt.Sprintf("Run manually this command if you want to complete the restore full: '%s --move-back --target-dir=%s --datadir=%s'", xtrabackup.Bin, restoreFullDir, xtrabackup.DataDir))

	} else if xtrabackup.Type == "differential" {
		//Prepare differential backup
		cmdString = fmt.Sprintf("%s --prepare --apply-log-only --target-dir=%s --incremental-dir=%s", xtrabackup.Bin, restoreFullDir, restoreDiffDir)
		opts = plugin.ExecOptions{
			Cmd:      cmdString,
			Stdout:   os.Stdout,
			ExpectRC: []int{0},
		}
		plugin.DEBUG("Executing: `%s`", cmdString)
		if err = plugin.ExecWithOptions(opts); err != nil {
			fmt.Fprintf(os.Stderr, "@R{\u2717 The Xtrabackup Prepare differential operation failed}\n")
			return err
		}
		fmt.Fprintf(os.Stderr, "@G{\u2713 The Xtrabackup Prepare differential operation is performed}\n")
	}

	// handling retro compatibility where move back isn't done in container
	if xtrabackup.Type == "differential" || xtrabackup.Type == "fullLegacy" {
		cmdString = fmt.Sprintf("%s --move-back --target-dir=%s --datadir=%s", xtrabackup.Bin, restoreFullDir, xtrabackup.DataDir)
		opts = plugin.ExecOptions{
			Cmd:      cmdString,
			Stdout:   os.Stdout,
			ExpectRC: []int{0},
		}
		plugin.DEBUG("Executing: `%s`", cmdString)
		if err = plugin.ExecWithOptions(opts); err != nil {
			fmt.Fprintf(os.Stderr, "@R{\u2717 Restoring MySQL server failed}\n")
			return err
		}
		fmt.Fprintf(os.Stderr, "@G{\u2713 Restored MySQL server}\n")
	}

	// change uid and gid of restore file
	err = filepath.Walk(xtrabackup.DataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if e := syscall.Chown(path, int(myuid), int(mygid)); e != nil {
			return e
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Changing files ownership failed}\n")
		return err
	}

	fmt.Fprintf(os.Stderr, "@G{\u2713 Changed files ownership}\n")
	// remove temporary target directory
	//return os.RemoveAll(xtrabackup.TargetDir)
	return nil
}

func (p XtraBackupPlugin) Store(endpoint plugin.ShieldEndpoint) (string, int64, error) {
	return "", 0, plugin.UNIMPLEMENTED
}

func (p XtraBackupPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p XtraBackupPlugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func getXtraBackupEndpoint(endpoint plugin.ShieldEndpoint) (XtraBackupEndpoint, error) {
	user, err := endpoint.StringValue("mysql_user")
	if err != nil {
		return XtraBackupEndpoint{}, err
	}
	plugin.DEBUG("MYSQL_USER: '%s'", user)

	password, err := endpoint.StringValue("mysql_password")
	if err != nil {
		return XtraBackupEndpoint{}, err
	}
	plugin.DEBUG("MYSQL_PWD: '%s'", password)

	databases, err := endpoint.StringValueDefault("mysql_databases", "")
	if err != nil {
		return XtraBackupEndpoint{}, err
	}
	plugin.DEBUG("MYSQL_DATABASES: '%s'", databases)

	dataDir, err := endpoint.StringValueDefault("mysql_datadir", DefaultDataDir)
	if err != nil {
		return XtraBackupEndpoint{}, err
	}
	plugin.DEBUG("MYSQL_DATADIR: '%s'", dataDir)

	socket, err := endpoint.StringValueDefault("mysql_socket", "")
	if err != nil {
		return XtraBackupEndpoint{}, err
	}
	plugin.DEBUG("MYSQL_SOCKET: '%s'", socket)

	targetDir, err := endpoint.StringValueDefault("mysql_temp_targetdir", DefaultTempTargetDir)
	if err != nil {
		return XtraBackupEndpoint{}, err
	}
	plugin.DEBUG("MYSQL_TEMP_TARGETDIR: '%s'", targetDir)

	xtrabackupBin, err := endpoint.StringValueDefault("mysql_xtrabackup", DefaultXtrabackup)
	if err != nil {
		return XtraBackupEndpoint{}, err
	}
	plugin.DEBUG("MYSQL_XTRABACKUP: '%s'", xtrabackupBin)

	tar, err := endpoint.StringValueDefault("mysql_tar", DefaultTar)
	if err != nil {
		return XtraBackupEndpoint{}, err
	}
	plugin.DEBUG("MYSQL_TAR: '%s'", tar)

	bckptype, err := endpoint.StringValueDefault("mysql_xtrabackup_type", DefaultBackupType)
	if err != nil {
		return XtraBackupEndpoint{}, err
	}
	plugin.DEBUG("MYSQL_XTRABACKUP_TYPE: '%s'", bckptype)

	return XtraBackupEndpoint{
		User:      user,
		Password:  password,
		Databases: databases,
		DataDir:   dataDir,
		TargetDir: targetDir,
		Socket:    socket,
		Bin:       xtrabackupBin,
		Tar:       tar,
		Type:      bckptype,
	}, nil
}
