package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"io"
	"os"
	"strings"
	"time"

	fmt "github.com/jhunt/go-ansi"
	"go.etcd.io/etcd/clientv3"

	"github.com/starkandwayne/shield/plugin"
)

func main() {
	p := EtcdPlugin{
		Name:    "Etcd Backup Plugin",
		Author:  "Jason Zhou, Pururva Lakkad, Naveed Ahmad, Sriniketh Varma Dasarraju",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
		Example: `
			{
			"endpoints"   : "https://192.168.42.45:2379"                                                                        # REQUIRED
        
			"auth"        : ""                                                                                             # is role based or cert based auth enabled on the etcd cluster
			"username"    : "admin",                                                                                           # username for role based authentication
			"password"    : "p@ssw0rd"                                                                                          # password for role based authentication
			"client_cert" : "-----BEGIN CERTIFICATE-----\n(cert contents)\n(... etc ...)\n-----END CERTIFICATE-----"            # path to client certificate
			"client_key"  : "-----BEGIN RSA PRIVATE KEY-----\n(cert contents)\n(... etc ...)\n-----END RSA PRIVATE KEY-----"    # path to client key
			"ca_cert"     : "-----BEGIN CERTIFICATE-----\n(cert contents)\n(... etc ...)\n-----END CERTIFICATE-----"            # path to CA certificate
			"overwrite"   : "false"                                                                                             # enable or disable full overwrite of the cluster
			"prefix"      : "starkandwayne/"                                                                                    # backup specific keys
			}
			`,
		Defaults: `
			{
			}
			`,
		Fields: []plugin.Field{
			plugin.Field{
				Mode:     "target",
				Name:     "url",
				Type:     "string",
				Title:    "Endpoint",
				Help:     "The client URL of the etcd cluster.",
				Example:  "https://192.168.42.45:2379",
				Required: true,
			},
			plugin.Field{
				Mode: "target",
				Name: "auth",
				Type: "enum",
				Enum: []string{
					"Role-Based Authentication",
					"Certificate-Based Authenticaiton",
					"Both",
				},
				Title:   "Authentication",
				Help:    "Type of authentication for accessing the ETCD cluster. No authentication is done if left blank.",
				Example: "Role-Based Authentication",
			},
			plugin.Field{
				Mode:    "target",
				Name:    "username",
				Type:    "string",
				Help:    "Username for role based authentication.",
				Title:   "Username",
				Example: "admin",
			},
			plugin.Field{
				Mode:    "target",
				Name:    "password",
				Type:    "password",
				Help:    "Password for role based authentication.",
				Title:   "Password",
				Example: "p@ssw0rd",
			},
			plugin.Field{
				Mode:    "target",
				Name:    "client_cert",
				Type:    "pem-x509",
				Help:    "Path to the certificate issued by the CA for the client connecting to the ETCD cluster.",
				Title:   "Client Certificate",
				Example: "-----BEGIN CERTIFICATE-----\n(cert contents)\n(... etc ...)\n-----END CERTIFICATE-----",
			},
			plugin.Field{
				Mode:    "target",
				Name:    "client_key",
				Type:    "pem-rsa-pk",
				Help:    "Path to the key issued by the CA for the client connecting to the ETCD cluster.",
				Title:   "Client Private Key",
				Example: "-----BEGIN RSA PRIVATE KEY-----\n(cert contents)\n(... etc ...)\n-----END RSA PRIVATE KEY-----",
			},
			plugin.Field{
				Mode:    "target",
				Name:    "ca_cert",
				Type:    "pem-x509",
				Help:    "Path to the CA certificate that issued the client cert and key.",
				Title:   "CA Certificate",
				Example: "-----BEGIN CERTIFICATE-----\n(cert contents)\n(... etc ...)\n-----END CERTIFICATE-----",
			},
			plugin.Field{
				Mode:    "target",
				Name:    "overwrite",
				Type:    "bool",
				Help:    "If this is enabled, only keys mentioned in the 'Prefix' field will be deleted. The values will be restored using the backup archive.",
				Title:   "Overwrite",
				Example: "false",
			},
			plugin.Field{
				Mode:    "target",
				Name:    "prefix",
				Type:    "string",
				Help:    "This is the input string for prefix based backup-restore. The trailing \"/\" is required.",
				Title:   "Prefix",
				Example: "starkandwayne/",
			},
		},
	}
	fmt.Fprintf(os.Stderr, "etcd plugin starting up...\n")
	plugin.Run(p)
}

