package main

import (
	"os"
	"strconv"

	fmt "github.com/jhunt/go-ansi"
	"github.com/jhunt/go-table"
	"github.com/spf13/cobra"

	"github.com/shieldproject/shield/client/v2/shield"
	"github.com/shieldproject/shield/tui"
)

var (
	jobsStore    string
	jobsTarget   string
	jobsPaused   bool
	jobsUnpaused bool

	createJobName     string
	createJobSummary  string
	createJobTarget   string
	createJobStore    string
	createJobSchedule string
	createJobRetain   string
	createJobPaused   bool
	createJobFixedKey bool
	createJobRetries  int

	updateJobName       string
	updateJobSummary    string
	updateJobTarget     string
	updateJobStore      string
	updateJobSchedule   string
	updateJobRetain     string
	updateJobFixedKey   bool
	updateJobNoFixedKey bool
	updateJobRetries    int
)

var jobsCmd = &cobra.Command{
	Use:   "jobs [NAME-or-UUID]",
	Short: "List backup jobs",
	RunE: func(cmd *cobra.Command, args []string) error {
		required(optTenant != "", "Missing required --tenant option.")
		required(!(jobsPaused && jobsUnpaused),
			"The --paused and --unpaused options are mutually exclusive.")
		required(len(args) <= 1, "Too many arguments.")

		c := clientFromConfig()
		filter := &shield.JobFilter{
			Fuzzy:  !optExact,
			Store:  jobsStore,
			Target: jobsTarget,
		}
		if jobsPaused || jobsUnpaused {
			filter.Paused = &jobsPaused
		}
		if len(args) == 1 {
			filter.Name = args[0]
			filter.UUID = args[0]
		}

		tenant, err := c.FindMyTenant(optTenant, true)
		bail(err)

		jobs, err := c.ListJobs(tenant, filter)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(jobs))
			return nil
		}

		tbl := table.NewTable("UUID", "Name", "Summary", "Schedule", "Status", "Retention", "SHIELD Agent", "Target", "Store", "Fixed-Key", "Retries")
		for _, job := range jobs {
			tbl.Row(job, uuid8full(job.UUID, optLong), job.Name, wrap(job.Summary, 35), job.Schedule, job.Status(), fmt.Sprintf("%dd (%d archives)", job.KeepDays, job.KeepN), job.Agent, job.Target.Name, job.Store.Name, job.FixedKey, job.Retries)
		}
		tbl.Output(os.Stdout)
		return nil
	},
}

var jobCmd = &cobra.Command{
	Use:   "job NAME-or-UUID",
	Short: "Show a single backup job",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			fail(2, "Usage: shield job NAME-or-UUID\n")
		}
		required(optTenant != "", "Missing required --tenant option.")

		c := clientFromConfig()
		tenant, err := c.FindMyTenant(optTenant, true)
		bail(err)

		job, err := c.FindJob(tenant, args[0], !optExact)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(job))
			return nil
		}

		r := tui.NewReport()
		r.Add("UUID", job.UUID)
		r.Add("Name", job.Name)
		r.Add("Status", job.Status())
		r.Break()

		r.Add("Schedule", job.Schedule)
		r.Add("Keep", fmt.Sprintf("%d days (%d archives)", job.KeepDays, job.KeepN))
		r.Add("Retries", fmt.Sprintf("%d tries", job.Retries))
		r.Break()

		r.Add("Data System", job.Target.Name)
		r.Add("Backup Plugin", job.Target.Plugin)
		r.Add("SHIELD Agent", job.Agent)
		r.Break()

		r.Add("Cloud Storage", job.Store.Name)
		r.Add("Storage Plugin", job.Store.Plugin)
		r.Break()

		r.Add("Fixed-Key", strconv.FormatBool(job.FixedKey))
		r.Add("Notes", job.Summary)

		r.Output(os.Stdout)
		return nil
	},
}

