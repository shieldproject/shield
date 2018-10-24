package main

import (
	fmt "github.com/jhunt/go-ansi"
	"strings"

	"github.com/starkandwayne/shield/client/v2/shield"
)

type ImportManifest struct {
	Core               string `yaml:"core"`
	Token              string `yaml:"token"`
	CA                 string `yaml:"ca"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`

	Global struct {
		Storage []struct {
			Name    string `yaml:"name"`
			Summary string `yaml:"summary"`
			Agent   string `yaml:"agent"`
			Plugin  string `yaml:"plugin"`

			Config map[string]interface{} `yaml:"config"`
		} `yaml:"storage,omitempty"`

		Policies []struct {
			Name    string `yaml:"name"`
			Summary string `yaml:"summary"`
			Days    int    `yaml:"days"`
		} `yaml:"policies,omitempty"`
	} `yaml:"global,omitempty"`

	Users []struct {
		Name       string `yaml:"name"`
		Username   string `yaml:"username"`
		Password   string `yaml:"password"`
		SystemRole string `yaml:"sysrole"`

		Tenants []struct {
			Name string `yaml:"name"`
			Role string `yaml:"role"`
		} `yaml:"tenants,omitempty"`
	} `yaml:"users,omitempty"`

	Tenants []ImportTenant `yaml:"tenants,omitempty"`
}

type ImportTenant struct {
	Name string `yaml:"name"`

	Members []ImportMembership `yaml:"members"`

	Storage []struct {
		Name    string `yaml:"name"`
		Summary string `yaml:"summary"`
		Agent   string `yaml:"agent"`
		Plugin  string `yaml:"plugin"`

		Config map[string]interface{} `yaml:"config"`
	} `yaml:"storage,omitempty"`

	Policies []struct {
		Name    string `yaml:"name"`
		Summary string `yaml:"summary"`
		Days    int    `yaml:"days"`
	} `yaml:"policies,omitempty"`

	Systems []struct {
		Name    string `yaml:"name"`
		Summary string `yaml:"summary"`
		Agent   string `yaml:"agent"`
		Plugin  string `yaml:"plugin"`

		Config map[string]interface{} `yaml:"config"`

		Jobs []struct {
			Name     string `yaml:"name"`
			When     string `yaml:"when"`
			Retain   string `yaml:"retain"`
			Storage  string `yaml:"storage"`
			FixedKey bool   `yaml:"fixed_key"`
			Paused   bool   `yaml:"paused"`
		} `yaml:"jobs,omitempty"`
	} `yaml:"systems,omitempty"`
}

type ImportMembership struct {
	User string `yaml:"user"`
	Role string `yaml:"role"`
}

