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

	bin, err := endpoint.StringValue("pg_bindir")
	if err != nil {
		return nil, err
	}
	DEBUG("PGBINDIR: '%s'", bin)

	return &PostgresConnectionInfo{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Bin:      bin,
	}, nil
}
