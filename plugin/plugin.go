package plugin

/*

Here are a bunch of frameworky-helper functions for use when creating a new backup/restore plugin. Important things to remember:

Use plugin.Run()  for starting your plugin execution.

Use plugin.PluginInfo to fill out the info for your plugin.
Make your plugin conform to the Plugin interface, by implementing Backup(), Restore(), Retrieve(), and Store(). If they don't make sense, just return plugin.UNSUPPORTED_ACTION, and a helpful errorm essage

plugin.Exec() can be used to easily run external commands sending their stdin/stdout to that of the plugin command. Keep in mind the commands don't get run in a shell, so things like '>', '<', '|' won't work the way you want them to, but you can just run /bin/bash -c <command> to solve that, right?

*/

import (
	"encoding/json"
	"fmt"
	"github.com/pborman/uuid"
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
	Backup(ShieldEndpoint) error
	Restore(ShieldEndpoint) error
	Store(ShieldEndpoint) (string, error)
	Retrieve(ShieldEndpoint, string) error
	Purge(ShieldEndpoint, string) error
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
			err = pluginInfo(p)
			if err != nil {
				fmt.Fprintf(stderr, "%s\n", err.Error())
				code = codeForError(err)
			}
		} else if action != "" {
			err = dispatch(p, action, opts)
			DEBUG("'%s' action returned %#v", action, err)
			if err != nil {
				switch err.(type) {
				case UnsupportedActionError:
					if err.(UnsupportedActionError).Action == "" {
						e := err.(UnsupportedActionError)
						e.Action = action
						err = e
					}
				}
				fmt.Fprintf(stderr, "%s\n", err.Error())
				code = codeForError(err)
			}
		} else {
			code = USAGE
			usage(fmt.Errorf("No plugin action was provided"))
		}
	}
	exit(code)
}

func dispatch(p Plugin, mode string, opts PluginOpts) error {
	var err error
	var key string
	var endpoint ShieldEndpoint

	DEBUG("'%s' action requested with options %#v", mode, opts)

	switch mode {
	case "backup":
		endpoint, err = getEndpoint(opts.Backup.Endpoint)
		if err != nil {
			return err
		}
		err = p.Backup(endpoint)
	case "restore":
		endpoint, err = getEndpoint(opts.Restore.Endpoint)
		if err != nil {
			return err
		}
		err = p.Restore(endpoint)
	case "store":
		endpoint, err = getEndpoint(opts.Store.Endpoint)
		if err != nil {
			return err
		}
		key, err = p.Store(endpoint)
		output, jsonErr := json.MarshalIndent(struct {
			Key string `json:"key"`
		}{Key: key}, "", "    ")
		if jsonErr != nil {
			return JSONError{Err: fmt.Sprintf("Could not JSON encode blob key: %s", jsonErr.Error())}
		}
		fmt.Fprintf(stdout, "%s\n", string(output))
	case "retrieve":
		endpoint, err = getEndpoint(opts.Retrieve.Endpoint)
		if err != nil {
			return err
		}
		if opts.Retrieve.Key == "" {
			return MissingRestoreKeyError{}
		}
		err = p.Retrieve(endpoint, opts.Retrieve.Key)
	case "purge":
		endpoint, err = getEndpoint(opts.Purge.Endpoint)
		if err != nil {
			return err
		}
		if opts.Purge.Key == "" {
			return MissingRestoreKeyError{}
		}
		err = p.Purge(endpoint, opts.Purge.Key)
	default:
		return UnsupportedActionError{Action: mode}
	}

	return err
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

func pluginInfo(p Plugin) error {
	json, err := json.MarshalIndent(p.Meta(), "", "    ")
	if err != nil {
		return JSONError{Err: fmt.Sprintf("Could not create plugin metadata output: %s", err.Error())}
	}
	fmt.Fprintf(stdout, "%s\n", json)
	return nil
}

func GenUUID() string {
	return uuid.New()
}

func codeForError(e error) int {
	var code int
	if e != nil {
		switch e.(type) {
		case UnsupportedActionError:
			code = UNSUPPORTED_ACTION
		case EndpointMissingRequiredDataError:
			code = ENDPOINT_MISSING_KEY
		case EndpointDataTypeMismatchError:
			code = ENDPOINT_BAD_DATA
		case ExecFailure:
			code = EXEC_FAILURE
		case JSONError:
			code = JSON_FAILURE
		case MissingRestoreKeyError:
			code = RESTORE_KEY_REQUIRED
		default:
			code = PLUGIN_FAILURE
		}
	} else {
		code = SUCCESS
	}

	return code
}
