// The `mongo` plugin for SHIELD implements generic backup + restore
// functionality for mongodb. It can be used against
// mongodb server with `mongodump` and `mongorestore` tools
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
// mongodb instance to back up, and how to connect to it. Your endpoint JSON
// should look something like this:
//
//    {
//        "mongo_host"     : "127.0.0.1",   # optional
//        "mongo_port"     : "27017",       # optional
//        "mongo_user"     : "username",    # optional
//        "mongo_password" : "password",    # optional
//        "mongo_database" : "db",          # optional
//        "mongo_bindir"   : "/path/to/bin" # optional
//    }
//
// Default Configuration
//
//    {
//        "mongo_host"        : "127.0.0.1",
//        "DefaultPort"       : "27017",
//        "DefaultMongoBinDir": "/var/vcap/packages/shield-mongo/bin"
//    }
//
// BACKUP DETAILS
//
// If `mongo_database` is specified in the plugin configuration, the `mongo` plugin backs up ONLY
// the specified database using `mongodump` command.
// If `mongo_database` is not specified, all databases are backed up.
//
// Backing up with the `mongo` plugin will not drop any existing connections to the database,
// or restart the service.
//
//
//RESTORE DETAILS
//
// To restore, the `mongo` plugin connects to the mongodb server using the `mongorestore` command.
// It then feeds in the backup data (`mongodump` output). Unlike the the `postgres` plugin,
// this plugin does NOT need to disconnect any open connections to mongodb to perform the
// restoration.
//
// Restoring with the `mongo` plugin should not interrupt established connections to the service.
//
// DEPENDENCIES
//
// This plugin relies on the `mongodump` and `mongorestore` utilities. Please ensure
// that they are present on the system that will be running the backups + restores
// for mongodb.
// TODO: add agent-mongodb job template to shield-boshrelease
// If you are using shield-boshrelease to deploy SHIELD, these tools
// are provided so long as you include the `agent-mongodb` job template along side
// your `shield agent`.
//
package main

import (
	"fmt"

	"github.com/starkandwayne/goutils/ansi"

	. "github.com/starkandwayne/shield/plugin"
)

var (
	DefaultHost        = "127.0.0.1"
	DefaultPort        = "27017"
	DefaultMongoBinDir = "/var/vcap/packages/shield-mongo/bin"
)

func main() {
	p := MongoPlugin{
		Name:    "Mongo Backup Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
		Example: `
{
  "mongo_host"     : "127.0.0.1",   # optional
  "mongo_port"     : "27017",       # optional
  "mongo_user"     : "username",    # optional
  "mongo_password" : "password",    # optional
  "mongo_database" : "db",          # optional
  "mongo_bindir"   : "/path/to/bin" # optional
}
`,
		Defaults: `
{
  "mongo_host"   : "127.0.0.1",
  "mongo_port"   : "27017",
  "mongo_bindir" : "/var/vcap/packages/shield-mongo/bin"
}
`,
	}

	Run(p)
}

type MongoPlugin PluginInfo

type MongoConnectionInfo struct {
	Host     string
	Port     string
	User     string
	Password string
	Bin      string
	Database string
}

func (p MongoPlugin) Meta() PluginInfo {
	return PluginInfo(p)
}

