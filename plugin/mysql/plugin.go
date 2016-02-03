package main

import (
	"fmt"

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

	cmd := fmt.Sprintf("%s/mysqldump --all-databases %s", mysql.Bin, connectionString(mysql))
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

// Store mysql - TODO
func (p MySQLPlugin) Store(endpoint ShieldEndpoint) (string, error) {
	return "", UNIMPLEMENTED
}

// Retrieve mysql - TODO
func (p MySQLPlugin) Retrieve(endpoint ShieldEndpoint, file string) error {
	return UNIMPLEMENTED
}

// Purge mysql - TODO
func (p MySQLPlugin) Purge(endpoint ShieldEndpoint, file string) error {
	return UNIMPLEMENTED
}

func connectionString(info *MySQLConnectionInfo) string {
	return fmt.Sprintf("-h %s -P %s -u %s -p %s", info.Host, info.Port, info.User, info.Password)
}

func mysqlConnectionInfo(endpoint ShieldEndpoint) (*MySQLConnectionInfo, error) {
	user, err := endpoint.StringValue("mysql_user")
	if err != nil {
		return nil, err
	}
	DEBUG("MYSQLUSER: '%s'", user)

	password, err := endpoint.StringValue("mysql_password")
	if err != nil {
		return nil, err
	}
	DEBUG("MYSQLPASSWORD: '%s'", password)

	host, err := endpoint.StringValue("mysql_host")
	if err != nil {
		return nil, err
	}
	DEBUG("MYSQLHOST: '%s'", host)

	port, err := endpoint.StringValue("mysql_port")
	if err != nil {
		return nil, err
	}
	DEBUG("MYSQLPORT: '%s'", port)

	bin := "/var/vcap/packages/shield-mysql/bin"
	DEBUG("MYSQLBINDIR: '%s'", bin)

	return &MySQLConnectionInfo{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Bin:      bin,
	}, nil
}
