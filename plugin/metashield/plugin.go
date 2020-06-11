package metashield

import (
	"io"
	"os"

	fmt "github.com/jhunt/go-ansi"

	"github.com/shieldproject/shield/client/v2/shield"
	"github.com/shieldproject/shield/plugin"
)

func New() plugin.Plugin {
	return ShieldPlugin{
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
		},
	}
}

func Run() {
	plugin.Run(New())
}

type ShieldPlugin plugin.PluginInfo

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

	return &shield.Client{
		URL:     url,
		Session: token,
	}, nil
}

// Validate user input
func (p ShieldPlugin) Validate(log io.Writer, endpoint plugin.ShieldEndpoint) error {
	var (
		s    string
		err  error
		fail bool
	)

	s, err = endpoint.StringValue("url")
	if err != nil {
		fmt.Fprintf(log, "@R{\u2717 url}                   @C{%s}\n", err)
		fail = true
	} else {
		fmt.Fprintf(log, "@G{\u2713 url}                   data in @C{%s} core will be backed up\n", s)
	}

	s, err = endpoint.StringValue("token")
	if err != nil {
		fmt.Fprintf(log, "@R{\u2717 token}                 @C{%s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Fprintf(log, "@R{\u2717 token}                 token was not provided\n")
		fail = true
	} else {
		fmt.Fprintf(log, "@G{\u2713 token}                 @C{%s}\n", plugin.Redact(s))
	}

	if fail {
		return fmt.Errorf("metashield: invalid configuration")
	}
	return nil
}

// Backup SHIELD data
func (p ShieldPlugin) Backup(out io.Writer, log io.Writer, endpoint plugin.ShieldEndpoint) error {
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

	io.Copy(out, src)
	fmt.Fprintf(log, "\n")
	return nil
}

// Restore SHIELD data
func (p ShieldPlugin) Restore(in io.Reader, log io.Writer, endpoint plugin.ShieldEndpoint) error {
	c, err := getClient(endpoint)
	if err != nil {
		return err
	}

	taskUUID := os.Getenv("SHIELD_TASK_UUID")
	if taskUUID == "" {
		return fmt.Errorf("SHIELD agent needs to be updated to have SHIELD_TASK_UUID environment variable")
	}
	return c.Import(taskUUID, os.Getenv("SHIELD_RESTORE_KEY"), in)
}

func (p ShieldPlugin) Store(in io.Reader, log io.Writer, endpoint plugin.ShieldEndpoint) (string, int64, error) {
	return "", 0, plugin.UNIMPLEMENTED
}

func (p ShieldPlugin) Retrieve(out io.Writer, log io.Writer, endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p ShieldPlugin) Purge(log io.Writer, endpoint plugin.ShieldEndpoint, key string) error {
	return plugin.UNIMPLEMENTED
}
