package main

import (
	"io/ioutil"
	"os"
	"regexp"
	"strconv"

	fmt "github.com/jhunt/go-ansi"
	"github.com/jhunt/go-cli"
	env "github.com/jhunt/go-envirotron"

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

	Curl struct{} `cli:"curl"`

	Status struct {
		Global bool `cli:"--global"`
	} `cli:"status"`

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
		Name    string   `cli:"-n, --name"`
		Summary string   `cli:"-s, --summary"`
		Agent   string   `cli:"-a, --agent"`
		Plugin  string   `cli:"-p, --plugin"`
		Data    []string `cli:"-d, --data"`
	} `cli:"create-store"`
	UpdateStore struct {
		Name      string   `cli:"-n, --name"`
		Summary   string   `cli:"-s, --summary"`
		Agent     string   `cli:"-a, --agent"`
		Plugin    string   `cli:"-p, --plugin"`
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
		Name    string   `cli:"-n, --name"`
		Summary string   `cli:"-s, --summary"`
		Agent   string   `cli:"-a, --agent"`
		Plugin  string   `cli:"-p, --plugin"`
		Data    []string `cli:"-d, --data"`
	} `cli:"create-global-store"`
	UpdateGlobalStore struct {
		Name      string   `cli:"-n, --name"`
		Summary   string   `cli:"-s, --summary"`
		Agent     string   `cli:"-a, --agent"`
		Plugin    string   `cli:"-p, --plugin"`
		ClearData bool     `cli:"--clear-data"`
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

	if opts.Help {
		fmt.Printf("USAGE: buckler COMMAND [OPTIONS] [ARGUMENTS]\n")
		os.Exit(0)
	}

	if opts.Version {
		fmt.Printf("buckler v¯\\_(ツ)_/¯\n")
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
			/* try to parse it as a literal PEM */
			/* check for the file on-disk if no interior newlines */
			cacert = opts.API.CACertificate /* FIXME cheating */
		}

		/* validate the SHIELD */
		c := &shield.Client{
			URL:     url,
			Debug:   opts.Debug,
			Trace:   opts.Trace,
			Session: "",
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
		URL:     config.Current.URL,
		Debug:   opts.Debug,
		Trace:   opts.Trace,
		Session: config.Current.Session,
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
		var status *shield.Status
		if opts.Status.Global {
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

		fmt.Printf("SHIELD %s v%s\n", status.SHIELD.Env, status.SHIELD.Version)

	/* }}} */

	case "auth-tokens": /* {{{ */
		tokens, err := c.ListAuthTokens()
		bail(err)

		for _, token := range tokens {
			if token.LastSeen == "" {
				token.LastSeen = "never"
			}
		}

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(tokens))
			break
		}

		tbl := tui.NewTable("Name", "Created at", "Last seen")
		for _, token := range tokens {
			tbl.Row(token, token.Name, token.CreatedAt, token.LastSeen)
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

		tbl := tui.NewTable("UUID", "Name", "Summary", "Plugin", "SHIELD Agent", "Configuration")
		for _, store := range stores {
			tbl.Row(store, store.UUID, store.Name, store.Summary, store.Plugin, store.Agent, asJSON(store.Config))
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

		r := tui.NewReport()
		r.Add("UUID", store.UUID)
		r.Add("Name", store.Name)
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
		}

		store, err := c.CreateStore(tenant, &shield.Store{
			Name:    opts.CreateStore.Name,
			Summary: opts.CreateStore.Summary,
			Agent:   opts.CreateStore.Agent,
			Plugin:  opts.CreateStore.Plugin,
			Config:  conf,
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

		tbl := tui.NewTable("UUID", "Name", "Summary", "Plugin", "SHIELD Agent", "Configuration")
		for _, store := range stores {
			tbl.Row(store, store.UUID, store.Name, store.Summary, store.Plugin, store.Agent, asJSON(store.Config))
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

		r := tui.NewReport()
		r.Add("UUID", store.UUID)
		r.Add("Name", store.Name)
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
		}

		store, err := c.CreateGlobalStore(&shield.Store{
			Name:    opts.CreateGlobalStore.Name,
			Summary: opts.CreateGlobalStore.Summary,
			Agent:   opts.CreateGlobalStore.Agent,
			Plugin:  opts.CreateGlobalStore.Plugin,
			Config:  conf,
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
	case "polic-templatey": /* {{{ */
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
		case "running", "pending", "cancelled":
			/* good enough to pass validation... */
		default:
			fail(3, "Invalid --status value of '%s' (must be one of all, running, pending, or cancelled).", opts.Tasks.Status)
		}

		if opts.Tasks.All {
			opts.Tasks.Status = ""
		}

		filter := &shield.TaskFilter{
			Status: opts.Tasks.Status,
			Limit:  &opts.Tasks.Limit,
		}
		if opts.Tasks.Active || opts.Tasks.Inactive {
			filter.Active = &opts.Tasks.Active
		}

		tenant, err := c.FindMyTenant(opts.Tenant, true)
		bail(err)

		tasks, err := c.ListTasks(tenant, filter)
		bail(err)

		if opts.JSON {
			fmt.Printf("%s\n", asJSON(tasks))
			break
		}

		/* FIXME: support --long / -l and maybe --output / -o "fmt-str" */
		tbl := tui.NewTable("UUID", "Type", "Status", "Owner", "Started at", "Completed at")
		for _, task := range tasks {
			tbl.Row(task, task.UUID, task.Type, task.Status, task.Owner, task.StartedAt, task.StoppedAt)
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
		if task.StartedAt != "" {
			stopped = "(running)"
			started = task.StartedAt
		}
		if task.StoppedAt != "" {
			stopped = task.StoppedAt
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
		if task.StartedAt != "" {
			stopped = "(running)"
			started = task.StartedAt
		}
		if task.StoppedAt != "" {
			stopped = task.StoppedAt
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

		if task.StoppedAt != "" {
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

	default:
		bail(fmt.Errorf("Unrecognized command '%s'", command))
	}
}
