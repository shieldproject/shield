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
	sessionsLimit    int
	sessionsUserUUID string
	sessionsIP       string
)

var sessionsCmd = &cobra.Command{
	Use:   "sessions [IP]",
	Short: "List active SHIELD sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		required(len(args) <= 1, "Too many arguments.")

		filter := &shield.SessionFilter{
			Limit: sessionsLimit,
		}
		if len(args) == 1 {
			filter.IP = args[0]
		}

		c := clientFromConfig()
		sessions, err := c.ListSessions(filter)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(sessions))
			return nil
		}

		tbl := table.NewTable("UUID", "Account", "Created At", "Last Seen", "IP Address", "User Agent")
		for _, session := range sessions {
			tbl.Row(session, uuid8full(session.UUID, optLong), session.UserAccount, strftime(session.CreatedAt), strftimenil(session.LastSeen, "(never)"), session.IP, session.UserAgent)
		}
		tbl.Output(os.Stdout)
		return nil
	},
}

var sessionCmd = &cobra.Command{
	Use:   "session UUID",
	Short: "Show details of a SHIELD session",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			fail(2, "Usage: shield session UUID\n")
		}

		c := clientFromConfig()
		session, err := c.GetSession(args[0])
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(session))
			return nil
		}

		r := tui.NewReport()
		r.Add("UUID", session.UUID)
		r.Add("Account", session.UserAccount)
		r.Add("Created At", strftime(session.CreatedAt))
		r.Add("Last Seen", strftimenil(session.LastSeen, "(never)"))
		r.Add("IP Address", session.IP)
		r.Add("User Agent", session.UserAgent)
		r.Output(os.Stdout)
		return nil
	},
}

var deleteSessionCmd = &cobra.Command{
	Use:   "delete-session UUID",
	Short: "Delete a SHIELD session",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			fail(2, "Usage: shield delete-session UUID\n")
		}

		c := clientFromConfig()
		session, err := c.GetSession(args[0])
		bail(err)

		if !confirm(optYes, "Delete session for user @Y{%s}?", session.UserAccount) {
			return nil
		}

		if session.CurrentSession {
			if !confirm(optYes, "This is your current session, are you really sure you want to delete it? You will have to reauthenticate.") {
				return nil
			}
		}
		res, err := c.DeleteSession(session)
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
	sessionsCmd.Flags().IntVarP(&sessionsLimit, "limit", "l", 0, "Limit number of sessions returned")
	sessionsCmd.Flags().StringVarP(&sessionsUserUUID, "user-uuid", "u", "", "Filter by user UUID")
	sessionsCmd.Flags().StringVar(&sessionsIP, "ip", "", "Filter by IP address")

	rootCmd.AddCommand(sessionsCmd)
	rootCmd.AddCommand(sessionCmd)
	rootCmd.AddCommand(deleteSessionCmd)
}
