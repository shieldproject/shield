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
//        "pg_password":"password-for-above-user",
//        "pg_host":"hostname-or-ip-of-pg-server",
//        "pg_port":"port-above-pg-server-listens-on"
//    }
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
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"

	. "github.com/starkandwayne/shield/plugin"
)

func main() {
	p := PostgresPlugin{
		Name:    "PostgreSQL Backup Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
	}

	Run(p)
}

type PostgresPlugin PluginInfo

type PostgresConnectionInfo struct {
	Host     string
	Port     string
	User     string
	Password string
	Bin      string
}

func (p PostgresPlugin) Meta() PluginInfo {
	return PluginInfo(p)
}

func (p PostgresPlugin) Backup(endpoint ShieldEndpoint) error {
	pg, err := pgConnectionInfo(endpoint)
	if err != nil {
		return err
	}

	setupEnvironmentVariables(pg)
	cmd := fmt.Sprintf("%s/pg_dumpall -c --no-password", pg.Bin)
	DEBUG("Executing: `%s`", cmd)
	return Exec(cmd, STDOUT)
}

func (p PostgresPlugin) Restore(endpoint ShieldEndpoint) error {
	pg, err := pgConnectionInfo(endpoint)
	if err != nil {
		return err
	}

	setupEnvironmentVariables(pg)

	cmd := exec.Command(fmt.Sprintf("%s/psql", pg.Bin), "-d", "postgres")
	DEBUG("Exec: %s/psql -d postgres", pg.Bin)
	DEBUG("Redirecting stdout and stderr to stderr")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	go func(out io.WriteCloser, in io.Reader) {
		DEBUG("Starting to read SQL statements from stdin...")
		b := bufio.NewScanner(in)
		reg := regexp.MustCompile("^DROP DATABASE (.*);$")
		i := 1
		for b.Scan() {
			m := reg.FindStringSubmatch(b.Text())
			if len(m) > 0 {
				DEBUG("Found dropped database '%s' on line %d", m[1], i)
				out.Write([]byte(fmt.Sprintf("UPDATE pg_database SET datallowconn = 'false' WHERE datname = '%s';\n", m[1])))
				out.Write([]byte(fmt.Sprintf("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s';\n", m[1])))
			}
			_, err = out.Write([]byte(b.Text() + "\n"))
			if err != nil {
				DEBUG("Scanner had an error: %s", err)
			}
			i++
		}
		DEBUG("Completed restore with %d lines of SQL", i)
		out.Close()
	}(stdin, os.Stdin)
	return cmd.Run()
}

func (p PostgresPlugin) Store(endpoint ShieldEndpoint) (string, error) {
	return "", UNIMPLEMENTED
}

func (p PostgresPlugin) Retrieve(endpoint ShieldEndpoint, file string) error {
	return UNIMPLEMENTED
}

func (p PostgresPlugin) Purge(endpoint ShieldEndpoint, file string) error {
	return UNIMPLEMENTED
}

func setupEnvironmentVariables(pg *PostgresConnectionInfo) {
	DEBUG("Setting up env:\n   PGUSER=%s, PGPASSWORD=%s, PGHOST=%s, PGPORT=%s", pg.User, pg.Password, pg.Host, pg.Port)
	os.Setenv("PGUSER", pg.User)
	os.Setenv("PGPASSWORD", pg.Password)
	os.Setenv("PGHOST", pg.Host)
	os.Setenv("PGPORT", pg.Port)
}

func pgConnectionInfo(endpoint ShieldEndpoint) (*PostgresConnectionInfo, error) {
	user, err := endpoint.StringValue("pg_user")
	if err != nil {
		return nil, err
	}
	DEBUG("PGUSER: '%s'", user)

	password, err := endpoint.StringValue("pg_password")
	if err != nil {
		return nil, err
	}
	DEBUG("PGPASSWORD: '%s'", password)

	host, err := endpoint.StringValue("pg_host")
	if err != nil {
		return nil, err
	}
	DEBUG("PGHOST: '%s'", host)

	port, err := endpoint.StringValue("pg_port")
	if err != nil {
		return nil, err
	}
	DEBUG("PGPORT: '%s'", port)

	bin := "/var/vcap/packages/postgres-9.4/bin"
	DEBUG("PGBINDIR: '%s'", bin)

	return &PostgresConnectionInfo{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Bin:      bin,
	}, nil
}