var createJobCmd = &cobra.Command{
	Use:   "create-job",
	Short: "Create a new backup job",
	RunE: func(cmd *cobra.Command, args []string) error {
		required(optTenant != "", "Missing required --tenant option.")

		c := clientFromConfig()
		tenant, err := c.FindMyTenant(optTenant, true)
		bail(err)

		if !optBatch {
			if createJobName == "" {
				createJobName = prompt("@C{Job Name}: ")
			}
			if createJobSummary == "" {
				createJobSummary = prompt("@C{Notes}: ")
			}
			for createJobTarget == "" {
				id := prompt("@C{Target Data System}: ")
				if len(id) > 0 && id[0] == '?' {
					SearchTargets(c, tenant, id[1:])
					continue
				}
				if target, err := c.FindTarget(tenant, id, !optExact); err != nil {
					fmt.Fprintf(os.Stderr, "@Y{%s}\n", err)
				} else {
					createJobTarget = target.UUID
					break
				}
			}
			for createJobStore == "" {
				id := prompt("@C{Cloud Storage}: ")
				if len(id) > 0 && id[0] == '?' {
					SearchStores(c, tenant, id[1:])
					continue
				}
				if store, err := c.FindUsableStore(tenant, id, !optExact); err != nil {
					fmt.Fprintf(os.Stderr, "@Y{%s}\n", err)
				} else {
					createJobStore = store.UUID
					break
				}
			}
			if createJobSchedule == "" {
				createJobSchedule = prompt("@C{Schedule}: ")
			}
			if createJobRetain == "" {
				createJobRetain = prompt("@C{Retain}: ")
			}
		} else {
			if id := createJobTarget; id != "" {
				if target, err := c.FindTarget(tenant, id, !optExact); err != nil {
					bail(err)
				} else {
					createJobTarget = target.UUID
				}
			}
			if id := createJobStore; id != "" {
				if store, err := c.FindUsableStore(tenant, id, !optExact); err != nil {
					bail(err)
				} else {
					createJobStore = store.UUID
				}
			}
		}

		job, err := c.CreateJob(tenant, &shield.Job{
			Name:       createJobName,
			Summary:    createJobSummary,
			TargetUUID: createJobTarget,
			StoreUUID:  createJobStore,
			Schedule:   createJobSchedule,
			Retain:     createJobRetain,
			Retries:    createJobRetries,
			Paused:     createJobPaused,
			FixedKey:   createJobFixedKey,
		})
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(job))
			return nil
		}

		r := tui.NewReport()
		r.Add("UUID", job.UUID)
		r.Add("Name", job.Name)
		r.Add("Summary", job.Summary)
		r.Output(os.Stdout)
		return nil
	},
}

var updateJobCmd = &cobra.Command{
	Use:   "update-job NAME-or-UUID",
	Short: "Update an existing backup job",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			fail(2, "Usage: shield update-job -t TENANT [OPTIONS] NAME-or-UUID\n")
		}
		required(optTenant != "", "Missing required --tenant option.")

		c := clientFromConfig()
		tenant, err := c.FindMyTenant(optTenant, true)
		bail(err)

		job, err := c.FindJob(tenant, args[0], !optExact)
		bail(err)

		if updateJobName != "" {
			job.Name = updateJobName
		}
		if updateJobSummary != "" {
			job.Summary = updateJobSummary
		}
		if id := updateJobTarget; id != "" {
			if target, err := c.FindTarget(tenant, id, !optExact); err != nil {
				bail(err)
			} else {
				job.TargetUUID = target.UUID
			}
		}
		if id := updateJobStore; id != "" {
			if store, err := c.FindUsableStore(tenant, id, !optExact); err != nil {
				bail(err)
			} else {
				job.StoreUUID = store.UUID
			}
		}
		if updateJobSchedule != "" {
			job.Schedule = updateJobSchedule
		}
		if updateJobRetain != "" {
			job.Retain = updateJobRetain
		}
		job.Retries = updateJobRetries

		if updateJobFixedKey {
			job.FixedKey = true
		}
		if updateJobNoFixedKey {
			job.FixedKey = false
		}

		_, err = c.UpdateJob(tenant, job)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(job))
			return nil
		}

		r := tui.NewReport()
		r.Add("UUID", job.UUID)
		r.Add("Name", job.Name)
		r.Add("Summary", job.Summary)
		r.Output(os.Stdout)
		return nil
	},
}

var deleteJobCmd = &cobra.Command{
	Use:   "delete-job NAME-or-UUID",
	Short: "Delete a backup job",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			fail(2, "Usage: shield delete-job -t TENANT NAME-or-UUID\n")
		}
		required(optTenant != "", "Missing required --tenant option.")

		c := clientFromConfig()
		tenant, err := c.FindMyTenant(optTenant, true)
		bail(err)

		job, err := c.FindJob(tenant, args[0], true)
		bail(err)

		if !confirm(optYes, "Delete job @Y{%s} in tenant @Y{%s}?", job.Name, tenant.Name) {
			return nil
		}
		r, err := c.DeleteJob(tenant, job)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(r))
			return nil
		}
		fmt.Printf("%s\n", r.OK)
		return nil
	},
}

var pauseJobCmd = &cobra.Command{
	Use:   "pause-job NAME-or-UUID",
	Short: "Pause a backup job",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			fail(2, "Usage: shield pause-job -t TENANT NAME-or-UUID\n")
		}
		required(optTenant != "", "Missing required --tenant option.")

		c := clientFromConfig()
		tenant, err := c.FindMyTenant(optTenant, true)
		bail(err)

		job, err := c.FindJob(tenant, args[0], true)
		bail(err)

		if !confirm(optYes, "Pause job @Y{%s} in tenant @Y{%s}?", job.Name, tenant.Name) {
			return nil
		}
		r, err := c.PauseJob(tenant, job)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(r))
			return nil
		}
		fmt.Printf("%s\n", r.OK)
		return nil
	},
}

