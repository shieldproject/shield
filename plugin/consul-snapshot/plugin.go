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
//        "address":"consul.service.consul:8500",                    # optional - can also be prefixed with http:// or https://
//        "ca-path":"/var/vcap/jobs/consul/consul/ca.cert",          # optional - required for connecting via https
//        "client-cert":"/var/vcap/jobs/consul/consul/consul.cert",  # optional - required when verify_incoming is set to true
//        "client-key":"/var/vcap/jobs/consul/consul/consul.key"     # optional - required when verify_incoming is set to true
//    }
//
// Default Configuration
//
//    {
//         "address" : "http://127.0.0.1:8500"
//         "consul" : "/var/vcap/packages/consul/bin/consul"
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
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/plugin"
)

var (
	DefaultAddress = "http://127.0.0.1:8500"
	DefaultConsul  = "/var/vcap/packages/consul/bin/consul"
)

func main() {
	p := ConsulPlugin{
		Name:    "Consul Backup Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
		Example: `
{
  "consul"      : "/var/vcap/packages/consul/bin/consul"     # Path to the consul binary
  "address"     : "consul.service.consul:8200",              # Optional hostname, with or without the http:// or https:// URL scheme.

  "ca-path"     : "/var/vcap/jobs/consul/consul/ca.cert"     # Path to the CA certificate to use for TLS when communicating with Consul via https.
  "client-cert" : "/var/vcap/jobs/consul/consul/consul.cert" # Path to a client cert file to use for TLS when 'verify_incoming' is enabled.
  "client-key"  : "/var/vcap/jobs/consul/consul/consul.key"  # Path to a client key file to use for TLS when 'verify_incoming' is enabled.

}
`,
		Defaults: `
{
  "consul"  : "/var/vcap/packages/consul/bin/consul"
  "address" : "http://127.0.0.1:8500"
}
`,
	}

	plugin.Run(p)
}

type ConsulPlugin plugin.PluginInfo

type ConsulConfig struct {
	Consul     string
	Address    string
	CaPath     string
	ClientCert string
	ClientKey  string
}

