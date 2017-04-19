// The `consul` plugin for SHIELD is intended to be a generic
// backup/restore plugin for a consul server.
//
// PLUGIN FEATURES
//
// This plugin implements functionality suitable for use with the following
// SHIELD Job components:
//
//   Target: yes
//   Store:  no
//
// PLUGIN CONFIGURATION
//
// The endpoint configuration passed to this plugin is used to identify
// what consul instance to back up, and how to connect to it. Your
// endpoint JSON should look something like this:
//
//    {
//        "host":"consul-endpoint",          # optional - can also be prefixed with http:// or https://
//        "skip_ssl_validation":false        # optional
//        "username":"basic-auth-username",  # optional
//        "password":"basic-auth-password"   # optional
//    }
//
// Default Configuration
//
//    {
//         "host" : "http://127.0.0.1:8500"
//    }
//
// BACKUP DETAILS
//
// The `consul` plugin makes uses the consul api to back up the entire kv store.
//
// RESTORE DETAILS
//
// The `consul` plugin will also restore the entire kv store.
//
// DEPENDENCIES
//
//
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/hashicorp/consul/api"
	"github.com/starkandwayne/goutils/ansi"
	. "github.com/starkandwayne/shield/plugin"
)

var (
	DefaultHostPort = "http://127.0.0.1:8500"
)

func main() {
	p := ConsulPlugin{
		Name:    "Consul Backup Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
		Example: `
{
  "host"     : "consul-endpoint",      # Optional hostname, with or without the
                                       # http:// or https:// URL scheme.

  "skip_ssl_validation" : false        # Skip certificate verification (not recommended)

  "username" : "basic-auth-username",  # Optional username and password
  "password" : "basic-auth-password"   # for HTTP Basic Authentication.
}
`,
		Defaults: `
{
  "host" : "http://127.0.0.1:8500"
}
`,
	}

	Run(p)
}

type ConsulPlugin PluginInfo

type ConsulConnectionInfo struct {
	Host     string
	Port     string
	User     string
	Password string
	Bin      string
	Database string
}

func (p ConsulPlugin) Meta() PluginInfo {
	return PluginInfo(p)
}

func (p ConsulPlugin) Validate(endpoint ShieldEndpoint) error {
	var (
		s    string
		b    bool
		err  error
		fail bool
	)

	s, err = endpoint.StringValueDefault("host", "")
	if err != nil {
		ansi.Printf("@R{\u2717 host                  %s}\n", err)
		fail = true
	} else if s == "" {
		ansi.Printf("@G{\u2717 host                  using default host @C{%s}}\n", DefaultHostPort)
	} else {
		ansi.Printf("@G{\u2713 host}                  @C{%s}\n", s)
	}

	b, err = endpoint.BooleanValueDefault("skip_ssl_validation", false)
	if err != nil {
		ansi.Printf("@R{\u2717 skip_ssl_validation   %s}\n", err)
		fail = true
	} else {
		ansi.Printf("@G{\u2713 skip_ssl_validation}   @C{%t}\n", b)
	}

	s, err = endpoint.StringValueDefault("username", "")
	if err != nil {
		ansi.Printf("@R{\u2717 username              %s}\n", err)
		fail = true
	} else if s == "" {
		ansi.Printf("@G{\u2713 username}              no username\n")
	} else {
		ansi.Printf("@G{\u2713 username}              @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("password", "")
	if err != nil {
		ansi.Printf("@R{\u2717 password              %s}\n", err)
		fail = true
	} else if s == "" {
		ansi.Printf("@G{\u2713 password}              no password\n")
	} else {
		ansi.Printf("@G{\u2713 password}              @C{%s}\n", s)
	}

	if fail {
		return fmt.Errorf("consul: invalid configuration")
	}
	return nil
}

func (p ConsulPlugin) Backup(endpoint ShieldEndpoint) error {

	encoder := json.NewEncoder(os.Stdout)

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

func (p ConsulPlugin) Restore(endpoint ShieldEndpoint) error {
	client, err := consulClient(endpoint)
	if err != nil {
		return err
	}

	kvClient := client.KV()
	decoder := json.NewDecoder(os.Stdin)

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

func (p ConsulPlugin) Store(endpoint ShieldEndpoint) (string, error) {
	return "", UNIMPLEMENTED
}

func (p ConsulPlugin) Retrieve(endpoint ShieldEndpoint, file string) error {
	return UNIMPLEMENTED
}

func (p ConsulPlugin) Purge(endpoint ShieldEndpoint, file string) error {
	return UNIMPLEMENTED
}

func consulClient(endpoint ShieldEndpoint) (*api.Client, error) {
	skipSSLVerify, err := endpoint.BooleanValueDefault("skip_ssl_validation", false)
	if err != nil {
		return nil, err
	}
	if skipSSLVerify {
		DEBUG("Skipping SSL Validation")
		os.Setenv(api.HTTPSSLVerifyEnvName, "false")
	}

	config := api.DefaultConfig()

	host, err := endpoint.StringValueDefault("host", DefaultHostPort)
	if err != nil {
		return nil, err
	}

	DEBUG("HOST: '%s'", host)
	config.Address = host

	username, err := endpoint.StringValueDefault("username", "")
	if err != nil {
		return nil, err
	}
	DEBUG("USERNAME: '%s'", username)

	password, err := endpoint.StringValueDefault("password", "")
	if err != nil {
		return nil, err
	}
	DEBUG("PASSWORD: '%s'", password)

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
