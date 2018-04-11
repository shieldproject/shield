// The `postgres` plugin for SHIELD is intended to be a generic
// backup/restore plugin for a postgres server. It can be used against
// any postgres server compatible with the `psql` and `pg_dumpall` tools
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
// The endpoint configuration passed to this plugin is used to identify
// what postgres instance to back up, and how to connect to it. Your
// endpoint JSON should look something like this:
//
//    {
//        "pg_user":"username-for-postgres",
//        "pg_password":"password-for-above-user",     # optional
//        "pg_host":"hostname-or-ip-of-pg-server",     # optional
//        "pg_port":"port-above-pg-server-listens-on", # optional
//        "pg_database": "name-of-db-to-backup",       # optional
//        "pg_bindir": "PostgreSQL binaries directory" # optional
//    }
//
// Default Configuration
//
//    {
//        "pg_port"  : "5432",
//        "pg_bindir": "/var/vcap/packages/postgres-9.4/bin"
//    }
//
// The `pg_port` field is optional. If specified, the plugin will connect to the
// given port to perform backups. If not specified plugin will connect to
// default postgres port 5432.
//
// The `pg_database` field is optional.  If specified, the plugin will only
// perform backups of the named database.  If not specified (the default), all
// databases will be backed up.
//
// The `pg_bindir` field is optional. It specifies where to find the PostgreSQL
// binaries such as pg_dump / pg_dumpall / pg_restore. If specified, the plugin
// will attempt to use binaries from within the given directory. If not specified
// the plugin will default to trying to use binaries in
// '/var/vcap/packages/postgres-9.4/bin', which is provided by the
// `agent-pgtools' package in the SHIELD BOSH release.
//
// BACKUP DETAILS
//
// The `postgres` plugin makes use of `pg_dumpall -c` to back up all databases
// on the postgres server it connects to. There is currently no filtering of
// individual databases to back up, unless that is done via the postgres users
// and roles. The dumps generated include SQL to clean up existing databses/tables,
// so that the restore will go smoothly.
//
// Backing up with the `postgres` plugin will not drop any existing connections to the
// database, or restart the service.
//
// RESTORE DETAILS
//
// To restore, the `postgres` plugin connects to the postgres server using the `psql`
// command. It then feeds in the backup data (`pg_dumpall` output). To work around
// cases where the databases being restored cannot be recreated due to existing connections,
// the plugin disallows incoming connections for each database, and disconnects the existing
// connections, prior to dropping the database. Once the database is recreated, connections
// are once again allowed into the database.
//
// Restoring with the `postgres` plugin will terminate existing connections to the database,
// but does not need to restart the postgres service.
//
// DEPENDENCIES
//
// This plugin relies on the `pg_dumpall` and `psql` commands. Please ensure that they
// are present on the system that will be running the backups + restores for postgres.
// If you are using shield-boshrelease to deploy SHIELD, these tools are provided, if you
// include the `agent-pgtools` job template along side your `shield-agent`.
//
package main

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"regexp"

	fmt "github.com/jhunt/go-ansi"

	"github.com/starkandwayne/shield/plugin"
)

var (
	DefaultPort = "5432"
)

func main() {
	p := PostgresPlugin{
		Name:    "PostgreSQL Backup Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
		Example: `
{
  "pg_user"     : "username",   # REQUIRED

  "pg_password" : "password",
  "pg_host"     : "10.0.0.1",
  "pg_port"     : "5432",             # Port that PostgreSQL is listening on
  "pg_database" : "db1",              # Limit backup/restore operation to this database
  "pg_bindir"   : "/path/to/pg/bin"   # Where to find the psql command
}
`,
		Defaults: `
{
  "pg_port"  : "5432",
  "pg_bindir": "/var/vcap/packages/postgres-9.4/bin"
}
`,
		Fields: []plugin.Field{
			plugin.Field{
				Mode:  "target",
				Name:  "pg_host",
				Type:  "string",
				Title: "PostgreSQL Host",
				Help:  "The hostname or IP address of your PostgreSQL server.",
			},
			plugin.Field{
				Mode:    "target",
				Name:    "pg_port",
				Type:    "port",
				Title:   "PostgreSQL Port",
				Help:    "The TCP port that PostgreSQL is bound to, listening for incoming connections.",
				Default: "5432",
			},
			plugin.Field{
				Mode:     "target",
				Name:     "pg_user",
				Type:     "string",
				Title:    "PostgreSQL Username",
				Help:     "Username to authenticate to PostgreSQL as.",
				Required: true,
			},
			plugin.Field{
				Mode:  "target",
				Name:  "pg_password",
				Type:  "password",
				Title: "PostgreSQL Password",
				Help:  "Password to authenticate to PostgreSQL as.",
			},
			plugin.Field{
				Mode:  "target",
				Name:  "pg_database",
				Type:  "string",
				Title: "Database to Backup",
				Help:  "Limit scope of the backup to include only this database.  By default, all databases will be backed up.",
			},
			plugin.Field{
				Mode:    "target",
				Name:    "pg_bindir",
				Type:    "abspath",
				Title:   "Path to PostgreSQL bin/ directory",
				Help:    "The absolute path to the bin/ directory that contains the `psql` command.",
				Default: "/var/vcap/packages/postgres-9.4/bin",
			},
		},
	}

	plugin.Run(p)
}

type PostgresPlugin plugin.PluginInfo

type PostgresConnectionInfo struct {
	Host     string
	Port     string
	User     string
	Password string
	Bin      string
	Database string
}

