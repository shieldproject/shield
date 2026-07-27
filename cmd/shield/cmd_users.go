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
	usersWithSystemRole string
	createUserName      string
	createUserAccount   string
	createUserPassword  string
	createUserSysRole   string
	updateUserName      string
	updateUserPassword  string
	updateUserSysRole   string
)

var usersCmd = &cobra.Command{
	Use:   "users [SEARCH]",
	Short: "List SHIELD users",
	RunE: func(cmd *cobra.Command, args []string) error {
		required(len(args) <= 1, "Too many arguments.")

		filter := &shield.UserFilter{
			Fuzzy:   !optExact,
			SysRole: usersWithSystemRole,
		}
		if len(args) == 1 {
			filter.Account = args[0]
		}

		c := clientFromConfig()
		users, err := c.ListUsers(filter)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(users))
			return nil
		}

		tbl := table.NewTable("UUID", "Name", "Account", "System Role")
		for _, user := range users {
			tbl.Row(user, uuid8full(user.UUID, optLong), user.Name, user.Account, user.SysRole)
		}
		tbl.Output(os.Stdout)
		return nil
	},
}

var userCmd = &cobra.Command{
	Use:   "user NAME-or-UUID",
	Short: "Show details of a SHIELD user",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			fail(2, "Usage: shield user NAME-or-UUID\n")
		}

		c := clientFromConfig()
		user, err := c.FindUser(args[0], !optExact)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(user))
			return nil
		}

		r := tui.NewReport()
		r.Add("UUID", user.UUID)
		r.Add("Name", user.Name)
		r.Add("Account", user.Account)
		r.Add("System Role", user.SysRole)
		r.Output(os.Stdout)
		return nil
	},
}

var createUserCmd = &cobra.Command{
	Use:   "create-user",
	Short: "Create a new SHIELD user",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !optBatch {
			if createUserName == "" {
				createUserName = prompt("@C{Display Name}: ")
			}
			if createUserAccount == "" {
				createUserAccount = prompt("@C{Username}: ")
			}
			for createUserPassword == "" {
				a := secureprompt("@Y{Choose a password}: ")
				b := secureprompt("@Y{Confirm password}: ")
				if a == "" {
					fmt.Fprintf(os.Stderr, "@R{password cannot be blank}\n")
				} else if a != b {
					fmt.Fprintf(os.Stderr, "@R{passwords do not match}\n")
				} else {
					createUserPassword = a
					break
				}
			}
		}

		c := clientFromConfig()
		user, err := c.CreateUser(&shield.User{
			Name:     createUserName,
			Account:  createUserAccount,
			Password: createUserPassword,
			SysRole:  createUserSysRole,
		})
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(user))
			return nil
		}

		r := tui.NewReport()
		r.Add("UUID", user.UUID)
		r.Add("Name", user.Name)
		r.Add("Account", user.Account)
		r.Add("System Role", user.SysRole)
		r.Output(os.Stdout)
		return nil
	},
}

var updateUserCmd = &cobra.Command{
	Use:   "update-user NAME-or-UUID",
	Short: "Update an existing SHIELD user",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			fail(2, "Usage: shield update-user [OPTIONS] NAME-or-UUID\n")
		}

		c := clientFromConfig()
		user, err := c.FindUser(args[0], !optExact)
		bail(err)

		if updateUserName != "" {
			user.Name = updateUserName
		}
		if updateUserPassword != "" {
			user.Password = updateUserPassword
		}
		if updateUserSysRole != "" {
			user.SysRole = updateUserSysRole
		}

		_, err = c.UpdateUser(user)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(user))
			return nil
		}

		/* FIXME: api doesn't return the user object... */
		r := tui.NewReport()
		r.Add("UUID", user.UUID)
		r.Add("Name", user.Name)
		r.Add("Account", user.Account)
		r.Add("System Role", user.SysRole)
		r.Output(os.Stdout)
		return nil
	},
}

var deleteUserCmd = &cobra.Command{
	Use:   "delete-user NAME-or-UUID",
	Short: "Delete a SHIELD user",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			fail(2, "Usage: shield delete-user NAME-or-UUID\n")
		}

		c := clientFromConfig()
		user, err := c.FindUser(args[0], true)
		bail(err)

		if !confirm(optYes, "Delete user @Y{%s}@local?", user.Name) {
			return nil
		}
		res, err := c.DeleteUser(user)
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
	usersCmd.Flags().StringVar(&usersWithSystemRole, "with-system-role", "", "Filter by system role")

	createUserCmd.Flags().StringVarP(&createUserName, "name", "n", "", "Display name")
	createUserCmd.Flags().StringVarP(&createUserAccount, "username", "u", "", "Username / account")
	createUserCmd.Flags().StringVarP(&createUserPassword, "password", "p", "", "Password")
	createUserCmd.Flags().StringVar(&createUserSysRole, "system-role", "", "System role to assign")

	updateUserCmd.Flags().StringVarP(&updateUserName, "name", "n", "", "New display name")
	updateUserCmd.Flags().StringVarP(&updateUserPassword, "password", "p", "", "New password")
	updateUserCmd.Flags().StringVar(&updateUserSysRole, "system-role", "", "New system role")

	rootCmd.AddCommand(usersCmd)
	rootCmd.AddCommand(userCmd)
	rootCmd.AddCommand(createUserCmd)
	rootCmd.AddCommand(updateUserCmd)
	rootCmd.AddCommand(deleteUserCmd)
}
