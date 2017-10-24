// The `mysql` plugin for SHIELD implements generic backup + restore
// functionality for a MySQL-compatible server. It can be used against
// any mysql server compatible with the `mysql` and `mysqldump` tools
// installed on the system where this plugin is run.
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
// The endpoint configuration passed to this plugin is used to identify what
// potgres instance to back up, and how to connect to it. Your endpoint JSON
// should look something like this:
//
//    {
//        "mysql_host"         : "127.0.0.1",    # optional
//        "mysql_port"         : "3306",         # optional
//        "mysql_user"         : "username",
//        "mysql_password"     : "password",
//        "mysql_read_replica" : "hostname/ip",  # optional
//        "mysql_database"     : "db",           # optional
//        "mysql_options"      : "--quick",      # optional
//        "mysql_bindir"       : "/path/to/bin"  # optional
//    }
//
// Default Configuration
//
//    {
//        "mysql_host"   : "127.0.0.1",
//        "mysql_port"   : "3306",
//        "mysql_bindir" : "/var/vcap/packages/shield-mysql/bin"
//    }
//
// BACKUP DETAILS
//
// If `mysql_database` is not specified in the plugin configuration, the `mysql` plugin makes
// use of `mysqldump --all-databases` to back up all databases on the mysql server it connects to.
// Otherwise, it backs up ONLY the specified database. The dumps generated include
// SQL to clean up existing tables of the databases, so that the restores will go smoothly.
//
// The mysql_options setting can apply mysqldump specific options like --force, --quick and/or
// --single-transaction
//
// Backing up with the `mysql` plugin will not drop any existing connections to the database,
// or restart the service.
//
//RESTORE DETAILS
//
// To restore, the `mysql` plugin connects to the mysql server using the `mysql` command.
// It then feeds in the backup data (`mysqldump` output). Unlike the the `postgres` plugin,
// this plugin does NOT need to disconnect any open connections to mysql to perform the
// restoration.
//
// Restoring with the `mysql` plugin should not interrupt established connections to the service.
//
// DEPENDENCIES
//
// This plugin relies on the `mysqldump` and `mysql` utilities. Please ensure
// that they are present on the system that will be running the backups + restores
// for postgres. If you are using shield-boshrelease to deploy SHIELD, these tools
// are provided so long as you include the `agent-mysql` job template along side
// your `shield agent`.
//
package main

import (
	"os"

	fmt "github.com/jhunt/go-ansi"

	"github.com/starkandwayne/shield/plugin"
)

var (
	DefaultHost = "127.0.0.1"
	DefaultPort = "3306"
)

func main() {
	p := MySQLPlugin{
		Name:    "MySQL Backup Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
		Example: `
{
  "mysql_host"         : "127.0.0.1",    # optional
  "mysql_port"         : "3306",         # optional
  "mysql_user"         : "username",
  "mysql_password"     : "password",
  "mysql_read_replica" : "hostname/ip",  # optional
  "mysql_database"     : "db",           # optional
  "mysql_options"      : "--quick",      # optional
  "mysql_bindir"       : "/path/to/bin"  # optional
}
`,
		Defaults: `
{
  "mysql_host"   : "127.0.0.1",
  "mysql_port"   : "3306",
  "mysql_bindir" : "/var/vcap/packages/shield-mysql/bin"
}
`,
		Fields: []plugin.Field{
			plugin.Field{
				Mode:     "target",
				Name:     "mysql_host",
				Type:     "string",
				Title:    "MySQL Host",
				Help:     "The hostname or IP address of your MySQL server.",
				Default:  "127.0.0.1",
				Required: true,
			},
			plugin.Field{
				Mode:     "target",
				Name:     "mysql_port",
				Type:     "port",
				Title:    "MySQL Port",
				Help:     "The TCP port that MySQL is bound to, listening for incoming connections.",
				Default:  "3306",
				Required: true,
			},
			plugin.Field{
				Mode:     "target",
				Name:     "mysql_user",
				Type:     "string",
				Title:    "MySQL Username",
				Help:     "Username to authenticate to MySQL as.",
				Required: true,
			},
			plugin.Field{
				Mode:     "target",
				Name:     "mysql_pass",
				Type:     "password",
				Title:    "MySQL Password",
				Help:     "Password to authenticate to MySQL as.",
				Required: true,
			},
			plugin.Field{
				Mode:  "target",
				Name:  "mysql_database",
				Type:  "string",
				Title: "Database to Backup",
				Help:  "Limit scope of the backup to include only this database.  By default, all databases will be backed up.",
			},
			plugin.Field{
				Mode:    "target",
				Name:    "mysql_options",
				Type:    "string",
				Title:   "Additional `mysqldump` options",
				Help:    "You can tune `mysqldump` (which performs the backup) by specifying additional options and command-line arguments.  If you don't know why you might need this, leave it blank.",
				Example: "--quick",
			},
			plugin.Field{
				Mode:  "target",
				Name:  "mysql_read_replica",
				Type:  "string",
				Title: "MySQL Read Replica",
				Help:  "An optional MySQL replica (possibly readonly) to use for backups, instead of the canonical host.  Restore operations will still be conducted against the real database host.",
			},
			plugin.Field{
				Mode:    "target",
				Name:    "mysql_bindir",
				Type:    "abspath",
				Title:   "Path to MySQL bin/ directory",
				Help:    "The absolute path to the bin/ directory that contains the `mysql` and `mysqldump` commands.",
				Default: "/var/vcap/packages/shield-mysql/bin",
			},
		},
	}

	plugin.Run(p)
}

