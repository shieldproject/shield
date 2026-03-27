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
	archivesTarget string
	archivesStore  string
	archivesLimit  int

	restoreArchiveTarget string

	purgeArchiveReason string

	annotateArchiveNotes string
)

var archivesCmd = &cobra.Command{
	Use:   "archives [UUID]",
	Short: "List backup archives",
	RunE: func(cmd *cobra.Command, args []string) error {
		required(optTenant != "", "Missing required --tenant option.")
		required(len(args) <= 1, "Too many arguments.")

		c := clientFromConfig()
		tenant, err := c.FindMyTenant(optTenant, true)
		bail(err)

		if archivesLimit == 0 {
			archivesLimit = 1000
		}
		if archivesTarget != "" {
			t, err := c.FindTarget(tenant, archivesTarget, !optExact)
			bail(err)
			archivesTarget = t.UUID
		}
		if archivesStore != "" {
			s, err := c.FindStore(tenant, archivesStore, !optExact)
			bail(err)
			archivesStore = s.UUID
		}

		filter := &shield.ArchiveFilter{
			Target: archivesTarget,
			Store:  archivesStore,
			Limit:  &archivesLimit,
			Fuzzy:  !optExact,
		}
		if len(args) == 1 {
			filter.UUID = args[0]
		}

		archives, err := c.ListArchives(tenant, filter)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(archives))
			return nil
		}

		tbl := table.NewTable("UUID", "Key", "Size", "Status", "Compression", "Encryption")
		for _, archive := range archives {
			tbl.Row(archive, uuid8full(archive.UUID, optLong), archive.Key, archive.Status, archive.Compression, archive.EncryptionType, formatBytes(archive.Size))
		}
		tbl.Output(os.Stdout)
		return nil
	},
}

var archiveCmd = &cobra.Command{
	Use:   "archive NAME-or-UUID",
	Short: "Show a single backup archive",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			fail(2, "Usage: shield archive NAME-or-UUID\n")
		}
		required(optTenant != "", "Missing required --tenant option.")

		c := clientFromConfig()
		tenant, err := c.FindMyTenant(optTenant, true)
		bail(err)

		archive, err := c.FindArchive(tenant, args[0], !optExact)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(archive))
			return nil
		}

		r := tui.NewReport()
		r.Add("UUID", archive.UUID)
		r.Add("Key", archive.Key)
		r.Add("Status", archive.Status)
		r.Add("Size", formatBytes(archive.Size))
		r.Add("Compression", archive.Compression)
		r.Add("Encryption", archive.EncryptionType)
		r.Add("Notes", archive.Notes)
		r.Output(os.Stdout)
		return nil
	},
}

var restoreArchiveCmd = &cobra.Command{
	Use:   "restore-archive NAME-or-UUID",
	Short: "Restore a backup archive",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			fail(2, "Usage: shield restore-archive NAME-or-UUID\n")
		}
		required(optTenant != "", "Missing required --tenant option.")

		c := clientFromConfig()
		tenant, err := c.FindMyTenant(optTenant, true)
		bail(err)

		archive, err := c.FindArchive(tenant, args[0], !optExact)
		bail(err)

		var target *shield.Target
		if id := restoreArchiveTarget; id != "" {
			target, err = c.FindTarget(tenant, id, !optExact)
			bail(err)
		}

		task, err := c.RestoreArchive(tenant, archive, target)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(task))
			return nil
		}

		fmt.Printf("Scheduled restore; task @C{%s}\n", task.UUID)
		return nil
	},
}

var purgeArchiveCmd = &cobra.Command{
	Use:   "purge-archive NAME-or-UUID",
	Short: "Purge a backup archive",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			fail(2, "Usage: shield purge-archive NAME-or-UUID\n")
		}
		required(optTenant != "", "Missing required --tenant option.")

		c := clientFromConfig()
		tenant, err := c.FindMyTenant(optTenant, true)
		bail(err)

		archive, err := c.FindArchive(tenant, args[0], !optExact)
		bail(err)

		if purgeArchiveReason != "" {
			archive.Notes = purgeArchiveReason
			_, err = c.UpdateArchive(tenant, archive)
			if err != nil && !optJSON {
				fmt.Fprintf(os.Stderr, "@Y{WARNING: Unable to update archive with reason for purge}: %s", err)
			}
		}

		rs, err := c.DeleteArchive(tenant, archive)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(rs))
			return nil
		}

		fmt.Printf("%s\n", rs.OK)
		return nil
	},
}

var annotateArchiveCmd = &cobra.Command{
	Use:   "annotate-archive NAME-or-UUID",
	Short: "Annotate a backup archive with notes",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			fail(2, "Usage: shield annotate-archive NAME-or-UUID --notes ...\n")
		}
		required(annotateArchiveNotes != "", "Missing required --notes option.")
		required(optTenant != "", "Missing required --tenant option.")

		c := clientFromConfig()
		tenant, err := c.FindMyTenant(optTenant, true)
		bail(err)

		archive, err := c.FindArchive(tenant, args[0], !optExact)
		bail(err)

		archive.Notes = annotateArchiveNotes
		archive, err = c.UpdateArchive(tenant, archive)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(archive))
			return nil
		}

		r := tui.NewReport()
		r.Add("UUID", archive.UUID)
		r.Add("Key", archive.Key)
		r.Add("Compression", archive.Compression)
		r.Add("Status", archive.Status)
		r.Add("Notes", archive.Notes)
		r.Output(os.Stdout)
		return nil
	},
}

func init() {
	archivesCmd.Flags().StringVar(&archivesTarget, "target", "", "Filter by target data system")
	archivesCmd.Flags().StringVar(&archivesStore, "store", "", "Filter by cloud storage system")
	archivesCmd.Flags().IntVarP(&archivesLimit, "limit", "l", 0, "Limit number of results")

	restoreArchiveCmd.Flags().StringVar(&restoreArchiveTarget, "target", "", "Restore to a different target")
	restoreArchiveCmd.Flags().StringVar(&restoreArchiveTarget, "to", "", "Restore to a different target")

	purgeArchiveCmd.Flags().StringVar(&purgeArchiveReason, "reason", "", "Reason for purging the archive")

	annotateArchiveCmd.Flags().StringVar(&annotateArchiveNotes, "notes", "", "Notes to annotate the archive with")

	rootCmd.AddCommand(archivesCmd)
	rootCmd.AddCommand(archiveCmd)
	rootCmd.AddCommand(restoreArchiveCmd)
	rootCmd.AddCommand(purgeArchiveCmd)
	rootCmd.AddCommand(annotateArchiveCmd)
}