type EtcdPlugin plugin.PluginInfo

type EtcdConfig struct {
	EtcdEndpoints  string
	Authentication string
	EtcdClient     *clientv3.Client
	Overwrite      bool
	Prefix         string
}

func (p EtcdPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func getEtcdConfig(endpoint plugin.ShieldEndpoint) (*EtcdConfig, error) {
	etcdEndpoint, err := endpoint.StringValue("url")
	if err != nil {
		return nil, err
	}

	auth, err := endpoint.StringValueDefault("auth", "")
	if err != nil {
		return nil, err
	}

	username, err := endpoint.StringValueDefault("username", "")
	if err != nil {
		return nil, err
	}

	password, err := endpoint.StringValueDefault("password", "")
	if err != nil {
		return nil, err
	}

	clientCert, err := endpoint.StringValueDefault("client_cert", "")
	if err != nil {
		return nil, err
	}

	clientKey, err := endpoint.StringValueDefault("client_key", "")
	if err != nil {
		return nil, err
	}

	caCert, err := endpoint.StringValueDefault("ca_cert", "")
	if err != nil {
		return nil, err
	}

	overwrite, err := endpoint.BooleanValueDefault("overwrite", false)
	if err != nil {
		return nil, err
	}

	prefix, err := endpoint.StringValueDefault("prefix", "")
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	check := caCertPool.AppendCertsFromPEM([]byte(caCert))
	if check != true {
		plugin.DEBUG("CA cert did't parse right.\n")
	}

	cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdEndpoint},
		DialTimeout: 2 * time.Second,
		Username:    username,
		Password:    password,
		TLS:         tlsConfig,
	})
	if err != nil {
		return nil, err
	}

	return &EtcdConfig{
		EtcdEndpoints:  etcdEndpoint,
		Authentication: auth,
		EtcdClient:     cli,
		Overwrite:      overwrite,
		Prefix:         prefix,
	}, nil
}

func (p EtcdPlugin) Validate(endpoint plugin.ShieldEndpoint) error {
	var (
		s    string
		b    bool
		err  error
		fail bool
	)
	s, err = endpoint.StringValue("url")
	if err != nil {
		fmt.Printf("@R{\u2717 url}                   @C{%s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 url}                   data in @C{%s} will be backed up\n", s)
	}

	s, err = endpoint.StringValueDefault("auth", "")
	if err != nil {
		fmt.Printf("@R{\u2717 authentication}        @C{%s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 authentication}        Not using any authentication methods\n")
	} else if s == "Role-Based Authentication" {
		fmt.Printf("@G{\u2713 authentication}        Role-Based authentication chosen\n")
	} else if s == "Certificate-Based Authenticaiton" {
		fmt.Printf("@G{\u2713 authentication}        Certificate-Based authentication chosen\n")
	} else if s == "Both" {
		fmt.Printf("@G{\u2713 authentication}        Both ways of authentication are chosed\n")
		b = true
	}

	if s == "Role-Based Authentication" || b {
		s, err = endpoint.StringValue("username")
		if err != nil {
			fmt.Printf("@R{\u2717 username}              @C{%s}\n", err)
			fail = true
		} else {
			fmt.Printf("@G{\u2713 username}              @C{%s}\n", plugin.Redact(s))
		}

		s, err = endpoint.StringValue("password")
		if err != nil {
			fmt.Printf("@R{\u2717 password}              @C{%s}\n", err)
			fail = true
		} else {
			fmt.Printf("@G{\u2713 password}              @C{%s}\n", plugin.Redact(s))
		}
	}

	if s == "Certificate-Based Authenticaiton" || b {
		s, err = endpoint.StringValue("client_cert")
		if err != nil {
			fmt.Printf("@R{\u2717 client_cert}           @C{%s}\n", err)
		} else {
			/* FIXME: validate that it is an X.509 PEM certificate */
			lines := strings.Split(s, "\n")
			fmt.Printf("@G{\u2713 client_cert}           @C{%s}\n", lines[0])
			if len(lines) > 1 {
				for _, line := range lines[1:] {
					fmt.Printf("                         @C{%s}\n", line)
				}
			}
		}

		s, err = endpoint.StringValue("client_key")
		if err != nil {
			fmt.Printf("@R{\u2717 client_key}            @C{%s}\n", err)
			fail = true
		} else {
			/* FIXME: validate that it is an X.509 PEM certificate */
			lines := strings.Split(s, "\n")
			fmt.Printf("@G{\u2713 ca_key}                @C{%s}\n", plugin.Redact(lines[0]))
			if len(lines) > 1 {
				for _, line := range lines[1:] {
					fmt.Printf("                         @C{%s}\n", plugin.Redact(line))
				}
			}
		}

		s, err = endpoint.StringValue("ca_cert")
		if err != nil {
			fmt.Printf("@R{\u2717 ca_cert}               @C{%s}\n", err)
			fail = true
		} else {
			/* FIXME: validate that it is an X.509 PEM certificate */
			lines := strings.Split(s, "\n")
			fmt.Printf("@G{\u2713 ca_cert}               @C{%s}\n", lines[0])
			if len(lines) > 1 {
				for _, line := range lines[1:] {
					fmt.Printf("                         @C{%s}\n", line)
				}
			}
		}
	}

	b, err = endpoint.BooleanValueDefault("overwrite", false)
	if err != nil {
		fmt.Printf("@R{\u2717 overwrite}             %s\n", err)
		fail = true
	} else if b {
		fmt.Printf("@G{\u2713 overwrite}             @C{%t} - While restoring the existing keys/values in the cluster are removed and then backed up\n", b)
	} else {
		fmt.Printf("@G{\u2713 overwrite}             @C{%t} - Keys/values will be appended to the existing etcd cluster when restored\n", b)
	}

	s, err = endpoint.StringValueDefault("prefix", "")
	if err != nil {
		fmt.Printf("@R{\u2717 prefix}                %s\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 prefix}                Prefix was not provided so everything will be backed-up/restored\n")
	} else {
		fmt.Printf("@G{\u2713 prefix}                @C{%s}\n", plugin.Redact(s))
	}

	if fail {
		return fmt.Errorf("etcd: invalid configuration")
	}

	return nil
}

