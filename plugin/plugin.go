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

	"github.com/pborman/uuid"
	"github.com/spf13/cobra"
)

type Opt struct {
	Endpoint string
	Key      string
	Text     bool
}

type Plugin interface {
	Validate(ShieldEndpoint) error
	Backup(ShieldEndpoint) error
	Restore(ShieldEndpoint) error
	Store(ShieldEndpoint) (string, int64, error)
	Retrieve(ShieldEndpoint, string) error
	Purge(ShieldEndpoint, string) error
	Meta() PluginInfo
}

type Field struct {
	Mode     string   `json:"mode"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Title    string   `json:"title,omitempty"`
	Help     string   `json:"help,omitempty"`
	Example  string   `json:"example,omitempty"`
	Default  string   `json:"default,omitempty"`
	Enum     []string `json:"enum,omitempty"`
	Required bool     `json:"required,omitempty"`
}

type PluginInfo struct {
	Name     string         `json:"name"`
	Author   string         `json:"author"`
	Version  string         `json:"version"`
	Features PluginFeatures `json:"features"`

	Example  string `json:"-"`
	Defaults string `json:"-"`

	Fields []Field `json:"fields"`
}

type PluginFeatures struct {
	Target string `json:"target"`
	Store  string `json:"store"`
}

var debug bool

func DEBUG(format string, args ...interface{}) {
	if debug {
		for _, line := range strings.Split(fmt.Sprintf(format, args...), "\n") {
			fmt.Fprintf(os.Stderr, "DEBUG> %s\n", line)
		}
	}
}

func Debugf(f string, args ...interface{}) {
	if debug {
		for _, line := range strings.Split(fmt.Sprintf(f, args...), "\n") {
			fmt.Fprintf(os.Stderr, "DEBUG> %s\n", line)
		}
	}
}
func Infof(f string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, f+"\n", args...)
}

func Run(p Plugin) {
	info := p.Meta()

	var endpoint string
	var key string
	var textFlag bool
	var debugFlag bool

	makeOpt := func() Opt {
		return Opt{Endpoint: endpoint, Key: key, Text: textFlag}
	}

	longHelp := fmt.Sprintf(`%s v%s - %s

OPTIONS
  -h, --help      Get some help.
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

  store --endpoint STORE-ENDPOINT-JSON [--text]

    Reads a compressed backup archive on standard input and attempts to
    persist it to the backing storage system indicated by --endpoint.
    Upon success, writes the STORAGE-HANDLE to standard output.

    If --text is given, the STORAGE-HANDLE is printed on a single line,
    without any additional whitespace or formatting.  By default, it will
    be printed inside of a JSON structure.

  retrieve --key STORAGE-HANDLE --endpoint STORE-ENDPOINT-JSON

    Retrieves a compressed backup archive from the backing storage,
    using the STORAGE-HANDLE given by a previous 'store' command, and
    writes it to standard output.

  purge --key STORAGE-HANDLE --endpoint STORE-ENDPOINT-JSON

    Removes a backup archive from the backing storage, using the
    STORAGE-HANDLE given by a previous 'store' command.`,
		info.Name, info.Version, info.Author)

	if info.Example != "" {
		longHelp += fmt.Sprintf("\n\nEXAMPLE ENDPOINT CONFIGURATION\n%s", info.Example)
	}
	if info.Defaults != "" {
		longHelp += fmt.Sprintf("\n\nDEFAULT ENDPOINT\n%s", info.Defaults)
	}

	rootCmd := &cobra.Command{
		Use:           info.Name,
		Short:         fmt.Sprintf("%s v%s - %s", info.Name, info.Version, info.Author),
		Version:       fmt.Sprintf("%s v%s - %s", info.Name, info.Version, info.Author),
		Long:          longHelp,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if debugFlag || os.Getenv("DEBUG") != "" {
				debug = true
			}
		},
	}

	rootCmd.PersistentFlags().BoolVarP(&debugFlag, "debug", "D", false, "Enable debug output")
	rootCmd.PersistentFlags().StringVarP(&endpoint, "endpoint", "e", "", "Endpoint JSON")
	rootCmd.PersistentFlags().StringVarP(&key, "key", "k", "", "Storage key")
	rootCmd.PersistentFlags().BoolVar(&textFlag, "text", false, "Text output for store key")

	infoCmd := &cobra.Command{
		Use:   "info",
		Short: "Print plugin information (name / version / author)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if os.Getenv("SHIELD_PEDANTIC_INFO") != "" {
				ok := true
				for i, f := range info.Fields {
					name := f.Name
					if f.Name == "" {
						fmt.Fprintf(os.Stderr, "!! %s: field #%d has no name\n", info.Name, i+1)
						name = fmt.Sprintf("field #%d", i+1)
						ok = false
					}
					if f.Type == "" {
						fmt.Fprintf(os.Stderr, "!! %s: %s has no type\n", info.Name, name)
						ok = false
					}
					if f.Title == "" {
						fmt.Fprintf(os.Stderr, "!! %s: %s has no title\n", info.Name, name)
						ok = false
					}
					if f.Help == "" {
						fmt.Fprintf(os.Stderr, "!! %s: %s has no help\n", info.Name, name)
						ok = false
					} else if !strings.HasSuffix(f.Help, ".") {
						fmt.Fprintf(os.Stderr, "!! %s: %s help field does not end in a period.\n", info.Name, name)
						ok = false
					}
					if f.Type == "enum" {
						if len(f.Enum) == 0 {
							fmt.Fprintf(os.Stderr, "!! %s: %s is defined as an enum, but specifies no allowed values.\n", info.Name, name)
							ok = false
						}
					} else {
						if len(f.Enum) != 0 {
							fmt.Fprintf(os.Stderr, "!! %s: %s is not defined as an enum, but has the following allowed values: [%s]\n", info.Name, name, f.Enum)
							ok = false
						}
					}
				}
				if !ok {
					os.Exit(1)
				}
			}
			out, err := json.MarshalIndent(info, "", "    ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
				os.Exit(JSON_FAILURE)
			}
			fmt.Printf("%s\n", out)
			return nil
		},
	}

	makeSubCmd := func(use, short, command string) *cobra.Command {
		return &cobra.Command{
			Use:   use,
			Short: short,
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				err := dispatch(p, command, makeOpt())
				if err != nil {
					DEBUG("'%s' action returned error: %s", command, err)
					if e, ok := err.(UnsupportedActionError); ok && e.Action == "" {
						e.Action = command
						err = e
					}
					fmt.Fprintf(os.Stderr, "%s\n", err)
					os.Exit(codeForError(err))
				}
				return nil
			},
		}
	}

	rootCmd.AddCommand(
		infoCmd,
		makeSubCmd("validate", "Validate endpoint JSON/configuration", "validate"),
		makeSubCmd("backup", "Backup a target", "backup"),
		makeSubCmd("restore", "Replay a backup archive to a target", "restore"),
		makeSubCmd("store", "Store a backup archive", "store"),
		makeSubCmd("retrieve", "Stream a backup archive from storage", "retrieve"),
		makeSubCmd("purge", "Delete a backup archive from storage", "purge"),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "!!! %s\n", err)
		os.Exit(USAGE)
	}
}

func dispatch(p Plugin, mode string, opt Opt) error {
	var (
		err      error
		key      string
		size     int64
		endpoint ShieldEndpoint
	)

	if debug {
		DEBUG("'%s' action requested with the following options:", mode)
		if opt.Endpoint != "" {
			DEBUG("  --endpoint '%s'", opt.Endpoint)
		}
		if opt.Key != "" {
			DEBUG("  --key '%s'", opt.Key)
		}
		if opt.Text {
			DEBUG("  --text")
		}
	}

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

		key, size, err = p.Store(endpoint)
		if err != nil {
			return err
		}
		if opt.Text {
			fmt.Printf("%s\n", key)

		} else {
			output, err := json.MarshalIndent(struct {
				Key  string `json:"key"`
				Size int64  `json:"archive_size"`
			}{
				Key:  key,
				Size: size,
			}, "", "    ")

			if err != nil {
				return JSONError{Err: fmt.Sprintf("Could not JSON encode blob key: %s", err)}
			}

			fmt.Printf("%s\n", string(output))
		}

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

func GenUUID() string {
	return uuid.New()
}

func Redact(raw string) string {
	return fmt.Sprintf("<redacted>%s</redacted>", raw)
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