func (p PostgresPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p PostgresPlugin) Validate(endpoint plugin.ShieldEndpoint) error {
	var (
		s    string
		err  error
		fail bool
	)

	s, err = endpoint.StringValueDefault("pg_host", "")
	if err != nil {
		fmt.Printf("@R{\u2717 pg_host      %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 pg_host}      using @C{localhost}\n")
	} else {
		fmt.Printf("@G{\u2713 pg_host}      @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("pg_port", "")
	if err != nil {
		fmt.Printf("@R{\u2717 pg_port      %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 pg_port}      using default port @C{%s}\n", DefaultPort)
	} else {
		fmt.Printf("@G{\u2713 pg_port}      @C{%s}\n", s)
	}

	s, err = endpoint.StringValue("pg_user")
	if err != nil {
		fmt.Printf("@R{\u2717 pg_user      %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 pg_user}      @C{%s}\n", plugin.Redact(s))
	}

	s, err = endpoint.StringValueDefault("pg_password", "")
	if err != nil {
		fmt.Printf("@R{\u2717 pg_password  %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 pg_password}  none (no credentials will be sent)\n")
	} else {
		fmt.Printf("@G{\u2713 pg_password}  @C{%s}\n", plugin.Redact(s))
	}

	s, err = endpoint.StringValueDefault("pg_database", "")
	if err != nil {
		fmt.Printf("@R{\u2717 pg_database  %s}\n", err)
	} else if s == "" {
		fmt.Printf("@G{\u2713 pg_database}  none (all databases will be backed up)\n")
	} else {
		fmt.Printf("@G{\u2713 pg_database}  @C{%s}\n", s)
	}

	if fail {
		return fmt.Errorf("postgres: invalid configuration")
	}
	return nil
}

func (p PostgresPlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	pg, err := pgConnectionInfo(endpoint)
	if err != nil {
		return err
	}

	setupEnvironmentVariables(pg)

	cmd := ""
	if pg.Database != "" {
		// Run dump all on the specified db
		cmd = fmt.Sprintf("%s/pg_dump %s -C -c --no-password", pg.Bin, pg.Database)
	} else {
		// Else run dump on all
		cmd = fmt.Sprintf("%s/pg_dumpall -c --no-password", pg.Bin)
	}
	plugin.DEBUG("Executing: `%s`", cmd)
	return plugin.Exec(cmd, plugin.STDOUT)
}

func (p PostgresPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	pg, err := pgConnectionInfo(endpoint)
	if err != nil {
		return err
	}

	setupEnvironmentVariables(pg)

	cmd := exec.Command(fmt.Sprintf("%s/psql", pg.Bin), "-d", "postgres")
	plugin.DEBUG("Exec: %s/psql -d postgres", pg.Bin)
	plugin.DEBUG("Redirecting stdout and stderr to stderr")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	scanErr := make(chan error)
	go func(out io.WriteCloser, in io.Reader, errChan chan<- error) {
		plugin.DEBUG("Starting to read SQL statements from stdin...")
		r := bufio.NewReader(in)
		reg := regexp.MustCompile("^DROP DATABASE (.*);$")
		i := 0
		for {
			thisLine := []byte{}
			isPrefix := true
			var err error
			for isPrefix {
				var tmpLine []byte
				tmpLine, isPrefix, err = r.ReadLine()
				if err != nil {
					if err == io.EOF {
						goto eof
					}
					errChan <- err
					return
				}
				thisLine = append(thisLine, tmpLine...)
			}
			m := reg.FindStringSubmatch(string(thisLine))
			if len(m) > 0 {
				plugin.DEBUG("Found dropped database '%s' on line %d", m[1], i)
				out.Write([]byte(fmt.Sprintf("UPDATE pg_database SET datallowconn = 'false' WHERE datname = '%s';\n", m[1])))
				out.Write([]byte(fmt.Sprintf("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s';\n", m[1])))
			}
			_, err = out.Write([]byte(string(thisLine) + "\n"))
			if err != nil {
				plugin.DEBUG("Error when writing to output: %s", err)
				errChan <- err
				return
			}
			i++
		}
	eof:
		plugin.DEBUG("Completed restore with %d lines of SQL", i)
		out.Close()
		errChan <- nil
	}(stdin, os.Stdin, scanErr)
	err = cmd.Run()
	if err != nil {
		return err
	}
	return <-scanErr
}

func (p PostgresPlugin) Store(endpoint plugin.ShieldEndpoint) (string, int64, error) {
	return "", 0, plugin.UNIMPLEMENTED
}

func (p PostgresPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p PostgresPlugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func setupEnvironmentVariables(pg *PostgresConnectionInfo) {
	plugin.DEBUG("Setting up env:\n   PGUSER=%s, PGPASSWORD=%s, PGHOST=%s, PGPORT=%s", pg.User, pg.Password, pg.Host, pg.Port)

	os.Setenv("PGUSER", pg.User)
	os.Setenv("PGPASSWORD", pg.Password)
	os.Setenv("PGHOST", pg.Host)
	os.Setenv("PGPORT", pg.Port)
}

func pgConnectionInfo(endpoint plugin.ShieldEndpoint) (*PostgresConnectionInfo, error) {
	user, err := endpoint.StringValue("pg_user")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("PGUSER: '%s'", user)

	password, err := endpoint.StringValueDefault("pg_password", "")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("PGPASSWORD: '%s'", password)

	host, err := endpoint.StringValueDefault("pg_host", "")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("PGHOST: '%s'", host)

	port, err := endpoint.StringValueDefault("pg_port", DefaultPort)
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("PGPORT: '%s'", port)

	database, err := endpoint.StringValueDefault("pg_database", "")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("PGDATABASE: '%s'", database)

	bin, err := endpoint.StringValueDefault("pg_bindir", "/var/vcap/packages/postgres-9.4/bin")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("PGBINDIR: '%s'", bin)

	return &PostgresConnectionInfo{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Bin:      bin,
		Database: database,
	}, nil
}