var unpauseJobCmd = &cobra.Command{
	Use:   "unpause-job NAME-or-UUID",
	Short: "Unpause a backup job",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			fail(2, "Usage: shield unpause-job -t TENANT NAME-or-UUID\n")
		}
		required(optTenant != "", "Missing required --tenant option.")

		c := clientFromConfig()
		tenant, err := c.FindMyTenant(optTenant, true)
		bail(err)

		job, err := c.FindJob(tenant, args[0], true)
		bail(err)

		if !confirm(optYes, "Unpause job @Y{%s} in tenant @Y{%s}?", job.Name, tenant.Name) {
			return nil
		}
		r, err := c.UnpauseJob(tenant, job)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(r))
			return nil
		}
		fmt.Printf("%s\n", r.OK)
		return nil
	},
}

var runJobCmd = &cobra.Command{
	Use:   "run-job NAME-or-UUID",
	Short: "Trigger an ad hoc run of a backup job",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			fail(2, "Usage: shield run-job -t TENANT NAME-or-UUID\n")
		}
		required(optTenant != "", "Missing required --tenant option.")

		c := clientFromConfig()
		tenant, err := c.FindMyTenant(optTenant, true)
		bail(err)

		job, err := c.FindJob(tenant, args[0], !optExact)
		bail(err)

		if !confirm(optYes, "Run job @Y{%s} in tenant @Y{%s}?", job.Name, tenant.Name) {
			return nil
		}
		r, err := c.RunJob(tenant, job)
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
	jobsCmd.Flags().StringVar(&jobsStore, "store", "", "Filter by cloud storage system")
	jobsCmd.Flags().StringVar(&jobsTarget, "target", "", "Filter by target data system")
	jobsCmd.Flags().BoolVar(&jobsPaused, "paused", false, "Show only paused jobs")
	jobsCmd.Flags().BoolVar(&jobsUnpaused, "unpaused", false, "Show only unpaused jobs")

	createJobCmd.Flags().StringVarP(&createJobName, "name", "n", "", "Job name")
	createJobCmd.Flags().StringVarP(&createJobSummary, "summary", "s", "", "Job notes/summary")
	createJobCmd.Flags().StringVar(&createJobTarget, "target", "", "Target data system")
	createJobCmd.Flags().StringVar(&createJobStore, "store", "", "Cloud storage system")
	createJobCmd.Flags().StringVar(&createJobSchedule, "schedule", "", "Backup schedule (cron)")
	createJobCmd.Flags().StringVar(&createJobRetain, "retain", "", "Retention policy")
	createJobCmd.Flags().BoolVar(&createJobPaused, "paused", false, "Create job in paused state")
	createJobCmd.Flags().BoolVar(&createJobFixedKey, "fixed-key", false, "Use fixed encryption key")
	createJobCmd.Flags().IntVar(&createJobRetries, "retries", 0, "Number of retries on failure")

	updateJobCmd.Flags().StringVarP(&updateJobName, "name", "n", "", "New job name")
	updateJobCmd.Flags().StringVarP(&updateJobSummary, "summary", "s", "", "New job notes/summary")
	updateJobCmd.Flags().StringVar(&updateJobTarget, "target", "", "New target data system")
	updateJobCmd.Flags().StringVar(&updateJobStore, "store", "", "New cloud storage system")
	updateJobCmd.Flags().StringVar(&updateJobSchedule, "schedule", "", "New backup schedule")
	updateJobCmd.Flags().StringVar(&updateJobRetain, "retain", "", "New retention policy")
	updateJobCmd.Flags().BoolVar(&updateJobFixedKey, "fixed-key", false, "Enable fixed encryption key")
	updateJobCmd.Flags().BoolVar(&updateJobNoFixedKey, "no-fixed-key", false, "Disable fixed encryption key")
	updateJobCmd.Flags().IntVar(&updateJobRetries, "retries", 0, "Number of retries on failure")

	rootCmd.AddCommand(jobsCmd)
	rootCmd.AddCommand(jobCmd)
	rootCmd.AddCommand(createJobCmd)
	rootCmd.AddCommand(updateJobCmd)
	rootCmd.AddCommand(deleteJobCmd)
	rootCmd.AddCommand(pauseJobCmd)
	rootCmd.AddCommand(unpauseJobCmd)
	rootCmd.AddCommand(runJobCmd)
}
