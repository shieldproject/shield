package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	fmt "github.com/jhunt/go-ansi"
	"github.com/jhunt/go-cli"
	env "github.com/jhunt/go-envirotron"
	"github.com/jhunt/go-table"
	"gopkg.in/yaml.v2"

	"github.com/shieldproject/shield/client/v2/shield"
	"github.com/shieldproject/shield/tui"
)

var Version = ""

var opts struct {
	Help    bool `cli:"-h, --help"`
	Version bool `cli:"-v, --version"`

	Quiet bool `cli:"-q, --quiet"`
	Yes   bool `cli:"-y, --yes"`
	Debug bool `cli:"-D, --debug"  env:"SHIELD_DEBUG"`
	Trace bool `cli:"-T, --trace"  env:"SHIELD_TRACE"`
	Batch bool `cli:"-b, --batch, --no-batch" env:"SHIELD_BATCH_MODE"`

	Core   string `cli:"-c, --core" env:"SHIELD_CORE"`
	Config string `cli:"--config" env:"SHIELD_CLI_CONFIG"`
	JSON   bool   `cli:"--json" env:"SHIELD_JSON_MODE"`
	Long   bool   `cli:"-L, --long"`

	Exact bool `cli:"--exact"`
	Fuzzy bool `cli:"--fuzzy"`

	HelpCommand struct{} `cli:"help"`

	Commands struct {
		Full bool `cli:"--full"`
		List bool `cli:"--list"`
	} `cli:"commands"`

	Curl     struct{} `cli:"curl"`
	TimeSpec struct{} `cli:"timespec"`

	Status struct{} `cli:"status"`

	Import struct {
		Example bool `cli:"--example"`
	} `cli:"import"`

	Events struct {
		Skip []string `cli:"--skip"`
	} `cli:"events"`

	PS struct{} `cli:"ps"`

	/* CORES {{{ */
	Cores struct{} `cli:"cores"`
	API   struct {
		SkipSSLValidation bool   `cli:"-k, --skip-ssl-validate"`
		CACertificate     string `cli:"--ca-certificate, --ca-cert"`
	} `cli:"api"`

	/* }}} */
	/* AUTHN {{{ */
	Login struct {
		Providers bool `cli:"--providers"`

		Username string `cli:"-u, --username, --user" env:"SHIELD_CORE_USERNAME"`
		Password string `cli:"-p, --password, --pass" env:"SHIELD_CORE_PASSWORD"`

		Token string `cli:"-a, --auth-token, --token" env:"SHIELD_CORE_TOKEN"`

		Via string `cli:"--via"`
	} `cli:"login"`
	Logout struct{} `cli:"logout"`
	ID     struct{} `cli:"id"`

	/* }}} */
	/* LOCKING / INIT {{{ */
	Init struct {
		Master string `cli:"--master" env:"SHIELD_CORE_MASTER"`
	} `cli:"init, initialize"`
	Lock   struct{} `cli:"lock"`
	Unlock struct {
		Master string `cli:"--master" env:"SHIELD_CORE_MASTER"`
	} `cli:"unlock"`
	Rekey struct {
		OldMaster   string `cli:"--old-master"`
		NewMaster   string `cli:"--new-master"`
		RotateFixed bool   `cli:"--rotate-fixed-key"`
	} `cli:"rekey"`

	/* }}} */
	/* AUTH TOKENS {{{ */
	AuthTokens      struct{} `cli:"auth-tokens"`
	CreateAuthToken struct{} `cli:"create-auth-token"`
	RevokeAuthToken struct{} `cli:"revoke-auth-token"`

	/* }}} */
	/* TARGETS {{{ */
	Targets struct {
		Used       bool   `cli:"--used"`
		Unused     bool   `cli:"--unused"`
		WithPlugin string `cli:"--with-plugin"`
	} `cli:"targets"`
	Target       struct{} `cli:"target"`
	DeletTarget  struct{} `cli:"delete-target"`
	CreateTarget struct {
		Name    string   `cli:"-n, --name"`
		Summary string   `cli:"-s, --summary"`
		Agent   string   `cli:"-a, --agent"`
		Plugin  string   `cli:"-p, --plugin"`
		Data    []string `cli:"-d, --data"`
	} `cli:"create-target"`
	UpdateTarget struct {
		Name      string   `cli:"-n, --name"`
		Summary   string   `cli:"-s, --summary"`
		Agent     string   `cli:"-a, --agent"`
		Plugin    string   `cli:"-p, --plugin"`
		ClearData bool     `cli:"--clear-data"`
		Data      []string `cli:"-d, --data"`
	} `cli:"update-target"`

	/* }}} */
	/* BUCKETS {{{ */
	Buckets struct{} `cli:"buckets"`
	Bucket  struct{} `cli:"bucket"`

	/* }}} */
	/* JOBS {{{ */
	Jobs struct {
		Bucket   string `cli:"--bucket"`
		Target   string `cli:"--target"`
		Paused   bool   `cli:"--paused"`
		Unpaused bool   `cli:"--unpaused"`
	} `cli:"jobs"`
	Job        struct{} `cli:"job"`
	DeleteJob  struct{} `cli:"delete-job"`
	PauseJob   struct{} `cli:"pause-job"`
	UnpauseJob struct{} `cli:"unpause-job"`
	RunJob     struct{} `cli:"run-job"`
	CreateJob  struct {
		Name     string `cli:"-n, --name"`
		Summary  string `cli:"-s, --summary"`
		Target   string `cli:"--target"`
		Bucket   string `cli:"--bucket"`
		Schedule string `cli:"--schedule"`
		Retain   string `cli:"--retain"`
		Paused   bool   `cli:"--paused"`
		FixedKey bool   `cli:"--fixed-key"`
	} `cli:"create-job"`
	UpdateJob struct {
		Name       string `cli:"-n, --name"`
		Summary    string `cli:"-s, --summary"`
		Target     string `cli:"--target"`
		Bucket     string `cli:"--bucket"`
		Schedule   string `cli:"--schedule"`
		Retain     string `cli:"--retain"`
		FixedKey   bool   `cli:"--fixed-key"`
		NoFixedKey bool   `cli:"--no-fixed-key"`
	} `cli:"update-job"`

	/* }}} */
	/* ARCHIVES {{{ */
	Archives struct {
		Target string `cli:"--target"`
		Limit  int    `cli:"-l, --limit"`
	} `cli:"archives"`
	Archive        struct{} `cli:"archive"`
	RestoreArchive struct {
		Target string `cli:"--target, --to"`
	} `cli:"restore-archive"`
	PurgeArchive struct {
		Reason string `cli:"--reason"`
	} `cli:"purge-archive"`
	AnnotateArchive struct {
		Notes string `cli:"--notes"`
	} `cli:"annotate-archive"`

	/* }}} */
	/* TASKS {{{ */
	Tasks struct {
		Status   string `cli:"-s, --status"`
		Active   bool   `cli:"--active"`
		Inactive bool   `cli:"--inactive"`
		All      bool   `cli:"-a, --all"`
		Target   string `cli:"--target"`
		Type     string `cli:"--type"`
		Limit    int    `cli:"-l, --limit"`
		Before   string `cli:"--before"`
	} `cli:"tasks"`
	Task       struct{} `cli:"task"`
	CancelTask struct{} `cli:"cancel"`

	/* }}} */
	/* USERS {{{ */
	Users struct {
		WithSystemRole string `cli:"--with-system-role"`
	} `cli:"users"`
	User       struct{} `cli:"user"`
	DeleteUser struct{} `cli:"delete-user"`
	Passwd     struct{} `cli:"passwd"`
	CreateUser struct {
		Name     string `cli:"-n, --name"`
		Account  string `cli:"-u, --username"`
		Password string `cli:"-p, --password"`
		SysRole  string `cli:"--system-role"`
	} `cli:"create-user"`
	UpdateUser struct {
		Name     string `cli:"-n, --name"`
		Password string `cli:"-p, --password"`
		SysRole  string `cli:"--system-role"`
	} `cli:"update-user"`
	/* }}} */
	/* SESSIONS {{{ */
	Sessions struct {
		Limit    int    `cli:"-l, --limit"`
		UserUUID string `cli:"-u, --user-uuid"`
		IP       string `cli:"--ip"`
	} `cli:"sessions"`
	Session       struct{} `cli:"session"`
	DeleteSession struct{} `cli:"delete-session"`
	/* }}} */
	/* AGENTS {{{ */
	Agents struct {
		Limit   int  `cli:"-l, --limit"`
		Visible bool `cli:"--visible"`
		Hidden  bool `cli:"--hidden"`
	} `cli:"agents"`
	Agent struct {
		Metadata bool `cli:"-m, --metadata"`
		Plugins  bool `cli:"-p, --plugins"`
	} `cli:"agent"`
	DeleteAgent struct{} `cli:"delete-agent"`
	HideAgent   struct{} `cli:"hide-agent"`
	ShowAgent   struct{} `cli:"show-agent"`
	/* }}} */
	/* FIXUPS {{{ */
	Fixups     struct{} `cli:"fixups"`
	Fixup      struct{} `cli:"fixup"`
	ApplyFixup struct{} `cli:"apply-fixup"`
	/* }}} */

	Op struct {
		Pry struct{} `cli:"pry"`
		IFK struct{} `cli:"ifk"`
	} `cli:"op"`
}