func (m *ImportManifest) Normalize() error {
	for i, tenant := range m.Tenants {
		/* tenant.name is required */
		if tenant.Name == "" {
			return fmt.Errorf("Tenant #%d is missing its `name' attribute", i+1)
		}

		for j, assign := range tenant.Members {
			/* user and role are required */
			if assign.User == "" {
				return fmt.Errorf("Tenant '%s', grant #%d is missing the `user' attribute", tenant.Name, j+1)
			}
			if assign.Role == "" {
				return fmt.Errorf("Tenant '%s', grant #%d to %s is missing the `role' attribute", tenant.Name, j+1, assign.User)
			}

			/* check that any membership grants are to @local */
			if !strings.HasSuffix(assign.User, "@local") {
				if strings.Contains(assign.User, "@") {
					return fmt.Errorf("Tenant '%s' defines membership for non-local user '%s'", tenant.Name, assign.User)
				} else {
					return fmt.Errorf("Tenant '%s' defines membership for non-local user '%s' (you forgot the @local)", tenant.Name, assign.User)
				}
			}
		}

		for j, storage := range tenant.Storage {
			if storage.Name == "" {
				return fmt.Errorf("Tenant '%s', storage system #%d is missing its `name' attribute", tenant.Name, j+1)
			}
			if storage.Agent == "" {
				return fmt.Errorf("Tenant '%s', storage system '%s' is missing its `agent' attribute", tenant.Name, storage.Name)
			}
			if storage.Plugin == "" {
				return fmt.Errorf("Tenant '%s', storage system '%s' is missing its `plugin' attribute", tenant.Name, storage.Name)
			}
		}

		for j, policy := range tenant.Policies {
			if policy.Name == "" {
				return fmt.Errorf("Tenant '%s', retention policy #%d is missing its `name' attribute", tenant.Name, j+1)
			}
			if policy.Days == 0 {
				return fmt.Errorf("Tenant '%s', retention policy '%s' is missing its `days' attribute", tenant.Name, policy.Name)
			}
		}

		for j, system := range tenant.Systems {
			if system.Name == "" {
				return fmt.Errorf("Tenant '%s', data system #%d is missing its `name' attribute", tenant.Name, j+1)
			}
			if system.Agent == "" {
				return fmt.Errorf("Tenant '%s', data system '%s' is missing its `agent' attribute", tenant.Name, system.Name)
			}
			if system.Plugin == "" {
				return fmt.Errorf("Tenant '%s', data system '%s' is missing its `plugin' attribute", tenant.Name, system.Name)
			}

			if len(system.Jobs) == 0 {
				return fmt.Errorf("Tenant '%s', data system '%s' has no jobs defined", tenant.Name, system.Name)
			}
			for k, job := range system.Jobs {
				if job.Name == "" {
					return fmt.Errorf("Tenant '%s', data system '%s', job #%d is missing its `name' attribute", tenant.Name, system.Name, k+1)
				}
				if job.When == "" {
					return fmt.Errorf("Tenant '%s', data system '%s', job '%s' is missing its `when' attribute", tenant.Name, system.Name, job.Name)
				}
				if job.Retain == "" {
					return fmt.Errorf("Tenant '%s', data system '%s', job '%s' is missing its `retain' attribute", tenant.Name, system.Name, job.Name)
				}
				if job.Storage == "" {
					return fmt.Errorf("Tenant '%s', data system '%s', job '%s' is missing its `storage' attribute", tenant.Name, system.Name, job.Name)
				}
			}
		}
	}

	/* convert $.global.users.tenants into autovivified $.tenants
	   entries, with appropriate .members[] sub-entries */
	for _, user := range m.Users {
		for _, memb := range user.Tenants {
			found := false
			for i, tenant := range m.Tenants {
				if tenant.Name == memb.Name {
					for _, assign := range tenant.Members {
						if user.Username+"@local" == assign.User {
							found = true
							if memb.Role != assign.Role {
								return fmt.Errorf("%s is assigned the %s role on %s via `users`, but is also assigned the %s role under the tenant definition", user.Username, memb.Role, tenant.Name, assign.Role)
							}
							break
						}
					}
					if !found {
						m.Tenants[i].Members = append(m.Tenants[i].Members, ImportMembership{
							user.Username + "@local",
							memb.Role,
						})
					}
					found = true
					break
				}
			}
			if !found {
				t := ImportTenant{Name: memb.Name}
				t.Members = make([]ImportMembership, 1)
				t.Members[0].User = user.Username
				t.Members[0].Role = memb.Role
				m.Tenants = append(m.Tenants, t)
			}
		}
	}

	return nil
}