func (p ConsulPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func getConsulConfig(endpoint plugin.ShieldEndpoint) (*ConsulConfig, error) {
	consul, err := endpoint.StringValueDefault("consul", DefaultConsul)
	if err != nil {
		return nil, err
	}

	address, err := endpoint.StringValueDefault("address", DefaultAddress)
	if err != nil {
		return nil, err
	}

	ca_path, err := endpoint.StringValueDefault("ca-path", "")
	if err != nil {
		return nil, err
	}

	client_cert, err := endpoint.StringValueDefault("client-cert", "")
	if err != nil {
		return nil, err
	}

	client_key, err := endpoint.StringValueDefault("client-key", "")
	if err != nil {
		return nil, err
	}

	return &ConsulConfig{
		Consul:     consul,
		Address:    address,
		CaPath:     ca_path,
		ClientCert: client_cert,
		ClientKey:  client_key,
	}, nil
}

func (p ConsulPlugin) Validate(endpoint plugin.ShieldEndpoint) error {
	var (
		s    string
		err  error
		fail bool
	)

	s, err = endpoint.StringValueDefault("consul", "")
	if err != nil {
		ansi.Printf("@R{\u2717 consul        %s}\n", err)
		fail = true
	} else if s == "" {
		ansi.Printf("@G{\u2713 consul}       using default consul @C{%s}\n", DefaultConsul)
	} else {
		ansi.Printf("@G{\u2713 consul}       @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("address", "")
	if err != nil {
		ansi.Printf("@R{\u2717 address       %s}\n", err)
		fail = true
	} else if s == "" {
		ansi.Printf("@G{\u2713 address}      using default address @C{%s}\n", DefaultAddress)
	} else {
		ansi.Printf("@G{\u2713 address}      @C{%s}\n", s)
	}

	addr := s
	s, err = endpoint.StringValueDefault("ca-path", "")
	if err != nil {
		ansi.Printf("@R{\u2717 ca-path       %s}\n", err)
		fail = true
	} else if s == "" && strings.HasPrefix(addr, "https") {
		ansi.Printf("@G{\u2717 ca-path       ca-path must be specified when using https}\n")
		fail = true
	} else {
		ansi.Printf("@G{\u2713 ca-path}      @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("client-cert", "")
	if err != nil {
		ansi.Printf("@R{\u2717 client-cert   %s}\n", err)
		fail = true
	} else {
		ansi.Printf("@G{\u2713 client-cert}  @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("client-key", "")
	if err != nil {
		ansi.Printf("@R{\u2717 client-key    %s}\n", err)
		fail = true
	} else {
		ansi.Printf("@G{\u2713 client-key}   @C{%s}\n", s)
	}

	if fail {
		return fmt.Errorf("consul: invalid configuration")
	}
	return nil
}

func (p ConsulPlugin) Backup(endpoint plugin.ShieldEndpoint) error {

	cfg, err := getConsulConfig(endpoint)
	if err != nil {
		return err
	}

	var flags string
	if cfg.CaPath != "" {
		flags = fmt.Sprintf("%s -ca-path='%s'", flags, cfg.CaPath)
	}
	if cfg.ClientCert != "" {
		flags = fmt.Sprintf("%s -client-cert='%s'", flags, cfg.ClientCert)
	}
	if cfg.ClientKey != "" {
		flags = fmt.Sprintf("%s -client-key='%s'", flags, cfg.ClientKey)
	}

	tmp_dir, err := ioutil.TempDir("", "consul")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp_dir)
	backup_file := fmt.Sprintf("%s/consul.back", tmp_dir)

	cmd := fmt.Sprintf("%s snapshot save -http-addr='%s' %s %s", cfg.Consul, cfg.Address, flags, backup_file)
	plugin.DEBUG("Executing `%s`", cmd)
	err = plugin.Exec(cmd, plugin.NOPIPE)
	if err != nil {
		return err
	}

	cmd = fmt.Sprintf("cat %s", backup_file)
	plugin.DEBUG("Executing `%s`", cmd)
	err = plugin.Exec(cmd, plugin.STDOUT)
	if err != nil {
		return err
	}

	return nil
}

func (p ConsulPlugin) Restore(endpoint plugin.ShieldEndpoint) error {

	cfg, err := getConsulConfig(endpoint)
	if err != nil {
		return err
	}

	var flags string
	if cfg.CaPath != "" {
		flags = fmt.Sprintf("%s -ca-path='%s'", flags, cfg.CaPath)
	}
	if cfg.ClientCert != "" {
		flags = fmt.Sprintf("%s -client-cert='%s'", flags, cfg.ClientCert)
	}
	if cfg.ClientKey != "" {
		flags = fmt.Sprintf("%s -client-key='%s'", flags, cfg.ClientKey)
	}

	tmp_dir, err := ioutil.TempDir("", "consul")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp_dir)
	backup_file := fmt.Sprintf("%s/consul.back", tmp_dir)

	cmd := fmt.Sprintf("tee %s", backup_file)
	plugin.DEBUG("Executing `%s`", cmd)
	err = plugin.Exec(cmd, plugin.STDIN)
	if err != nil {
		return err
	}

	cmd = fmt.Sprintf("%s snapshot restore -http-addr='%s' %s %s", cfg.Consul, cfg.Address, flags, backup_file)
	plugin.DEBUG("Executing `%s`", cmd)
	err = plugin.Exec(cmd, plugin.NOPIPE)
	if err != nil {
		return err
	}

	return nil
}

func (p ConsulPlugin) Store(endpoint plugin.ShieldEndpoint) (string, error) {
	return "", plugin.UNIMPLEMENTED
}

func (p ConsulPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p ConsulPlugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}
