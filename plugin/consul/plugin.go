package consul

import (
	"encoding/json"
	"io"
	"os"

	"github.com/hashicorp/consul/api"
	fmt "github.com/jhunt/go-ansi"

	"github.com/shieldproject/shield/plugin"
)

var (
	DefaultHostPort = "http://127.0.0.1:8500"
)

func New() plugin.Plugin {
	return ConsulPlugin{
		Name:    "Consul Backup Plugin",
		Author:  "SHIELD Core Team",
		Version: "0.0.1",
		Fields: []plugin.Field{
			plugin.Field{
				Mode:    "target",
				Name:    "host",
				Type:    "string",
				Title:   "Consul Host/Port",
				Help:    "The hostname or IP address port of your consul endpoint.",
				Example: "my.consul.tld:8500",
				Default: "127.0.0.1:8500",
			},
			plugin.Field{
				Mode:  "target",
				Name:  "skip_ssl_validation",
				Type:  "bool",
				Title: "Skip SSL Validation",
				Help:  "If your Consul certificate is invalid, expired, or signed by an unknown Certificate Authority, you can disable SSL validation.  This is not recommended from a security standpoint, however.",
			},
			plugin.Field{
				Mode:  "target",
				Name:  "username",
				Type:  "string",
				Title: "Consul Username",
				Help:  "Username to authenticate to Consul as (usually over HTTP Basic Auth).",
			},
			plugin.Field{
				Mode:  "target",
				Name:  "password",
				Type:  "password",
				Title: "Consul Password",
				Help:  "Password to authenticate to Consul as (usually over HTTP Basic Auth).",
			},
		},
	}
}

func Run() {
	plugin.Run(New())
}

type ConsulPlugin plugin.PluginInfo

func (p ConsulPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p ConsulPlugin) Validate(log io.Writer, endpoint plugin.ShieldEndpoint) error {
	var (
		s    string
		b    bool
		err  error
		fail bool
	)

	s, err = endpoint.StringValueDefault("host", "")
	if err != nil {
		fmt.Fprintf(log, "@R{\u2717 host                  %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Fprintf(log, "@G{\u2717 host                  using default host @C{%s}}\n", DefaultHostPort)
	} else {
		fmt.Fprintf(log, "@G{\u2713 host}                  @C{%s}\n", s)
	}

	b, err = endpoint.BooleanValueDefault("skip_ssl_validation", false)
	if err != nil {
		fmt.Fprintf(log, "@R{\u2717 skip_ssl_validation   %s}\n", err)
		fail = true
	} else {
		fmt.Fprintf(log, "@G{\u2713 skip_ssl_validation}   @C{%t}\n", b)
	}

	s, err = endpoint.StringValueDefault("username", "")
	if err != nil {
		fmt.Fprintf(log, "@R{\u2717 username              %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Fprintf(log, "@G{\u2713 username}              no username\n")
	} else {
		fmt.Fprintf(log, "@G{\u2713 username}              @C{%s}\n", plugin.Redact(s))
	}

	s, err = endpoint.StringValueDefault("password", "")
	if err != nil {
		fmt.Fprintf(log, "@R{\u2717 password              %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Fprintf(log, "@G{\u2713 password}              no password\n")
	} else {
		fmt.Fprintf(log, "@G{\u2713 password}              @C{%s}\n", plugin.Redact(s))
	}

	if fail {
		return fmt.Errorf("consul: invalid configuration")
	}
	return nil
}

func (p ConsulPlugin) Backup(out io.Writer, log io.Writer, endpoint plugin.ShieldEndpoint) error {

	encoder := json.NewEncoder(out)

	client, err := consulClient(endpoint)
	if err != nil {
		return err
	}

	kv := client.KV()

	kvs, _, err := kv.List("/", nil)
	if err != nil {
		return err
	}

	for _, kv := range kvs {
		encoder.Encode(kv)
	}

	return err
}

func (p ConsulPlugin) Restore(in io.Reader, log io.Writer, endpoint plugin.ShieldEndpoint) error {
	client, err := consulClient(endpoint)
	if err != nil {
		return err
	}

	kvClient := client.KV()
	decoder := json.NewDecoder(in)

	var kvs []api.KVPair
	var kv api.KVPair

	for {
		if err := decoder.Decode(&kv); err == io.EOF {
			break
		} else if err != nil {
			return err
		} else {
			kvs = append(kvs, kv)
		}
	}
	for _, kv := range kvs {
		_, err := kvClient.Put(&kv, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func consulClient(endpoint plugin.ShieldEndpoint) (*api.Client, error) {
	skipSSLVerify, err := endpoint.BooleanValueDefault("skip_ssl_validation", false)
	if err != nil {
		return nil, err
	}
	if skipSSLVerify {
		plugin.DEBUG("Skipping SSL Validation")
		os.Setenv(api.HTTPSSLVerifyEnvName, "false")
	}

	config := api.DefaultConfig()

	host, err := endpoint.StringValueDefault("host", DefaultHostPort)
	if err != nil {
		return nil, err
	}

	plugin.DEBUG("HOST: '%s'", host)
	config.Address = host

	username, err := endpoint.StringValueDefault("username", "")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("USERNAME: '%s'", username)

	password, err := endpoint.StringValueDefault("password", "")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("PASSWORD: '%s'", password)

	if username != "" && password != "" {
		config.HttpAuth = &api.HttpBasicAuth{
			Username: username,
			Password: password,
		}
	}

	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}
