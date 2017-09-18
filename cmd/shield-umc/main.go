package main

import (
	"os"
	"strings"

	"github.com/jhunt/go-cli"
	env "github.com/jhunt/go-envirotron"
	"github.com/pborman/uuid"
	fmt "github.com/starkandwayne/goutils/ansi"

	_ "github.com/mattn/go-sqlite3"

	"github.com/starkandwayne/shield/db"
)

var Version = ""

type Options struct {
	Help    bool   `cli:"-h, --help"`
	Version bool   `cli:"-v, --version"`
	Config  string `cli:"-c, --config"`

	Driver string `cli:"-T, --type" env:"SHIELD_DB_TYPE"`
	DSN    string `cli:"-d, --dsn"  env:"SHIELD_DB_DSN"`

	Tenants struct {
	} `cli:"tenants"`

	NewTenant struct {
		UUID string `cli:"-U, --uuid"`
		Name string `cli:"-N, --name"`
	} `cli:"new-tenant"`

	UpdateTenant struct {
		UUID string `cli:"-U, --uuid"`
		Name string `cli:"-N, --name"`
	} `cli:"update-tenant"`

	Users struct {
		ShowMemberships bool `cli:"-l, --memberships"`
	} `cli:"users"`

	NewUser struct {
		UUID     string `cli:"-U, --uuid"`
		Name     string `cli:"-N, --name"`
		Username string `cli:"-u, --username"`
		Password string `cli:"-p, --password"`
	} `cli:"new-user"`

	UpdateUser struct {
		UUID     string `cli:"-U, --uuid"`
		Name     string `cli:"-N, --name"`
		Username string `cli:"-u, --username"`
		Password string `cli:"-p, --password"`
	} `cli:"update-user"`

	Invite struct {
		Role   string `cli:"-r, --role"`
		System bool   `cli:"-s, --system"`
		Tenant string `cli:"-t, --tenant"`
	} `cli:"invite"`

	Banish struct {
		System bool   `cli:"-s, --system"`
		Tenant string `cli:"-t, --tenant"`
	} `cli:"banish"`
}

