package main

import (
	"os"

	fmt "github.com/jhunt/go-ansi"
	"github.com/jhunt/go-table"
	"github.com/spf13/cobra"

	"github.com/shieldproject/shield/client/v2/shield"
	"github.com/shieldproject/shield/tui"
)

var (
	targetsUsed       bool
	targetsUnused     bool
	targetsWithPlugin string

	createTargetName        string
	createTargetSummary     string
	createTargetAgent       string
	createTargetPlugin      string
	createTargetData        []string
	createTargetCompression string

	updateTargetName        string
	updateTargetSummary     string
	updateTargetAgent       string
	updateTargetPlugin      string
	updateTargetCompression string
	updateTargetClearData   bool
	updateTargetData        []string
)

var targetsCmd = &cobra.Command{
	Use:   "targets [NAME-or-UUID]",
	Short: "List backup targets in the current tenant",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		required(optTenant != "", "Missing required --tenant option.")
		required(!(targetsUsed && targetsUnused), "The --used and --unused options are mutually exclusive.")

		c := clientFromConfig()
		filter := &shield.TargetFilter{
			Plugin: targetsWithPlugin,
			Fuzzy:  !optExact,
		}
		if len(args) == 1 {
			filter.Name = args[0]
			filter.UUID = args[0]
		}
		if targetsUsed || targetsUnused {
			x := targetsUsed
			filter.Used = &x
		}

		tenant, err := c.FindMyTenant(optTenant, true)
		bail(err)

		targets, err := c.ListTargets(tenant, filter)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(targets))
			return nil
		}

		tbl := table.NewTable("UUID", "Name", "Summary", "Plugin", "SHIELD Agent")
		for _, target := range targets {
			tbl.Row(target, uuid8full(target.UUID, optLong), target.Name, wrap(target.Summary, 35), target.Plugin, target.Agent)
		}
		tbl.Output(os.Stdout)
		return nil
	},
}

var targetCmd = &cobra.Command{
	Use:   "target NAME-or-UUID",
	Short: "Show details of a backup target",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		required(optTenant != "", "Missing required --tenant option.")
		c := clientFromConfig()
		tenant, err := c.FindMyTenant(optTenant, true)
		bail(err)

		t, err := c.FindTarget(tenant, args[0], !optExact)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(t))
			return nil
		}

		r := tui.NewReport()
		r.Add("UUID", t.UUID)
		r.Add("Name", t.Name)
		r.Add("Summary", wrap(t.Summary, 35))
		r.Add("Compression", t.Compression)
		r.Add("SHIELD Agent", t.Agent)
		r.Add("Backup Plugin", t.Plugin)
		r.Break()
		r.Add("Configuration", asJSON(t.Config))
		r.Output(os.Stdout)
		return nil
	},
}

var createTargetCmd = &cobra.Command{
	Use:   "create-target",
	Short: "Create a new backup target in the current tenant",
	RunE: func(cmd *cobra.Command, args []string) error {
		required(optTenant != "", "Missing required --tenant option.")
		c := clientFromConfig()
		tenant, err := c.FindMyTenant(optTenant, true)
		bail(err)

		conf, err := dataConfig(createTargetData)
		bail(err)

		if !optBatch {
			if createTargetName == "" {
				createTargetName = prompt("@C{Target Name}: ")
			}
			if createTargetSummary == "" {
				createTargetSummary = prompt("@C{Description}: ")
			}
			if createTargetAgent == "" {
				createTargetAgent = prompt("@C{SHIELD Agent (IP:port)}: ")
			}
			if createTargetPlugin == "" {
				createTargetPlugin = prompt("@C{Backup Plugin}: ")
			}
		}

		if createTargetCompression == "" {
			createTargetCompression = "bzip2"
		}

		t, err := c.CreateTarget(tenant, &shield.Target{
			Name:        createTargetName,
			Summary:     createTargetSummary,
			Agent:       createTargetAgent,
			Plugin:      createTargetPlugin,
			Compression: createTargetCompression,
			Config:      conf,
		})
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(t))
			return nil
		}

		r := tui.NewReport()
		r.Add("UUID", t.UUID)
		r.Add("Name", t.Name)
		r.Add("Summary", t.Summary)
		r.Add("Compression", t.Compression)
		r.Add("SHIELD Agent", t.Agent)
		r.Add("Backup Plugin", t.Plugin)
		r.Output(os.Stdout)
		return nil
	},
}

