package main

import (
	"fmt"
	"time"

	"github.com/starkandwayne/shield/plugin"
)

func main() {
	p := RedisBrokerPlugin{
		Name:    "Pivotal Redis Broker Backup Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
	}

	plugin.Run(p)
}

type RedisBrokerPlugin plugin.PluginInfo

type RedisEndpoint struct {
	Mode string
}

func (p RedisBrokerPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p RedisBrokerPlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	err := plugin.Exec("tar -c -C /var/vcap/store .", plugin.STDOUT)
	if err != nil {
		return err
	}

	return nil
}

func (p RedisBrokerPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	redis, err := getRedisEndpoint(endpoint)
	if err != nil {
		return err
	}

	var services = []string{"cf-redis-broker"}
	if redis.Mode == "dedicated" {
		services = []string{"redis", "redis-agent"}
	}

	for _, svc := range services {
		err = plugin.Exec(fmt.Sprintf("/var/vcap/bosh/bin/monit stop %s", svc), plugin.STDOUT)
		if err != nil {
			return err
		}
	}

	err = plugin.Exec("bash -c \"while [[ $(/var/vcap/bosh/bin/monit summary | grep redis | grep running) ]]; do sleep 1; done\"", plugin.STDOUT)
	if err != nil {
		return err
	}

	// Don't look for errors here, because pkill will return non-zero if there
	// were no processes to kill in the first place.
	// FIXME: handle this better, so we know we're pkilling properly
	plugin.Exec("pkill redis-server", plugin.STDOUT)
	time.Sleep(2 * time.Second)
	plugin.Exec("pkill -9 redis-server", plugin.STDOUT)
	time.Sleep(1 * time.Second)

	err = plugin.Exec("tar -x -C /var/vcap/store . ", plugin.STDIN)
	if err != nil {
		return err
	}

	err = plugin.Exec("bash -c 'yes | find /var/vcap/store -name appendonly.aof -exec /var/vcap/packages/redis/bin/redis-check-aof --fix {} \\;'", plugin.STDOUT)
	if err != nil {
		return err
	}

	for _, svc := range services {
		err = plugin.Exec(fmt.Sprintf("/var/vcap/bosh/bin/monit start %s", svc), plugin.STDOUT)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p RedisBrokerPlugin) Store(endpoint plugin.ShieldEndpoint) (string, error) {
	return "", plugin.UNIMPLEMENTED
}

func (p RedisBrokerPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p RedisBrokerPlugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func getRedisEndpoint(endpoint plugin.ShieldEndpoint) (RedisEndpoint, error) {
	mode, err := endpoint.StringValue("redis_type")
	if err != nil {
		return RedisEndpoint{}, err
	}
	return RedisEndpoint{Mode: mode}, nil
}
