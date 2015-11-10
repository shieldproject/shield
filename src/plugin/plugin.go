package plugin

/*

Here are a bunch of frameworky-helper functions for use when creating a new backup/restore plugin. Important things to remember:

Use plugin.Run()  for starting your plugin execution.

Use plugin.PluginInfo to fill out the info for your plugin.
Make your plugin conform to the Plugin interface, by implementing Backup(), Restore(), Retrieve(), and Store(). If they don't make sense, just return plugin.UNSUPPORTED_ACTION, and a helpful errorm essage

plugin.Exec() can be used to easily run external commands sending their stdin/stdout to that of the plugin command. Keep in mind the commands don't get run in a shell, so things like '>', '<', '|' won't work the way you want them to, but you can just run /bin/bash -c <command> to solve that, right?

*/

import (
	"code.google.com/p/go-uuid/uuid"
	"encoding/json"
	"fmt"
	"github.com/voxelbrain/goptions"
	"os"
	"strings"
)

type PluginOpts struct {
	Debug   bool `goptions:"-D, --debug, description='Enable debugging'"`
	Version bool `goptions:"-v, --version, description='Display version information'"`
	Action  goptions.Verbs
	Info    struct {
	} `goptions:"info"`
	Backup struct {
		Endpoint string `goptions:"-e, --endpoint, obligatory, description='JSON string representing backup target'"`
	} `goptions:"backup"`
	Restore struct {
		Endpoint string `goptions:"-e, --endpoint, obligatory, description='JSON string representing backup target'"`
	} `goptions:"restore"`
	Store struct {
		Endpoint string `goptions:"-e, --endpoint, obligatory, description='JSON string representing store endpoint'"`
	} `goptions:"store"`
	Retrieve struct {
		Endpoint string `goptions:"-e, --endpoint, obligatory, description='JSON string representing retrieve endpoint'"`
		Key      string `goptions:"-k, --key, obligatory, description='Key of blob to retrieve from storage'"`
	} `goptions:"retrieve"`
	Purge struct {
		Endpoint string `goptions:"-e, --endpoint, obligatory, description='JSON string representing purge endpoint'"`
		Key      string `goptions:"-k, --key, obligatory, description='Key of blob to purge from storage'"`
	} `goptions:"purge"`
}

type Plugin interface {
	Backup(ShieldEndpoint) (int, error)
	Restore(ShieldEndpoint) (int, error)
	Store(ShieldEndpoint) (string, int, error)
	Retrieve(ShieldEndpoint, string) (int, error)
	Purge(ShieldEndpoint, string) (int, error)
	Meta() PluginInfo
}

type PluginInfo struct {
	Name     string         `json:"name"`
	Author   string         `json:"author"`
	Version  string         `json:"version"`
	Features PluginFeatures `json:"features"`
}

type PluginFeatures struct {
	Target string `json:"target"`
	Store  string `json:"store"`
}

var debug bool

func DEBUG(format string, args ...interface{}) {
	if debug {
		content := fmt.Sprintf(format, args...)
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			lines[i] = "DEBUG> " + line
		}
		content = strings.Join(lines, "\n")
		fmt.Fprintf(stderr, "%s\n", content)
	}
}

var stdout = os.Stdout
var stderr = os.Stderr

var usage = func(err error) {
	fmt.Fprintf(stderr, "%s\n", err.Error())
	goptions.PrintHelp()
}
var exit = func(code int) {
	os.Exit(code)
}

func Run(p Plugin) {
	var code int
	var action string

	opts, err := getPluginOptions()
	if err != nil {
		usage(err)
		code = USAGE
	} else {
		action = string(opts.Action)

		if action == "info" {
			code, err = pluginInfo(p)
			if err != nil {
				fmt.Fprintf(stderr, "%s\n", err.Error())
			}
		} else if action != "" {
			code, err = dispatch(p, action, opts)
			if err != nil {
				fmt.Fprintf(stderr, "%s\n", err.Error())
			}
		} else {
			code = USAGE
			usage(fmt.Errorf("No plugin action was provided"))
		}
	}
	exit(code)
}

func dispatch(p Plugin, mode string, opts PluginOpts) (int, error) {
	var code int
	var err error
	var key string
	var endpoint ShieldEndpoint

	DEBUG("'%s' action requested with options %#v", mode, opts)

	switch mode {
	case "backup":
		endpoint, err = getEndpoint(opts.Backup.Endpoint)
		if err != nil {
			return JSON_FAILURE, fmt.Errorf("Error trying parse --endpoint value as JSON: %s", err.Error())
		}
		code, err = p.Backup(endpoint)
	case "restore":
		endpoint, err = getEndpoint(opts.Restore.Endpoint)
		if err != nil {
			return JSON_FAILURE, fmt.Errorf("Error trying parse --endpoint value as JSON: %s", err.Error())
		}
		code, err = p.Restore(endpoint)
	case "store":
		endpoint, err = getEndpoint(opts.Store.Endpoint)
		if err != nil {
			return JSON_FAILURE, fmt.Errorf("Error trying parse --endpoint value as JSON: %s", err.Error())
		}
		key, code, err = p.Store(endpoint)
		var output []byte
		output, err = json.MarshalIndent(struct {
			Key string `json:"key"`
		}{Key: key}, "", "    ")
		if err != nil {
			return JSON_FAILURE, err
		}
		fmt.Fprintf(stdout, "%s\n", string(output))
	case "retrieve":
		endpoint, err = getEndpoint(opts.Retrieve.Endpoint)
		if err != nil {
			return JSON_FAILURE, fmt.Errorf("Error trying parse --endpoint value as JSON: %s", err.Error())
		}
		if opts.Retrieve.Key == "" {
			return RESTORE_KEY_REQUIRED, fmt.Errorf("retrieving requires --key, but it was not provided")
		}
		code, err = p.Retrieve(endpoint, opts.Retrieve.Key)
	case "purge":
		endpoint, err = getEndpoint(opts.Purge.Endpoint)
		if err != nil {
			return JSON_FAILURE, fmt.Errorf("Error trying parse --endpoint value as JSON: %s", err.Error())
		}
		if opts.Purge.Key == "" {
			return RESTORE_KEY_REQUIRED, fmt.Errorf("purging requires --key, but it was not provided")
		}
		code, err = p.Purge(endpoint, opts.Purge.Key)
	default:
		return UNSUPPORTED_ACTION, fmt.Errorf("Sorry, '%s' is not a supported action for S.H.I.E.L.D plugins", mode)
	}

	DEBUG("'%s' action returned %d", mode, code)
	return code, err
}

func getPluginOptions() (PluginOpts, error) {
	var opts PluginOpts
	err := goptions.Parse(&opts)
	if err != nil {
		return opts, err
	}

	if os.Getenv("DEBUG") != "" && strings.ToLower(os.Getenv("DEBUG")) != "false" && os.Getenv("DEBUG") != "0" {
		debug = true
	}

	if opts.Debug {
		debug = true
	}

	return opts, err
}

func pluginInfo(p Plugin) (int, error) {
	json, err := json.MarshalIndent(p.Meta(), "", "    ")
	if err != nil {
		return JSON_FAILURE, fmt.Errorf("Could not create plugin metadata output: %s", err.Error())
	}
	fmt.Fprintf(stdout, "%s\n", json)
	return SUCCESS, nil
}

func GenUUID() string {
	return uuid.New()
}
