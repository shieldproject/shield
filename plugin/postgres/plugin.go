package main

import (
	"fmt"
	"github.com/starkandwayne/shield/plugin"
	"os"
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
	DB       string
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
	return plugin.Exec(fmt.Sprintf("%s/pg_dump %s -cC --format p --no-password %s", pg.Bin, pg.DumpArgs, pg.DB), plugin.STDOUT)
}

func (p PostgresPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	pg, err := pgConnectionInfo(endpoint)
	if err != nil {
		return err
	}

	setupEnvironmentVariables(pg)
	return plugin.Exec(fmt.Sprintf("%s/psql -d %s", pg.Bin, pg.DB), plugin.STDIN)
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

	db, err := endpoint.StringValue("pg_database")
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
		DB:       db,
		DumpArgs: dumpArgs,
		Bin:      bin,
	}, nil
}
