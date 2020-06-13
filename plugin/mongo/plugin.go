package mongo

import (
	"io"
	"strings"

	fmt "github.com/jhunt/go-ansi"

	"github.com/shieldproject/shield/plugin"
)

var (
	DefaultHost        = "127.0.0.1"
	DefaultPort        = "27017"
	DefaultMongoBinDir = "/var/vcap/packages/shield-mongo/bin"
)

func New() plugin.Plugin {
	return MongoPlugin{
		Name:    "MongoDB Backup Plugin",
		Author:  "Szlachta, Jacek",
		Version: "0.0.1",
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
				Name:    "mongo_authdb",
				Type:    "string",
				Title:   "Authentication Database",
				Help:    "The database to authenticate against.",
				Example: "admin",
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

			plugin.Field{
				Mode:    "target",
				Name:    "mongorestore_options",
				Type:    "string",
				Title:   "mongorestore options",
				Help:    "You can tune `mongorestore` (only) by specifying additional options and command-line arguments.  If you don't know why you might need this, leave it blank.",
				Example: "--ssl",
			},

			plugin.Field{
				Mode:    "target",
				Name:    "mongodump_options",
				Type:    "string",
				Title:   "mongodump options",
				Help:    "You can tune `mongodump` (only) by specifying additional options and command-line arguments.  If you don't know why you might need this, leave it blank.",
				Example: "--ssl",
			},
		},
	}

}

func Run() {
	plugin.Run(New())
}

type MongoPlugin plugin.PluginInfo

type MongoConnectionInfo struct {
	Host           string
	Port           string
	User           string
	Password       string
	Bin            string
	Database       string
	AuthDB         string
	Options        string
	DumpOptions    string
	RestoreOptions string
}

func (p MongoPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p MongoPlugin) Validate(log io.Writer, endpoint plugin.ShieldEndpoint) error {
	var (
		s    string
		err  error
		fail bool
	)

	s, err = endpoint.StringValueDefault("mongo_host", "")
	if err != nil {
		fmt.Fprintf(log, "@R{\u2717 mongo_host          %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Fprintf(log, "@G{\u2713 mongo_host}          using default host @C{%s}\n", DefaultHost)
	} else {
		fmt.Fprintf(log, "@G{\u2713 mongo_host}          @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("mongo_port", "")
	if err != nil {
		fmt.Fprintf(log, "@R{\u2717 mongo_port          %s}\n", err)
	} else if s == "" {
		fmt.Fprintf(log, "@G{\u2713 mongo_port}          using default port @C{%s}\n", DefaultPort)
	} else {
		fmt.Fprintf(log, "@G{\u2713 mongo_port}          @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("mongo_user", "")
	if err != nil {
		fmt.Fprintf(log, "@R{\u2717 mongo_user          %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Fprintf(log, "@G{\u2713 mongo_user}          (none)\n")
	} else {
		fmt.Fprintf(log, "@G{\u2713 mongo_user}          @C{%s}\n", plugin.Redact(s))
	}

	s, err = endpoint.StringValueDefault("mongo_password", "")
	if err != nil {
		fmt.Fprintf(log, "@R{\u2717 mongo_password      %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Fprintf(log, "@G{\u2713 mongo_password}      (none)\n")
	} else {
		fmt.Fprintf(log, "@G{\u2713 mongo_password}      @C{%s}\n", plugin.Redact(s))
	}

	if fail {
		return fmt.Errorf("mongo: invalid configuration")
	}
	return nil
}

// Backup mongo database
func (p MongoPlugin) Backup(out io.Writer, log io.Writer, endpoint plugin.ShieldEndpoint) error {
	mongo, err := mongoConnectionInfo(endpoint)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("%s/mongodump %s %s", mongo.Bin, connectionString(mongo), mongo.DumpOptions)
	plugin.DEBUG("Executing: `%s`", cmd)
	return plugin.Exec(cmd, nil, out, log)
}

// Restore mongo database
func (p MongoPlugin) Restore(in io.Reader, log io.Writer, endpoint plugin.ShieldEndpoint) error {
	mongo, err := mongoConnectionInfo(endpoint)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("%s/mongorestore %s %s", mongo.Bin, connectionString(mongo), mongo.RestoreOptions)
	plugin.DEBUG("Exec: %s", cmd)
	return plugin.Exec(cmd, in, log, log)
}

func connectionString(info *MongoConnectionInfo) string {
	opts := fmt.Sprintf("--archive --host %s", info.Host)

	if info.Options != "" {
		opts += fmt.Sprintf(" %s ", info.Options)
	}

	if info.Database != "" {
		opts += fmt.Sprintf(" --db %s", info.Database)
	}

	if info.User != "" && info.Password != "" {
		opts += fmt.Sprintf(" --authenticationDatabase %s --username %s --password %s",
			info.AuthDB, info.User, info.Password)
	}

	if !strings.ContainsAny(info.Host, ":") {
		opts += fmt.Sprintf(" --port %s", info.Port)
	}

	return opts
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

	authdb, err := endpoint.StringValueDefault("mongo_authdb", "admin")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("MONGO_AUTHDB: '%s'", authdb)

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

	dumpOptions, err := endpoint.StringValueDefault("mongodump_options", "")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("MONGODUMP_OPTIONS: '%s'", dumpOptions)

	restoreOptions, err := endpoint.StringValueDefault("mongorestore_options", "")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("MONGORESTORE_OPTIONS: '%s'", restoreOptions)

	return &MongoConnectionInfo{
		Host:           host,
		Port:           port,
		User:           user,
		Password:       password,
		Bin:            bin,
		Database:       db,
		AuthDB:         authdb,
		Options:        options,
		DumpOptions:    dumpOptions,
		RestoreOptions: restoreOptions,
	}, nil
}
