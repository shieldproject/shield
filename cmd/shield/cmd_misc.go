package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
	fmt "github.com/jhunt/go-ansi"
	"github.com/jhunt/go-table"
	"github.com/spf13/cobra"

	"github.com/shieldproject/shield/client/v2/shield"
)

var (
	cmdCommandsList  bool
	cmdCurlFile      string
	cmdStatusGlobal  bool
	cmdEventsSkip    []string
	cmdImportExample bool
)

var commandsCmd = &cobra.Command{
	Use:   "commands [group ...]",
	Short: "Print list of shield commands",
	RunE: func(cmd *cobra.Command, args []string) error {
		set := make(map[string]bool)
		for _, want := range args {
			set[want] = true
		}

		first := true
		blank := func() {
			if !cmdCommandsList {
				fmt.Printf("\n")
			}
		}
		header := func(s string) {
			if !cmdCommandsList {
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
			if cmdCommandsList {
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
			printc("  global-stores            List shared cloud storage systems.\n")
			printc("  global-store             Display details for a single shared cloud storage system.\n")
			printc("  create-global-store      Configure a new shared cloud storage system.\n")
			printc("  update-global-store      Reconfigure a shared cloud storage system.\n")
			printc("  delete-global-store      Decomission an unused shared cloud storage system.\n")
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
			printc("  delete-tenant            Remove a tenant\n")
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

		if cmdCommandsList {
			sort.Strings(save)
			for _, s := range save {
				fmt.Printf(s)
			}
		}
		return nil
	},
}

var curlCmd = &cobra.Command{
	Use:   "curl [METHOD] RELATIVE-URL [BODY]",
	Short: "Issue raw HTTP requests to the targeted SHIELD Core",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 || len(args) > 3 {
			fail(2, "Usage: shield curl [OPTIONS] [METHOD] RELATIVE-URL [BODY]\n")
		}
		c := clientFromConfig()

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

		if cmdCurlFile != "" {
			b, err := ioutil.ReadFile(cmdCurlFile)
			bail(err)
			body = string(b)
		}

		code, response, err := c.Curl(method, path, body)
		bail(err)
		fmt.Printf("%s\n", asJSON(response))
		if code >= 400 {
			os.Exit(code / 100)
		}
		return nil
	},
}

var timespecCmd = &cobra.Command{
	Use:   "timespec \"a schedule string\"",
	Short: "Explain Timespec scheduling strings",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := clientFromConfig()
		ok, spec, err := c.CheckTimespec(strings.Join(args, " "))
		if err != nil {
			bail(fmt.Errorf("Failed to check timespec: %s\n", err))
		}
		if !ok {
			fail(1, "Invalid timespec\n")
		}
		fmt.Printf("%s\n", spec)
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of the targeted SHIELD Core",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := clientFromConfig()
		info, err := c.Info()
		bail(err)

		var status *shield.Status
		if cmdStatusGlobal || optTenant == "" {
			status, err = c.GlobalStatus()
			bail(err)
		} else {
			required(optTenant != "", "Missing required --tenant option.")
			tenant, err := c.FindMyTenant(optTenant, true)
			bail(err)
			status, err = c.TenantStatus(tenant)
			bail(err)
		}

		if optJSON {
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
			return nil
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
		return nil
	},
}

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Watch the event stream from the targeted SHIELD Core",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := clientFromConfig()
		skip := make(map[string]bool)
		for _, what := range cmdEventsSkip {
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

			if optJSON {
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
		return nil
	},
}

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "Display scheduler thread and backlog status",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := clientFromConfig()
		ps, err := c.SchedulerStatus()
		bail(err)

		none := fmt.Sprintf("@K{(none)}")
		oops := fmt.Sprintf("@R{(oops)}")

		tbl := table.NewTable("#", "Op", "Status", "Task", "Tenant", "System", "Store", "Job", "Archive", "Agent")
		for _, worker := range ps.Workers {
			if worker.Idle {
				tbl.Row(worker, worker.ID, none, fmt.Sprintf("@C{idle}"),
					none, none, none, none, none, none)
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

				tenant := oops
				if worker.Tenant != nil {
					tenant = worker.Tenant.Name
					if worker.Tenant.UUID != "" {
						tenant = fmt.Sprintf("@W{%s}\n(%s)", worker.Tenant.Name, uuid8(worker.Tenant.UUID))
					}
				}

				store := none
				if worker.Store != nil {
					store = fmt.Sprintf("@W{%s}\n(%s)", worker.Store.Name, uuid8(worker.Store.UUID))
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
					archive = fmt.Sprintf("@W{%s}\n(%s; %s)\n", worker.Archive.UUID, "-", "-")
				}

				agent := oops
				if worker.Agent != "" {
					agent = fmt.Sprintf("@Y{%s}", worker.Agent)
				}

				tbl.Row(worker, worker.ID, op, status, task, tenant, system, store, job, archive, agent)
			}
		}
		fmt.Printf("@M{Scheduler Threads}\n\n")
		tbl.Output(os.Stdout)

		fmt.Printf("\n\n")
		fmt.Printf("@M{Task Backlog}\n\n")
		if len(ps.Backlog) > 0 {
			tbl = table.NewTable("Priority", "#", "Op", "Task", "System", "Store", "Job", "Archive", "Agent")
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

				store := none
				if t.Store != nil {
					store = fmt.Sprintf("@W{%s}\n(%s)", t.Store.Name, uuid8(t.Store.UUID))
				}

				job := none
				if t.Job != nil {
					job = fmt.Sprintf("@W{%s}\n(%s)", t.Job.Name, uuid8(t.Job.UUID))
				}

				archive := none
				if t.Archive != nil {
					archive = fmt.Sprintf("@W{%s}\n(%s; %s)\n", t.Archive.UUID, "-", "-")
				}

				agent := oops
				if t.Agent != "" {
					agent = fmt.Sprintf("@Y{%s}", t.Agent)
				}
				tbl.Row(t, t.Priority, t.Position, op, task, system, store, job, archive, agent)
			}
			tbl.Output(os.Stdout)
		} else {
			fmt.Printf("  none\n")
		}

		fmt.Printf("\n\n")
		return nil
	},
}

