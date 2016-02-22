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
//        "mysql_user":"username-for-mysql",
//        "mysql_password":"password-for-above-user",
//        "mysql_host":"hostname-or-ip-of-mysql-server",
//        "mysql_port":"port-above-mysql-server-listens-on",
//        "mysql_read_replica":"hostname-or-ip-of-mysql-replica-server",  #OPTIONAL
//        "mysql_database": "your-database-name"  #OPTIONAL
//    }
//
// BACKUP DETAILS
//
// If `mysql_database` is not specified in the plugin configuration, the `mysql` plugin makes
// use of `mysqldump --all-databases` to back up all databases on the mysql server it connects to.
// Otherwise, it backs up ONLY the specified database. The dumps generated include
// SQL to clean up existing tables of the databases, so that the restores will go smoothly.
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
	"fmt"
	"os"

	. "github.com/starkandwayne/shield/plugin"
)

func main() {
	p := MySQLPlugin{
		Name:    "MySQL Backup Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
	}

	Run(p)
}

type MySQLPlugin PluginInfo

type MySQLConnectionInfo struct {
	Host     string
	Port     string
	User     string
	Password string
	Bin      string
	Replica  string
	Database string
}

func (p MySQLPlugin) Meta() PluginInfo {
	return PluginInfo(p)
}

// Backup mysql database
func (p MySQLPlugin) Backup(endpoint ShieldEndpoint) error {
	mysql, err := mysqlConnectionInfo(endpoint)
	if err != nil {
		return err
	}

	if mysql.Replica != "" {
		mysql.Host = mysql.Replica
	}

	cmd := fmt.Sprintf("%s/mysqldump %s", mysql.Bin, connectionString(mysql))
	DEBUG("Executing: `%s`", cmd)
	return Exec(cmd, STDOUT)
}

// Restore mysql database
func (p MySQLPlugin) Restore(endpoint ShieldEndpoint) error {
	mysql, err := mysqlConnectionInfo(endpoint)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("%s/mysql %s", mysql.Bin, connectionString(mysql))
	DEBUG("Exec: %s", cmd)
	return Exec(cmd, STDIN)
}

func (p MySQLPlugin) Store(endpoint ShieldEndpoint) (string, error) {
	return "", UNIMPLEMENTED
}

func (p MySQLPlugin) Retrieve(endpoint ShieldEndpoint, file string) error {
	return UNIMPLEMENTED
}

func (p MySQLPlugin) Purge(endpoint ShieldEndpoint, file string) error {
	return UNIMPLEMENTED
}

func connectionString(info *MySQLConnectionInfo) string {
	// use env variable for communicating password, so it's less likely to appear in our logs/ps output
	os.Setenv("MYSQL_PWD", info.Password)

	db := "--all-databases"
	if info.Database != "" {
		db = info.Database
	}

	return fmt.Sprintf("%s -h %s -P %s -u %s", db, info.Host, info.Port, info.User)
}

func mysqlConnectionInfo(endpoint ShieldEndpoint) (*MySQLConnectionInfo, error) {
	user, err := endpoint.StringValue("mysql_user")
	if err != nil {
		return nil, err
	}
	DEBUG("MYSQL_USER: '%s'", user)

	password, err := endpoint.StringValue("mysql_password")
	if err != nil {
		return nil, err
	}
	DEBUG("MYSQL_PWD: '%s'", password)

	host, err := endpoint.StringValue("mysql_host")
	if err != nil {
		return nil, err
	}
	DEBUG("MYSQL_HOST: '%s'", host)

	port, err := endpoint.StringValue("mysql_port")
	if err != nil {
		return nil, err
	}
	DEBUG("MYSQL_PORT: '%s'", port)

	replica, _ := endpoint.StringValue("mysql_read_replica")
	DEBUG("MYSQL_READ_REPLICA: '%s'", replica)

	db, _ := endpoint.StringValue("mysql_database")
	DEBUG("MYSQL_DB: '%s'", db)

	bin := "/var/vcap/packages/shield-mysql/bin"
	DEBUG("MYSQL_BIN_DIR: '%s'", bin)

	return &MySQLConnectionInfo{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Bin:      bin,
		Replica:  replica,
		Database: db,
	}, nil
}
