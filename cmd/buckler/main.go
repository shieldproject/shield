package main

import (
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	fmt "github.com/jhunt/go-ansi"
	"github.com/jhunt/go-cli"
	env "github.com/jhunt/go-envirotron"
	"gopkg.in/yaml.v2"

	"github.com/starkandwayne/shield/client/v2/shield"
	"github.com/starkandwayne/shield/tui"
)

var opts struct {
	Help  bool `cli:"-h, --help"`
	Quiet bool `cli:"-q, --quiet"`
	Yes   bool `cli:"-y, --yes"`
	Debug bool `cli:"-D, --debug"  env:"SHIELD_DEBUG"`
	Trace bool `cli:"-T, --trace"  env:"SHIELD_TRACE"`
	Batch bool `cli:"-b, --batch, --no-batch" env:"SHIELD_BATCH_MODE"`

	Version bool `cli:"-v, --version"`

	Core   string `cli:"-c, --core" env:"SHIELD_CORE"`
	Config string `cli:"--config" env:"SHIELD_CLI_CONFIG"`
	JSON   bool   `cli:"--json" env:"SHIELD_JSON_MODE"`

	Exact  bool   `cli:"--exact"`
	Fuzzy  bool   `cli:"--fuzzy"`
	Tenant string `cli:"-t, --tenant" env:"SHIELD_TENANT"`

	HelpCommand struct{} `cli:"help"`

	Commands struct {
		Full bool `cli:"--full"`
		List bool `cli:"--list"`
	} `cli:"commands"`

	Curl struct{} `cli:"curl"`

	Status struct {
		Global bool `cli:"--global"`
	} `cli:"status"`

	Import struct{} `cli:"import"`

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
	Unlock struct {
		Master string `cli:"--master" env:"SHIELD_CORE_MASTER"`
	} `cli:"unlock"`
	Rekey struct {
		OldMaster string `cli:"--old-master"`
		NewMaster string `cli:"--new-master"`
	} `cli:"rekey"`

	/* }}} */
	/* AUTH TOKENS {{{ */
	AuthTokens      struct{} `cli:"auth-tokens"`
	CreateAuthToken struct{} `cli:"create-auth-token"`
	RevokeAuthToken struct{} `cli:"revoke-auth-token"`

	/* }}} */
	/* TENANTS {{{ */
	Tenants    struct{} `cli:"tenants"`
	ShowTenant struct {
		Members bool `cli:"--members"`
	} `cli:"tenant"`
	CreateTenant struct {
		Name string `cli:"-n, --name"`
	} `cli:"create-tenant"`
	UpdateTenant struct {
		Name string `cli:"-n, --name"`
	} `cli:"update-tenant"`

	/* }}} */
	/* MEMBERSHIP {{{ */
	Banish struct{} `cli:"banish"`
	Invite struct {
		Role string `cli:"-r, --role"`
	} `cli:"invite"`
	/* FIXME: delete-tenant */

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
	/* STORES {{{ */
	Stores struct {
		Used       bool   `cli:"--used"`
		Unused     bool   `cli:"--unused"`
		WithPlugin string `cli:"--with-plugin"`
	} `cli:"stores"`
	Store       struct{} `cli:"store"`
	DeleteStore struct{} `cli:"delete-store"`
	CreateStore struct {
		Name      string   `cli:"-n, --name"`
		Summary   string   `cli:"-s, --summary"`
		Agent     string   `cli:"-a, --agent"`
		Plugin    string   `cli:"-p, --plugin"`
		Threshold string   `cli:"--threshold"`
		Data      []string `cli:"-d, --data"`
	} `cli:"create-store"`
	UpdateStore struct {
		Name      string   `cli:"-n, --name"`
		Summary   string   `cli:"-s, --summary"`
		Agent     string   `cli:"-a, --agent"`
		Plugin    string   `cli:"-p, --plugin"`
		Threshold string   `cli:"--threshold"`
		ClearData bool     `cli:"--clear-data"`
		Data      []string `cli:"-d, --data"`
	} `cli:"update-store"`

	/* }}} */
	/* GLOBAL STORES {{{ */
	GlobalStores struct {
		Used       bool   `cli:"--used"`
		Unused     bool   `cli:"--unused"`
		WithPlugin string `cli:"--with-plugin"`
	} `cli:"global-stores"`
	GlobalStore       struct{} `cli:"global-store"`
	DeleteGlobalStore struct{} `cli:"delete-global-store"`
	CreateGlobalStore struct {
		Name      string   `cli:"-n, --name"`
		Summary   string   `cli:"-s, --summary"`
		Agent     string   `cli:"-a, --agent"`
		Plugin    string   `cli:"-p, --plugin"`
		Threshold string   `cli:"--threshold"`
		Data      []string `cli:"-d, --data"`
	} `cli:"create-global-store"`
	UpdateGlobalStore struct {
		Name      string   `cli:"-n, --name"`
		Summary   string   `cli:"-s, --summary"`
		Agent     string   `cli:"-a, --agent"`
		Plugin    string   `cli:"-p, --plugin"`
		ClearData bool     `cli:"--clear-data"`
		Threshold string   `cli:"--threshold"`
		Data      []string `cli:"-d, --data"`
	} `cli:"update-global-store"`

	/* }}} */
	/* POLICIES {{{ */
	Policies struct {
		Used   bool `cli:"--used"`
		Unused bool `cli:"--unused"`
	} `cli:"policies"`
	Policy       struct{} `cli:"policy"`
	DeletePolicy struct{} `cli:"delete-policy"`
	CreatePolicy struct {
		Name    string `cli:"-n, --name"`
		Summary string `cli:"-s, --summary"`
		Days    int    `cli:"-d, --days"`
	} `cli:"create-policy"`
	UpdatePolicy struct {
		Name    string `cli:"-n, --name"`
		Summary string `cli:"-s, --summary"`
		Days    int    `cli:"-d, --days"`
	} `cli:"update-policy"`

	/* }}} */
	/* POLICY TEMPLATES {{{ */
	PolicyTemplates      struct{} `cli:"policy-templates"`
	PolicyTemplate       struct{} `cli:"policy-template"`
	DeletePolicyTemplate struct{} `cli:"delete-policy-template"`
	CreatePolicyTemplate struct {
		Name string `cli:"-n, --name"`
		Days int    `cli:"-d, --days"`
	} `cli:"create-policy-template"`
	UpdatePolicyTemplate struct {
		Name string `cli:"-n, --name"`
		Days int    `cli:"-d, --days"`
	} `cli:"update-policy-template"`

	/* }}} */
	/* JOBS {{{ */
	Jobs struct {
		Store    string `cli:"--store"`
		Target   string `cli:"--target"`
		Policy   string `cli:"--policy"`
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
		Store    string `cli:"--store"`
		Policy   string `cli:"--policy"`
		Schedule string `cli:"--schedule"`
		Paused   bool   `cli:"--paused"`
	} `cli:"create-job"`
	UpdateJob struct {
		Name     string `cli:"-n, --name"`
		Summary  string `cli:"-s, --summary"`
		Target   string `cli:"--target"`
		Store    string `cli:"--store"`
		Policy   string `cli:"--policy"`
		Schedule string `cli:"--schedule"`
	} `cli:"update-job"`

	/* }}} */
	/* ARCHIVES {{{ */
	Archives       struct{} `cli:"archives"`
	Archive        struct{} `cli:"archive"`
	RestoreArchive struct {
		Target string `cli:"--target, --to"`
	} `cli:"restore-archive"`
	PurgeArchive struct{} `cli:"purge-archive"`

	/* }}} */
	/* TASKS {{{ */
	Tasks struct {
		Status   string `cli:"-s, --status"`
		Active   bool   `cli:"--active"`
		Inactive bool   `cli:"--inactive"`
		Target   string `cli:"--target"`
		All      bool   `cli:"-a, --all"`
		Limit    int    `cli:"-l, --limit"`
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
		fmt.Printf("USAGE: @G{buckler} COMMAND [OPTIONS] [ARGUMENTS]\n")
		fmt.Printf("\n")
		fmt.Printf("@B{Global options:}\n")
		fmt.Printf("  -h, --help     Show this help screen.\n")
		fmt.Printf("  -v, --version  Print the version and exit.\n")
		fmt.Printf("\n")
		fmt.Printf("      --config   An alternate client configuration file to use. (@W{$SHIELD_CLI_CONFIG})\n")
		fmt.Printf("  -c, --core     Which SHIELD core to communicate with. (@W{$SHIELD_CORE})\n")
		fmt.Printf("  -t, --tenant   Which SHIELD tenant to operate within. (@W{$SHIELD_TENANT})\n")
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
		fmt.Printf("  For example, the $SHIELD_CORE environment variable causes buckler to\n")
		fmt.Printf("  behave as if the user called `buckler --core \"$SHIELD_CORE\"`.\n")
		fmt.Printf("\n")
		fmt.Printf("  Here are the environment variable / command-line flag correlations:\n")
		fmt.Printf("\n")
		fmt.Printf("    SHIELD_CLI_CONFIG=@C{/path/to/.shield}      --config @C{/path/to/.shield}\n")
		fmt.Printf("    SHIELD_CORE=@C{prod-shield}                 --core @C{prod-shield}\n")
		fmt.Printf("    SHIELD_TENANT=@C{infrastructure}            --tenant @C{infrastructure}\n")
		fmt.Printf("    SHIELD_BATCH_MODE=@M{1}                     --batch\n")
		fmt.Printf("    SHIELD_JSON_MODE=@M{y}                      --json\n")
		fmt.Printf("    SHIELD_DEBUG=@M{1} SHIELD_TRACE=@M{yes}         --debug --trace\n")
		fmt.Printf("\n")
		fmt.Printf("For a list of common buckler commands, try `buckler commands`\n")
		fmt.Printf("\n")
		fmt.Printf("\n")
		os.Exit(0)
	}

	if opts.Version {
		fmt.Printf("buckler v¯\\_(ツ)_/¯\n")
		os.Exit(0)
	}

	if command == "help" {
		command = args[0]
		args = args[1:]
		opts.Help = true
	}

	if command == "commands" { /* {{{ */
		if opts.Help {
			fmt.Printf("USAGE: @G{buckler} @C{commands} [--list] [group [group ...]]\n")
			fmt.Printf("\n")
			fmt.Printf("  Summarizes the things that buckler can do.\n")
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
			fmt.Printf("    @C{tenants}    Tenant (and membership) management.\n")
			fmt.Printf("    @C{targets}    Target Data System management.\n")
			fmt.Printf("    @C{storage}    Cloud Storage management.\n")
			fmt.Printf("    @C{policies}   Retenion Policy management.\n")
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
			printc("  status                   Show the status of the targeted SHIELD Core.\n")
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
			printc("  unlock                   Unlock a SHIELD Core (i.e. after a reboot).\n")
			printc("  rekey                    Change a SHIELD Core master (unlock) password.\n")
			blank()
			printc("  global-stores            List shared cloud storage systems.\n")
			printc("  global-store             Display details for a single shared cloud storage system.\n")
			printc("  create-global-store      Configure a new shared cloud storage system.\n")
			printc("  update-global-store      Reconfigure a shared cloud storage system.\n")
			printc("  delete-global-store      Decomission an unused shared cloud storage system.\n")
			blank()
			printc("  policy-templates         List retention policy templates.\n")
			printc("  policy-template          Display details for a single retention policy template.\n")
			printc("  create-policy-template   Configure a new retention policy template.\n")
			printc("  update-policy-template   Reconfigure a retention policy template.\n")
			printc("  delete-policy-template   Decomission an unused retention policy template.\n")
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
		if show("tenant", "tenants") {
			header("Tenant Management")
			printc("  tenants                  List all SHIELD Tenants.\n")
			printc("  tenant                   Display the details for a single SHIELD Tenant.\n")
			printc("  create-tenant            Create a new SHIELD Tenant.\n")
			printc("  update-tenant            Update the metadata for a single tenant.\n")
			blank()
			printc("  invite                   Invite a local user to a SHIELD Tenant.\n")
			printc("  banish                   Remove a local user from a SHIELD Tenant.\n")
		}
		if show("target", "targets") {
			header("Target Data Systems")
			printc("  targets                  List all target data systems.\n")
			printc("  target                   Display the details for a single target data system.\n")
			printc("  create-target            Configure a new target data system.\n")
			printc("  update-target            Reconfigure a target data system.\n")
			printc("  delete-target            Decomission an unused target data system.\n")
		}
		if show("store", "stores", "storage") {
			header("Cloud Storage Systems")
			printc("  stores                   List all cloud storage systems.\n")
			printc("  store                    Display the details for a single cloud storage system.\n")
			printc("  create-store             Configure a new cloud storage system.\n")
			printc("  update-store             Reconfigure a cloud storage system.\n")
			printc("  delete-store             Decomission an unused cloud storage system.\n")
		}
		if show("retention", "policies", "policy") {
			header("Retention Policies")
			printc("  policies                 List all retention policies.\n")
			printc("  policy                   Display the details for a single retention policy.\n")
			printc("  create-policy            Configure a new retention policy.\n")
			printc("  update-policy            Reconfigure a retention policy.\n")
			printc("  delete-policy            Decomission an unused retention policy.\n")
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
		tbl := tui.NewTable("Name", "URL", "Verify TLS?")
		/* FIXME need stable sort */
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
			fail(2, "Usage: buckler %s URL ALIAS\n", command)
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

				     buckler api test https://shield.example.com \
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
			URL:           url,
			Debug:         opts.Debug,
			Trace:         opts.Trace,
			Session:       "",
			CACertificate: cacert,
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
	}

	if opts.Core == "" {
		bail(fmt.Errorf("Missing required --core option (and no SHIELD_CORE environment variable was set)."))
	}
	bail(config.Select(opts.Core))

	c := &shield.Client{
		URL:           config.Current.URL,
		Debug:         opts.Debug,
		Trace:         opts.Trace,
		Session:       config.Current.Session,
		CACertificate: config.Current.CACertificate,
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

			tbl := tui.NewTable("Name", "Description", "Type")
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
		fmt.Printf("\n")

		if len(id.Tenants) == 0 {
			fmt.Printf("@Y{you are not assigned to any tenants}\n")
		} else {
			tbl := tui.NewTable("UUID", "Name", "Role")
			for _, tenant := range id.Tenants {
				tbl.Row(tenant, tenant.UUID, tenant.Name, tenant.Role)
			}
			fmt.Printf("@G{Tenants}\n")
			tbl.Output(os.Stdout)
		}

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
		err := c.Initialize(opts.Init.Master)
		bail(err)

		fmt.Printf("SHIELD core unlocked successfully.\n")

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
		err := c.Rekey(opts.Rekey.OldMaster, opts.Rekey.NewMaster)
		bail(err)

		fmt.Printf("SHIELD core rekeyed successfully.\n")

	/* }}} */

	case "curl": /* {{{ */
		if len(args) < 1 || len(args) > 3 {
			fail(2, "Usage: buckler %s [METHOD] RELATIVE-URL [BODY]\n", command)
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
	case "status": /* {{{ */
		info, err := c.Info()
		bail(err)

		var status *shield.Status
		if opts.Status.Global || opts.Tenant == "" {
			status, err = c.GlobalStatus()
			bail(err)

		} else {
			required(opts.Tenant != "", "Missing required --tenant option.")
			tenant, err := c.FindMyTenant(opts.Tenant, true)
			bail(err)

			status, err = c.TenantStatus(tenant)
			bail(err)
		}

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(status))
			break
		}

		if info.Version == "" {
			fmt.Printf("@W{SHIELD %s} @R{(development)}\n", info.Env)
		} else {
			fmt.Printf("@W{SHIELD %s} v@G{%s}\n", info.Env, info.Version)
		}
		fmt.Printf("API Version @G{%d}\n", info.API)
		if info.MOTD != "" {
			fmt.Printf("\n---[ MOTD ]-------------------------------------\n")
			fmt.Printf("%s", info.MOTD)
			fmt.Printf("\n------------------------------------------------\n\n")
		}

	/* }}} */

	case "auth-tokens": /* {{{ */
		tokens, err := c.ListAuthTokens()
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(tokens))
			break
		}

		tbl := tui.NewTable("Name", "Created at", "Last seen")
		for _, token := range tokens {
			tbl.Row(token, token.Name, strftime(token.CreatedAt), strftimenil(token.LastSeen, "(never)"))
		}
		tbl.Output(os.Stdout)

	/* }}} */
	case "create-auth-token": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s TOKEN-NAME\n", command)
		}

		t, err := c.CreateAuthToken(&shield.AuthToken{Name: args[0]})
		bail(err)
		fmt.Printf("@C{%s}\n", t.Session)

	/* }}} */
	case "revoke-auth-token": /* {{{ */
		if len(args) == 0 {
			fail(2, "Usage: buckler %s TOKEN-NAME [OTHER-TOKEN ...]\n", command)
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

	case "import": /* {{{ */
		if len(args) < 1 {
			fail(2, "Usage: buckler %s /path/to/manifest.yml ...\n", command)
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

			err = m.Deploy(c)
			bailon(file, err)
		}
	/* }}} */

	case "tenants": /* {{{ */
		required(len(args) <= 1, "Too many arguments.")
		filter := &shield.TenantFilter{
			Fuzzy: !opts.Exact,
		}
		if len(args) == 1 {
			filter.Name = args[0]
		}

		tenants, err := c.ListTenants(filter)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(tenants))
			break
		}

		tbl := tui.NewTable("UUID", "Name")
		for _, tenant := range tenants {
			tbl.Row(tenant, tenant.UUID, tenant.Name)
		}
		tbl.Output(os.Stdout)

	/* }}} */
	case "tenant": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s NAME-or-UUID\n", command)
		}

		tenant, err := c.FindTenant(args[0], !opts.Exact)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(tenant))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", tenant.UUID)
		r.Add("Name", tenant.Name)
		r.Output(os.Stdout)

		if opts.ShowTenant.Members {
			fmt.Printf("\n")
			t := tui.NewTable("UUID", "Name", "Account", "Role")
			for _, mem := range tenant.Members {
				t.Row(mem, mem.UUID, mem.Name, fmt.Sprintf("%s@%s", mem.Account, mem.Backend), mem.Role)
			}
			t.Output(os.Stdout)
		}

	/* }}} */
	case "create-tenant": /* {{{ */
		if !opts.Batch {
			if opts.CreateTenant.Name == "" {
				opts.CreateTenant.Name = prompt("@C{Tenant Name}: ")
			}
		}

		t, err := c.CreateTenant(&shield.Tenant{
			Name: opts.CreateTenant.Name,
		})
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(t))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", t.UUID)
		r.Add("Name", t.Name)
		r.Output(os.Stdout)

	/* }}} */
	case "update-tenant": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s [OPTIONS] NAME-or-UUID\n", command)
		}
		t, err := c.FindTenant(args[0], true)
		bail(err)

		if opts.UpdateTenant.Name != "" {
			t.Name = opts.UpdateTenant.Name
		}

		t, err = c.UpdateTenant(t)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(t))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", t.UUID)
		r.Add("Name", t.Name)
		r.Output(os.Stdout)

	/* }}} */
	case "invite": /* {{{ */
		if len(args) < 1 {
			fail(2, "Usage: buckler %s -r ROLE USER [USER ...]\n", command)
		}
		tenant, err := c.FindTenant(opts.Tenant, true)
		bail(err)

		switch opts.Invite.Role {
		case "":
			opts.Invite.Role = "operator"
		case "operator", "engineer", "admin":
		default:
			bail(fmt.Errorf("Invalid --role value '%s' (must be one of operator, engineer, or admin)", opts.Invite.Role))
		}

		users := make([]*shield.User, len(args))
		for i := range args {
			user, err := c.FindUser(args[i], !opts.Exact)
			bail(err)

			users[i] = user
		}

		r, err := c.Invite(tenant, opts.Invite.Role, users)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(r))
			break
		}
		fmt.Printf("%s\n", r.OK)

		/* }}} */
	case "banish": /* {{{ */
		if len(args) < 1 {
			fail(2, "Usage: buckler %s USER [USER ...]\n", command)
		}
		tenant, err := c.FindTenant(opts.Tenant, true)
		bail(err)

		users := make([]*shield.User, len(args))
		for i := range args {
			user, err := c.FindUser(args[i], !opts.Exact)
			bail(err)

			users[i] = user
		}

		r, err := c.Banish(tenant, users)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(r))
			break
		}
		fmt.Printf("%s\n", r.OK)

		/* }}} */

	case "targets": /* {{{ */
		required(opts.Tenant != "", "Missing required --tenant option.")
		required(!(opts.Targets.Used && opts.Targets.Unused),
			"The --used and --unused options are mutually exclusive.")
		required(len(args) <= 1, "Too many arguments.")

		filter := &shield.TargetFilter{
			Plugin: opts.Targets.WithPlugin,
			Fuzzy:  !opts.Exact,
		}
		if len(args) == 1 {
			filter.Name = args[0]
		}
		if opts.Targets.Used || opts.Targets.Unused {
			x := opts.Targets.Used
			filter.Used = &x
		}

		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		targets, err := c.ListTargets(tenant, filter)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(targets))
			break
		}

		tbl := tui.NewTable("UUID", "Name", "Summary", "Plugin", "SHIELD Agent", "Configuration")
		for _, target := range targets {
			tbl.Row(target, target.UUID, target.Name, target.Summary, target.Plugin, target.Agent, asJSON(target.Config))
		}
		tbl.Output(os.Stdout)

	/* }}} */
	case "target": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s NAME-or-UUID\n", command)
		}

		required(opts.Tenant != "", "Missing required --tenant option.")
		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		t, err := c.FindTarget(tenant, args[0], !opts.Exact)
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
	case "create-target": /* {{{ */
		required(opts.Tenant != "", "Missing required --tenant option.")
		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

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

		t, err := c.CreateTarget(tenant, &shield.Target{
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
			fail(2, "Usage: buckler %s -t TENANT [OPTIONS] NAME-or-UUID\n", command)
		}
		required(opts.Tenant != "", "Missing required --tenant option.")
		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		t, err := c.FindTarget(tenant, args[0], true)
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
		if opts.UpdateTarget.ClearData {
			t.Config = conf
		} else {
			for k, v := range conf {
				t.Config[k] = v
			}
		}

		t, err = c.UpdateTarget(tenant, t)
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
	case "delete-target": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s -t TENANT [OPTIONS] NAME-or-UUID\n", command)
		}
		required(opts.Tenant != "", "Missing required --tenant option.")
		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		t, err := c.FindTarget(tenant, args[0], true)
		bail(err)

		if !confirm(opts.Yes, "Delete target @Y{%s} in tenant @Y{%s}?", t.Name, tenant.Name) {
			break
		}
		r, err := c.DeleteTarget(tenant, t)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(r))
			break
		}
		fmt.Printf("%s\n", r.OK)

	/* }}} */

	case "stores": /* {{{ */
		required(opts.Tenant != "", "Missing required --tenant option.")
		required(!(opts.Stores.Used && opts.Stores.Unused),
			"The --used and --unused options are mutually exclusive.")
		required(len(args) <= 1, "Too many arguments.")

		filter := &shield.StoreFilter{
			Plugin: opts.Stores.WithPlugin,
			Fuzzy:  !opts.Exact,
		}
		if len(args) == 1 {
			filter.Name = args[0]
		}
		if opts.Stores.Used || opts.Stores.Unused {
			x := opts.Stores.Used
			filter.Used = &x
		}

		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		stores, err := c.ListStores(tenant, filter)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(stores))
			break
		}

		tbl := tui.NewTable("UUID", "Name", "Summary", "Plugin", "SHIELD Agent", "Configuration", "Healthy?")
		for _, store := range stores {
			health := fmt.Sprintf("@G{yes}")
			if !store.Healthy {
				health = fmt.Sprintf("@R{no}")
			}
			tbl.Row(store, store.UUID, store.Name, store.Summary, store.Plugin, store.Agent, asJSON(store.Config), health)
		}
		tbl.Output(os.Stdout)

	/* }}} */
	case "store": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s NAME-or-UUID\n", command)
		}

		required(opts.Tenant != "", "Missing required --tenant option.")
		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		store, err := c.FindStore(tenant, args[0], !opts.Exact)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(store))
			break
		}

		health := fmt.Sprintf("@G{yes}")
		if !store.Healthy {
			health = fmt.Sprintf("@R{no}")
		}

		r := tui.NewReport()
		r.Add("UUID", store.UUID)
		r.Add("Name", store.Name)
		r.Add("Healthy?", health)
		r.Add("Summary", store.Summary)
		r.Add("SHIELD Agent", store.Agent)
		r.Add("Backup Plugin", store.Plugin)
		r.Output(os.Stdout)

	/* }}} */
	case "create-store": /* {{{ */
		required(opts.Tenant != "", "Missing required --tenant option.")

		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		conf, err := dataConfig(opts.CreateStore.Data)
		bail(err)

		if !opts.Batch {
			if opts.CreateStore.Name == "" {
				opts.CreateStore.Name = prompt("@C{Store Name}: ")
			}
			if opts.CreateStore.Summary == "" {
				opts.CreateStore.Summary = prompt("@C{Description}: ")
			}
			if opts.CreateStore.Agent == "" {
				opts.CreateStore.Agent = prompt("@C{SHIELD Agent (IP:port)}: ")
			}
			if opts.CreateStore.Plugin == "" {
				opts.CreateStore.Plugin = prompt("@C{Backup Plugin}: ")
			}
			if opts.CreateStore.Threshold == "" {
				opts.CreateStore.Threshold = prompt("@C{Threshold}: ")
			}
		}

		thold, err := parseBytes(opts.CreateStore.Threshold)
		bail(err)

		store, err := c.CreateStore(tenant, &shield.Store{
			Name:      opts.CreateStore.Name,
			Summary:   opts.CreateStore.Summary,
			Agent:     opts.CreateStore.Agent,
			Plugin:    opts.CreateStore.Plugin,
			Threshold: thold,
			Config:    conf,
		})
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(store))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", store.UUID)
		r.Add("Name", store.Name)
		r.Add("Summary", store.Summary)
		r.Add("SHIELD Agent", store.Agent)
		r.Add("Backup Plugin", store.Plugin)
		r.Add("Threshold", formatBytes(store.Threshold))
		r.Output(os.Stdout)

	/* }}} */
	case "update-store": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s -t TENANT [OPTIONS] NAME-or-UUID\n", command)
		}
		required(opts.Tenant != "", "Missing required --tenant option.")

		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		store, err := c.FindStore(tenant, args[0], true)
		bail(err)

		conf, err := dataConfig(opts.UpdateStore.Data)
		bail(err)

		if opts.UpdateStore.Name != "" {
			store.Name = opts.UpdateStore.Name
		}
		if opts.UpdateStore.Summary != "" {
			store.Summary = opts.UpdateStore.Summary
		}
		if opts.UpdateStore.Agent != "" {
			store.Agent = opts.UpdateStore.Agent
		}
		if opts.UpdateStore.Plugin != "" && store.Plugin != opts.UpdateStore.Plugin {
			opts.UpdateStore.ClearData = true
			store.Plugin = opts.UpdateStore.Plugin
		}
		if opts.UpdateStore.Threshold != "" {
			thold, err := parseBytes(opts.UpdateStore.Threshold)
			fmt.Printf("threshold is '%s' -> %d\n", opts.UpdateStore.Threshold, thold)
			bail(err)
			store.Threshold = thold
		}
		if opts.UpdateStore.ClearData {
			store.Config = conf
		} else {
			for k, v := range conf {
				store.Config[k] = v
			}
		}

		store, err = c.UpdateStore(tenant, store)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(store))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", store.UUID)
		r.Add("Name", store.Name)
		r.Add("Summary", store.Summary)
		r.Add("SHIELD Agent", store.Agent)
		r.Add("Backup Plugin", store.Plugin)
		r.Add("Threshold", formatBytes(store.Threshold))
		r.Output(os.Stdout)

	/* }}} */
	case "delete-store": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s -t TENANT [OPTIONS] NAME-or-UUID\n", command)
		}

		required(opts.Tenant != "", "Missing required --tenant option.")

		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		store, err := c.FindStore(tenant, args[0], true)
		bail(err)

		if !confirm(opts.Yes, "Delete store @Y{%s} in tenant @Y{%s}?", store.Name, tenant.Name) {
			break
		}
		r, err := c.DeleteStore(tenant, store)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(r))
			break
		}
		fmt.Printf("%s\n", r.OK)

	/* }}} */

	case "global-stores": /* {{{ */
		required(!(opts.GlobalStores.Used && opts.GlobalStores.Unused),
			"The --used and --unused options are mutually exclusive.")
		required(len(args) <= 1, "Too many arguments.")

		filter := &shield.StoreFilter{
			Plugin: opts.Stores.WithPlugin,
			Fuzzy:  !opts.Exact,
		}
		if len(args) == 1 {
			filter.Name = args[0]
		}
		if opts.GlobalStores.Used || opts.GlobalStores.Unused {
			x := opts.GlobalStores.Used
			filter.Used = &x
		}

		stores, err := c.ListGlobalStores(filter)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(stores))
			break
		}

		tbl := tui.NewTable("UUID", "Name", "Summary", "Plugin", "SHIELD Agent", "Configuration", "Healthy?")
		for _, store := range stores {
			health := fmt.Sprintf("@G{yes}")
			if !store.Healthy {
				health = fmt.Sprintf("@R{no}")
			}
			tbl.Row(store, store.UUID, store.Name, store.Summary, store.Plugin, store.Agent, asJSON(store.Config), health)
		}
		tbl.Output(os.Stdout)

	/* }}} */
	case "global-store": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s NAME-or-UUID\n", command)
		}

		store, err := c.FindGlobalStore(args[0], !opts.Exact)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(store))
			break
		}

		health := fmt.Sprintf("@G{yes}")
		if !store.Healthy {
			health = fmt.Sprintf("@R{no}")
		}

		r := tui.NewReport()
		r.Add("UUID", store.UUID)
		r.Add("Name", store.Name)
		r.Add("Healthy?", health)
		r.Add("Summary", store.Summary)
		r.Add("SHIELD Agent", store.Agent)
		r.Add("Backup Plugin", store.Plugin)
		r.Output(os.Stdout)

	/* }}} */
	case "create-global-store": /* {{{ */
		conf, err := dataConfig(opts.CreateGlobalStore.Data)
		bail(err)

		if !opts.Batch {
			if opts.CreateGlobalStore.Name == "" {
				opts.CreateGlobalStore.Name = prompt("@C{Store Name}: ")
			}
			if opts.CreateGlobalStore.Summary == "" {
				opts.CreateGlobalStore.Summary = prompt("@C{Description}: ")
			}
			if opts.CreateGlobalStore.Agent == "" {
				opts.CreateGlobalStore.Agent = prompt("@C{SHIELD Agent (IP:port)}: ")
			}
			if opts.CreateGlobalStore.Plugin == "" {
				opts.CreateGlobalStore.Plugin = prompt("@C{Backup Plugin}: ")
			}
			if opts.CreateGlobalStore.Threshold == "" {
				opts.CreateGlobalStore.Threshold = prompt("@C{Threshold}: ")
			}
		}

		thold, err := parseBytes(opts.CreateStore.Threshold)
		bail(err)

		store, err := c.CreateGlobalStore(&shield.Store{
			Name:      opts.CreateGlobalStore.Name,
			Summary:   opts.CreateGlobalStore.Summary,
			Agent:     opts.CreateGlobalStore.Agent,
			Plugin:    opts.CreateGlobalStore.Plugin,
			Threshold: thold,
			Config:    conf,
		})
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(store))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", store.UUID)
		r.Add("Name", store.Name)
		r.Add("Summary", store.Summary)
		r.Add("SHIELD Agent", store.Agent)
		r.Add("Backup Plugin", store.Plugin)
		r.Add("Threshold", formatBytes(store.Threshold))
		r.Output(os.Stdout)

	/* }}} */
	case "update-global-store": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s -t TENANT [OPTIONS] NAME-or-UUID\n", command)
		}

		store, err := c.FindGlobalStore(args[0], true)
		bail(err)

		conf, err := dataConfig(opts.UpdateGlobalStore.Data)
		bail(err)

		if opts.UpdateGlobalStore.Name != "" {
			store.Name = opts.UpdateGlobalStore.Name
		}
		if opts.UpdateGlobalStore.Summary != "" {
			store.Summary = opts.UpdateGlobalStore.Summary
		}
		if opts.UpdateGlobalStore.Agent != "" {
			store.Agent = opts.UpdateGlobalStore.Agent
		}
		if opts.UpdateGlobalStore.Plugin != "" && store.Plugin != opts.UpdateGlobalStore.Plugin {
			opts.UpdateGlobalStore.ClearData = true
			store.Plugin = opts.UpdateGlobalStore.Plugin
		}
		if opts.UpdateGlobalStore.Threshold != "" {
			thold, err := parseBytes(opts.UpdateGlobalStore.Threshold)
			bail(err)
			store.Threshold = thold
		}
		if opts.UpdateGlobalStore.ClearData {
			store.Config = conf
		} else {
			for k, v := range conf {
				store.Config[k] = v
			}
		}

		store, err = c.UpdateGlobalStore(store)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(store))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", store.UUID)
		r.Add("Name", store.Name)
		r.Add("Summary", store.Summary)
		r.Add("SHIELD Agent", store.Agent)
		r.Add("Backup Plugin", store.Plugin)
		r.Add("Threshold", formatBytes(store.Threshold))
		r.Output(os.Stdout)

	/* }}} */
	case "delete-global-store": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s -t TENANT [OPTIONS] NAME-or-UUID\n", command)
		}

		store, err := c.FindGlobalStore(args[0], true)
		bail(err)

		if !confirm(opts.Yes, "Delete @R{global} store @Y{%s}?", store.Name) {
			break
		}
		r, err := c.DeleteGlobalStore(store)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(r))
			break
		}
		fmt.Printf("%s\n", r.OK)

	/* }}} */

	case "policies": /* {{{ */
		required(opts.Tenant != "", "Missing required --tenant option.")
		required(!(opts.Policies.Used && opts.Policies.Unused),
			"The --used and --unused options are mutually exclusive.")
		required(len(args) <= 1, "Too many arguments.")

		filter := &shield.PolicyFilter{
			Fuzzy: !opts.Exact,
		}
		if len(args) == 1 {
			filter.Name = args[0]
		}
		if opts.Policies.Used || opts.Policies.Unused {
			x := opts.Policies.Used
			filter.Used = &x
		}

		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		policies, err := c.ListPolicies(tenant, filter)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(policies))
			break
		}

		tbl := tui.NewTable("UUID", "Name", "Retention Period")
		for _, p := range policies {
			tbl.Row(p, p.UUID, p.Name, fmt.Sprintf("%dd", p.Expires))
		}
		tbl.Output(os.Stdout)

	/* }}} */
	case "policy": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s NAME-or-UUID\n", command)
		}

		required(opts.Tenant != "", "Missing required --tenant option.")
		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		p, err := c.FindPolicy(tenant, args[0], !opts.Exact)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(p))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", p.UUID)
		r.Add("Name", p.Name)
		r.Add("Retention Period", fmt.Sprintf("%dd", p.Expires))
		r.Output(os.Stdout)

	/* }}} */
	case "create-policy": /* {{{ */
		required(opts.Tenant != "", "Missing required --tenant option.")
		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		if !opts.Batch {
			if opts.CreatePolicy.Name == "" {
				opts.CreatePolicy.Name = prompt("@C{Policy Name}: ")
			}
			if opts.CreatePolicy.Summary == "" {
				opts.CreatePolicy.Summary = prompt("@C{Summary}: ")
			}
			if opts.CreatePolicy.Days == 0 {
				for {
					s := prompt("@C{Retention Period (days)}: ")
					if d, err := strconv.Atoi(s); err != nil && d > 0 {
						opts.CreatePolicy.Days = d
						break
					}
					fmt.Fprintf(os.Stderr, "@R{invalid expiry (must be numeric and greater than zero)}\n")
				}
			}
		}

		p, err := c.CreatePolicy(tenant, &shield.Policy{
			Name:    opts.CreatePolicy.Name,
			Summary: opts.CreatePolicy.Summary,
			Expires: opts.CreatePolicy.Days,
		})
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(p))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", p.UUID)
		r.Add("Name", p.Name)
		r.Add("Retention Period", fmt.Sprintf("%dd", p.Expires))
		r.Output(os.Stdout)

	/* }}} */
	case "update-policy": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s -t TENANT [OPTIONS] NAME-or-UUID\n", command)
		}
		required(opts.Tenant != "", "Missing required --tenant option.")
		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		p, err := c.FindPolicy(tenant, args[0], true)
		bail(err)

		if opts.UpdatePolicy.Name != "" {
			p.Name = opts.UpdatePolicy.Name
		}
		if opts.UpdatePolicy.Summary != "" {
			p.Summary = opts.UpdatePolicy.Summary
		}
		if opts.UpdatePolicy.Days != 0 {
			p.Expires = opts.UpdatePolicy.Days
		}

		p, err = c.UpdatePolicy(tenant, p)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(p))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", p.UUID)
		r.Add("Name", p.Name)
		r.Add("Retention Period", fmt.Sprintf("%dd", p.Expires))
		r.Output(os.Stdout)

	/* }}} */
	case "delete-policy": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s -t TENANT [OPTIONS] NAME-or-UUID\n", command)
		}
		required(opts.Tenant != "", "Missing required --tenant option.")
		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		policy, err := c.FindPolicy(tenant, args[0], true)
		bail(err)

		if !confirm(opts.Yes, "Delete policy @Y{%s} in tenant @Y{%s}?", policy.Name, tenant.Name) {
			break
		}
		r, err := c.DeletePolicy(tenant, policy)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(r))
			break
		}
		fmt.Printf("%s\n", r.OK)

	/* }}} */

	case "policy-templates": /* {{{ */
		templates, err := c.ListPolicyTemplates(nil)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(templates))
			break
		}

		tbl := tui.NewTable("UUID", "Name", "Retention Period")
		for _, p := range templates {
			tbl.Row(p, p.UUID, p.Name, fmt.Sprintf("%dd", p.Expires))
		}
		tbl.Output(os.Stdout)

	/* }}} */
	case "policy-template": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s NAME-or-UUID\n", command)
		}

		p, err := c.FindPolicyTemplate(args[0], !opts.Exact)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(p))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", p.UUID)
		r.Add("Name", p.Name)
		r.Add("Retention Period", fmt.Sprintf("%dd", p.Expires))
		r.Output(os.Stdout)

	/* }}} */
	case "create-policy-template": /* {{{ */
		if !opts.Batch {
			if opts.CreatePolicyTemplate.Name == "" {
				opts.CreatePolicyTemplate.Name = prompt("@C{Policy Template Name}: ")
			}
			if opts.CreatePolicyTemplate.Days == 0 {
				for {
					s := prompt("@C{Retention Period (days)}: ")
					if d, err := strconv.Atoi(s); err != nil && d > 0 {
						opts.CreatePolicyTemplate.Days = d
						break
					}
					fmt.Fprintf(os.Stderr, "@R{invalid expiry (must be numeric and greater than zero)}\n")
				}
			}
		}

		p, err := c.CreatePolicyTemplate(&shield.Policy{
			Name:    opts.CreatePolicyTemplate.Name,
			Expires: opts.CreatePolicyTemplate.Days,
		})
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(p))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", p.UUID)
		r.Add("Name", p.Name)
		r.Add("Retention Period", fmt.Sprintf("%dd", p.Expires))
		r.Output(os.Stdout)

	/* }}} */
	case "update-policy-template": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s -t TENANT [OPTIONS] NAME-or-UUID\n", command)
		}
		p, err := c.FindPolicyTemplate(args[0], true)
		bail(err)

		if opts.UpdatePolicy.Name != "" {
			p.Name = opts.UpdatePolicy.Name
		}
		if opts.UpdatePolicy.Days != 0 {
			p.Expires = opts.UpdatePolicy.Days
		}

		p, err = c.UpdatePolicyTemplate(p)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(p))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", p.UUID)
		r.Add("Name", p.Name)
		r.Add("Retention Period", fmt.Sprintf("%dd", p.Expires))
		r.Output(os.Stdout)

	/* }}} */
	case "delete-policy-template": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s -t TENANT [OPTIONS] NAME-or-UUID\n", command)
		}

		policy, err := c.FindPolicyTemplate(args[0], true)
		bail(err)

		if !confirm(opts.Yes, "Delete policy template @Y{%s}?", policy.Name) {
			break
		}
		r, err := c.DeletePolicyTemplate(policy)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(r))
			break
		}
		fmt.Printf("%s\n", r.OK)

	/* }}} */

	case "jobs": /* {{{ */
		required(opts.Tenant != "", "Missing required --tenant option.")
		required(!(opts.Jobs.Paused && opts.Jobs.Unpaused),
			"The --paused and --unpaused options are mutually exclusive.")
		required(len(args) <= 1, "Too many arguments.")

		filter := &shield.JobFilter{
			Fuzzy:  !opts.Exact,
			Store:  opts.Jobs.Store,
			Target: opts.Jobs.Target,
			Policy: opts.Jobs.Policy,
		}
		if opts.Jobs.Paused || opts.Jobs.Unpaused {
			filter.Paused = &opts.Jobs.Paused
		}
		if len(args) == 1 {
			filter.Name = args[0]
		}

		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		jobs, err := c.ListJobs(tenant, filter)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(jobs))
			break
		}

		/*
			tbl := table.New(
				opts.Output,
				&table.Map{
					"%uuid": "UUID",
					"%name": "Name",
				},
				jobs...
			)
		*/
		/* FIXME: support --long / -l and maybe --output / -o "fmt-str" */
		tbl := tui.NewTable("UUID", "Name", "Summary", "Schedule", "Status", "Policy", "SHIELD Agent", "Target", "Store")
		for _, job := range jobs {
			tbl.Row(job, job.UUID, job.Name, job.Summary, job.Schedule, job.Status(), job.Policy.Name, job.Agent, job.Target.Name, job.Store.Name)
		}
		tbl.Output(os.Stdout)

	/* }}} */
	case "job": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s NAME-or-UUID\n", command)
		}

		required(opts.Tenant != "", "Missing required --tenant option.")
		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		job, err := c.FindJob(tenant, args[0], !opts.Exact)
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
		r.Add("Policy", job.Policy.Name)
		r.Add("Expires in", fmt.Sprintf("%d days", job.Expiry))
		r.Break()

		r.Add("Data System", job.Target.Name)
		r.Add("Backup Plugin", job.Target.Plugin)
		r.Add("SHIELD Agent", job.Target.Agent)
		r.Break()

		r.Add("Cloud Storage", job.Store.Name)
		r.Add("Storage Plugin", job.Store.Plugin)
		r.Break()

		r.Add("Notes", job.Summary)

		r.Output(os.Stdout)

	/* }}} */
	case "create-job": /* {{{ */
		required(opts.Tenant != "", "Missing required --tenant option.")

		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

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
					SearchTargets(c, tenant, id[1:])
					continue
				}
				if target, err := c.FindTarget(tenant, id, !opts.Exact); err != nil {
					fmt.Fprintf(os.Stderr, "@Y{%s}\n", err)
				} else {
					opts.CreateJob.Target = target.UUID
					break
				}
			}
			for opts.CreateJob.Store == "" {
				id := prompt("@C{Cloud Storage}: ")
				if len(id) > 0 && id[0] == '?' {
					SearchStores(c, tenant, id[1:])
					continue
				}
				if store, err := c.FindUsableStore(tenant, id, !opts.Exact); err != nil {
					fmt.Fprintf(os.Stderr, "@Y{%s}\n", err)
				} else {
					opts.CreateJob.Store = store.UUID
					break
				}
			}
			for opts.CreateJob.Policy == "" {
				id := prompt("@C{Retention Policy}: ")
				if len(id) > 0 && id[0] == '?' {
					SearchPolicies(c, tenant, id[1:])
					continue
				}
				if policy, err := c.FindPolicy(tenant, id, !opts.Exact); err != nil {
					fmt.Fprintf(os.Stderr, "@Y{%s}\n", err)
				} else {
					opts.CreateJob.Policy = policy.UUID
					break
				}
			}
			if opts.CreateJob.Schedule == "" {
				opts.CreateJob.Schedule = prompt("@C{Schedule}: ")
			}
		} else {
			if id := opts.CreateJob.Target; id != "" {
				if target, err := c.FindTarget(tenant, id, !opts.Exact); err != nil {
					bail(err)
				} else {
					opts.CreateJob.Target = target.UUID
				}
			}
			if id := opts.CreateJob.Store; id != "" {
				if store, err := c.FindUsableStore(tenant, id, !opts.Exact); err != nil {
					bail(err)
				} else {
					opts.CreateJob.Store = store.UUID
				}
			}
			if id := opts.CreateJob.Policy; id != "" {
				if policy, err := c.FindPolicy(tenant, id, !opts.Exact); err != nil {
					bail(err)
				} else {
					opts.CreateJob.Policy = policy.UUID
				}
			}
		}

		job, err := c.CreateJob(tenant, &shield.Job{
			Name:       opts.CreateJob.Name,
			Summary:    opts.CreateJob.Summary,
			TargetUUID: opts.CreateJob.Target,
			StoreUUID:  opts.CreateJob.Store,
			PolicyUUID: opts.CreateJob.Policy,
			Schedule:   opts.CreateJob.Schedule,
			Paused:     opts.CreateJob.Paused,
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
			fail(2, "Usage: buckler %s -t TENANT [OPTIONS] NAME-or-UUID\n", command)
		}
		required(opts.Tenant != "", "Missing required --tenant option.")

		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		job, err := c.FindJob(tenant, args[0], !opts.Exact)
		bail(err)

		if opts.UpdateJob.Name != "" {
			job.Name = opts.UpdateJob.Name
		}
		if opts.UpdateJob.Summary != "" {
			job.Summary = opts.UpdateJob.Summary
		}
		if id := opts.UpdateJob.Target; id != "" {
			if target, err := c.FindTarget(tenant, id, !opts.Exact); err != nil {
				bail(err)
			} else {
				job.TargetUUID = target.UUID
			}
		}
		if id := opts.UpdateJob.Store; id != "" {
			if store, err := c.FindUsableStore(tenant, id, !opts.Exact); err != nil {
				bail(err)
			} else {
				job.StoreUUID = store.UUID
			}
		}
		if id := opts.UpdateJob.Policy; id != "" {
			if policy, err := c.FindPolicy(tenant, id, !opts.Exact); err != nil {
				bail(err)
			} else {
				job.PolicyUUID = policy.UUID
			}
		}
		if opts.UpdateJob.Schedule != "" {
			job.Schedule = opts.UpdateJob.Schedule
		}

		job, err = c.UpdateJob(tenant, job)
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
			fail(2, "Usage: buckler %s -t TENANT [OPTIONS] NAME-or-UUID\n", command)
		}

		required(opts.Tenant != "", "Missing required --tenant option.")

		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		job, err := c.FindJob(tenant, args[0], true)
		bail(err)

		if !confirm(opts.Yes, "Delete job @Y{%s} in tenant @Y{%s}?", job.Name, tenant.Name) {
			break
		}
		r, err := c.DeleteJob(tenant, job)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(r))
			break
		}
		fmt.Printf("%s\n", r.OK)

	/* }}} */
	case "pause-job": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s -t TENANT [OPTIONS] NAME-or-UUID\n", command)
		}

		required(opts.Tenant != "", "Missing required --tenant option.")

		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		job, err := c.FindJob(tenant, args[0], true)
		bail(err)

		if !confirm(opts.Yes, "Pause job @Y{%s} in tenant @Y{%s}?", job.Name, tenant.Name) {
			break
		}
		r, err := c.PauseJob(tenant, job)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(r))
			break
		}
		fmt.Printf("%s\n", r.OK)

	/* }}} */
	case "unpause-job": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s -t TENANT [OPTIONS] NAME-or-UUID\n", command)
		}

		required(opts.Tenant != "", "Missing required --tenant option.")

		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		job, err := c.FindJob(tenant, args[0], true)
		bail(err)

		if !confirm(opts.Yes, "Unpause job @Y{%s} in tenant @Y{%s}?", job.Name, tenant.Name) {
			break
		}
		r, err := c.UnpauseJob(tenant, job)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(r))
			break
		}
		fmt.Printf("%s\n", r.OK)

	/* }}} */
	case "run-job": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s -t TENANT [OPTIONS] NAME-or-UUID\n", command)
		}

		required(opts.Tenant != "", "Missing required --tenant option.")

		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		job, err := c.FindJob(tenant, args[0], !opts.Exact)
		bail(err)

		if !confirm(opts.Yes, "Run job @Y{%s} in tenant @Y{%s}?", job.Name, tenant.Name) {
			break
		}
		r, err := c.RunJob(tenant, job)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(r))
			break
		}
		fmt.Printf("%s\n", r.OK)

	/* }}} */

	case "archives": /* {{{ */
		required(opts.Tenant != "", "Missing required --tenant option.")

		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		archives, err := c.ListArchives(tenant, nil)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(archives))
			break
		}

		/*
			tbl := table.New(
				opts.Output,
				&table.Map{
					"%uuid": "UUID",
					"%name": "Name",
				},
				archives...
			)
		*/
		/* FIXME: support --long / -l and maybe --output / -o "fmt-str" */
		tbl := tui.NewTable("UUID", "Key", "Status")
		for _, archive := range archives {
			tbl.Row(archive, archive.UUID, archive.Key, archive.Status)
		}
		tbl.Output(os.Stdout)

	/* }}} */
	case "archive": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s NAME-or-UUID\n", command)
		}

		required(opts.Tenant != "", "Missing required --tenant option.")
		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		archive, err := c.GetArchive(tenant, args[0])
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(archive))
			break
		}

		r := tui.NewReport()
		r.Add("UUID", archive.UUID)
		r.Add("Key", archive.Key)
		r.Add("Status", archive.Status)
		r.Output(os.Stdout)

	/* }}} */
	case "restore-archive": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s NAME-or-UUID\n", command)
		}

		required(opts.Tenant != "", "Missing required --tenant option.")
		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		archive, err := c.GetArchive(tenant, args[0])
		bail(err)

		var target *shield.Target
		if id := opts.RestoreArchive.Target; id != "" {
			target, err = c.FindTarget(tenant, id, !opts.Exact)
			bail(err)
		}

		rs, err := c.RestoreArchive(tenant, archive, target)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(rs))
			break
		}

		fmt.Printf("%s\n", rs.OK)

	/* }}} */
	case "purge-archive": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s NAME-or-UUID\n", command)
		}

		required(opts.Tenant != "", "Missing required --tenant option.")
		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		archive, err := c.GetArchive(tenant, args[0])
		bail(err)

		rs, err := c.DeleteArchive(tenant, archive)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(rs))
			break
		}

		fmt.Printf("%s\n", rs.OK)

	/* }}} */

	/* FIXME: allow partial search by UUID */

	case "tasks": /* {{{ */
		required(opts.Tenant != "", "Missing required --tenant option.")
		required(!(opts.Tasks.Active && opts.Tasks.Inactive),
			"The --active and --inactive options are mutually exclusive.")
		required(!(opts.Tasks.All && opts.Tasks.Inactive),
			"The --all and --inactive options are mutually exclusive.")
		required(!(opts.Tasks.All && opts.Tasks.Active),
			"The --all and --active options are mutually exclusive.")
		required(len(args) <= 0, "Too many arguments.")

		if opts.Tasks.Limit == 0 {
			opts.Tasks.Limit = 1000 /* arbitrary upper-limit */
		}

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

		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		if opts.Tasks.Target != "" {
			t, err := c.FindTarget(tenant, opts.Tasks.Target, !opts.Exact)
			bail(err)
			opts.Tasks.Target = t.UUID
		}

		filter := &shield.TaskFilter{
			Status: opts.Tasks.Status,
			Limit:  &opts.Tasks.Limit,
			Target: opts.Tasks.Target,
		}
		if opts.Tasks.Active || opts.Tasks.Inactive {
			filter.Active = &opts.Tasks.Active
		}

		tasks, err := c.ListTasks(tenant, filter)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(tasks))
			break
		}

		/* FIXME: support --long / -l and maybe --output / -o "fmt-str" */
		tbl := tui.NewTable("UUID", "Type", "Status", "Owner", "Started at", "Completed at")
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
			tbl.Row(task, task.UUID, task.Type, task.Status, task.Owner, started, stopped)
		}
		tbl.Output(os.Stdout)

	/* }}} */
	case "task": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s UUID\n", command)
		}

		required(opts.Tenant != "", "Missing required --tenant option.")
		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		task, err := c.GetTask(tenant, args[0])
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

		if job, err := c.GetJob(tenant, task.JobUUID); err == nil {
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
			fail(2, "Usage: buckler %s -t TENANT [OPTIONS] UUID\n", command)
		}

		required(opts.Tenant != "", "Missing required --tenant option.")

		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		task, err := c.GetTask(tenant, args[0])
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

		if job, err := c.GetJob(tenant, task.JobUUID); err == nil {
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
		rs, err := c.CancelTask(tenant, task)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(rs))
			break
		}
		fmt.Printf("%s\n", rs.OK)

		/* }}} */

	/* FIXME: global tasks */

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

		tbl := tui.NewTable("UUID", "Name", "Account", "System Role")
		for _, user := range users {
			tbl.Row(user, user.UUID, user.Name, user.Account, user.SysRole)
		}
		tbl.Output(os.Stdout)

	/* }}} */
	case "user": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s NAME-or-UUID\n", command)
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
			fail(2, "Usage: buckler %s -t TENANT [OPTIONS] NAME-or-UUID\n", command)
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

		user, err = c.UpdateUser(user)
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
			fail(2, "Usage: buckler %s -t TENANT [OPTIONS] NAME-or-UUID\n", command)
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

		tbl := tui.NewTable("UUID", "Account", "Created At", "Last Seen", "IP Address", "User Agent")
		for _, session := range sessions {
			tbl.Row(session, session.UUID, session.UserAccount, strftime(session.CreatedAt), strftimenil(session.LastSeen, "(nerver)"), session.IP, session.UserAgent)
		}
		tbl.Output(os.Stdout)

	/* }}} */
	case "session": /* {{{ */
		if len(args) != 1 {
			fail(2, "Usage: buckler %s UUID\n", command)
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
			fail(2, "Usage: buckler %s UUID\n", command)
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

	default:
		bail(fmt.Errorf("Unrecognized command '%s'", command))
	}
}
