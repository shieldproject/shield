package main

import (
	"github.com/starkandwayne/shield/plugin"
	"time"
)

func main() {
	p := RedisPlugin{
		Name:    "Redis Backup Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
	}

	plugin.Run(p)
}

type RedisPlugin plugin.PluginInfo

func (p RedisPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p RedisPlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	err := plugin.Exec("tar -c -C /var/vcap/store .", plugin.STDOUT)
	if err != nil {
		return err
	}

	return nil
}

func (p RedisPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	err := plugin.Exec("/var/vcap/bosh/bin/monit stop all", plugin.STDOUT)
	if err != nil {
		return err
	}

	err = plugin.Exec("/bin/bash -c \"while [[ $(/var/vcap/bosh/bin/monit summary | /bin/grep running) ]]; do /bin/sleep 1; done\"", plugin.STDOUT)
	if err != nil {
		return err
	}

	err = plugin.Exec("/bin/pkill redis-server", plugin.STDOUT)
	if err != nil {
		return err
	}

	time.Sleep(2 * time.Second)

	err = plugin.Exec("/bin/pkill -9 redis-server", plugin.STDOUT)
	if err != nil {
		return err
	}

	time.Sleep(1 * time.Second)

	err = plugin.Exec("tar -x -C /var/vcap/store . ", plugin.STDIN)
	if err != nil {
		return err
	}

	err = plugin.Exec("/bin/bash -c '/usr/bin/yes | /usr/bin/find /var/vcap/store -name appendonly.aof -exec /var/vcap/packages/redis/bin/redis-check-aof --fix {} \\;'", plugin.STDOUT)
	if err != nil {
		return err
	}

	err = plugin.Exec("/var/vcap/bosh/bin/monit start all", plugin.STDOUT)
	if err != nil {
		return err
	}
	return nil
}

func (p RedisPlugin) Store(endpoint plugin.ShieldEndpoint) (string, error) {
	return "", plugin.UNIMPLEMENTED
}

func (p RedisPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p RedisPlugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}
