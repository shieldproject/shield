package main

import (
	"encoding/json"
	"os"
	"strings"

	fmt "github.com/jhunt/go-ansi"
	"github.com/jhunt/go-table"
	"github.com/spf13/cobra"

	"github.com/shieldproject/shield/client/v2/shield"
	"github.com/shieldproject/shield/tui"
)

var (
	agentsLimit   int
	agentsVisible bool
	agentsHidden  bool
	agentMetadata bool
	agentPlugins  bool
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "List SHIELD agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		required(len(args) <= 1, "Too many arguments.")
		required(!(agentsVisible && agentsHidden),
			"The --visible and --hidden options are mutually exclusive.")

		filter := &shield.AgentFilter{}
		if agentsVisible || agentsHidden {
			filter.Hidden = &agentsHidden
		}

		c := clientFromConfig()
		agents, err := c.ListAgents(filter)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(agents))
			return nil
		}

		tbl := table.NewTable("UUID", "Name", "Version", "Address", "Status", "Last Seen", "Last Checked", "Problems")
		for _, agent := range agents {
			st := agent.Status
			if agent.Hidden {
				st += " (hidden)"
			}
			tbl.Row(agent, uuid8full(agent.UUID, optLong), agent.Name, agent.Version, agent.Address, st, strftime(agent.LastSeenAt), strftimenil(agent.LastCheckedAt, "(never)"), len(agent.Problems))
		}
		tbl.Output(os.Stdout)
		return nil
	},
}

var agentCmd = &cobra.Command{
	Use:   "agent UUID",
	Short: "Show details of a SHIELD agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			fail(2, "Usage: shield agent UUID\n")
		}

		c := clientFromConfig()
		agent, err := c.FindAgent(args[0], !optExact)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(agent))
			return nil
		}

		r := tui.NewReport()
		r.Add("UUID", agent.UUID)
		r.Add("Name", agent.Name)
		r.Add("Version", agent.Version)
		r.Add("Address", agent.Address)
		r.Add("Status", agent.Status)
		r.Add("Last Seen", strftime(agent.LastSeenAt))
		r.Add("Last Checked", strftimenil(agent.LastCheckedAt, "(never)"))

		if agentMetadata {
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

		if agentPlugins {
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
		return nil
	},
}

var hideAgentCmd = &cobra.Command{
	Use:   "hide-agent UUID",
	Short: "Hide a SHIELD agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			fail(2, "Usage: shield hide-agent UUID\n")
		}

		c := clientFromConfig()
		agent, err := c.FindAgent(args[0], !optExact)
		bail(err)

		res, err := c.HideAgent(agent)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(res))
			return nil
		}
		fmt.Printf("%s\n", res.OK)
		return nil
	},
}

var showAgentCmd = &cobra.Command{
	Use:   "show-agent UUID",
	Short: "Show (unhide) a SHIELD agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			fail(2, "Usage: shield show-agent UUID\n")
		}

		c := clientFromConfig()
		agent, err := c.FindAgent(args[0], !optExact)
		bail(err)

		res, err := c.ShowAgent(agent)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(res))
			return nil
		}
		fmt.Printf("%s\n", res.OK)
		return nil
	},
}

var deleteAgentCmd = &cobra.Command{
	Use:   "delete-agent UUID",
	Short: "Delete a SHIELD agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			fail(2, "Usage: shield delete-agent UUID\n")
		}

		c := clientFromConfig()
		agent, err := c.FindAgent(args[0], !optExact)
		bail(err)

		if !confirm(optYes, "Delete agent @Y{%s} at @Y{%s}?", agent.Name, agent.Address) {
			return nil
		}
		res, err := c.DeleteAgent(agent)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(res))
			return nil
		}
		fmt.Printf("%s\n", res.OK)
		return nil
	},
}

func init() {
	agentsCmd.Flags().IntVarP(&agentsLimit, "limit", "l", 0, "Limit number of agents returned")
	agentsCmd.Flags().BoolVar(&agentsVisible, "visible", false, "Show only visible agents")
	agentsCmd.Flags().BoolVar(&agentsHidden, "hidden", false, "Show only hidden agents")

	agentCmd.Flags().BoolVarP(&agentMetadata, "metadata", "m", false, "Show agent metadata")
	agentCmd.Flags().BoolVarP(&agentPlugins, "plugins", "p", false, "Show agent plugins")

	rootCmd.AddCommand(agentsCmd)
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(hideAgentCmd)
	rootCmd.AddCommand(showAgentCmd)
	rootCmd.AddCommand(deleteAgentCmd)
}