var importCmd = &cobra.Command{
	Use:   "import /path/to/manifest.yml ...",
	Short: "Import SHIELD configuration from a manifest file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cmdImportExample {
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
                         #   manager  - handles tenants and tenant
                         #              role assignments.
                         #
                         #   engineer - for the technical stuff
                         #              (mostly just global cloud storage)
                         #

  - name:     J User
    username: juser
    password: password
    sysrole:  ~          # juser has no system-level privileges
                         # (but they can still be invited to tenants)

global:
  # These cloud storage systems will be usable by all tenants,
  # but the specific configuration will be hidden from anyone
  # lacking system-level (sysrole) privileges.
  #
  storage:
    - name:    Global Storage
      summary: Shared global cloud storage, for use by anyone.
      agent:   '10.0.0.6:5444'
      plugin:  webdav
      config:                           # this configuration depends entirely
        url: http://webdav/global       # on the store plugin used (here, webdav)

tenants:
  - name: A Tenant
    members:
      - { user: juser@local, role: admin }

    storage:
      - name:    Local Storage
        summary: Dedicated cloud storage, just for this tenant.
        agent:   '10.0.0.6:5444'
        plugin:  webdav
        config:
          url: http://webdav/a-tenant

    systems:
      - name:    A System
        summary: A protected data system, owned by A Tenant.
        agent:   10.255.6.7:5444
        plugin:  fs
        config:
          base_dir: /tmp

        jobs:
          - name:     Daily
            when:     daily 4:10am
            paused:   no
            storage:  Local Storage
            retain:   4d
            retries:  3

          - name:     Weekly
            when:     sundays at 2:45am
            paused:   yes
            storage:  Local Storage
            retain:   28d
            retries:  2
`)
			return nil
		}

		if len(args) < 1 {
			fail(2, "Usage: shield import /path/to/manifest.yml ...\n")
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
				bail(cliConfig.Select(optCore))

				m.Core = cliConfig.Current.URL
				m.Token = cliConfig.Current.Session
				m.CA = cliConfig.Current.CACertificate
				m.InsecureSkipVerify = cliConfig.Current.InsecureSkipVerify
			}

			err = m.Deploy(&shield.Client{
				Debug: optDebug,
				Trace: optTrace,
			})
			bailon(file, err)
		}
		return nil
	},
}

func init() {
	commandsCmd.Flags().BoolVar(&cmdCommandsList, "list", false, "List commands alphabetically")
	commandsCmd.Flags().BoolVar(&cmdCommandsList, "full", false, "Show full command list")
	curlCmd.Flags().StringVar(&cmdCurlFile, "file", "", "Read request body from file")
	statusCmd.Flags().BoolVar(&cmdStatusGlobal, "global", false, "Show global status")
	eventsCmd.Flags().StringArrayVar(&cmdEventsSkip, "skip", nil, "Event types to skip")
	importCmd.Flags().BoolVar(&cmdImportExample, "example", false, "Print example import manifest")

	rootCmd.AddCommand(commandsCmd)
	rootCmd.AddCommand(curlCmd)
	rootCmd.AddCommand(timespecCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(eventsCmd)
	rootCmd.AddCommand(psCmd)
	rootCmd.AddCommand(importCmd)
}
