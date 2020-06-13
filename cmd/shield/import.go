package main

import (
	fmt "github.com/jhunt/go-ansi"

	"github.com/shieldproject/shield/client/v2/shield"
)

type ImportManifest struct {
	Core               string `yaml:"core"`
	Token              string `yaml:"token"`
	CA                 string `yaml:"ca"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`

	Users []struct {
		Name       string `yaml:"name"`
		Username   string `yaml:"username"`
		Password   string `yaml:"password"`
		SystemRole string `yaml:"sysrole"`
	} `yaml:"users,omitempty"`

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
			Bucket   string `yaml:"bucket"`
			FixedKey bool   `yaml:"fixed_key"`
			Paused   bool   `yaml:"paused"`
		} `yaml:"jobs,omitempty"`
	} `yaml:"systems,omitempty"`
}

func (m *ImportManifest) Normalize() error {
	for i, system := range m.Systems {
		if system.Name == "" {
			return fmt.Errorf("Data system #%d is missing its `name' attribute", i+1)
		}
		if system.Agent == "" {
			return fmt.Errorf("Data system '%s' is missing its `agent' attribute", system.Name)
		}
		if system.Plugin == "" {
			return fmt.Errorf("Data system '%s' is missing its `plugin' attribute", system.Name)
		}

		if len(system.Jobs) == 0 {
			return fmt.Errorf("Data system '%s' has no jobs defined", system.Name)
		}
		for k, job := range system.Jobs {
			if job.Name == "" {
				return fmt.Errorf("Data system '%s', job #%d is missing its `name' attribute", system.Name, k+1)
			}
			if job.When == "" {
				return fmt.Errorf("Data system '%s', job '%s' is missing its `when' attribute", system.Name, job.Name)
			}
			if job.Retain == "" {
				return fmt.Errorf("Data system '%s', job '%s' is missing its `retain' attribute", system.Name, job.Name)
			}
			if job.Bucket == "" {
				return fmt.Errorf("Data system '%s', job '%s' is missing its `bucket' attribute", system.Name, job.Name)
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

	fmt.Printf("\n@G{Importing Data Systems...}\n")
	for _, system := range m.Systems {
		sys, _ := c.FindTarget(system.Name, false)
		if sys == nil {
			fmt.Printf("creating data system @M{%s}, using the @Y{%s} plugin\n", system.Name, system.Plugin)
			sys, err = c.CreateTarget(&shield.Target{
				Name:    system.Name,
				Summary: system.Summary,
				Agent:   system.Agent,
				Plugin:  system.Plugin,
				Config:  system.Config,
			})
			if err != nil {
				return fmt.Errorf("failed to create data system '%s': %s", system.Name, err)
			}
		} else {
			fmt.Printf("updating data system @M{%s}, using the @Y{%s} plugin\n", system.Name, system.Plugin)
			sys.Summary = system.Summary
			sys.Agent = system.Agent
			sys.Plugin = system.Plugin
			sys.Config = system.Config
			_, err := c.UpdateTarget(sys)
			if err != nil {
				return fmt.Errorf("failed to update data system '%s': %s", system.Name, err)
			}
		}

		for _, job := range system.Jobs {
			bucket, err := c.FindBucket(job.Bucket, false)
			if err != nil {
				return fmt.Errorf("unable to find storage bucket '%s' for '%s' job on data system '%s': %s", job.Bucket, job.Name, sys.Name, err)
			}

			ll, err := c.ListJobs(&shield.JobFilter{
				Fuzzy:  false,
				Name:   job.Name,
				Target: sys.UUID,
				Bucket: bucket.Key,
			})
			if err != nil || len(ll) == 0 {
				fmt.Printf("@C{%s}> creating @M{%s} job, running at @W{%s}\n", sys.Name, system.Name, job.Name, job.When)
				_, err = c.CreateJob(&shield.Job{
					Name:       job.Name,
					TargetUUID: sys.UUID,
					Bucket:     bucket.Key,
					Schedule:   job.When,
					Retain:     job.Retain,
					Paused:     job.Paused,
				})
				if err != nil {
					return fmt.Errorf("failed to configure job '%s' of data system '%s': %s", job.Name, system.Name, err)
				}
			} else if len(ll) > 1 {
				return fmt.Errorf("failed to configure job '%s' of data system '%s': too many matching jobs", job.Name, system.Name)
			} else {
				fmt.Printf("@C{%s}> updating @M{%s} job, running at @W{%s}\n", system.Name, job.Name, job.When)
				ll[0].Bucket = job.Bucket
				ll[0].Schedule = job.When
				ll[0].Retain = job.Retain
				_, err := c.UpdateJob(ll[0])
				if err != nil {
					return fmt.Errorf("failed to reconfigure job '%s' of data system '%s': %s", job.Name, system.Name, err)
				}
			}
		}
		fmt.Printf("\n")
	}
	return nil
}
