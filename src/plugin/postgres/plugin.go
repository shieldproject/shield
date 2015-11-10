package main

import (
	"fmt"
	"os"
	"plugin"
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

func (p PostgresPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p PostgresPlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	pgUser, err := endpoint.StringValue("pg_user")
	if err != nil {
		return err
	}
	os.Setenv("PGUSER", pgUser)

	pgPass, err := endpoint.StringValue("pg_password")
	if err != nil {
		return err
	}
	os.Setenv("PGPASSWORD", pgPass)

	pgHost, err := endpoint.StringValue("pg_host")
	if err != nil {
		return err
	}
	os.Setenv("PGHOST", pgHost)

	pgPort, err := endpoint.StringValue("pg_port")
	if err != nil {
		return err
	}
	os.Setenv("PGPORT", pgPort)

	pgDB, err := endpoint.StringValue("pg_database")
	if err != nil {
		return err
	}

	pgDumpBin, err := endpoint.StringValue("pg_dump")
	if err != nil {
		return err
	}

	pgDumpArgs, err := endpoint.StringValue("pg_dump_args")
	if err != nil {
		return err
	}

	return plugin.Exec(fmt.Sprintf("%s %s -cC --format p --no-password %s", pgDumpBin, pgDumpArgs, pgDB), plugin.STDOUT)
}

func (p PostgresPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	pgUser, err := endpoint.StringValue("pg_user")
	if err != nil {
		return err
	}
	os.Setenv("PGUSER", pgUser)

	pgPass, err := endpoint.StringValue("pg_password")
	if err != nil {
		return err
	}
	os.Setenv("PGPASSWORD", pgPass)

	pgHost, err := endpoint.StringValue("pg_host")
	if err != nil {
		return err
	}
	os.Setenv("PGHOST", pgHost)

	pgPort, err := endpoint.StringValue("pg_port")
	if err != nil {
		return err
	}
	os.Setenv("PGPORT", pgPort)

	pgDB, err := endpoint.StringValue("pg_database")
	if err != nil {
		return err
	}

	pgBin, err := endpoint.StringValue("pg_psql")
	if err != nil {
		return err
	}

	return plugin.Exec(fmt.Sprintf("%s -d %s", pgBin, pgDB), plugin.STDIN)
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