type MySQLPlugin plugin.PluginInfo

type MySQLConnectionInfo struct {
	Host     string
	Port     string
	User     string
	Password string
	Bin      string
	Replica  string
	Database string
	Options  string
}

func (p MySQLPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p MySQLPlugin) Validate(endpoint plugin.ShieldEndpoint) error {
	var (
		s    string
		err  error
		fail bool
	)

	s, err = endpoint.StringValueDefault("mysql_host", DefaultHost)
	if err != nil {
		fmt.Printf("@R{\u2717 mysql_host          %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 mysql_host}          using default host @C{%s}\n", DefaultHost)
	} else {
		fmt.Printf("@G{\u2713 mysql_host}          @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("mysql_port", "")
	if err != nil {
		fmt.Printf("@R{\u2717 mysql_port          %s}\n", err)
	} else if s == "" {
		fmt.Printf("@G{\u2713 mysql_port}          using default port @C{%s}\n", DefaultPort)
	} else {
		fmt.Printf("@G{\u2713 mysql_port}          @C{%s}\n", s)
	}

	s, err = endpoint.StringValue("mysql_user")
	if err != nil {
		fmt.Printf("@R{\u2717 mysql_user          %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 mysql_user}          @C{%s}\n", s)
	}

	s, err = endpoint.StringValue("mysql_password")
	if err != nil {
		fmt.Printf("@R{\u2717 mysql_password      %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 mysql_password}      @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("mysql_read_replica", "")
	if err != nil {
		fmt.Printf("@R{\u2717 mysql_read_replica  %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 mysql_read_replica}  no read replica given\n")
	} else {
		fmt.Printf("@G{\u2713 mysql_read_replica}  @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("mysql_database", "")
	if err != nil {
		fmt.Printf("@R{\u2717 mysql_database      %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 mysql_database}      backing up *all* databases\n")
	} else {
		fmt.Printf("@G{\u2713 mysql_database}      @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("mysql_options", "")
	if err != nil {
		fmt.Printf("@R{\u2717 mysql_options       %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 mysql_options}       no options given\n")
	} else {
		fmt.Printf("@G{\u2713 mysql_options}       @C{%s}\n", s)
	}

	if fail {
		return fmt.Errorf("mysql: invalid configuration")
	}
	return nil
}

// Backup mysql database
func (p MySQLPlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	mysql, err := mysqlConnectionInfo(endpoint)
	if err != nil {
		return err
	}

	if mysql.Replica != "" {
		mysql.Host = mysql.Replica
	}

	cmd := fmt.Sprintf("%s/mysqldump %s %s", mysql.Bin, mysql.Options, connectionString(mysql, true))
	plugin.DEBUG("Executing: `%s`", cmd)
	return plugin.Exec(cmd, plugin.STDOUT)
}

// Restore mysql database
func (p MySQLPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	mysql, err := mysqlConnectionInfo(endpoint)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("%s/mysql %s", mysql.Bin, connectionString(mysql, false))
	plugin.DEBUG("Exec: %s", cmd)
	return plugin.Exec(cmd, plugin.STDIN)
}

func (p MySQLPlugin) Store(endpoint plugin.ShieldEndpoint) (string, int64, error) {
	return "", 0, plugin.UNIMPLEMENTED
}

func (p MySQLPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p MySQLPlugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func connectionString(info *MySQLConnectionInfo, backup bool) string {
	// use env variable for communicating password, so it's less likely to appear in our logs/ps output
	os.Setenv("MYSQL_PWD", info.Password)

	var db string
	if info.Database != "" {
		db = info.Database
	} else if backup {
		db = "--all-databases"
	}

	return fmt.Sprintf("%s -h %s -P %s -u %s", db, info.Host, info.Port, info.User)
}

func mysqlConnectionInfo(endpoint plugin.ShieldEndpoint) (*MySQLConnectionInfo, error) {
	user, err := endpoint.StringValue("mysql_user")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("MYSQL_USER: '%s'", user)

	password, err := endpoint.StringValue("mysql_password")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("MYSQL_PWD: '%s'", password)

	host, err := endpoint.StringValueDefault("mysql_host", DefaultHost)
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("MYSQL_HOST: '%s'", host)

	port, err := endpoint.StringValueDefault("mysql_port", DefaultPort)
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("MYSQL_PORT: '%s'", port)

	replica, err := endpoint.StringValueDefault("mysql_read_replica", "")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("MYSQL_READ_REPLICA: '%s'", replica)

	options, err := endpoint.StringValueDefault("mysql_options", "")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("MYSQL_OPTIONS: '%s'", options)

	db, err := endpoint.StringValueDefault("mysql_database", "")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("MYSQL_DB: '%s'", db)

	bin, err := endpoint.StringValueDefault("mysql_bindir", "/var/vcap/packages/shield-mysql/bin")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("MYSQL_BINDIR: '%s'", bin)

	return &MySQLConnectionInfo{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Bin:      bin,
		Replica:  replica,
		Database: db,
		Options:  options,
	}, nil
}