var updateTargetCmd = &cobra.Command{
	Use:   "update-target NAME-or-UUID",
	Short: "Update an existing backup target",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		required(optTenant != "", "Missing required --tenant option.")
		c := clientFromConfig()
		tenant, err := c.FindMyTenant(optTenant, true)
		bail(err)

		t, err := c.FindTarget(tenant, args[0], true)
		bail(err)

		conf, err := dataConfig(updateTargetData)
		bail(err)

		if updateTargetName != "" {
			t.Name = updateTargetName
		}
		if updateTargetSummary != "" {
			t.Summary = updateTargetSummary
		}
		if updateTargetAgent != "" {
			t.Agent = updateTargetAgent
		}
		if updateTargetPlugin != "" && t.Plugin != updateTargetPlugin {
			updateTargetClearData = true
			t.Plugin = updateTargetPlugin
		}
		if updateTargetCompression != "" {
			t.Compression = updateTargetCompression
		}

		if t.Config == nil {
			t.Config = make(map[string]interface{})
		}
		if updateTargetClearData {
			t.Config = conf
		} else {
			for k, v := range conf {
				t.Config[k] = v
			}
		}

		_, err = c.UpdateTarget(tenant, t)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(t))
			return nil
		}

		r := tui.NewReport()
		r.Add("UUID", t.UUID)
		r.Add("Name", t.Name)
		r.Add("Summary", t.Summary)
		r.Add("Compression", t.Compression)
		r.Add("SHIELD Agent", t.Agent)
		r.Add("Backup Plugin", t.Plugin)
		r.Add("Configuration", asJSON(t.Config))
		r.Output(os.Stdout)
		return nil
	},
}

var deleteTargetCmd = &cobra.Command{
	Use:   "delete-target NAME-or-UUID",
	Short: "Delete a backup target from the current tenant",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		required(optTenant != "", "Missing required --tenant option.")
		c := clientFromConfig()
		tenant, err := c.FindMyTenant(optTenant, true)
		bail(err)

		t, err := c.FindTarget(tenant, args[0], true)
		bail(err)

		if !confirm(optYes, "Delete target @Y{%s} in tenant @Y{%s}?", t.Name, tenant.Name) {
			return nil
		}
		r, err := c.DeleteTarget(tenant, t)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(r))
			return nil
		}
		fmt.Printf("%s\n", r.OK)
		return nil
	},
}

func init() {
	targetsCmd.Flags().BoolVar(&targetsUsed, "used", false, "Show only targets that are in use")
	targetsCmd.Flags().BoolVar(&targetsUnused, "unused", false, "Show only targets not in use")
	targetsCmd.Flags().StringVar(&targetsWithPlugin, "with-plugin", "", "Filter by plugin name")

	createTargetCmd.Flags().StringVarP(&createTargetName, "name", "n", "", "Target name")
	createTargetCmd.Flags().StringVarP(&createTargetSummary, "summary", "s", "", "Target description")
	createTargetCmd.Flags().StringVarP(&createTargetAgent, "agent", "a", "", "SHIELD agent address (IP:port)")
	createTargetCmd.Flags().StringVarP(&createTargetPlugin, "plugin", "p", "", "Backup plugin name")
	createTargetCmd.Flags().StringArrayVarP(&createTargetData, "data", "d", nil, "Plugin configuration (key=value)")
	createTargetCmd.Flags().StringVarP(&createTargetCompression, "compression", "C", "", "Compression algorithm (default: bzip2)")

	updateTargetCmd.Flags().StringVarP(&updateTargetName, "name", "n", "", "New target name")
	updateTargetCmd.Flags().StringVarP(&updateTargetSummary, "summary", "s", "", "New target description")
	updateTargetCmd.Flags().StringVarP(&updateTargetAgent, "agent", "a", "", "New SHIELD agent address (IP:port)")
	updateTargetCmd.Flags().StringVarP(&updateTargetPlugin, "plugin", "p", "", "New backup plugin name")
	updateTargetCmd.Flags().StringVarP(&updateTargetCompression, "compression", "C", "", "New compression algorithm")
	updateTargetCmd.Flags().BoolVar(&updateTargetClearData, "clear-data", false, "Clear existing plugin configuration")
	updateTargetCmd.Flags().StringArrayVarP(&updateTargetData, "data", "d", nil, "Plugin configuration (key=value)")

	rootCmd.AddCommand(targetsCmd)
	rootCmd.AddCommand(targetCmd)
	rootCmd.AddCommand(createTargetCmd)
	rootCmd.AddCommand(updateTargetCmd)
	rootCmd.AddCommand(deleteTargetCmd)
}
