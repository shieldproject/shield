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
	"os"
	"strings"

	"github.com/jhunt/go-cli"
	env "github.com/jhunt/go-envirotron"
	"github.com/pborman/uuid"
)

type Opt struct {
	HelpShort bool   `cli:"-h"`
	HelpFull  bool   `cli:"--help"`
	Debug     bool   `cli:"-D, --debug",env:"DEBUG"`
	Version   bool   `cli:"-v, --version"`
	Endpoint  string `cli:"-e,--endpoint"`
	Key       string `cli:"-k, --key"`

	Info     struct{} `cli:"info"`
	Example  struct{} `cli:"example"`
	Validate struct{} `cli:"validate"`
	Backup   struct{} `cli:"backup"`
	Restore  struct{} `cli:"restore"`
	Store    struct{} `cli:"store"`
	Retrieve struct{} `cli:"retrieve"`
	Purge    struct{} `cli:"purge"`
}

type Plugin interface {
	Validate(ShieldEndpoint) error
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

	Example  string `json:"-"`
	Defaults string `json:"-"`
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
		fmt.Fprintf(os.Stderr, "%s\n", content)
	}
}

func Run(p Plugin) {
	var opt Opt
	info := p.Meta()
	env.Override(&opt)
	command, args, err := cli.Parse(&opt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "!!! %s\n", err.Error())
		fmt.Fprintf(os.Stderr, "USAGE: %s [OPTIONS...] COMMAND [OPTIONS...]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Try %s --help for more information.\n", os.Args[0])
		os.Exit(USAGE)
	}
	if opt.Debug {
		debug = true
	}

	if opt.HelpShort {
		fmt.Fprintf(os.Stderr, "%s v%s - %s\n", info.Name, info.Version, info.Author)
		fmt.Fprintf(os.Stderr, "USAGE: %s [OPTIONS...] COMMAND [OPTIONS...]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, `OPTIONS
  -h, --help      Get some help. (--help provides more detail; -h, less)
  -D, --debug     Enable debugging.
  -v, --version   Print the version of this plugin and exit.

COMMANDS
  info                         Print plugin information (name / version / author)
  validate -e JSON             Validate endpoint JSON/configuration
  backup   -e JSON             Backup a target
  restore  -e JSON             Replay a backup archive to a target
  store    -e JSON             Store a backup archive
  retrieve -e JSON -k KEY      Stream a backup archive from storage
  purge    -e JSON -k KEY      Delete a backup archive from storage
`)
		if info.Example != "" {
			fmt.Fprintf(os.Stderr, "\nEXAMPLE ENDPOINT CONFIGURATION\n%s\n", info.Example)
		}
		if info.Defaults != "" {
			fmt.Fprintf(os.Stderr, "\nDEFAULT ENDPOINT\n%s\n", info.Defaults)
		}
		os.Exit(0)
	}

	if opt.HelpFull {
		fmt.Fprintf(os.Stderr, "%s v%s - %s\n", info.Name, info.Version, info.Author)
		fmt.Fprintf(os.Stderr, "USAGE: %s [OPTIONS...] COMMAND [OPTIONS...]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, `OPTIONS
  -h, --help      Get some help. (--help provides more detail; -h, less)
  -D, --debug     Enable debugging.
  -v, --version   Print the version of this plugin and exit.

  -e, --endpoint  JSON string representing what to backup / where to back it up.



GENERAL COMMANDS

  info

    Print information about this plugin, in JSON format, to standard output.


  validate --endpoint ENDPOINT-JSON

    Validates the given ENDPOINT-JSON to ensure that it is (a) well-formed
    JSON data, and (b) is semantically valid for this plugin.  Checks that
    required configuration is set, and verifies the format and suitability
    of the given configuration.



BACKUP COMMANDS

  backup --endpoint TARGET-ENDPOINT-JSON

    Perform a backup of the indicated target endpoint.  The raw (uncompressed)
    backup archive will be written to standard output.

  restore --endpoint TARGET-ENDPOINT-JSON

    Reads a raw (uncompressed) backup archive on standard input and attempts to
    replay it to the given target.


STORAGE COMMANDS

  store --endpoint STORE-ENDPOINT-JSON

    Reads a compressed backup archive on standard input and attempts to
    persist it to the backing storage system indicated by --endpoint.
    Upon success, writes the STORAGE-HANDLE to standard output.

  retrieve --key STORAGE-HANDLE --endpoint STORE-ENDPOINT-JSON

    Retrieves a compressed backup archive from the backing storage,
    using the STORAGE-HANDLE given by a previous 'store' command, and
    writes it to standard output.

  purge --key STORAGE-HANDLE --endpoint STORE-ENDPOINT-JSON

    Removes a backup archive from the backing storage, using the
    STORAGE-HANDLE given by a previous 'store' command.
`)
		os.Exit(0)
	}

	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "extra arguments found, starting at %v\n", args[0])
		fmt.Fprintf(os.Stderr, "USAGE: %s [OPTIONS...] COMMAND [OPTIONS...]\n\n", info.Name)
		os.Exit(USAGE)
	}

	if opt.Version {
		fmt.Printf("%s v%s - %s\n", info.Name, info.Version, info.Author)
		os.Exit(0)
	}

	switch command {
	case "info":
		json, err := json.MarshalIndent(info, "", "    ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(JSON_FAILURE)
		}
		fmt.Printf("%s\n", json)
		os.Exit(0)

	default:
		err = dispatch(p, command, opt)
		DEBUG("'%s' action returned %#v", command, err)
		if err != nil {
			switch err.(type) {
			case UnsupportedActionError:
				if err.(UnsupportedActionError).Action == "" {
					e := err.(UnsupportedActionError)
					e.Action = command
					err = e
				}
			}
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
			os.Exit(codeForError(err))
		}
	}
	os.Exit(0)
}

func dispatch(p Plugin, mode string, opt Opt) error {
	var err error
	var key string
	var endpoint ShieldEndpoint

	DEBUG("'%s' action requested with options %#v", mode, opt)

	switch mode {
	case "validate":
		endpoint, err = getEndpoint(opt.Endpoint)
		if err != nil {
			return err
		}
		err = p.Validate(endpoint)
	case "backup":
		endpoint, err = getEndpoint(opt.Endpoint)
		if err != nil {
			return err
		}
		err = p.Backup(endpoint)
	case "restore":
		endpoint, err = getEndpoint(opt.Endpoint)
		if err != nil {
			return err
		}
		err = p.Restore(endpoint)
	case "store":
		endpoint, err = getEndpoint(opt.Endpoint)
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
		fmt.Printf("%s\n", string(output))
	case "retrieve":
		endpoint, err = getEndpoint(opt.Endpoint)
		if err != nil {
			return err
		}
		if opt.Key == "" {
			return MissingRestoreKeyError{}
		}
		err = p.Retrieve(endpoint, opt.Key)

	case "purge":
		endpoint, err = getEndpoint(opt.Endpoint)
		if err != nil {
			return err
		}
		if opt.Key == "" {
			return MissingRestoreKeyError{}
		}
		err = p.Purge(endpoint, opt.Key)
	default:
		return UnsupportedActionError{Action: mode}
	}

	return err
}

func pluginInfo(p Plugin) error {
	json, err := json.MarshalIndent(p.Meta(), "", "    ")
	if err != nil {
		return JSONError{Err: fmt.Sprintf("Could not create plugin metadata output: %s", err.Error())}
	}
	fmt.Printf("%s\n", json)
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
