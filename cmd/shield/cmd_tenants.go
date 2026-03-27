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
	tenantShowMembers   bool
	createTenantName    string
	updateTenantName    string
	deleteTenantRecurse bool
	inviteRole          string
)

var tenantsCmd = &cobra.Command{
	Use:   "tenants [NAME-or-UUID]",
	Short: "List SHIELD tenants",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := clientFromConfig()
		filter := &shield.TenantFilter{
			Fuzzy: !optExact,
		}
		if len(args) == 1 {
			filter.Name = args[0]
			filter.UUID = args[0]
		}

		tenants, err := c.ListTenants(filter)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(tenants))
			return nil
		}

		tbl := table.NewTable("UUID", "Name")
		for _, tenant := range tenants {
			tbl.Row(tenant, uuid8full(tenant.UUID, optLong), tenant.Name)
		}
		tbl.Output(os.Stdout)
		return nil
	},
}

var tenantCmd = &cobra.Command{
	Use:   "tenant NAME-or-UUID",
	Short: "Show details of a SHIELD tenant",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := clientFromConfig()
		tenant, err := c.FindTenant(args[0], !optExact)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(tenant))
			return nil
		}

		r := tui.NewReport()
		r.Add("UUID", tenant.UUID)
		r.Add("Name", tenant.Name)
		r.Output(os.Stdout)

		if tenantShowMembers {
			fmt.Printf("\n")
			t := table.NewTable("UUID", "Name", "Account", "Role")
			for _, mem := range tenant.Members {
				t.Row(mem, mem.UUID, mem.Name, fmt.Sprintf("%s@%s", mem.Account, mem.Backend), mem.Role)
			}
			t.Output(os.Stdout)
		}
		return nil
	},
}

var createTenantCmd = &cobra.Command{
	Use:   "create-tenant",
	Short: "Create a new SHIELD tenant",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := clientFromConfig()
		if !optBatch {
			if createTenantName == "" {
				createTenantName = prompt("@C{Tenant Name}: ")
			}
		}

		t, err := c.CreateTenant(&shield.Tenant{
			Name: createTenantName,
		})
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(t))
			return nil
		}

		r := tui.NewReport()
		r.Add("UUID", t.UUID)
		r.Add("Name", t.Name)
		r.Output(os.Stdout)
		return nil
	},
}

var updateTenantCmd = &cobra.Command{
	Use:   "update-tenant NAME-or-UUID",
	Short: "Update an existing SHIELD tenant",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := clientFromConfig()
		t, err := c.FindTenant(args[0], true)
		bail(err)

		if updateTenantName != "" {
			t.Name = updateTenantName
		}

		_, err = c.UpdateTenant(t)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(t))
			return nil
		}

		r := tui.NewReport()
		r.Add("UUID", t.UUID)
		r.Add("Name", t.Name)
		r.Output(os.Stdout)
		return nil
	},
}

var deleteTenantCmd = &cobra.Command{
	Use:   "delete-tenant NAME-or-UUID",
	Short: "Delete a SHIELD tenant",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := clientFromConfig()
		t, err := c.FindTenant(args[0], true)
		bail(err)

		if !confirm(optYes, "Are you sure you want to delete all configuration under this tenant?") {
			return nil
		}

		_, err = c.DeleteTenant(t, deleteTenantRecurse)
		bail(err)

		if optJSON {
			fmt.Printf("%s has been deleted\n", asJSON(t))
			return nil
		}

		r := tui.NewReport()
		r.Add("UUID", t.UUID)
		r.Add("Name", t.Name)
		r.Output(os.Stdout)
		return nil
	},
}

var inviteCmd = &cobra.Command{
	Use:   "invite USER [USER ...]",
	Short: "Invite users to the current tenant",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := clientFromConfig()
		required(optTenant != "", "Missing required --tenant option.")
		tenant, err := c.FindTenant(optTenant, true)
		bail(err)

		switch inviteRole {
		case "":
			inviteRole = "operator"
		case "operator", "engineer", "admin":
		default:
			bail(fmt.Errorf("Invalid --role value '%s' (must be one of operator, engineer, or admin)", inviteRole))
		}

		users := make([]*shield.User, len(args))
		for i := range args {
			user, err := c.FindUser(args[i], !optExact)
			bail(err)
			users[i] = user
		}

		r, err := c.Invite(tenant, inviteRole, users)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(r))
			return nil
		}
		fmt.Printf("%s\n", r.OK)
		return nil
	},
}

var banishCmd = &cobra.Command{
	Use:   "banish USER [USER ...]",
	Short: "Remove users from the current tenant",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := clientFromConfig()
		required(optTenant != "", "Missing required --tenant option.")
		tenant, err := c.FindTenant(optTenant, true)
		bail(err)

		users := make([]*shield.User, len(args))
		for i := range args {
			user, err := c.FindUser(args[i], !optExact)
			bail(err)
			users[i] = user
		}

		r, err := c.Banish(tenant, users)
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
	tenantCmd.Flags().BoolVar(&tenantShowMembers, "members", false, "Show tenant members")

	createTenantCmd.Flags().StringVarP(&createTenantName, "name", "n", "", "Tenant name")

	updateTenantCmd.Flags().StringVarP(&updateTenantName, "name", "n", "", "New tenant name")

	deleteTenantCmd.Flags().BoolVarP(&deleteTenantRecurse, "recursive", "r", false, "Recursively delete all tenant resources")

	inviteCmd.Flags().StringVarP(&inviteRole, "role", "r", "", "Role to assign (operator, engineer, admin)")

	rootCmd.AddCommand(tenantsCmd)
	rootCmd.AddCommand(tenantCmd)
	rootCmd.AddCommand(createTenantCmd)
	rootCmd.AddCommand(updateTenantCmd)
	rootCmd.AddCommand(deleteTenantCmd)
	rootCmd.AddCommand(inviteCmd)
	rootCmd.AddCommand(banishCmd)
}
