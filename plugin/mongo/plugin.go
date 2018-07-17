package main

import (
	fmt "github.com/jhunt/go-ansi"

	"github.com/starkandwayne/shield/plugin"
)

var (
	DefaultHost        = "127.0.0.1"
	DefaultPort        = "27017"
	DefaultMongoBinDir = "/var/vcap/packages/shield-mongo/bin"
)

func main() {
	p := MongoPlugin{
		Name:    "MongoDB Backup Plugin",
		Author:  "Szlachta, Jacek",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
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
  "mongo_options"  : "--ssl"        # optional
}
`,
		Defaults: `
{
  "mongo_host"   : "127.0.0.1",
  "mongo_port"   : "27017",
  "mongo_bindir" : "/var/vcap/packages/shield-mongo/bin"
}
`,
		Fields: []plugin.Field{
			plugin.Field{
				Mode:    "target",
				Name:    "mongo_host",
				Type:    "string",
				Title:   "MongoDB Host",
				Help:    "The hostname or IP address of your MongoDB server.",
				Default: "127.0.0.1",
			},
			plugin.Field{
				Mode:    "target",
				Name:    "mongo_port",
				Type:    "port",
				Title:   "MongoDB Port",
				Help:    "The TCP port that MongoDB is bound to, listening for incoming connections.",
				Default: "27017",
			},
			plugin.Field{
				Mode:  "target",
				Name:  "mongo_user",
				Title: "MongoDB Username",
				Type:  "string",
				Help:  "Username to authenticate to MongoDB as.",
			},
			plugin.Field{
				Mode:  "target",
				Name:  "mongo_password",
				Type:  "password",
				Title: "MongoDB Password",
				Help:  "The password to authenticate to MongoDB as.",
			},
			plugin.Field{
				Mode:    "target",
				Name:    "mongo_database",
				Type:    "string",
				Title:   "Backup Database",
				Help:    "Limit the scope of the backup to the named database.  By default, all databases are backed up.",
				Example: "salesdb1",
			},

			plugin.Field{
				Mode:    "target",
				Name:    "mongo_bindir",
				Type:    "abspath",
				Title:   "Path to the MongoDB bin/ directory",
				Help:    "The absolute path to the bin/ directory that houses the `mongodump` and `mongorestore` commands.",
				Default: "/var/vcap/packages/shield-mongo/bin",
			},

			plugin.Field{
				Mode:    "target",
				Name:    "mongo_options",
				Type:    "string",
				Title:   "Mongo options",
				Help:    "You can tune `mongodump` and `mongorestore` by specifying additional options and command-line arguments.  If you don't know why you might need this, leave it blank.",
				Example: "--ssl",
			},
		},
	}

	plugin.Run(p)
}

type MongoPlugin plugin.PluginInfo

type MongoConnectionInfo struct {
	Host     string
	Port     string
	User     string
	Password string
	Bin      string
	Database string
	Options  string
}

func (p MongoPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p MongoPlugin) Validate(endpoint plugin.ShieldEndpoint) error {
	var (
		s    string
		err  error
		fail bool
	)

	s, err = endpoint.StringValueDefault("mongo_host", "")
	if err != nil {
		fmt.Printf("@R{\u2717 mongo_host          %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 mongo_host}          using default host @C{%s}\n", DefaultHost)
	} else {
		fmt.Printf("@G{\u2713 mongo_host}          @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("mongo_port", "")
	if err != nil {
		fmt.Printf("@R{\u2717 mongo_port          %s}\n", err)
	} else if s == "" {
		fmt.Printf("@G{\u2713 mongo_port}          using default port @C{%s}\n", DefaultPort)
	} else {
		fmt.Printf("@G{\u2713 mongo_port}          @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("mongo_user", "")
	if err != nil {
		fmt.Printf("@R{\u2717 mongo_user          %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 mongo_user}          (none)\n")
	} else {
		fmt.Printf("@G{\u2713 mongo_user}          @C{%s}\n", plugin.Redact(s))
	}

	s, err = endpoint.StringValueDefault("mongo_password", "")
	if err != nil {
		fmt.Printf("@R{\u2717 mongo_password      %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 mongo_password}      (none)\n")
	} else {
		fmt.Printf("@G{\u2713 mongo_password}      @C{%s}\n", plugin.Redact(s))
	}

	if fail {
		return fmt.Errorf("mongo: invalid configuration")
	}
	return nil
}

// Backup mongo database
func (p MongoPlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	mongo, err := mongoConnectionInfo(endpoint)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("%s/mongodump %s", mongo.Bin, connectionString(mongo, true))
	plugin.DEBUG("Executing: `%s`", cmd)
	return plugin.Exec(cmd, plugin.STDOUT)
}

// Restore mongo database
func (p MongoPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	mongo, err := mongoConnectionInfo(endpoint)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("%s/mongorestore %s", mongo.Bin, connectionString(mongo, false))
	plugin.DEBUG("Exec: %s", cmd)
	return plugin.Exec(cmd, plugin.STDIN)
}

func (p MongoPlugin) Store(endpoint plugin.ShieldEndpoint) (string, int64, error) {
	return "", 0, plugin.UNIMPLEMENTED
}

func (p MongoPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p MongoPlugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func connectionString(info *MongoConnectionInfo, backup bool) string {

	var options string
	if info.Options != "" {
		options = fmt.Sprintf(" %s ", info.Options)
	}

	var db string
	if info.Database != "" {
		db = fmt.Sprintf(" --db %s", info.Database)
	}

	var auth string
	if info.User != "" && info.Password != "" {
		auth = fmt.Sprintf(" --authenticationDatabase admin --username %s --password %s",
			info.User, info.Password)
	}

	return fmt.Sprintf("--archive --host %s --port %s%s%s%s",
		info.Host, info.Port, auth, db, options)
}

func mongoConnectionInfo(endpoint plugin.ShieldEndpoint) (*MongoConnectionInfo, error) {
	user, err := endpoint.StringValueDefault("mongo_user", "")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("MONGO_USER: '%s'", user)

	password, err := endpoint.StringValueDefault("mongo_password", "")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("MONGO_PWD: '%s'", password)

	host, err := endpoint.StringValueDefault("mongo_host", DefaultHost)
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("MONGO_HOST: '%s'", host)

	port, err := endpoint.StringValueDefault("mongo_port", DefaultPort)
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("MONGO_PORT: '%s'", port)

	db, err := endpoint.StringValueDefault("mongo_database", "")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("MONGO_DB: '%s'", db)

	bin, err := endpoint.StringValueDefault("mongo_bindir", DefaultMongoBinDir)
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("MONGO_BIN_DIR: '%s'", bin)

	options, err := endpoint.StringValueDefault("mongo_options", "")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("MONGO_OPTIONS: '%s'", options)

	return &MongoConnectionInfo{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Bin:      bin,
		Database: db,
		Options:  options,
	}, nil
}
