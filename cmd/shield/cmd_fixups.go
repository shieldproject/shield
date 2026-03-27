package main

import (
	"os"

	fmt "github.com/jhunt/go-ansi"
	"github.com/jhunt/go-table"
	"github.com/spf13/cobra"

	"github.com/shieldproject/shield/tui"
)

var fixupsCmd = &cobra.Command{
	Use:   "fixups",
	Short: "List available SHIELD fixups",
	RunE: func(cmd *cobra.Command, args []string) error {
		required(len(args) == 0, "Too many arguments.")

		c := clientFromConfig()
		fixups, err := c.ListFixups(nil)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(fixups))
			return nil
		}

		tbl := table.NewTable("ID", "Name", "Created at", "Applied at")
		for _, fixup := range fixups {
			tbl.Row(fixup, fixup.ID, fixup.Name, strftime(fixup.CreatedAt), strftimenil(fixup.AppliedAt, "(never)"))
		}
		tbl.Output(os.Stdout)
		return nil
	},
}

var fixupCmd = &cobra.Command{
	Use:   "fixup ID",
	Short: "Show details of a SHIELD fixup",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			fail(2, "Usage: shield fixup ID\n")
		}

		c := clientFromConfig()
		fixup, err := c.GetFixup(args[0])
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(fixup))
			return nil
		}

		r := tui.NewReport()
		r.Add("ID", fixup.ID)
		r.Add("Name", fixup.Name)
		r.Add("Created at", strftime(fixup.CreatedAt))
		r.Add("Applied at", strftimenil(fixup.AppliedAt, "(never)"))
		r.Add("Summary", wrap(fixup.Summary, 65))
		r.Output(os.Stdout)
		return nil
	},
}

var applyFixupCmd = &cobra.Command{
	Use:   "apply-fixup ID",
	Short: "Apply a SHIELD fixup",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			fail(2, "Usage: shield apply-fixup ID\n")
		}

		c := clientFromConfig()
		res, err := c.ApplyFixup(args[0])
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
	rootCmd.AddCommand(fixupsCmd)
	rootCmd.AddCommand(fixupCmd)
	rootCmd.AddCommand(applyFixupCmd)
}
