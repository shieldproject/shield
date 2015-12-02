package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"

	"github.com/starkandwayne/shield/plugin"
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
	}

	plugin.Run(p)
}

type PostgresPlugin plugin.PluginInfo

type PostgresConnectionInfo struct {
	Host     string
	Port     string
	User     string
	Password string
	BDB      string
	RDB      string
	DumpArgs string
	Bin      string
}

func (p PostgresPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p PostgresPlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	pg, err := pgConnectionInfo(endpoint)
	if err != nil {
		return err
	}

	setupEnvironmentVariables(pg)
	cmd := fmt.Sprintf("%s/pg_dump %s -cC --format p --no-password %s", pg.Bin, pg.DumpArgs, pg.BDB)
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
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	go func(out io.WriteCloser, in io.Reader) {
		b := bufio.NewScanner(in)
		reg := regexp.MustCompile("^DROP DATABASE (.*);$")
		for b.Scan() {
			m := reg.FindStringSubmatch(b.Text())
			if len(m) > 0 {
				plugin.DEBUG("Match found: %s", m[1])
				out.Write([]byte(fmt.Sprintf("UPDATE pg_database SET datallowconn = 'false' WHERE datname = '%s';\n", m[1])))
				out.Write([]byte(fmt.Sprintf("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s';\n", m[1])))
			}
			_, err = out.Write([]byte(b.Text() + "\n"))
			if err != nil {
				plugin.DEBUG("Scanner had an error: %s", err)
			}
		}
		out.Close()
	}(stdin, os.Stdin)
	return cmd.Run()
}

func (p PostgresPlugin) Store(endpoint plugin.ShieldEndpoint) (string, error) {
	return "", plugin.UNIMPLEMENTED
}

func (p PostgresPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p PostgresPlugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func setupEnvironmentVariables(pg *PostgresConnectionInfo) {
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

	password, err := endpoint.StringValue("pg_password")
	if err != nil {
		return nil, err
	}

	host, err := endpoint.StringValue("pg_host")
	if err != nil {
		return nil, err
	}

	port, err := endpoint.StringValue("pg_port")
	if err != nil {
		return nil, err
	}

	bdb, err := endpoint.StringValue("pg_db_tobkp")
	if err != nil {
		return nil, err
	}

	rdb, err := endpoint.StringValue("pg_db_tores")
	if err != nil {
		return nil, err
	}

	bin, err := endpoint.StringValue("pg_bindir")
	if err != nil {
		return nil, err
	}

	dumpArgs, err := endpoint.StringValue("pg_dump_args")
	if err != nil {
		return nil, err
	}

	return &PostgresConnectionInfo{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		BDB:      bdb,
		RDB:      rdb,
		DumpArgs: dumpArgs,
		Bin:      bin,
	}, nil
}
