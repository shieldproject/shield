package redis

import (
	"io"
	"time"

	fmt "github.com/jhunt/go-ansi"

	"github.com/shieldproject/shield/plugin"
)

func New() plugin.Plugin {
	return RedisBrokerPlugin{
		Name:    "Pivotal Redis Broker Backup Plugin",
		Author:  "SHIELD Core Team",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
		Example: `
{
  "redis_type" : "shared"    # Type of Redis Broker backups to run.
                             # Must be either 'shared' or 'dedicated'
}
`,
		Defaults: `
{
  # there are no defaults.
  # all keys are required.
}
`,
		Fields: []plugin.Field{
			plugin.Field{
				Mode: "target",
				Name: "redis_type",
				Type: "enum",
				Enum: []string{
					"shared",
					"dedicated",
				},
				Title:    "Type of Redis Broker",
				Help:     "The CF Redis Broker can run in either `shared` or `dedicated` mode, which affects how it gets backed up.",
				Required: true,
			},
		},
	}

}

func Run() {
	plugin.Run(New())
}

type RedisBrokerPlugin plugin.PluginInfo

type RedisEndpoint struct {
	Mode string
}

func (p RedisBrokerPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p RedisBrokerPlugin) Validate(log io.Writer, endpoint plugin.ShieldEndpoint) error {
	var (
		s    string
		err  error
		fail bool
	)

	s, err = endpoint.StringValue("redis_type")
	if err != nil {
		fmt.Fprintf(log, "@R{\u2717 redis_type  %s}\n", err)
		fail = true
	} else {
		fmt.Fprintf(log, "@G{\u2713 redis_type}  @C{%s}\n", s)
	}

	if fail {
		return fmt.Errorf("postgres: invalid configuration")
	}
	return nil
}

func (p RedisBrokerPlugin) Backup(out io.Writer, log io.Writer, endpoint plugin.ShieldEndpoint) error {
	// Use ExecWithOptions here, to allow tar to exit 1 as warning about
	// files changing/shrinking/disappearing is ok in this specific case
	// redis-check-aof is used in restore to fix corruption on the last
	// command in the AOF, and the file is written to every second. At
	// worst, the restored data appears to have been backed up one second
	// prior to when it actually was

	opts := plugin.ExecOptions{
		Cmd:      "tar -c --warning no-file-changed --warning no-file-shrank --warning no-file-removed -C /var/vcap/store .",
		Stdout:   out,
		ExpectRC: []int{0, 1},
	}
	err := plugin.ExecWithOptions(opts)
	if err != nil {
		return err
	}

	return nil
}

func (p RedisBrokerPlugin) Restore(in io.Reader, log io.Writer, endpoint plugin.ShieldEndpoint) error {
	redis, err := getRedisEndpoint(endpoint)
	if err != nil {
		return err
	}

	var services = []string{"cf-redis-broker"}
	if redis.Mode == "dedicated" {
		services = []string{"redis", "redis-agent"}
	}

	for _, svc := range services {
		err = plugin.Exec(fmt.Sprintf("/var/vcap/bosh/bin/monit stop %s", svc), nil, log, log)
		if err != nil {
			return err
		}
	}

	err = plugin.Exec("bash -c \"while [[ $(/var/vcap/bosh/bin/monit summary | grep redis | grep running) ]]; do sleep 1; done\"", in, log, log)
	if err != nil {
		return err
	}

	// Don't look for errors here, because pkill will return non-zero if there
	// were no processes to kill in the first place.
	plugin.Exec("pkill redis-server", nil, log, log)
	time.Sleep(2 * time.Second)
	plugin.Exec("pkill -9 redis-server", nil, log, log)
	time.Sleep(1 * time.Second)

	err = plugin.Exec("tar -x -C /var/vcap/store . ", in, log, log)
	if err != nil {
		return err
	}

	err = plugin.Exec("bash -c 'yes | find /var/vcap/store -name appendonly.aof -exec /var/vcap/packages/redis/bin/redis-check-aof --fix {} \\;'", nil, log, log)
	if err != nil {
		return err
	}

	for _, svc := range services {
		err = plugin.Exec(fmt.Sprintf("/var/vcap/bosh/bin/monit start %s", svc), nil, log, log)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p RedisBrokerPlugin) Store(in io.Reader, log io.Writer, endpoint plugin.ShieldEndpoint) (string, int64, error) {
	return "", 0, plugin.UNIMPLEMENTED
}

func (p RedisBrokerPlugin) Retrieve(out io.Writer, log io.Writer, endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p RedisBrokerPlugin) Purge(log io.Writer, endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func getRedisEndpoint(endpoint plugin.ShieldEndpoint) (RedisEndpoint, error) {
	mode, err := endpoint.StringValue("redis_type")
	if err != nil {
		return RedisEndpoint{}, err
	}
	return RedisEndpoint{Mode: mode}, nil
}