func main() {
	var opt Options
	env.Override(&opt)

	command, args, err := cli.Parse(&opt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{%s}\n", err)
		os.Exit(1)
	}

	if command == "" && opt.Help {
		fmt.Fprintf(os.Stderr, `USAGE: shield-umc [options] <command> [options]

    -h, --help
    -v, --version

Commands:

    tenants        List all SHIELD tenants
    new-tenant     Create a new tenant
    update-tenant  Rename a tenant

    users          List all (local) users
    new-user       Create a new user
    update-user    Update a user's details

    invite         Invite one or more users to a tenant
    banish         Remove one of more users form a tenant

To get more in-depth command help,
run 'shield-umc -h command-name'
`)
		os.Exit(0)
	}

	if opt.Version {
		if Version == "" {
			fmt.Printf("shield-umc (development)%s\n", Version)
		} else {
			fmt.Printf("shield-umc v%s\n", Version)
		}
		os.Exit(0)
	}

	if opt.Driver == "" {
		fmt.Fprintf(os.Stderr, "@R{missing required --type option}\n")
		os.Exit(1)
	}
	if opt.DSN == "" {
		fmt.Fprintf(os.Stderr, "@R{missing required --dsn option}\n")
		os.Exit(1)
	}

	database := &db.DB{
		Driver: opt.Driver,
		DSN:    opt.DSN,
	}
	if err := database.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "@R{%s}\n", err)
		os.Exit(1)
	}
	defer database.Disconnect()

	switch command {
	case "tenants":
		tenants, err := database.GetAllTenants()
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{failed to list tenants: %s}\n", err)
			os.Exit(2)
		}
		for _, tenant := range tenants {
			fmt.Fprintf(os.Stdout, "%s   %s\n", tenant.UUID, tenant.Name)
		}
		os.Exit(0)

	case "new-tenant":
		if opt.NewTenant.Name == "" {
			fmt.Fprintf(os.Stderr, "@R{missing required --name option}\n")
			os.Exit(1)
		}
		if strings.ToLower(opt.NewTenant.Name) == "system" {
			fmt.Fprintf(os.Stderr, "@Y{system} is a reserved name; you cannot create a tenant named @Y{%s}\n", opt.NewTenant.Name)
			os.Exit(1)
		}
		id := uuid.NewRandom()
		if opt.NewTenant.UUID != "" {
			id = uuid.Parse(opt.NewTenant.UUID)
		}
		err := database.Exec(`INSERT INTO tenants (uuid, name) VALUES (?, ?)`, id.String(), opt.NewTenant.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{failed to create tenant %s/%s: %s}\n", id, opt.NewTenant.Name, err)
			os.Exit(2)
		}
		fmt.Fprintf(os.Stdout, "%s\n", id)
		os.Exit(0)

	case "update-tenant":
		if opt.UpdateTenant.Name == "" {
			fmt.Fprintf(os.Stderr, "@R{missing required --name option}\n")
			os.Exit(1)
		}
		if opt.UpdateTenant.UUID != "" {
			fmt.Fprintf(os.Stderr, "@R{missing required --uuid option}\n")
			os.Exit(1)
		}
		err := database.Exec(`UPDATE tenants SET name = ? WHERE uuid = ?`, opt.UpdateTenant.Name, opt.UpdateTenant.UUID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{failed to update tenant %s/%s: %s}\n", opt.UpdateTenant.UUID, opt.UpdateTenant.Name, err)
			os.Exit(2)
		}
		os.Exit(0)

	case "users":
		users, err := database.GetAllUsers(&db.UserFilter{Backend: "local"})
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{failed to list users: %s}\n", err)
			os.Exit(2)
		}

		if opt.Users.ShowMemberships {
			for _, user := range users {
				memberships, err := database.GetMembershipsForUser(user.UUID)
				if err != nil {
					fmt.Fprintf(os.Stderr, "@R{failed to list memberships for '%s@%s': %s}\n",
						user.Account, user.Backend, err)
					continue
				}

				fmt.Fprintf(os.Stdout, "@C{%s@%s} (@M{%s}):\n", user.Account, user.Backend, user.Name)
				sys := "(system)"
				if user.SysRole != "" {
					fmt.Fprintf(os.Stdout, "    @Y{%-20s}  %-36s  %s\n", user.SysRole, sys, sys)
				} else {
					fmt.Fprintf(os.Stdout, "    @Y{%-20s}  %-36s  %s\n", "(none)", sys, sys)
				}
				for _, membership := range memberships {
					fmt.Fprintf(os.Stdout, "    @G{%-20s}  %s  %s\n",
						membership.Role, membership.TenantUUID, membership.TenantName)
				}
				fmt.Fprintf(os.Stdout, "\n")
			}
		} else {
			for _, user := range users {
				fmt.Fprintf(os.Stdout, "%s   %-20s   %s\n", user.UUID, user.Account, user.Name)
			}
		}
		os.Exit(0)

	case "new-user":
		if opt.NewUser.Username == "" {
			fmt.Fprintf(os.Stderr, "@R{missing required --username option}\n")
			os.Exit(1)
		}
		if opt.NewUser.Password == "" {
			fmt.Fprintf(os.Stderr, "@R{missing required --password option}\n")
			os.Exit(1)
		}
		if opt.NewUser.Name == "" {
			opt.NewUser.Name = opt.NewUser.Username
		}
		id := uuid.NewRandom()
		if opt.NewUser.UUID != "" {
			id = uuid.Parse(opt.NewUser.UUID)
		}
		user := &db.User{
			UUID:    id,
			Name:    opt.NewUser.Name,
			Account: opt.NewUser.Username,
			Backend: "local",
		}
		user.SetPassword(opt.NewUser.Password)
		if _, err := database.CreateUser(user); err != nil {
			fmt.Fprintf(os.Stderr, "@R{failed to create user %s/%s: %s}\n", opt.NewUser.UUID, opt.NewUser.Username, err)
			os.Exit(2)
		}
		os.Exit(0)

	case "update-user":

	case "invite":
		if opt.Invite.Role == "" {
			fmt.Fprintf(os.Stderr, "@R{missing required --role option}\n")
			os.Exit(1)
		}
		if !opt.Invite.System && opt.Invite.Tenant == "" {
			fmt.Fprintf(os.Stderr, "@R{either --tenant UUID must be given, or --system must be specified}\n")
			os.Exit(1)
		}
		if len(args) < 1 {
			fmt.Fprintf(os.Stderr, "@R{no invitee (user) UUIDs specified}\n")
			os.Exit(1)
		}

		rc := 0
		if opt.Invite.System {
			for _, id := range args {
				user, err := database.GetUserByID(finduser(database, id))
				if err != nil {
					fmt.Fprintf(os.Stderr, "@R{failed to set system role to %s for %s: %s}\n",
						opt.Invite.Role, id, err)
					rc = 1
					continue
				}
				if user == nil {
					fmt.Fprintf(os.Stderr, "@R{failed to set system role to %s for %s: no such user}\n",
						opt.Invite.Role, id)
					rc = 1
					continue
				}

				user.SysRole = opt.Invite.Role
				err = database.UpdateUser(user)
				if err != nil {
					fmt.Fprintf(os.Stderr, "@R{failed to set system role to %s for %s: %s}\n",
						opt.Invite.Role, id, err)
					rc = 1
					continue
				}

				fmt.Fprintf(os.Stderr, "Set @Y{system} role to @C{%s} for @C{%s}\n",
					opt.Invite.Role, id)
			}

		} else {
			tenant := findtenant(database, opt.Invite.Tenant)
			for _, id := range args {
				uid := finduser(database, id)
				err := database.AddUserToTenant(uid, tenant, opt.Invite.Role)
				if err != nil {
					fmt.Fprintf(os.Stderr, "@R{failed to invite %s to be a %s on %s: %s}\n",
						id, opt.Invite.Role, opt.Invite.Tenant, err)
					rc = 1
					continue
				}
				fmt.Fprintf(os.Stderr, "Added @C{%s} as a %s on @C{%s}\n", id, opt.Invite.Role, opt.Invite.Tenant)
			}
		}
		os.Exit(rc)

	case "banish":
		if !opt.Banish.System && opt.Banish.Tenant == "" {
			fmt.Fprintf(os.Stderr, "@R{either --tenant UUID must be given, or --system must be specified}\n")
			os.Exit(1)
		}
		if len(args) < 1 {
			fmt.Fprintf(os.Stderr, "@R{no user UUIDs specified; who shall I banish?}\n")
			os.Exit(1)
		}

		rc := 0
		if opt.Banish.System {
			for _, id := range args {
				user, err := database.GetUserByID(finduser(database, id))
				if err != nil {
					fmt.Fprintf(os.Stderr, "@R{failed to remove system role from %s: %s}\n", id, err)
					rc = 1
					continue
				}
				if user == nil {
					fmt.Fprintf(os.Stderr, "@R{failed to set system role to %s for %s: no such user}\n",
						opt.Invite.Role, id)
					rc = 1
					continue
				}

				user.SysRole = ""
				err = database.UpdateUser(user)
				if err != nil {
					fmt.Fprintf(os.Stderr, "@R{failed to remove system role from %s: %s}\n", id, err)
					rc = 1
					continue
				}

				fmt.Fprintf(os.Stderr, "Removed @Y{system} role from @Y{%s}\n", id)
			}

		} else {
			tenant := findtenant(database, opt.Banish.Tenant)
			for _, id := range args {
				err := database.RemoveUserFromTenant(id, tenant)
				if err != nil {
					fmt.Fprintf(os.Stderr, "@R{failed to banish %s from %s: %s}\n",
						id, opt.Banish.Tenant, err)
					rc = 1
					continue
				}
				fmt.Fprintf(os.Stderr, "Banished @Y{%s} from @Y{%s}\n", id, opt.Banish.Tenant)
			}
		}
		os.Exit(rc)

	default:
		fmt.Fprintf(os.Stderr, "command @Y{%s} is not implemented\n")
		os.Exit(2)
	}
}
