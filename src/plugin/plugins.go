package plugin

import (
	"encoding/json"
	"fmt"
	"github.com/voxelbrain/goptions"
	"os"
	"strings"
)

type PluginOpts struct {
	Debug    bool `goptions:"-D, --debug, description='Enable debugging'"`
	Version  bool `goptions:"-v, --version, description='Display version information'"`
	Action   goptions.Verbs
	Info     struct{} `goptions:"info"`
	Backup   struct{} `goptions:"backup"`
	Restore  struct{} `goptions:"restore"`
	Store    struct{} `goptions:"store"`
	Retrieve struct{} `goptions:"retrieve"`
}

type Plugin interface {
	Backup(ShieldEndpoint) (int, error)
	Restore(ShieldEndpoint) (int, error)
	Store(ShieldEndpoint) (int, error)
	Retrieve(ShieldEndpoint) (int, error)
	Meta() PluginInfo
}

type PluginInfo struct {
	Name     string
	Author   string
	Version  string
	Features PluginFeatures
}

type PluginFeatures struct {
	Target bool
	Store  bool
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
	} else {
		var envVar string
		if action == "backup" || action == "restore" {
			envVar = "SHIELD_TARGET_ENDPOINT"
		} else if action == "store" || action == "retrieve" {
			envVar = "SHIELD_STORE_ENDPOINT"
		}

		code, err = dispatch(p, action, envVar)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
	}
	os.Exit(code)
}

func pluginInfo(p Plugin) (int, error) {
	json, err := json.Marshal(p.Meta())
	if err != nil {
		return JSON_FAILURE, fmt.Errorf("Could not create plugin metadata output: %s", err.Error())
	}
	fmt.Printf("%s\n", json)
	return 0, nil
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

type ShieldEndpoint map[string]interface{}

func getEndpoint(env string) (ShieldEndpoint, error) {
	t := os.Getenv(env)
	if t == "" {
		return nil, fmt.Errorf("No %s variable was set", env)
	}

	endpoint := make(ShieldEndpoint)
	err := json.Unmarshal([]byte(t), &endpoint)
	if err != nil {
		return nil, err
	}

	return endpoint, nil
}

func dispatch(p Plugin, mode string, envVar string) (int, error) {
	var code int
	var err error

	endpoint, err := getEndpoint(envVar)
	if err != nil {
		return ENDPOINT_REQUIRED, fmt.Errorf("Error trying to %s: %s", mode, err.Error())
	}
	DEBUG("'%s' action requested agains endpoint %#v", mode, endpoint)

	switch mode {
	case "backup":
		code, err = p.Backup(endpoint)
	case "restore":
		code, err = p.Restore(endpoint)
	case "store":
		code, err = p.Store(endpoint)
	case "retrieve":
		code, err = p.Retrieve(endpoint)
	default:
		return UNSUPPORTED_ACTION, fmt.Errorf("Sorry, '%s' is not a supported action for S.H.I.E.L.D plugins", mode)
	}

	DEBUG("'%s' returned %d", mode, code)
	return code, err
}
