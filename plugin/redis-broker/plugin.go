// The `redis-broker` plugin for SHIELD implements backup + restore functionality
// for Piovtal's cf-redis-release (Redis Service + Broker for CloudFoundry). It is
// specific to Pivotal's implementation, which can be found at https://github.com/pivotal-cf/cf-redis-release
//
// It is unlikely that this plugin will work with other Redis instances.
//
// PLUGIN FEATURES
//
// This plugin implements functionality suitable for use with the following SHIELD
// Job components:
//
//    Target: yes
//    Store:  no
//
// PLUGIN CONFIGURATION
//
// The endpoint configuration passed to this plugin is used to identify what type of
// redis VM to back up. Your endpoint JSON should look something like this:
//
//    {
//        "redis_type":<dedicated|broker>"
//    }
//
// BACKUP DETAILS
//
// The `redis-broker` plugin backs up all data in `/var/vcap/store`, to
// grab all of the redis data, whether it is a dedicated-vm, or shared-vm instance.
// Redis data for this BOSH release is stored in the appendonly.aof file, and written
// to every second.
//
// RESTORE DETAILS
//
// Restoration steps for the `redis-broker` plugin depend on the type of redis being backed
// up.
//
// If `redis_type` is set to `broker`, the restoration stops the redis service-broker process,
// kills all instances of `redis-server`, and untars the backup into /var/vcap/store. Once
// complete, it runs `redis-check-aof --fix` against all appendonly.aof files, to resolve any
// potential corruption caused by backups happening mid-write. Lastly, it starts up the
// redis service-broker process, which in turn spawns all of the redis-server's to be running
// on this shared-vm.
//
// If `redis_type_ is set to `dedicated`, the restoration stops the redis-server and redis-agent
// process, untars the backup into /var/vcap/store, runs `redis-check-aof --fix` against any
// appendonly.aof files, to resolve any potential corruption caused by backups happening
// mid-write. Lastly, it starts up the redis-server and redis-agent processes.
//
// Restores with the `redis-broker` plugin are service-impacting, as the redis-servers are shut
// down for the duration of the restore. Additionally, the redis-agent and CloudFoundry service
// broker processes are disabled, to prevent creation of new services of either shared-vm or
// dedicated-vm plans during the restore.
//
// DEPENDENCIES
//
// None.
package main

import (
	"fmt"
	"os"
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
	// Use ExecWithOptions here, to allow tar to exit 1 as warning about
	// files changing/shrinking/disappearing is ok in this specific case
	// redis-check-aof is used in restore to fix corruption on the last
	// command in the AOF, and the file is written to every second. At
	// worst, the restored data appears to have been backed up one second
	// prior to when it actually was

	opts := plugin.ExecOptions{
		Cmd:      "tar -c --warning no-file-changed --warning no-file-shrank --warning no-file-removed -C /var/vcap/store .",
		Stdout:   os.Stdout,
		ExpectRC: []int{0, 1},
	}
	err := plugin.ExecWithOptions(opts)
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
