package main

import (
	"io"
	"os"
	"strings"

	fmt "github.com/jhunt/go-ansi"

	"github.com/shieldproject/shield/client/v2/shield"
	"github.com/shieldproject/shield/plugin"
)

func main() {
	p := ShieldPlugin{
		Name:    "SHIELD Backup Plugin",
		Author:  "Stark and Wayne",
		Version: "1.0.0",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
		Fields: []plugin.Field{
			plugin.Field{
				Mode:     "target",
				Name:     "url",
				Type:     "string",
				Title:    "SHIELD Core",
				Help:     "The SHIELD core URL.",
				Example:  "http://192.168.43.32",
				Required: true,
			},
			plugin.Field{
				Mode:     "target",
				Name:     "token",
				Type:     "password",
				Title:    "Auth Token",
				Help:     "Token for authentication against the SHIELD core.",
				Example:  "daa9a25d-8f52-4b9a-b9c8-2730e0e4a9eb",
				Required: true,
			},
			plugin.Field{
				Mode:    "target",
				Name:    "core_ca_cert",
				Type:    "pem-x509",
				Help:    "CA certificate in pem format that is associated with your shield core.",
				Title:   "CA Certificate",
				Example: "-----BEGIN CERTIFICATE-----\n(cert contents)\n(... etc ...)\n-----END CERTIFICATE-----",
				Default: "",
			},
			plugin.Field{
				Mode:    "target",
				Name:    "skip_ssl_validation",
				Type:    "bool",
				Title:   "Skip SSL Validation",
				Help:    "If your SHIELD certificate is invalid, expired, or signed by an unknown Certificate Authority, you can disable SSL validation.  This is not recommended from a security standpoint, however.",
				Default: "false",
			},
		},
	}
	fmt.Fprintf(os.Stderr, "SHIELD plugin starting up...\n")
	plugin.Run(p)
}

type ShieldPlugin plugin.PluginInfo

type Client struct {
	url   string
	token string
}

func (p ShieldPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func getClient(endpoint plugin.ShieldEndpoint) (*shield.Client, error) {
	url, err := endpoint.StringValue("url")
	if err != nil {
		return nil, err
	}

	token, err := endpoint.StringValue("token")
	if err != nil {
		return nil, err
	}

	ca, err := endpoint.StringValue("core_ca_cert")
	if err != nil {
		return nil, err
	}

	ssl, err := endpoint.BooleanValueDefault("skip_ssl_validation", false)
	if err != nil {
		return nil, err
	}

	return &shield.Client{
		URL:                url,
		Session:            token,
		CACertificate:      ca,
		InsecureSkipVerify: ssl,
	}, nil
}

// Validate user input
func (p ShieldPlugin) Validate(endpoint plugin.ShieldEndpoint) error {
	var (
		s    string
		ssl  bool
		err  error
		fail bool
	)

	s, err = endpoint.StringValue("url")
	if err != nil {
		fmt.Printf("@R{\u2717 url}                   @C{%s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 url}                   data in @C{%s} core will be backed up\n", s)
	}

	s, err = endpoint.StringValue("token")
	if err != nil {
		fmt.Printf("@R{\u2717 token}                 @C{%s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@R{\u2717 token}                 token was not provided\n")
		fail = true
	} else {
		fmt.Printf("@G{\u2713 token}                 @C{%s}\n", plugin.Redact(s))
	}

	s, err = endpoint.StringValueDefault("core_ca_cert", "")
	if err != nil {
		fmt.Printf("@R{\u2717 core_ca_cert}               @C{%s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 core_ca_cert}               CA cert was not provided.\n")
	} else {
		lines := strings.Split(s, "\n")
		fmt.Printf("@G{\u2713 core_ca_cert}               @C{%s}\n", lines[0])
		if len(lines) > 1 {
			for _, line := range lines[1:] {
				fmt.Printf("                         @C{%s}\n", line)
			}
		}
	}

	ssl, err = endpoint.BooleanValueDefault("skip_ssl_validation", false)
	if err != nil {
		fmt.Printf("@R{\u2717 skip_ssl_validation  %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 skip_ssl_validation}  @C{%t}\n", ssl)
	}

	if fail {
		return fmt.Errorf("metashield: invalid configuration")
	}
	return nil
}

// Backup SHIELD data
func (p ShieldPlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	c, err := getClient(endpoint)
	if err != nil {
		return err
	}

	taskUUID := os.Getenv("SHIELD_TASK_UUID")
	if taskUUID == "" {
		return fmt.Errorf("SHIELD agent needs to be updated to have SHIELD_TASK_UUID environment variable")
	}

	src, err := c.Export(taskUUID)
	if err != nil {
		return err
	}

	io.Copy(os.Stdout, src)
	fmt.Printf("\n")
	return nil
}

// Restore SHIELD data
func (p ShieldPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	c, err := getClient(endpoint)
	if err != nil {
		return err
	}

	taskUUID := os.Getenv("SHIELD_TASK_UUID")
	if taskUUID == "" {
		return fmt.Errorf("SHIELD agent needs to be updated to have SHIELD_TASK_UUID environment variable")
	}
	return c.Import(taskUUID, os.Getenv("SHIELD_RESTORE_KEY"), os.Stdin)
}

func (p ShieldPlugin) Store(endpoint plugin.ShieldEndpoint) (string, int64, error) {
	return "", 0, plugin.UNIMPLEMENTED
}

func (p ShieldPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p ShieldPlugin) Purge(endpoint plugin.ShieldEndpoint, key string) error {
	return plugin.UNIMPLEMENTED
}