// Backup ETCD data
func (p EtcdPlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	etcd, err := getEtcdConfig(endpoint)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	defer etcd.EtcdClient.Close()

	resp, err := etcd.EtcdClient.Get(ctx, etcd.Prefix, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortDescend))
	if err != nil {
		return err
	}

	for _, ev := range resp.Kvs {
		fmt.Printf("%s : %s\n", base64.StdEncoding.EncodeToString([]byte(ev.Key)), base64.StdEncoding.EncodeToString([]byte(ev.Value)))
	}

	return nil
}

// Restore ETCD data
func (p EtcdPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	reader := bufio.NewReader(os.Stdin)

	etcd, err := getEtcdConfig(endpoint)
	if err != nil {
		return err
	}

	defer etcd.EtcdClient.Close()

	if etcd.Overwrite {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, err := etcd.EtcdClient.Delete(ctx, etcd.Prefix, clientv3.WithPrefix())
		defer cancel()
		if err != nil {
			return err
		}
	}

	for {
		line, buffer, err := reader.ReadLine()
		if err == io.EOF {
			break
		}

		lineBuffer := []byte{}
		lineBuffer = append(lineBuffer, line...)
		for buffer {
			partLine := []byte{}
			partLine, buffer, err = reader.ReadLine()
			if err == io.EOF {
				break
			}
			lineBuffer = append(lineBuffer, partLine...)
		}

		datakey, err := base64.StdEncoding.DecodeString(strings.Split(string(lineBuffer), " : ")[0])
		if err != nil {
			fmt.Printf("error decoding key: %s", err)
			return err
		}

		dataval, err := base64.StdEncoding.DecodeString(strings.Split(string(lineBuffer), " : ")[1])
		if err != nil {
			fmt.Printf("error decoding value: %s", err)
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, err = etcd.EtcdClient.Put(ctx, fmt.Sprintf("%s", datakey), fmt.Sprintf("%s", dataval))
		defer cancel()
		if err != nil {
			return err
		}
	}
	return nil
}

func (p EtcdPlugin) Store(endpoint plugin.ShieldEndpoint) (string, int64, error) {
	return "", 0, plugin.UNIMPLEMENTED
}

func (p EtcdPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p EtcdPlugin) Purge(endpoint plugin.ShieldEndpoint, key string) error {
	return plugin.UNIMPLEMENTED
}