func (m *ImportManifest) Deploy(c *shield.Client) error {
	if m.Core == "" {
		return fmt.Errorf("Missing requird 'core' top-level key in the import manifest.\n")
	}
	if m.Token == "" {
		return fmt.Errorf("Missing requird 'token' top-level key in the import manifest.\n")
	}

	fmt.Printf("@W{Connecting to }@G{%s}\n", m.Core)
	c = &shield.Client{
		URL:                m.Core,
		Session:            m.Token,
		CACertificate:      m.CA,
		InsecureSkipVerify: m.InsecureSkipVerify,

		/* stuff we keep from the passed client */
		Debug: c.Debug,
		Trace: c.Trace,
	}

	id, err := c.AuthID()
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %s\n", c.URL, err)
	}
	if id.Unauthenticated {
		return fmt.Errorf("authentication to %s failed!", c.URL)
	}
	if !id.Is.System.Manager {
		return fmt.Errorf("insufficient system privileges (you must be a system manager or admin)")
	}

	fmt.Printf("@G{Importing Users...}\n")
	for _, user := range m.Users {
		u, _ := c.FindUser(user.Username, false)
		if u == nil {
			if user.SystemRole == "" {
				fmt.Printf("creating user @C{%s} (%s) with no system role\n", user.Username, user.Name)
			} else {
				fmt.Printf("creating user @C{%s} (%s) with the @Y{%s} system role\n", user.Username, user.Name, user.SystemRole)
			}
			_, err = c.CreateUser(&shield.User{
				Name:     user.Name,
				Account:  user.Username,
				Password: user.Password,
				SysRole:  user.SystemRole,
			})
			if err != nil {
				return fmt.Errorf("failed to create local user '%s': %s", user.Username, err)
			}
		} else {
			fmt.Printf("creating user @C{%s} (%s)\n", user.Username, user.Name)
			u.Name = user.Name
			_, err = c.UpdateUser(u)
			if err != nil {
				return fmt.Errorf("failed to update local user '%s': %s", user.Username, err)
			}
		}
	}

	fmt.Printf("\n@G{Importing Shared Storage...}\n")
	for _, storage := range m.Global.Storage {
		s, _ := c.FindGlobalStore(storage.Name, false)
		if s == nil {
			fmt.Printf("creating cloud storage system @M{%s}, using the @Y{%s} plugin\n", storage.Name, storage.Plugin)
			_, err = c.CreateGlobalStore(&shield.Store{
				Name:    storage.Name,
				Summary: storage.Summary,
				Agent:   storage.Agent,
				Plugin:  storage.Plugin,
				Config:  storage.Config,
			})
			if err != nil {
				return fmt.Errorf("failed to create storage system '%s': %s", storage.Name, err)
			}
		} else {
			fmt.Printf("updating cloud storage system @M{%s}, using the @Y{%s} plugin\n", storage.Name, storage.Plugin)
			s.Summary = storage.Summary
			s.Agent = storage.Agent
			s.Plugin = storage.Plugin
			s.Config = storage.Config
			_, err := c.UpdateGlobalStore(s)
			if err != nil {
				return fmt.Errorf("failed to update storage system '%s': %s", storage.Name, err)
			}
		}
	}

	fmt.Printf("\n@G{Importing Tenants...}\n")
	for _, tenant := range m.Tenants {
		fmt.Printf("importing tenant @C{%s}...\n", tenant.Name)
		t, _ := c.FindTenant(tenant.Name, false)
		if t == nil {
			fmt.Printf("creating tenant @C{%s}\n", tenant.Name)
			t, err = c.CreateTenant(&shield.Tenant{
				Name: tenant.Name,
			})
			if err != nil {
				return fmt.Errorf("failed to create tenant '%s': %s", tenant.Name, err)
			}
		}

		fmt.Printf("@C{%s}> @G{inviting local users...}\n", tenant.Name)
		for _, assign := range tenant.Members {
			fmt.Printf("@C{%s}> inviting @M{%s} in the @Y{%s} role\n", tenant.Name, assign.User, assign.Role)
			user, err := c.ListUsers(&shield.UserFilter{
				Fuzzy:   false,
				Account: strings.TrimSuffix(assign.User, "@local"),
			})
			if err != nil || len(user) == 0 {
				return fmt.Errorf("unable to find local SHIELD user '%s': %s", assign.User, err)
			} else if len(user) > 1 {
				return fmt.Errorf("multiple local SHIELD users name '%s' found!", assign.User)
			}
			_, err = c.Invite(t, assign.Role, user)
			if err != nil {
				return fmt.Errorf("unable to invite local SHIELD user '%s' to '%s' as role '%s': %s", assign.User, tenant.Name, assign.Role, err)
			}
		}

		fmt.Printf("@C{%s}> @G{importing cloud storage systems...}\n", tenant.Name)
		for _, storage := range tenant.Storage {
			s, _ := c.FindStore(t, storage.Name, false)
			if s == nil {
				fmt.Printf("@C{%s}> creating cloud storage system @M{%s}, using the @Y{%s} plugin\n", tenant.Name, storage.Name, storage.Plugin)
				_, err = c.CreateStore(t, &shield.Store{
					Name:    storage.Name,
					Summary: storage.Summary,
					Agent:   storage.Agent,
					Plugin:  storage.Plugin,
					Config:  storage.Config,
				})
				if err != nil {
					return fmt.Errorf("failed to create storage system '%s' for tenant '%s': %s", storage.Name, tenant.Name, err)
				}
			} else {
				fmt.Printf("@C{%s}> updating cloud storage system @M{%s}, using the @Y{%s} plugin\n", tenant.Name, storage.Name, storage.Plugin)
				s.Summary = storage.Summary
				s.Agent = storage.Agent
				s.Plugin = storage.Plugin
				s.Config = storage.Config
				_, err := c.UpdateStore(t, s)
				if err != nil {
					return fmt.Errorf("failed to update storage system '%s' on tenant '%s': %s", storage.Name, tenant.Name, err)
				}
			}
		}

		fmt.Printf("@C{%s}> @G{importing data systems...}\n", tenant.Name)
		for _, system := range tenant.Systems {
			sys, _ := c.FindTarget(t, system.Name, false)
			if sys == nil {
				fmt.Printf("@C{%s}> creating data system @M{%s}, using the @Y{%s} plugin\n", tenant.Name, system.Name, system.Plugin)
				sys, err = c.CreateTarget(t, &shield.Target{
					Name:    system.Name,
					Summary: system.Summary,
					Agent:   system.Agent,
					Plugin:  system.Plugin,
					Config:  system.Config,
				})
				if err != nil {
					return fmt.Errorf("failed to create data system '%s' for tenant '%s': %s", system.Name, tenant.Name, err)
				}
			} else {
				fmt.Printf("@C{%s}> updating data system @M{%s}, using the @Y{%s} plugin\n", tenant.Name, system.Name, system.Plugin)
				sys.Summary = system.Summary
				sys.Agent = system.Agent
				sys.Plugin = system.Plugin
				sys.Config = system.Config
				_, err := c.UpdateTarget(t, sys)
				if err != nil {
					return fmt.Errorf("failed to update data system '%s' on tenant '%s': %s", system.Name, tenant.Name, err)
				}
			}

			for _, job := range system.Jobs {
				store, err := c.FindUsableStore(t, job.Storage, false)
				if err != nil {
					return fmt.Errorf("unable to find storage system '%s' for '%s' job on tenant '%s': %s", job.Storage, job.Name, tenant.Name, err)
				}

				ll, err := c.ListJobs(t, &shield.JobFilter{
					Fuzzy:  false,
					Name:   job.Name,
					Target: sys.UUID,
					Store:  store.UUID,
				})
				if err != nil || len(ll) == 0 {
					fmt.Printf("@C{%s} :: @B{%s}> creating @M{%s} job, running at @W{%s}\n", tenant.Name, system.Name, job.Name, job.When)
					_, err = c.CreateJob(t, &shield.Job{
						Name:       job.Name,
						TargetUUID: sys.UUID,
						StoreUUID:  store.UUID,
						Schedule:   job.When,
						Retain:     job.Retain,
						FixedKey:   job.FixedKey,
						Paused:     job.Paused,
					})
					if err != nil {
						return fmt.Errorf("failed to configure job '%s' of data system '%s' for tenant '%s': %s", job.Name, system.Name, tenant.Name, err)
					}
				} else if len(ll) > 1 {
					return fmt.Errorf("failed to configure job '%s' of data system '%s' for tenant '%s': too many matching jobs", job.Name, system.Name, tenant.Name)
				} else {
					fmt.Printf("@C{%s} :: @B{%s}> updating @M{%s} job, running at @W{%s}\n", tenant.Name, system.Name, job.Name, job.When)
					ll[0].StoreUUID = store.UUID
					ll[0].Schedule = job.When
					ll[0].Retain = job.Retain
					_, err := c.UpdateJob(t, ll[0])
					if err != nil {
						return fmt.Errorf("failed to reconfigure job '%s' of data system '%s' on tenant '%s': %s", job.Name, system.Name, tenant.Name, err)
					}
				}
			}
		}
		fmt.Printf("\n")
	}
	return nil
}