func main() {
	opts.Config = fmt.Sprintf("%s/.shield", os.Getenv("HOME"))
	env.Override(&opts)

	command, args, err := cli.Parse(&opts)
	bail(err)

	if opts.JSON {
		opts.Yes = true
	}
	if opts.Quiet {
		opts.Trace = false
		opts.Debug = false
	}
	if opts.Batch {
		opts.Yes = true
	}
	if opts.Fuzzy {
		opts.Exact = false
	}

	if command == "" && !opts.Help && !opts.Version {
		if len(args) > 0 {
			bail(fmt.Errorf("Unrecognized command '%s'", args[0]))
		}

		opts.Help = true
	}
	if command == "help" && len(args) == 0 {
		opts.Help = true
		command = ""
	}

	if opts.Help && command == "" {
		fmt.Printf("USAGE: @G{shield} COMMAND [OPTIONS] [ARGUMENTS]\n")
		fmt.Printf("\n")
		fmt.Printf("@B{Global options:}\n")
		fmt.Printf("  -h, --help     Show this help screen.\n")
		fmt.Printf("  -v, --version  Print the version and exit.\n")
		fmt.Printf("\n")
		fmt.Printf("      --config   An alternate client configuration file to use. (@W{$SHIELD_CLI_CONFIG})\n")
		fmt.Printf("  -c, --core     Which SHIELD core to communicate with. (@W{$SHIELD_CORE})\n")
		fmt.Printf("\n")
		fmt.Printf("      --exact    Perform lookups against SHIELD data without using fuzzy matching.\n")
		fmt.Printf("      --fuzzy    Perform lookups against SHIELD data using fuzzy matching.\n")
		fmt.Printf("  -y, --yes      Answer all prompts affirmatively.\n")
		fmt.Printf("  -b, --batch    Batch mode; no questions will be asked. (@W{$SHIELD_BATCH_MODE})\n")
		fmt.Printf("      --no-batch\n")
		fmt.Printf("\n")
		fmt.Printf("  -q, --quiet    Suppress all output.\n")
		fmt.Printf("      --json     Format output as JSON. (@W{$SHIELD_JSON_MODE})\n")
		fmt.Printf("  -D, --debug    Enable debugging output. (@W{$SHIELD_DEBUG})\n")
		fmt.Printf("  -T, --trace    Trace HTTP communication with the SHIELD core.  (@W{$SHIELD_TRACE})\n")
		fmt.Printf("\n")
		fmt.Printf("@B{Environment variables:}\n")
		fmt.Printf("\n")
		fmt.Printf("  Some global options can be specified by setting environment variables.\n")
		fmt.Printf("  For example, the $SHIELD_CORE environment variable causes shield to\n")
		fmt.Printf("  behave as if the user called `shield --core \"$SHIELD_CORE\"`.\n")
		fmt.Printf("\n")
		fmt.Printf("  Here are the environment variable / command-line flag correlations:\n")
		fmt.Printf("\n")
		fmt.Printf("    SHIELD_CLI_CONFIG=@C{/path/to/.shield}      --config @C{/path/to/.shield}\n")
		fmt.Printf("    SHIELD_CORE=@C{prod-shield}                 --core @C{prod-shield}\n")
		fmt.Printf("    SHIELD_BATCH_MODE=@M{1}                     --batch\n")
		fmt.Printf("    SHIELD_JSON_MODE=@M{y}                      --json\n")
		fmt.Printf("    SHIELD_DEBUG=@M{1} SHIELD_TRACE=@M{yes}         --debug --trace\n")
		fmt.Printf("\n")
		fmt.Printf("For a list of common shield commands, try `shield commands`\n")
		fmt.Printf("\n")
		fmt.Printf("\n")
		os.Exit(0)
	}

	if opts.Version {
		if Version == "" || Version == "dev" {
			fmt.Printf("shield (development)\n")
		} else {
			fmt.Printf("shield v%s\n", Version)
		}
		os.Exit(0)
	}

	if command == "help" {
		command = args[0]
		args = args[1:]
		opts.Help = true
	}

	if command == "commands" { /* {{{ */
		if opts.Help {
			fmt.Printf("USAGE: @G{shield} @C{commands} [--list] [group [group ...]]\n")
			fmt.Printf("\n")
			fmt.Printf("  Summarizes the things that shield can do.\n")
			fmt.Printf("\n")
			fmt.Printf("  By default, all commands will be shown, grouped according to\n")
			fmt.Printf("  their function.  For example, authentication-related commands\n")
			fmt.Printf("  will be grouped together.\n")
			fmt.Printf("\n")
			fmt.Printf("  The @M{--list} argument countermands this behavior, listing all\n")
			fmt.Printf("  commands, alphabetically, regardless of functional similarities.\n")
			fmt.Printf("  Great for scripts and grepping!\n")
			fmt.Printf("\n")
			fmt.Printf("@B{Groups}\n")
			fmt.Printf("\n")
			fmt.Printf("  You can target your help query to a subset of functional groups\n")
			fmt.Printf("  by naming those groups on the command-line.  This works regardless\n")
			fmt.Printf("  of whether or not @M{--list} is in force.\n")
			fmt.Printf("\n")
			fmt.Printf("  Currently defined groups are:\n")
			fmt.Printf("\n")
			fmt.Printf("    @C{auth}       Authentication and SHIELD Endpoint management.\n")
			fmt.Printf("    @C{misc}       Commands that don't really fit elsewhere...\n")
			fmt.Printf("    @C{admin}      Administrative commands, for SHIELD site operators.\n")
			fmt.Printf("    @C{targets}    Target Data System management.\n")
			fmt.Printf("    @C{storage}    Cloud Storage management.\n")
			fmt.Printf("    @C{jobs}       Scheduled Backup Job management.\n")
			fmt.Printf("    @C{archives}   Backup Archive (and restore!) management.\n")
			fmt.Printf("    @C{tasks}      Task management.\n")
			fmt.Printf("\n\n")
			os.Exit(0)
		}

		set := make(map[string]bool)
		for _, want := range args {
			set[want] = true
		}

		first := true
		blank := func() {
			if !opts.Commands.List {
				fmt.Printf("\n")
			}
		}
		header := func(s string) {
			if !opts.Commands.List {
				if !first {
					fmt.Printf("\n\n")
				}
				first = false
				fmt.Printf("@G{%s:}\n\n", s)
			}
		}
		show := func(ss ...string) bool {
			if len(args) == 0 {
				return true
			}
			for _, accept := range ss {
				for _, have := range args {
					if accept == have {
						return true
					}
				}
			}
			return false
		}

		save := make([]string, 0)
		printc := func(s string) {
			if opts.Commands.List {
				save = append(save, s)
			} else {
				fmt.Printf(s)
			}
		}

		if show("misc", "miscellaneous") {
			header("Miscellaneous")
			printc("  commands                 Print this list of commands.\n")
			printc("  curl                     Issue raw HTTP requests to the targeted SHIELD Core.\n")
			printc("  timespec                 Explain Timespec scheduling strings.\n")
			printc("  status                   Show the status of the targeted SHIELD Core.\n")
			printc("  events                   Watch the even stream from the targeted SHIELD Core.\n")
		}
		if show("auth", "authentication") {
			header("Authentication (auth)")
			printc("  cores                    Print list of targeted SHIELD Cores.\n")
			printc("  api                      Target a new SHIELD Core, saving it in the configuration.\n")
			printc("  login                    Authenticate to the designated SHIELD Core.\n")
			printc("  logout                   Sign out of the current authenticated session.\n")
			printc("  id                       Display information about the current session.\n")
			printc("  passwd                   Change your password.\n")
			blank()
			printc("  auth-tokens              List your personal authentication tokens.\n")
			printc("  create-auth-token        Issue a new personal authentication token.\n")
			printc("  revoke-auth-token        Revoke an issued authentication token\n")
		}
		if show("admin", "administration", "administrative") {
			header("Administrative Tasks")
			printc("  init                     Initialize a new SHIELD Core.\n")
			printc("  lock                     Lock a SHIELD Core.\n")
			printc("  unlock                   Unlock a SHIELD Core (i.e. after a reboot).\n")
			printc("  rekey                    Change a SHIELD Core master (unlock) password.\n")
			blank()
			printc("  users                    List all of the local user accounts.\n")
			printc("  user                     Display the details for a single local user account.\n")
			printc("  create-user              Create a new local user account.\n")
			printc("  update-user              Modify the account settings of a local user.\n")
			printc("  delete-user              Delete a local user account.\n")
			blank()
			printc("  sessions                 List all authenticated sessions.\n")
			printc("  session                  Display the details of a single session.\n")
			printc("  delete-session           Revoke (forcibly de-authenticate) a session.\n")
		}
		if show("target", "targets") {
			header("Target Data Systems")
			printc("  targets                  List all target data systems.\n")
			printc("  target                   Display the details for a single target data system.\n")
			printc("  create-target            Configure a new target data system.\n")
			printc("  update-target            Reconfigure a target data system.\n")
			printc("  delete-target            Decomission an unused target data system.\n")
		}
		if show("bucket", "buckets", "storage") {
			header("Cloud Storage Buckets")
			printc("  buckets                  List all cloud storage buckets.\n")
			printc("  bucket                   Display the details for a single cloud storage bucket.\n")
		}
		if show("job", "jobs") {
			header("Scheduled Backup Jobs")
			printc("  jobs                     List configured backup jobs.\n")
			printc("  job                      Display the details for a single backup job.\n")
			printc("  create-job               Configure a new backup job.\n")
			printc("  update-job               Reconfigure a scheduled backup job.\n")
			printc("  delete-job               Decomission a scheduled backup job.\n")
			blank()
			printc("  pause-job                Pause a backup job, so that it doesn't get scheduled.\n")
			printc("  unpause-job              Unpause a backup job, so that it gets scheduled.\n")
			printc("  run-job                  Schedule an ad hoc run of a backup job.\n")
		}
		if show("archive", "archives", "backup", "backups") {
			header("Backup Data Archives")
			printc("  archives                 List all backup archives (valid or otherwise).\n")
			printc("  archive                  Display the details for a single backup archive.\n")
			printc("  restore-archive          Restore a backup archive to its original target system, or a new one.\n")
			printc("  purge-archive            Remove a backup archive from its cloud storage, and mark it invalid.\n")
			printc("  annotate-archive         Add notes about this archive, for the benefit of other operators.\n")
		}
		if show("task", "tasks") {
			header("Task Management")
			printc("  tasks                    List all tasks, running or otherwise.\n")
			printc("  task                     Display the details for a single task.\n")
			printc("  cancel                   Cancel a running task.\n")
		}
		blank()
		blank()

		if opts.Commands.List {
			sort.Strings(save)
			for _, s := range save {
				fmt.Printf(s)
			}
		}
		return
	}
	/* }}} */
	if opts.Help {
		ShowHelp(command)
		os.Exit(0)
	}

	config, err := ReadConfig(opts.Config, opts.Config+"_config")
	bail(err)

	switch command {
	case "cores": /* {{{ */
		tbl := table.NewTable("Name", "URL", "Verify TLS?")
		for alias, core := range config.SHIELDs {
			vfy := fmt.Sprintf("@G{yes}")
			if core.InsecureSkipVerify {
				vfy = fmt.Sprintf("@R{NO}")
			}
			tbl.Row(core, alias, core.URL, vfy)
		}
		tbl.Output(os.Stdout)
		return

	/* }}} */
	case "api": /* {{{ */
		if len(args) != 2 {
			fail(2, "Usage: shield %s URL ALIAS\n", command)
		}

		url := args[0]
		alias := args[1]

		if ok, _ := regexp.MatchString("^http", alias); ok {
			t := alias
			alias = url
			url = t
		}

		cacert := ""
		if opts.API.CACertificate != "" {
			if strings.Contains(opts.API.CACertificate, "\n") {
				/* embedded newlines detected in option value;
				   assume this is a literal PEM blob, perhaps provided via

				     shield api test https://shield.example.com \
				        --ca-certificate "$(vault read secret/ca/certs)"
				*/
				cacert = opts.API.CACertificate
			} else {
				/* no embedded newlines in option value;
				   check for the file on-disk if no interior newlines */
				b, err := ioutil.ReadFile(opts.API.CACertificate)
				bail(err)
				cacert = string(b)
			}
		}

		/* validate the SHIELD */
		c := &shield.Client{
			URL:                url,
			Debug:              opts.Debug,
			Trace:              opts.Trace,
			Session:            "",
			InsecureSkipVerify: opts.API.SkipSSLValidation,
			CACertificate:      cacert,
			TrustSystemCAs:     true,
		}
		nfo, err := c.Info()
		bail(err)

		fmt.Printf("@C{%s}  (@B{%s})  @G{OK}\n@W{SHIELD} @Y{%s}\n", alias, url, nfo.Env)
		config.Add(alias, SHIELD{
			URL:                url,
			InsecureSkipVerify: opts.API.SkipSSLValidation,
			CACertificate:      cacert,
		})
		bail(config.Write())
		return
		/* }}} */

	case "import": /* {{{ */
		if opts.Import.Example {
			fmt.Printf(`---
#
# THIS IS AN EXAMPLE import file to demonstrate the
# syntax used for a shield import.
#

# Local Users can be defined:
users:
  - name:     Administrator
    username: admin
    password: sekrit
    sysrole:  admin      # valid system roles are:
                         #
                         #   admin    - full access to SHIELD
                         #
                         #   manager  - handles users and system
                         #              role assignments.
                         #
                         #   engineer - for the technical stuff
                         #
                         #   operator - read-only access to (re)run
                         #              jobs and kick off restores.
                         #

  - name:     J User
    username: juser
    password: password
    sysrole:  ~          # juser has no system-level privileges

systems:
  - name:    A System
    summary: A protected data system
    agent:   10.255.6.7:5444
    plugin:  fs
    config:
      base_dir: /tmp

    jobs:
      - name:     Daily
        when:     daily 4:10am
        paused:   no
        bucket:   local
        retain:   4d

      - name:     Weekly
        when:     sundays at 2:45am
        paused:   yes
        bucket:   local
        retain:   28d
`)
			return
		}
		if len(args) < 1 {
			fail(2, "Usage: shield %s /path/to/manifest.yml ...\n", command)
		}
		readin := false
		for _, file := range args {
			var (
				m   ImportManifest
				b   []byte
				err error
			)

			if file == "-" {
				if readin {
					bail(fmt.Errorf("a second '-' file was encountered; but we already read standard input!"))
				}
				readin = true
				b, err = ioutil.ReadAll(os.Stdin)
				file = "<stdin>"
			} else {
				b, err = ioutil.ReadFile(file)
			}
			bailon(file, err)

			err = yaml.Unmarshal(b, &m)
			bailon(file, err)

			err = m.Normalize()
			bailon(file, err)

			if m.Core == "" {
				bail(config.Select(opts.Core))

				m.Core = config.Current.URL
				m.Token = config.Current.Session
				m.CA = config.Current.CACertificate
				m.InsecureSkipVerify = config.Current.InsecureSkipVerify
			}

			err = m.Deploy(&shield.Client{
				Debug: opts.Debug,
				Trace: opts.Trace,
			})
			bailon(file, err)
		}
		return
		/* }}} */
	}

	if opts.Core == "" {
		bail(fmt.Errorf("Missing required --core option (and no SHIELD_CORE environment variable was set)."))
	}
	bail(config.Select(opts.Core))

	c := &shield.Client{
		URL:                config.Current.URL,
		Debug:              opts.Debug,
		Trace:              opts.Trace,
		Session:            config.Current.Session,
		InsecureSkipVerify: config.Current.InsecureSkipVerify,
		CACertificate:      config.Current.CACertificate,
		TrustSystemCAs:     true,
	}

	switch command {
	case "login": /* {{{ */
		if opts.Login.Providers {
			providers, err := c.AuthProviders()
			bail(err)

			if opts.JSON {
				fmt.Printf("%s\n", asJSON(providers))
				break
			}

			tbl := table.NewTable("Name", "Description", "Type")
			for _, provider := range providers {
				tbl.Row(provider, provider.Identifier, provider.Name, provider.Type)
			}
			tbl.Output(os.Stdout)
			break
		}

		if opts.Login.Token != "" {
			err := c.Authenticate(&shield.TokenAuth{
				Token: opts.Login.Token,
			})
			bail(err)

		} else if opts.Login.Username != "" {
			if opts.Login.Password == "" {
				opts.Login.Password = secureprompt("@Y{SHIELD Password:} ")
			}
			err := c.Authenticate(&shield.LocalAuth{
				Username: opts.Login.Username,
				Password: opts.Login.Password,
			})
			bail(err)

		} else if opts.Login.Via != "" {
			provider, err := c.AuthProviderAnonymous(opts.Login.Via)
			bail(err)

			fmt.Printf("Visit the following URL in your favorite web browser:\n\n")
			fmt.Printf("  @B{%s%s}\n\n", c.URL, provider.CLIEntry)
			fmt.Printf("Then, enter the token you get, below.\n\n")

			err = c.Authenticate(&shield.TokenAuth{
				Token: prompt("@Y{Token:} "),
			})
			bail(err)

		} else if opts.Batch {
			bail(fmt.Errorf("Unable to login interactively under `--batch` mode"))

		} else {
			opts.Login.Username = prompt("@C{SHIELD Username:} ")
			opts.Login.Password = secureprompt("@Y{SHIELD Password:} ")
			err := c.Authenticate(&shield.LocalAuth{
				Username: opts.Login.Username,
				Password: opts.Login.Password,
			})
			bail(err)
		}

		config.Current.Session = c.Session
		bail(config.Write())

		fmt.Printf("logged in successfully\n")

	/* }}} */
	case "logout": /* {{{ */
		bail(c.Logout())

		config.Current.Session = ""
		bail(config.Write())

		fmt.Printf("logged out successfully\n")

	/* }}} */
	case "id": /* {{{ */
		id, err := c.AuthID()
		bail(err)

		if id.Unauthenticated {
			fmt.Printf("@Y{not authenticated}\n")
			break
		}

		r := tui.NewReport()
		r.Add("Display Name", id.User.Name)
		r.Add("Username", id.User.Account)
		r.Add("Designation", id.User.Account+"@"+id.User.Backend)
		if id.User.SysRole != "" {
			r.Add("System Role", id.User.SysRole)
		} else {
			r.Add("System Role", fmt.Sprintf("@Y{none}"))
		}
		fmt.Printf("@G{Account Details}\n")
		r.Output(os.Stdout)

	/* }}} */

	case "init": /* {{{ */
		if opts.Init.Master == "" {
			a := secureprompt("@Y{New SHIELD Core master password}: ")
			b := secureprompt("@Y{Confirm new master password}: ")
			if a == "" {
				fail(3, "@R{master password cannot be blank!}\n")
			} else if a != b {
				fail(3, "@R{master password mismatch!}\n")
			}
			opts.Init.Master = a
		}
		fixedKey, err := c.Initialize(opts.Init.Master)
		bail(err)

		fmt.Printf("SHIELD core unlocked successfully.\n")

		if fixedKey != "" {
			fmt.Printf("@R{BELOW IS YOUR FIXED KEY FOR RECOVERING FIXED-KEY BACKUPS.}\n")
			fmt.Printf("@R{SAVE THIS IN A SECURE LOCATION.}\n")
			fmt.Printf("----------------------------------------------------------------\n")
			fmt.Printf("@Y{" + c.SplitKey(fixedKey, 64) + "}")
			fmt.Printf("\n----------------------------------------------------------------\n")
		} else {
			bail(fmt.Errorf("Failed to initialize Fixed Key!"))
		}

	/* }}} */
	case "lock": /* {{{ */
		bail(c.Lock())
		fmt.Printf("SHIELD core locked successfully.\n")

	/* }}} */
	case "unlock": /* {{{ */
		if opts.Unlock.Master == "" {
			opts.Unlock.Master = secureprompt("@Y{SHIELD Core master password:} ")
		}
		err := c.Unlock(opts.Unlock.Master)
		bail(err)

		fmt.Printf("SHIELD core unlocked successfully.\n")

	/* }}} */
	case "rekey": /* {{{ */
		if opts.Rekey.OldMaster == "" {
			opts.Rekey.OldMaster = secureprompt("@Y{Current master password:} ")
		}
		if opts.Rekey.NewMaster == "" {
			a := secureprompt("@C{New SHIELD Core master password:} ")
			b := secureprompt("@C{Confirm new master password}: ")
			if a == "" {
				fail(3, "@R{master password cannot be blank!}\n")
			} else if a != b {
				fail(3, "@R{new master password mismatch!}\n")
			}
			opts.Rekey.NewMaster = a
		}
		fixedKey, err := c.Rekey(opts.Rekey.OldMaster, opts.Rekey.NewMaster, opts.Rekey.RotateFixed)
		bail(err)

		if fixedKey != "" {
			fmt.Printf("@R{BELOW IS YOUR FIXED KEY FOR RECOVERING FIXED-KEY BACKUPS.}\n")
			fmt.Printf("@R{SAVE THIS IN A SECURE LOCATION.}\n")
			fmt.Printf("----------------------------------------------------------------\n")
			fmt.Printf("@Y{" + c.SplitKey(fixedKey, 64) + "}")
			fmt.Printf("\n----------------------------------------------------------------\n")
		} else if opts.Rekey.RotateFixed {
			bail(fmt.Errorf("Failed to initialize Fixed Key!"))
		}

		fmt.Printf("SHIELD core rekeyed successfully.\n")

	/* }}} */

	case "curl": /* {{{ */
		if len(args) < 1 || len(args) > 3 {
			fail(2, "Usage: shield %s [METHOD] RELATIVE-URL [BODY]\n", command)
		}

		var method, path, body string
		switch len(args) {
		case 1:
			method = "GET"
			path = args[0]
		case 2:
			method = args[0]
			path = args[1]
		case 3:
			method = args[0]
			path = args[1]
			body = args[2]
		}

		if body == "-" {
			b, err := ioutil.ReadAll(os.Stdin)
			bail(err)
			body = string(b)
		}

		code, response, err := c.Curl(method, path, body)
		bail(err)
		fmt.Printf("%s\n", asJSON(response))
		if code >= 400 {
			os.Exit(code / 100)
		}

	/* }}} */
	case "timespec": /* {{{ */
		if len(args) < 1 {
			fail(2, "Usage: shield %s \"a schedule string\"\n", command)
		}
		ok, spec, err := c.CheckTimespec(strings.Join(args, " "))
		if err != nil {
			bail(fmt.Errorf("Failed to check timespec: %s\n", err))
		}
		if !ok {
			fail(1, "Invalid timespec\n")
		}

		fmt.Printf("%s\n", spec)
		return

	/* }}} */
	case "status": /* {{{ */
		info, err := c.Info()
		bail(err)

		status, err := c.GlobalStatus()
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(
				struct {
					Shield  *shield.Info         `json:"shield"`
					Health  shield.StatusHealth  `json:"health"`
					Storage shield.StatusStorage `json:"storage"`
					Jobs    shield.StatusJobs    `json:"jobs"`
					Stats   shield.StatusStats   `json:"stats"`
				}{
					Shield:  info,
					Health:  status.Health,
					Storage: status.Storage,
					Jobs:    status.Jobs,
					Stats:   status.Stats,
				}))
			break
		}

		fmt.Printf("@B{###############################################}\n\n")
		if info.Version == "" {
			fmt.Printf("  @W{SHIELD %s} @Y{(development)} :: api @C{v%d}\n", info.Env, info.API)
		} else {
			fmt.Printf("  @W{SHIELD %s} @C{v%s} :: api @C{v%d}\n", info.Env, info.Version, info.API)
		}
		fmt.Printf("\n\n")

		if info.MOTD != "" {
			fmt.Printf("@B{##} @M{MESSAGE OF THE DAY} @B{#########################}\n\n")
			fmt.Printf("%s\n\n", wrap(info.MOTD, 60))
		}

		fmt.Printf("@B{##} @M{CURRENT HEALTH} @B{#############################}\n\n")
		good := "✔"
		bad := "✘"
		if status.Health.Core == "unlocked" {
			fmt.Printf("   @G{%s} core is %s\n", good, status.Health.Core)
		} else {
			fmt.Printf("   @R{%s} core is %s\n", bad, status.Health.Core)
		}
		if status.Health.StorageOK {
			fmt.Printf("   @G{%s} cloud storage is connected\n", good)
		} else {
			fmt.Printf("   @R{%s} cloud storage is @R{FAILING}\n", bad)
		}
		if status.Health.JobsOK {
			fmt.Printf("   @G{%s} jobs are running successfully\n", good)
		} else {
			fmt.Printf("   @R{%s} jobs are @R{FAILING}\n", bad)
		}
		fmt.Printf("\n")

		fmt.Printf("  @C{%d} @W{systems} / @C{%d} @W{jobs} / @C{%d} @W{archives}\n", status.Stats.Systems, status.Stats.Jobs, status.Stats.Archives)
		fmt.Printf("  @C{%s} @W{total storage} used\n", formatBytes(status.Stats.StorageUsed))
		fmt.Printf("\n\n")

		fmt.Printf("@B{##} @M{STORAGE HEALTH} @B{#############################}\n\n")
		for _, s := range status.Storage {
			if s.Health {
				fmt.Printf("   @G{%s} %s is @G{OK}\n", good, s.Name)
			} else {
				fmt.Printf("   @R{%s} %s is @R{FAILING}\n", bad, s.Name)
			}
		}
		fmt.Printf("\n\n")

		fmt.Printf("@B{##} @M{BACKUP JOB HEALTH} @B{##########################}\n\n")
		for _, j := range status.Jobs {
			if j.Healthy {
				fmt.Printf("   @G{%s} %s/%s is @G{OK}\n", good, j.Target, j.Job)
			} else {
				fmt.Printf("   @R{%s} %s/%s is @R{FAILING}\n", bad, j.Target, j.Job)
			}
		}
		fmt.Printf("\n\n")

	/* }}} */
	case "events": /* {{{ */
		skip := make(map[string]bool)
		for _, what := range opts.Events.Skip {
			skip[what] = true
		}

		header := false
		err := c.StreamEvents(func(ev shield.Event) {
			if _, ok := skip[ev.Event]; ok {
				return
			}
			if _, ok := skip[ev.Queue]; ok {
				return
			}

			if opts.JSON {
				fmt.Printf("%s\n", asJSON(ev))
				return
			}

			if !header {
				header = true
				fmt.Printf("@W{Queue}                                         @W{Event Type}             @W{Object Type}            @W{Object}\n")
				fmt.Printf("-------------------------------------------   --------------------   --------------------   -----------------\n")
			}
			fmt.Printf("@C{%-43s}   %-20s   @Y{%-20s}   ", ev.Queue, ev.Event, ev.Type)

			b, err := json.MarshalIndent(ev.Data, "", "  ")
			if err != nil {
				fmt.Printf("%s\n", ev.Data)
			} else {
				prefix := fmt.Sprintf("%92s", " ")
				for i, s := range strings.Split(string(b), "\n") {
					if i == 0 {
						fmt.Printf("%s\n", s)
					} else {
						fmt.Printf("%s%s\n", prefix, s)
					}
				}
			}
			fmt.Printf("\n")
		})
		bail(err)

	/* }}} */
	case "ps": /* {{{ */
		ps, err := c.SchedulerStatus()
		bail(err)

		none := fmt.Sprintf("@K{(none)}")
		oops := fmt.Sprintf("@R{(oops)}")

		tbl := table.NewTable("#", "Op", "Status", "Task", "System", "Job", "Archive", "Agent")
		for _, worker := range ps.Workers {
			if worker.Idle {
				tbl.Row(worker, worker.ID, none, fmt.Sprintf("@C{idle}"),
					none, none, none, none, none)

			} else {
				op := oops
				if worker.Op != "" {
					op = fmt.Sprintf("@G{%s}", worker.Op)
				}

				status := oops
				if worker.Status != "" {
					status = fmt.Sprintf("(%s)", worker.Status)
				}

				task := oops
				if worker.TaskUUID != "" {
					task = fmt.Sprintf("@Y{%s}", uuid8(worker.TaskUUID))
				}

				system := none
				if worker.System != nil {
					system = fmt.Sprintf("@W{%s}\n(%s)", worker.System.Name, uuid8(worker.System.UUID))
				}

				job := none
				if worker.Job != nil {
					job = fmt.Sprintf("@W{%s}\n(%s)", worker.Job.Name, uuid8(worker.Job.UUID))
				}

				archive := none
				if worker.Archive != nil {
					archive = fmt.Sprintf("@W{%s}\n(%s; %s)\n", worker.Archive.UUID, "-", "-") /* FIXME */
				}

				agent := oops
				if worker.Agent != "" {
					agent = fmt.Sprintf("@Y{%s}", worker.Agent)
				}

				tbl.Row(worker, worker.ID, op, status, task, system, job, archive, agent)
			}
		}
		fmt.Printf("@M{Scheduler Threads}\n\n")
		tbl.Output(os.Stdout)

		fmt.Printf("\n\n")
		fmt.Printf("@M{Task Backlog}\n\n")
		if len(ps.Backlog) > 0 {
			tbl = table.NewTable("Priority", "#", "Op", "Task", "System", "Job", "Archive", "Agent")
			for _, t := range ps.Backlog {
				op := oops
				if t.Op != "" {
					op = fmt.Sprintf("@G{%s}", t.Op)
				}

				task := oops
				if t.TaskUUID != "" {
					task = fmt.Sprintf("@Y{%s}", uuid8(t.TaskUUID))
				}

				system := none
				if t.System != nil {
					system = fmt.Sprintf("@W{%s}\n(%s)", t.System.Name, uuid8(t.System.UUID))
				}

				job := none
				if t.Job != nil {
					job = fmt.Sprintf("@W{%s}\n(%s)", t.Job.Name, uuid8(t.Job.UUID))
				}

				archive := none
				if t.Archive != nil {
					archive = fmt.Sprintf("@W{%s}\n(%s; %s)\n", t.Archive.UUID, "-", "-") /* FIXME */
				}

				agent := oops
				if t.Agent != "" {
					agent = fmt.Sprintf("@Y{%s}", t.Agent)
				}
				tbl.Row(t, t.Priority, t.Position, op, task, system, job, archive, agent)
			}
			tbl.Output(os.Stdout)

		} else {
			fmt.Printf("  none\n")
		}

		fmt.Printf("\n\n")

	/* }}} */

	case "auth-tokens": /* {{{ */
		tokens, err := c.ListAuthTokens()
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(tokens))
			break
		}

		tbl := table.NewTable("Name", "Created at", "Last seen")
		for _, token := range tokens {
			tbl.Row(token, token.Name, strftime(token.CreatedAt), strftimenil(token.LastSeen, "(never)"))
		}
		tbl.Output(os.Stdout)

	/* }}} */
	case "create-auth-token": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s TOKEN-NAME\n", command)
		}

		t, err := c.CreateAuthToken(&shield.AuthToken{Name: args[0]})
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(t))
			break
		}

		fmt.Printf("@C{%s}\n", t.Session)

	/* }}} */
	case "revoke-auth-token": /* {{{ */
		if len(args) == 0 {
			fail(2, "Usage: shield %s TOKEN-NAME [OTHER-TOKEN ...]\n", command)
		}

		tokens, err := c.ListAuthTokens()
		bail(err)

		rc := 0
		for _, revoke := range args {
			found := false
			for _, token := range tokens {
				if token.Name == revoke {
					found = true
					if err := c.RevokeAuthToken(token); err != nil {
						fmt.Fprintf(os.Stderr, "%s: %s\n", token.Name, err)
						rc = 3
					} else {
						fmt.Fprintf(os.Stderr, "%s: revoked\n", token.Name)
					}
					break
				}
			}
			if !found {
				fmt.Fprintf(os.Stderr, "%s: token not found\n", revoke)
				rc = 3
			}
		}
		os.Exit(rc)

	/* }}} */

	case "targets": /* {{{ */
		required(!(opts.Targets.Used && opts.Targets.Unused),
			"The --used and --unused options are mutually exclusive.")
		required(len(args) <= 1, "Too many arguments.")

		filter := &shield.TargetFilter{
			Plugin: opts.Targets.WithPlugin,
			Fuzzy:  !opts.Exact,
		}
		if len(args) == 1 {
			filter.Name = args[0]
			filter.UUID = args[0]
		}
		if opts.Targets.Used || opts.Targets.Unused {
			x := opts.Targets.Used
			filter.Used = &x
		}

		targets, err := c.ListTargets(filter)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(targets))
			break
		}

		tbl := table.NewTable("UUID", "Name", "Summary", "Plugin", "SHIELD Agent")
		for _, target := range targets {
			tbl.Row(target, uuid8full(target.UUID, opts.Long), target.Name, wrap(target.Summary, 35), target.Plugin, target.Agent)
		}
		tbl.Output(os.Stdout)

	/* }}} */
	case "target": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s NAME-or-UUID\n", command)
		}

		t, err := c.FindTarget(args[0], !opts.Exact)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(t))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", t.UUID)
		r.Add("Name", t.Name)
		r.Add("Summary", wrap(t.Summary, 35))
		r.Add("SHIELD Agent", t.Agent)
		r.Add("Backup Plugin", t.Plugin)
		r.Break()
		r.Add("Configuration", asJSON(t.Config))
		r.Output(os.Stdout)

	/* }}} */
	case "create-target": /* {{{ */
		conf, err := dataConfig(opts.CreateTarget.Data)
		bail(err)

		if !opts.Batch {
			if opts.CreateTarget.Name == "" {
				opts.CreateTarget.Name = prompt("@C{Target Name}: ")
			}
			if opts.CreateTarget.Summary == "" {
				opts.CreateTarget.Summary = prompt("@C{Description}: ")
			}
			if opts.CreateTarget.Agent == "" {
				opts.CreateTarget.Agent = prompt("@C{SHIELD Agent (IP:port)}: ")
			}
			if opts.CreateTarget.Plugin == "" {
				opts.CreateTarget.Plugin = prompt("@C{Backup Plugin}: ")
			}
		}

		t, err := c.CreateTarget(&shield.Target{
			Name:    opts.CreateTarget.Name,
			Summary: opts.CreateTarget.Summary,
			Agent:   opts.CreateTarget.Agent,
			Plugin:  opts.CreateTarget.Plugin,
			Config:  conf,
		})
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(t))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", t.UUID)
		r.Add("Name", t.Name)
		r.Add("Summary", t.Summary)
		r.Add("SHIELD Agent", t.Agent)
		r.Add("Backup Plugin", t.Plugin)
		r.Output(os.Stdout)

	/* }}} */
	case "update-target": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s [OPTIONS] NAME-or-UUID\n", command)
		}
		t, err := c.FindTarget(args[0], true)
		bail(err)

		conf, err := dataConfig(opts.UpdateTarget.Data)
		bail(err)

		if opts.UpdateTarget.Name != "" {
			t.Name = opts.UpdateTarget.Name
		}
		if opts.UpdateTarget.Summary != "" {
			t.Summary = opts.UpdateTarget.Summary
		}
		if opts.UpdateTarget.Agent != "" {
			t.Agent = opts.UpdateTarget.Agent
		}
		if opts.UpdateTarget.Plugin != "" && t.Plugin != opts.UpdateTarget.Plugin {
			opts.UpdateTarget.ClearData = true
			t.Plugin = opts.UpdateTarget.Plugin
		}

		if t.Config == nil {
			t.Config = make(map[string]interface{})
		}
		if opts.UpdateTarget.ClearData {
			t.Config = conf
		} else {
			for k, v := range conf {
				t.Config[k] = v
			}
		}

		_, err = c.UpdateTarget(t)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(t))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", t.UUID)
		r.Add("Name", t.Name)
		r.Add("Summary", t.Summary)
		r.Add("SHIELD Agent", t.Agent)
		r.Add("Backup Plugin", t.Plugin)
		r.Add("Configuration", asJSON(t.Config))
		r.Output(os.Stdout)

	/* }}} */
	case "delete-target": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s [OPTIONS] NAME-or-UUID\n", command)
		}
		t, err := c.FindTarget(args[0], true)
		bail(err)

		if !confirm(opts.Yes, "Delete target @Y{%s}?", t.Name) {
			break
		}
		r, err := c.DeleteTarget(t)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(r))
			break
		}
		fmt.Printf("%s\n", r.OK)

	/* }}} */

	case "buckets": /* {{{ */
		required(len(args) == 0, "Too many arguments.")

		buckets, err := c.ListBuckets()
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(buckets))
			break
		}

		tbl := table.NewTable("Key", "Name", "Description", "Encryption")
		for _, bucket := range buckets {
			tbl.Row(bucket, bucket.Key, bucket.Name, wrap(bucket.Description, 35), bucket.Encryption)
		}
		tbl.Output(os.Stdout)

	/* }}} */
	case "bucket": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s NAME-or-KEY\n", command)
		}

		bucket, err := c.FindBucket(args[0], !opts.Exact)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(bucket))
			break
		}

		r := tui.NewReport()
		r.Add("Key", bucket.Key)
		r.Add("Name", bucket.Name)
		r.Add("Description", bucket.Description)
		r.Add("Encryption", bucket.Encryption)
		r.Output(os.Stdout)

	/* }}} */

	case "jobs": /* {{{ */
		required(!(opts.Jobs.Paused && opts.Jobs.Unpaused),
			"The --paused and --unpaused options are mutually exclusive.")
		required(len(args) <= 1, "Too many arguments.")

		filter := &shield.JobFilter{
			Fuzzy:  !opts.Exact,
			Bucket: opts.Jobs.Bucket,
			Target: opts.Jobs.Target,
		}
		if opts.Jobs.Paused || opts.Jobs.Unpaused {
			filter.Paused = &opts.Jobs.Paused
		}
		if len(args) == 1 {
			filter.Name = args[0]
			filter.UUID = args[0]
		}

		jobs, err := c.ListJobs(filter)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(jobs))
			break
		}

		tbl := table.NewTable("UUID", "Name", "Summary", "Schedule", "Status", "Retention", "SHIELD Agent", "Target", "Bucket", "Fixed-Key")
		for _, job := range jobs {
			tbl.Row(job, uuid8full(job.UUID, opts.Long), job.Name, wrap(job.Summary, 35), job.Schedule, job.Status(), fmt.Sprintf("%dd (%d archives)", job.KeepDays, job.KeepN), job.Agent, job.Target.Name, job.Bucket, job.FixedKey)
		}
		tbl.Output(os.Stdout)

	/* }}} */
	case "job": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s NAME-or-UUID\n", command)
		}

		job, err := c.FindJob(args[0], !opts.Exact)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(job))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", job.UUID)
		r.Add("Name", job.Name)
		r.Add("Status", job.Status())
		r.Break()

		r.Add("Schedule", job.Schedule)
		r.Add("Keep", fmt.Sprintf("%d days (%d archives)", job.KeepDays, job.KeepN))
		r.Break()

		r.Add("Data System", job.Target.Name)
		r.Add("Backup Plugin", job.Target.Plugin)
		r.Add("SHIELD Agent", job.Agent)
		r.Break()

		r.Add("Cloud Storage", job.Bucket)
		r.Break()

		r.Add("Fixed-Key", strconv.FormatBool(job.FixedKey))
		r.Add("Notes", job.Summary)

		r.Output(os.Stdout)

	/* }}} */
	case "create-job": /* {{{ */
		if !opts.Batch {
			if opts.CreateJob.Name == "" {
				opts.CreateJob.Name = prompt("@C{Job Name}: ")
			}
			if opts.CreateJob.Summary == "" {
				opts.CreateJob.Summary = prompt("@C{Notes}: ")
			}
			for opts.CreateJob.Target == "" {
				id := prompt("@C{Target Data System}: ")
				if len(id) > 0 && id[0] == '?' {
					SearchTargets(c, id[1:])
					continue
				}
				if target, err := c.FindTarget(id, !opts.Exact); err != nil {
					fmt.Fprintf(os.Stderr, "@Y{%s}\n", err)
				} else {
					opts.CreateJob.Target = target.UUID
					break
				}
			}
			for opts.CreateJob.Bucket == "" {
				id := prompt("@C{Cloud Storage}: ")
				if len(id) > 0 && id[0] == '?' {
					SearchBuckets(c, id[1:])
					continue
				}
				if bucket, err := c.FindBucket(id, !opts.Exact); err != nil {
					fmt.Fprintf(os.Stderr, "@Y{%s}\n", err)
				} else {
					opts.CreateJob.Bucket = bucket.Key
					break
				}
			}
			if opts.CreateJob.Schedule == "" {
				opts.CreateJob.Schedule = prompt("@C{Schedule}: ")
			}
			if opts.CreateJob.Retain == "" {
				opts.CreateJob.Retain = prompt("@C{Retain}: ")
			}

			if opts.CreateJob.Summary == "" {
				opts.CreateJob.Summary = prompt("@C{Notes}: ")
			}

		} else {
			if id := opts.CreateJob.Target; id != "" {
				if target, err := c.FindTarget(id, !opts.Exact); err != nil {
					bail(err)
				} else {
					opts.CreateJob.Target = target.UUID
				}
			}
			if id := opts.CreateJob.Bucket; id != "" {
				if bucket, err := c.FindBucket(id, !opts.Exact); err != nil {
					bail(err)
				} else {
					opts.CreateJob.Bucket = bucket.Key
				}
			}
		}

		job, err := c.CreateJob(&shield.Job{
			Name:       opts.CreateJob.Name,
			Summary:    opts.CreateJob.Summary,
			TargetUUID: opts.CreateJob.Target,
			Bucket:     opts.CreateJob.Bucket,
			Schedule:   opts.CreateJob.Schedule,
			Retain:     opts.CreateJob.Retain,
			Paused:     opts.CreateJob.Paused,
			FixedKey:   opts.CreateJob.FixedKey,
		})
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(job))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", job.UUID)
		r.Add("Name", job.Name)
		r.Add("Summary", job.Summary)
		r.Output(os.Stdout)

	/* }}} */
	case "update-job": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s [OPTIONS] NAME-or-UUID\n", command)
		}
		job, err := c.FindJob(args[0], !opts.Exact)
		bail(err)

		if opts.UpdateJob.Name != "" {
			job.Name = opts.UpdateJob.Name
		}
		if opts.UpdateJob.Summary != "" {
			job.Summary = opts.UpdateJob.Summary
		}
		if id := opts.UpdateJob.Target; id != "" {
			if target, err := c.FindTarget(id, !opts.Exact); err != nil {
				bail(err)
			} else {
				job.TargetUUID = target.UUID
			}
		}
		if id := opts.UpdateJob.Bucket; id != "" {
			if bucket, err := c.FindBucket(id, !opts.Exact); err != nil {
				bail(err)
			} else {
				job.Bucket = bucket.Key
			}
		}
		if opts.UpdateJob.Schedule != "" {
			job.Schedule = opts.UpdateJob.Schedule
		}
		if opts.UpdateJob.Retain != "" {
			job.Retain = opts.UpdateJob.Retain
		}

		if opts.UpdateJob.FixedKey {
			job.FixedKey = true
		}
		if opts.UpdateJob.NoFixedKey {
			job.FixedKey = false
		}

		_, err = c.UpdateJob(job)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(job))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", job.UUID)
		r.Add("Name", job.Name)
		r.Add("Summary", job.Summary)
		r.Output(os.Stdout)

	/* }}} */
	case "delete-job": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s [OPTIONS] NAME-or-UUID\n", command)
		}

		job, err := c.FindJob(args[0], true)
		bail(err)

		if !confirm(opts.Yes, "Delete job @Y{%s} (for system @W{%s})?", job.Name, job.Target.Name) {
			break
		}
		r, err := c.DeleteJob(job)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(r))
			break
		}
		fmt.Printf("%s\n", r.OK)

	/* }}} */
	case "pause-job": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s [OPTIONS] NAME-or-UUID\n", command)
		}

		job, err := c.FindJob(args[0], true)
		bail(err)

		if !confirm(opts.Yes, "Pause job @Y{%s} (for system @W{%s})?", job.Name, job.Target.Name) {
			break
		}
		r, err := c.PauseJob(job)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(r))
			break
		}
		fmt.Printf("%s\n", r.OK)

	/* }}} */
	case "unpause-job": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s [OPTIONS] NAME-or-UUID\n", command)
		}

		job, err := c.FindJob(args[0], true)
		bail(err)

		if !confirm(opts.Yes, "Unpause job @Y{%s} (for system @W{%s})?", job.Name, job.Target.Name) {
			break
		}
		r, err := c.UnpauseJob(job)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(r))
			break
		}
		fmt.Printf("%s\n", r.OK)

	/* }}} */
	case "run-job": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s [OPTIONS] NAME-or-UUID\n", command)
		}

		job, err := c.FindJob(args[0], !opts.Exact)
		bail(err)

		if !confirm(opts.Yes, "Run job @Y{%s} (for system @W{%s})?", job.Name, job.Target.Name) {
			break
		}
		r, err := c.RunJob(job)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(r))
			break
		}
		fmt.Printf("%s\n", r.OK)

	/* }}} */

	case "archives": /* {{{ */
		required(len(args) <= 1, "Too many arguments.")

		if opts.Archives.Limit == 0 {
			opts.Archives.Limit = 1000 /* sane upper limit */
		}
		if opts.Archives.Target != "" {
			t, err := c.FindTarget(opts.Archives.Target, !opts.Exact)
			bail(err)
			opts.Archives.Target = t.UUID
		}

		filter := &shield.ArchiveFilter{
			Target: opts.Archives.Target,
			Limit:  &opts.Archives.Limit,
			Fuzzy:  !opts.Exact,
		}
		if len(args) == 1 {
			filter.UUID = args[0]
		}

		archives, err := c.ListArchives(filter)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(archives))
			break
		}

		tbl := table.NewTable("UUID", "Key", "Size", "Status", "Encryption")
		for _, archive := range archives {
			tbl.Row(archive, uuid8full(archive.UUID, opts.Long), archive.Key, archive.Status, archive.EncryptionType, formatBytes(archive.Size))
		}
		tbl.Output(os.Stdout)

	/* }}} */
	case "archive": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s NAME-or-UUID\n", command)
		}

		archive, err := c.FindArchive(args[0], !opts.Exact)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(archive))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", archive.UUID)
		r.Add("Key", archive.Key)
		r.Add("Status", archive.Status)
		r.Add("Size", formatBytes(archive.Size))
		r.Add("Encryption", archive.EncryptionType)
		r.Add("Notes", archive.Notes)
		r.Output(os.Stdout)

	/* }}} */
	case "restore-archive": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s NAME-or-UUID\n", command)
		}

		archive, err := c.FindArchive(args[0], !opts.Exact)
		bail(err)

		var target *shield.Target
		if id := opts.RestoreArchive.Target; id != "" {
			target, err = c.FindTarget(id, !opts.Exact)
			bail(err)
		}

		task, err := c.RestoreArchive(archive, target)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(task))
			break
		}

		fmt.Printf("Scheduled restore; task @C{%s}\n", task.UUID)

	/* }}} */
	case "purge-archive": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s NAME-or-UUID\n", command)
		}

		archive, err := c.FindArchive(args[0], !opts.Exact)
		bail(err)

		if opts.PurgeArchive.Reason != "" {
			archive.Notes = opts.PurgeArchive.Reason
			_, err = c.UpdateArchive(archive)
			if err != nil && !opts.JSON {
				fmt.Fprintf(os.Stderr, "@Y{WARNING: Unable to update archive with reason for purge}: %s", err)
			}
		}

		rs, err := c.DeleteArchive(archive)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(rs))
			break
		}

		fmt.Printf("%s\n", rs.OK)

	/* }}} */
	case "annotate-archive": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s NAME-or-UUID --notes ...\n", command)
		}

		required(opts.AnnotateArchive.Notes != "", "Missing required --notes option.")

		archive, err := c.FindArchive(args[0], !opts.Exact)
		bail(err)

		archive.Notes = opts.AnnotateArchive.Notes
		archive, err = c.UpdateArchive(archive)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(archive))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", archive.UUID)
		r.Add("Key", archive.Key)
		r.Add("Status", archive.Status)
		r.Add("Notes", archive.Notes)
		r.Output(os.Stdout)

	/* }}} */

	case "tasks": /* {{{ */
		required(!(opts.Tasks.Active && opts.Tasks.Inactive),
			"The --active and --inactive options are mutually exclusive.")
		required(!(opts.Tasks.All && opts.Tasks.Inactive),
			"The --all and --inactive options are mutually exclusive.")
		required(!(opts.Tasks.All && opts.Tasks.Active),
			"The --all and --active options are mutually exclusive.")
		required(len(args) <= 0, "Too many arguments.")

		switch opts.Tasks.Status {
		case "":
			/* not specified; which is ok... */
		case "all":
			opts.Tasks.All = true
		case "pending", "scheduled", "running", "canceled", "failed", "done":
			/* good enough to pass validation... */
		default:
			fail(3, "Invalid --status value of '%s'\n(must be one of all, pending, running,\n cnaceled, failed, or done).", opts.Tasks.Status)
		}

		if opts.Tasks.All {
			opts.Tasks.Status = ""
		}

		var err error
		if opts.Tasks.Target != "" {
			t, err := c.FindTarget(opts.Tasks.Target, !opts.Exact)
			bail(err)
			opts.Tasks.Target = t.UUID
		}

		var timeBefore int64
		if opts.Tasks.Before != "" {
			timeBefore = strptime(opts.Tasks.Before)
		} else {
			timeBefore = time.Now().Unix()
		}

		filter := &shield.TaskFilter{
			Status: opts.Tasks.Status,
			Limit:  &opts.Tasks.Limit,
			Target: opts.Tasks.Target,
			Type:   opts.Tasks.Type,
			Before: timeBefore,
		}

		if opts.Tasks.Active || opts.Tasks.Inactive {
			filter.Active = &opts.Tasks.Active
		}

		tasks, err := c.ListTasks(filter)
		bail(err)

		if opts.Tasks.Limit > 30 {
			for i := 0; i < opts.Tasks.Limit/30; i++ {
				filter := &shield.TaskFilter{
					Status: opts.Tasks.Status,
					Limit:  &opts.Tasks.Limit,
					Target: opts.Tasks.Target,
					Type:   opts.Tasks.Type,
					Before: timeBefore,
				}

				if opts.Tasks.Active || opts.Tasks.Inactive {
					filter.Active = &opts.Tasks.Active
				}

				sometasks, err := c.ListTasks(filter)
				bail(err)
				//current request smaller than API limit means we're EoDB
				if len(sometasks) < 30 {
					break
				}

				tasks = append(tasks, sometasks...)
				timeBefore = tasks[len(tasks)-1].RequestedAt
				time.Sleep(100 * time.Millisecond)
			}
		}

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(tasks))
			break
		}

		tbl := table.NewTable("UUID", "Type", "Status", "Owner", "Requested at", "Started at", "Completed at")
		for _, task := range tasks {
			started := "(pending)"
			stopped := "(not yet started)"
			if task.StartedAt != 0 {
				stopped = "(running)"
				started = strftime(task.StartedAt)
			}
			if task.StoppedAt != 0 {
				stopped = strftime(task.StoppedAt)
			}
			tbl.Row(task, uuid8full(task.UUID, opts.Long), task.Type, task.Status, task.Owner, strftime(task.RequestedAt), started, stopped)
		}
		tbl.Output(os.Stdout)

	/* }}} */
	case "task": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s UUID\n", command)
		}

		task, err := c.FindTask(args[0], !opts.Exact)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(task))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", task.UUID)
		r.Add("Owner", task.Owner)
		r.Add("Type", task.Type)
		r.Add("Status", task.Status)
		r.Break()

		started := "(pending)"
		stopped := "(not yet started)"
		if task.StartedAt != 0 {
			stopped = "(running)"
			started = strftime(task.StartedAt)
		}
		if task.StoppedAt != 0 {
			stopped = strftime(task.StoppedAt)
		}
		r.Add("Started at", started)
		r.Add("Stopped at", stopped)
		r.Break()

		if job, err := c.GetJob(task.JobUUID); err == nil && job != nil {
			r.Add("Job", fmt.Sprintf("%s (%s)", job.Name, task.JobUUID))
		}
		if task.ArchiveUUID != "" {
			r.Add("Archive UUID", task.ArchiveUUID)
		}
		r.Break()

		r.Add("Log", task.Log)
		r.Output(os.Stdout)

	/* }}} */
	case "cancel": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s [OPTIONS] UUID\n", command)
		}

		task, err := c.FindTask(args[0], !opts.Exact)
		bail(err)

		r := tui.NewReport()
		r.Add("Owner", task.Owner)
		r.Add("Type", task.Type)
		r.Add("Status", task.Status)
		r.Break()

		started := "(pending)"
		stopped := "(not yet started)"
		if task.StartedAt != 0 {
			stopped = "(running)"
			started = strftime(task.StartedAt)
		}
		if task.StoppedAt != 0 {
			stopped = strftime(task.StoppedAt)
		}
		r.Add("Started at", started)
		r.Add("Stopped at", stopped)
		r.Break()

		if job, err := c.GetJob(task.JobUUID); err == nil {
			r.Add("Job", fmt.Sprintf("%s (%s)", job.Name, task.JobUUID))
		}
		if task.ArchiveUUID != "" {
			r.Add("Archive UUID", task.ArchiveUUID)
		}
		r.Output(os.Stdout)

		if task.StoppedAt != 0 {
			fail(1, "This task cannot be cancelled, as it has already completed.\n")
		}
		if !confirm(opts.Yes, "Cancel this task?") {
			break
		}
		rs, err := c.CancelTask(task)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(rs))
			break
		}
		fmt.Printf("%s\n", rs.OK)

		/* }}} */

	case "users": /* {{{ */
		required(len(args) <= 1, "Too many arguments.")

		filter := &shield.UserFilter{
			Fuzzy:   !opts.Exact,
			SysRole: opts.Users.WithSystemRole,
		}
		if len(args) == 1 {
			filter.Account = args[0]
		}

		users, err := c.ListUsers(filter)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(users))
			break
		}

		tbl := table.NewTable("UUID", "Name", "Account", "System Role")
		for _, user := range users {
			tbl.Row(user, uuid8full(user.UUID, opts.Long), user.Name, user.Account, user.SysRole)
		}
		tbl.Output(os.Stdout)

	/* }}} */
	case "user": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s NAME-or-UUID\n", command)
		}

		user, err := c.FindUser(args[0], !opts.Exact)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(user))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", user.UUID)
		r.Add("Name", user.Name)
		r.Add("Account", user.Account)
		r.Add("System Role", user.SysRole)
		r.Output(os.Stdout)

	/* }}} */
	case "create-user": /* {{{ */
		if !opts.Batch {
			if opts.CreateUser.Name == "" {
				opts.CreateUser.Name = prompt("@C{Display Name}: ")
			}
			if opts.CreateUser.Account == "" {
				opts.CreateUser.Account = prompt("@C{Username}: ")
			}
			for opts.CreateUser.Password == "" {
				a := secureprompt("@Y{Choose a password}: ")
				b := secureprompt("@Y{Confirm password}: ")
				if a == "" {
					fmt.Fprintf(os.Stderr, "@R{password cannot be blank}\n")
				} else if a != b {
					fmt.Fprintf(os.Stderr, "@R{passwords do not match}\n")
				} else {
					opts.CreateUser.Password = a
					break
				}
			}
		}

		user, err := c.CreateUser(&shield.User{
			Name:     opts.CreateUser.Name,
			Account:  opts.CreateUser.Account,
			Password: opts.CreateUser.Password,
			SysRole:  opts.CreateUser.SysRole,
		})
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(user))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", user.UUID)
		r.Add("Name", user.Name)
		r.Add("Account", user.Account)
		r.Add("System Role", user.SysRole)
		r.Output(os.Stdout)

	/* }}} */
	case "update-user": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s [OPTIONS] NAME-or-UUID\n", command)
		}
		user, err := c.FindUser(args[0], !opts.Exact)
		bail(err)

		if opts.UpdateUser.Name != "" {
			user.Name = opts.UpdateUser.Name
		}
		if opts.UpdateUser.Password != "" {
			user.Password = opts.UpdateUser.Password
		}
		if opts.UpdateUser.SysRole != "" {
			user.SysRole = opts.UpdateUser.SysRole
		}

		_, err = c.UpdateUser(user)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(user))
			break
		}

		/* FIXME: api doesn't return the user object... */
		r := tui.NewReport()
		r.Add("UUID", user.UUID)
		r.Add("Name", user.Name)
		r.Add("Account", user.Account)
		r.Add("System Role", user.SysRole)
		r.Output(os.Stdout)

	/* }}} */
	case "delete-user": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s [OPTIONS] NAME-or-UUID\n", command)
		}

		user, err := c.FindUser(args[0], true)
		bail(err)

		if !confirm(opts.Yes, "Delete user @Y{%s}@local?", user.Name) {
			break
		}
		r, err := c.DeleteUser(user)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(r))
			break
		}
		fmt.Printf("%s\n", r.OK)

	/* }}} */
	case "passwd": /* {{{ */
		if opts.Batch {
			bail(fmt.Errorf("Password changes cannot be done in batch mode."))
		}

		old := secureprompt("@Y{Current pasword}: ")
		a := secureprompt("@C{Pick a new password}: ")
		b := secureprompt("@C{Confirm new password}: ")
		fmt.Printf("old=%s; a=%s; b=%s;\n", old, a, b)
		if a == "" {
			bail(fmt.Errorf("passwords cannot be blank"))
		}
		if a != b {
			bail(fmt.Errorf("passwords do not match"))
		}

		r, err := c.ChangePassword(old, a)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(r))
			break
		}
		fmt.Printf("%s\n", r.OK)

	/* }}} */

	case "sessions": /* {{{ */
		required(len(args) <= 1, "Too many arguments.")

		filter := &shield.SessionFilter{
			Limit: opts.Sessions.Limit,
		}
		if len(args) == 1 {
			filter.IP = args[0]
		}

		sessions, err := c.ListSessions(filter)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(sessions))
			break
		}

		tbl := table.NewTable("UUID", "Account", "Created At", "Last Seen", "IP Address", "User Agent")
		for _, session := range sessions {
			tbl.Row(session, uuid8full(session.UUID, opts.Long), session.UserAccount, strftime(session.CreatedAt), strftimenil(session.LastSeen, "(never)"), session.IP, session.UserAgent)
		}
		tbl.Output(os.Stdout)

	/* }}} */
	case "session": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s UUID\n", command)
		}

		session, err := c.GetSession(args[0])
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(session))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", session.UUID)
		r.Add("Account", session.UserAccount)
		r.Add("Created At", strftime(session.CreatedAt))
		r.Add("Last Seen", strftimenil(session.LastSeen, "(never)"))
		r.Add("IP Address", session.IP)
		r.Add("User Agent", session.UserAgent)
		r.Output(os.Stdout)

	/* }}} */
	case "delete-session": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s UUID\n", command)
		}

		session, err := c.GetSession(args[0])
		bail(err)

		if !confirm(opts.Yes, "Delete session for user @Y{%s}?", session.UserAccount) {
			break
		}

		if session.CurrentSession {
			if !confirm(opts.Yes, "This is your current session, are you really sure you want to delete it? You will have to reauthenticate.") {
				break
			}
		}
		r, err := c.DeleteSession(session)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(r))
			break
		}
		fmt.Printf("%s\n", r.OK)

	/* }}} */

	case "agents": /* {{{ */
		required(len(args) <= 1, "Too many arguments.")
		required(!(opts.Agents.Visible && opts.Agents.Hidden),
			"The --visible and --hidden options are mutually exclusive.")

		filter := &shield.AgentFilter{}
		if opts.Agents.Visible || opts.Agents.Hidden {
			filter.Hidden = &opts.Agents.Hidden
		}

		agents, err := c.ListAgents(filter)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(agents))
			break
		}

		tbl := table.NewTable("UUID", "Name", "Version", "Address", "Status", "Last Seen", "Last Checked", "Problems")
		for _, agent := range agents {
			st := agent.Status
			if agent.Hidden {
				st += " (hidden)"
			}
			tbl.Row(agent, uuid8full(agent.UUID, opts.Long), agent.Name, agent.Version, agent.Address, st, strftime(agent.LastSeenAt), strftimenil(agent.LastCheckedAt, "(never)"), len(agent.Problems))
		}
		tbl.Output(os.Stdout)

	/* }}} */
	case "agent": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s UUID\n", command)
		}

		agent, err := c.FindAgent(args[0], !opts.Exact)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(agent))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", agent.UUID)
		r.Add("Name", agent.Name)
		r.Add("Version", agent.Version)
		r.Add("Address", agent.Address)
		r.Add("Status", agent.Status)
		r.Add("Last Seen", strftime(agent.LastSeenAt))
		r.Add("Last Checked", strftimenil(agent.LastCheckedAt, "(never)"))

		if opts.Agent.Metadata {
			r.Break()

			b, err := json.MarshalIndent(agent.Metadata, "", "  ")
			if err != nil {
				r.Add("Metadata", fmt.Sprintf("<error: %s>"))
			} else {
				r.Add("Metadata", string(b))
			}
		}

		r.Break()
		if agent.LastError != "" {
			r.Add("Last Error", agent.LastError)
		}
		r.Add("Problems", wrap(strings.Join(agent.Problems, "\n\n"), 70))
		r.Output(os.Stdout)

		if opts.Agent.Plugins {
			plugins, err := shield.ParseAgentMetadata(agent.Metadata)
			bail(err)

			tbl := table.NewTable("Type", "Plugin", "Version", "Name", "Author(s)")
			for _, p := range plugins {
				t := "-"
				if p.CanStore && p.CanTarget {
					t = "store / target"
				} else if p.CanStore {
					t = "store"
				} else if p.CanTarget {
					t = "target"
				}
				tbl.Row(agent, t, p.ID, p.Version, p.Name, p.Author)
			}
			fmt.Printf("\n")
			tbl.Output(os.Stdout)
			fmt.Printf("\n")
		}

	/* }}} */
	case "hide-agent": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s UUID\n", command)
		}

		agent, err := c.FindAgent(args[0], !opts.Exact)
		bail(err)

		r, err := c.HideAgent(agent)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(r))
			break
		}
		fmt.Printf("%s\n", r.OK)

	/* }}} */
	case "show-agent": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s UUID\n", command)
		}

		agent, err := c.FindAgent(args[0], !opts.Exact)
		bail(err)

		r, err := c.ShowAgent(agent)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(r))
			break
		}
		fmt.Printf("%s\n", r.OK)

	/* }}} */
	case "delete-agent": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s UUID\n", command)
		}

		agent, err := c.FindAgent(args[0], !opts.Exact)
		bail(err)

		if !confirm(opts.Yes, "Delete agent @Y{%s} at @Y{%s}?", agent.Name, agent.Address) {
			break
		}
		r, err := c.DeleteAgent(agent)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(r))
			break
		}
		fmt.Printf("%s\n", r.OK)

	/* }}} */

	case "fixups": /* {{{ */
		required(len(args) == 0, "Too many arguments.")

		fixups, err := c.ListFixups(nil)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(fixups))
			break
		}

		tbl := table.NewTable("ID", "Name", "Created at", "Applied at")
		for _, fixup := range fixups {
			tbl.Row(fixup, fixup.ID, fixup.Name, strftime(fixup.CreatedAt), strftimenil(fixup.AppliedAt, "(never)"))
		}
		tbl.Output(os.Stdout)

	/* }}} */
	case "fixup": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s ID\n", command)
		}

		fixup, err := c.GetFixup(args[0])
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(fixup))
			break
		}

		r := tui.NewReport()
		r.Add("ID", fixup.ID)
		r.Add("Name", fixup.Name)
		r.Add("Created at", strftime(fixup.CreatedAt))
		r.Add("Applied at", strftimenil(fixup.AppliedAt, "(never)"))
		r.Add("Summary", wrap(fixup.Summary, 65))
		r.Output(os.Stdout)

	/* }}} */
	case "apply-fixup": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: shield %s ID\n", command)
		}

		r, err := c.ApplyFixup(args[0])
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(r))
			break
		}
		fmt.Printf("%s\n", r.OK)

	/* }}} */

	default:
		bail(fmt.Errorf("Unrecognized command '%s'", command))
	}
}