func (p MongoPlugin) Validate(endpoint ShieldEndpoint) error {
	var (
		s    string
		err  error
		fail bool
	)

	s, err = endpoint.StringValueDefault("mongo_host", "")
	if err != nil {
		ansi.Printf("@R{\u2717 mongo_host          %s}\n", err)
		fail = true
	} else if s == "" {
		ansi.Printf("@G{\u2713 mongo_host}          using default host @C{%s}\n", DefaultHost)
	} else {
		ansi.Printf("@G{\u2713 mongo_host}          @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("mongo_port", "")
	if err != nil {
		ansi.Printf("@R{\u2717 mongo_port          %s}\n", err)
	} else if s == "" {
		ansi.Printf("@G{\u2713 mongo_port}          using default port @C{%s}\n", DefaultPort)
	} else {
		ansi.Printf("@G{\u2713 mongo_port}          @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("mongo_user", "")
	if err != nil {
		ansi.Printf("@R{\u2717 mongo_user          %s}\n", err)
		fail = true
	} else if s == "" {
		ansi.Printf("@G{\u2713 mongo_user}          (none)\n")
	} else {
		ansi.Printf("@G{\u2713 mongo_user}          @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("mongo_password", "")
	if err != nil {
		ansi.Printf("@R{\u2717 mongo_password      %s}\n", err)
		fail = true
	} else if s == "" {
		ansi.Printf("@G{\u2713 mongo_password}      (none)\n")
	} else {
		ansi.Printf("@G{\u2713 mongo_password}      @C{%s}\n", s)
	}

	if fail {
		return fmt.Errorf("mongo: invalid configuration")
	}
	return nil
}

// Backup mongo database
func (p MongoPlugin) Backup(endpoint ShieldEndpoint) error {
	mongo, err := mongoConnectionInfo(endpoint)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("%s/mongodump %s", mongo.Bin, connectionString(mongo, true))
	DEBUG("Executing: `%s`", cmd)
	return Exec(cmd, STDOUT)
}

// Restore mongo database
func (p MongoPlugin) Restore(endpoint ShieldEndpoint) error {
	mongo, err := mongoConnectionInfo(endpoint)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("%s/mongorestore %s", mongo.Bin, connectionString(mongo, false))
	DEBUG("Exec: %s", cmd)
	return Exec(cmd, STDIN)
}

func (p MongoPlugin) Store(endpoint ShieldEndpoint) (string, error) {
	return "", UNIMPLEMENTED
}

func (p MongoPlugin) Retrieve(endpoint ShieldEndpoint, file string) error {
	return UNIMPLEMENTED
}

func (p MongoPlugin) Purge(endpoint ShieldEndpoint, file string) error {
	return UNIMPLEMENTED
}

func connectionString(info *MongoConnectionInfo, backup bool) string {

	var db string
	if info.Database != "" {
		db = fmt.Sprintf(" --db %s", info.Database)
	}

	var auth string
	if info.User != "" && info.Password != "" {
		auth = fmt.Sprintf(" --authenticationDatabase admin --username %s --password %s",
			info.User, info.Password)
	}

	return fmt.Sprintf("--archive --host %s --port %s%s%s",
		info.Host, info.Port, auth, db)
}

func mongoConnectionInfo(endpoint ShieldEndpoint) (*MongoConnectionInfo, error) {
	user, err := endpoint.StringValueDefault("mongo_user", "")
	if err != nil {
		return nil, err
	}
	DEBUG("MONGO_USER: '%s'", user)

	password, err := endpoint.StringValueDefault("mongo_password", "")
	if err != nil {
		return nil, err
	}
	DEBUG("MONGO_PWD: '%s'", password)

	host, err := endpoint.StringValueDefault("mongo_host", DefaultHost)
	if err != nil {
		return nil, err
	}
	DEBUG("MONGO_HOST: '%s'", host)

	port, err := endpoint.StringValueDefault("mongo_port", DefaultPort)
	if err != nil {
		return nil, err
	}
	DEBUG("MONGO_PORT: '%s'", port)

	db, err := endpoint.StringValueDefault("mongo_database", "")
	if err != nil {
		return nil, err
	}
	DEBUG("MONGO_DB: '%s'", db)

	bin, err := endpoint.StringValueDefault("mongo_bindir", DefaultMongoBinDir)
	if err != nil {
		return nil, err
	}
	DEBUG("MONGO_BIN_DIR: '%s'", bin)

	return &MongoConnectionInfo{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Bin:      bin,
		Database: db,
	}, nil
}
