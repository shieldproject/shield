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
	"github.com/mattn/go-shellwords"
	"github.com/voxelbrain/goptions"
	"os"
	"os/exec"
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
	Name     string
	Author   string
	Version  string
	Features PluginFeatures
}

type PluginFeatures struct {
	Target string
	Store  string
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
		fmt.Fprintf(os.Stderr, "%s\n", content)
	}
}

func Run(p Plugin) {
	opts := getPluginOptions()
	action := string(opts.Action)

	var code int
	var err error

	if action == "info" {
		code, err = pluginInfo(p)
	} else if action != "" {
		code, err = dispatch(p, action, opts)
	} else {
		goptions.PrintHelp()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
	}
	os.Exit(code)
}

const STDIN = 1
const STDOUT = 2

func Exec(flags int, cmdString string) (int, error) {
	cmdArgs, err := shellwords.Parse(cmdString)
	if err != nil {
		return EXEC_FAILURE, fmt.Errorf("Could not parse '%s' into exec-able command: %s", cmdString, err.Error)
	}
	DEBUG("Executing '%s' with arguments %v", cmdArgs[0], cmdArgs[1:])

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	if flags&STDOUT == STDOUT {
		cmd.Stdout = os.Stdout
	}
	if flags&STDIN == STDIN {
		cmd.Stdin = os.Stdin
	}
	err = cmd.Run()
	if err != nil {
		return EXEC_FAILURE, fmt.Errorf("Unable to exec '%s': %s", cmdArgs[0], err.Error())
	}
	return SUCCESS, nil
}

func pluginInfo(p Plugin) (int, error) {
	json, err := json.Marshal(p.Meta())
	if err != nil {
		return JSON_FAILURE, fmt.Errorf("Could not create plugin metadata output: %s", err.Error())
	}
	fmt.Printf("%s\n", json)
	return SUCCESS, nil
}

func getPluginOptions() PluginOpts {
	var opts PluginOpts
	err := goptions.Parse(&opts)
	if err != nil {
		goptions.PrintHelp()
		os.Exit(USAGE)
	}

	if os.Getenv("DEBUG") != "" && strings.ToLower(os.Getenv("DEBUG")) != "false" && os.Getenv("DEBUG") != "0" {
		debug = true
	}

	if opts.Debug {
		debug = true
	}

	return opts
}

func dispatch(p Plugin, mode string, opts PluginOpts) (int, error) {
	var code int
	var err error
	var key string

	DEBUG("'%s' action requested with options %#v", mode, opts)

	switch mode {
	case "backup":
		endpoint, err := getEndpoint(opts.Backup.Endpoint)
		if err != nil {
			return ENDPOINT_REQUIRED, fmt.Errorf("Error trying parse --endpoint value as JSON: %s", err.Error())
		}
		code, err = p.Backup(endpoint)
	case "restore":
		endpoint, err := getEndpoint(opts.Restore.Endpoint)
		if err != nil {
			return ENDPOINT_REQUIRED, fmt.Errorf("Error trying parse --endpoint value: %s", err.Error())
		}
		code, err = p.Restore(endpoint)
	case "store":
		endpoint, err := getEndpoint(opts.Store.Endpoint)
		if err != nil {
			return ENDPOINT_REQUIRED, fmt.Errorf("Error trying parse --endpoint value: %s", err.Error())
		}
		key, code, err = p.Store(endpoint)
		output, err := json.Marshal(struct{ key string }{key: key})
		if err != nil {
			return JSON_FAILURE, err
		}
		fmt.Printf("%s\n", string(output))
	case "retrieve":
		endpoint, err := getEndpoint(opts.Retrieve.Endpoint)
		if err != nil {
			return ENDPOINT_REQUIRED, fmt.Errorf("Error trying parse --endpoint value: %s", err.Error())
		}
		if opts.Retrieve.Key == "" {
			return RESTORE_KEY_REQUIRED, fmt.Errorf("retrieving requires --key, but it was not provided")
		}
		code, err = p.Retrieve(endpoint, opts.Retrieve.Key)
	case "purge":
		endpoint, err := getEndpoint(opts.Purge.Endpoint)
		if err != nil {
			return ENDPOINT_REQUIRED, fmt.Errorf("Error trying parse --endpoint value: %s", err.Error())
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

func GenUUID() string {
	return uuid.New()
}
