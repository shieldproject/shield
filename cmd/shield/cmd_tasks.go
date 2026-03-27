package main

import (
	"os"
	"time"

	fmt "github.com/jhunt/go-ansi"
	"github.com/jhunt/go-table"
	"github.com/spf13/cobra"

	"github.com/shieldproject/shield/client/v2/shield"
	"github.com/shieldproject/shield/tui"
)

var (
	tasksStatus   string
	tasksActive   bool
	tasksInactive bool
	tasksAll      bool
	tasksTarget   string
	tasksStore    string
	tasksType     string
	tasksLimit    int
	tasksBefore   string
)

var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "List tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		required(!(tasksActive && tasksInactive),
			"The --active and --inactive options are mutually exclusive.")
		required(!(tasksAll && tasksInactive),
			"The --all and --inactive options are mutually exclusive.")
		required(!(tasksAll && tasksActive),
			"The --all and --active options are mutually exclusive.")
		required(len(args) <= 0, "Too many arguments.")
		required(tasksTarget == "" || optTenant != "",
			"You must select a tenant (via --tenant) if you want to filter by target / system")

		switch tasksStatus {
		case "":
			// not specified; ok
		case "all":
			tasksAll = true
		case "pending", "scheduled", "running", "canceled", "failed", "done":
			// valid
		default:
			fail(3, "Invalid --status value of '%s'\n(must be one of all, pending, running,\n canceled, failed, or done).", tasksStatus)
		}

		if tasksAll {
			tasksStatus = ""
		}

		c := clientFromConfig()

		var tenant *shield.Tenant
		var err error
		if optTenant != "" {
			tenant, err = c.FindMyTenant(optTenant, true)
			bail(err)

			if tasksTarget != "" {
				t, err := c.FindTarget(tenant, tasksTarget, !optExact)
				bail(err)
				tasksTarget = t.UUID
			}

			if tasksStore != "" {
				s, err := c.FindStore(tenant, tasksStore, !optExact)
				bail(err)
				tasksStore = s.UUID
			}
		} else {
			if tasksStore != "" {
				s, err := c.FindGlobalStore(tasksStore, !optExact)
				bail(err)
				tasksStore = s.UUID
			}
		}

		var timeBefore int64
		if tasksBefore != "" {
			timeBefore = strptime(tasksBefore)
		} else {
			timeBefore = time.Now().Unix()
		}

		filter := &shield.TaskFilter{
			Status: tasksStatus,
			Limit:  &tasksLimit,
			Target: tasksTarget,
			Store:  tasksStore,
			Type:   tasksType,
			Before: timeBefore,
		}

		if tasksActive || tasksInactive {
			filter.Active = &tasksActive
		}

		tasks, err := c.ListTasks(tenant, filter)
		bail(err)

		if tasksLimit > 30 {
			for i := 0; i < tasksLimit/30; i++ {
				f := &shield.TaskFilter{
					Status: tasksStatus,
					Limit:  &tasksLimit,
					Target: tasksTarget,
					Type:   tasksType,
					Before: timeBefore,
				}

				if tasksActive || tasksInactive {
					f.Active = &tasksActive
				}

				sometasks, err := c.ListTasks(tenant, f)
				bail(err)
				if len(sometasks) < 30 {
					break
				}

				tasks = append(tasks, sometasks...)
				timeBefore = tasks[len(tasks)-1].RequestedAt
				time.Sleep(100 * time.Millisecond)
			}
		}

		if optJSON {
			fmt.Printf("%s\n", asJSON(tasks))
			return nil
		}

		tbl := table.NewTable("UUID", "Type", "Status", "Owner", "Requested at", "Started at", "Completed at")
		for _, task := range tasks {
			started := "(pending)"
			stopped := "(not yet started)"
			if task.StartedAt != 0 {
				stopped = "(running)"
				started = strftime(task.StartedAt)
			}
			if task.StoppedAt != 0 {
				stopped = strftime(task.StoppedAt)
			}
			tbl.Row(task, uuid8full(task.UUID, optLong), task.Type, task.Status, task.Owner, strftime(task.RequestedAt), started, stopped)
		}
		tbl.Output(os.Stdout)
		return nil
	},
}

var taskCmd = &cobra.Command{
	Use:   "task UUID",
	Short: "Show a single task",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			fail(2, "Usage: shield task UUID\n")
		}

		c := clientFromConfig()

		var tenant *shield.Tenant
		var err error
		if optTenant != "" {
			tenant, err = c.FindMyTenant(optTenant, true)
			bail(err)
		}

		task, err := c.FindTask(tenant, args[0], !optExact)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(task))
			return nil
		}

		r := tui.NewReport()
		r.Add("UUID", task.UUID)
		r.Add("Owner", task.Owner)
		r.Add("Type", task.Type)
		r.Add("Status", task.Status)
		r.Break()

		started := "(pending)"
		stopped := "(not yet started)"
		if task.StartedAt != 0 {
			stopped = "(running)"
			started = strftime(task.StartedAt)
		}
		if task.StoppedAt != 0 {
			stopped = strftime(task.StoppedAt)
		}
		r.Add("Started at", started)
		r.Add("Stopped at", stopped)
		r.Break()

		if job, err := c.GetJob(tenant, task.JobUUID); err == nil && job != nil {
			r.Add("Job", fmt.Sprintf("%s (%s)", job.Name, task.JobUUID))
		}
		if task.ArchiveUUID != "" {
			r.Add("Archive UUID", task.ArchiveUUID)
		}
		r.Break()

		r.Add("Log", task.Log)
		r.Output(os.Stdout)
		return nil
	},
}

var cancelCmd = &cobra.Command{
	Use:   "cancel UUID",
	Short: "Cancel a running task",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			fail(2, "Usage: shield cancel -t TENANT UUID\n")
		}
		required(optTenant != "", "Missing required --tenant option.")

		c := clientFromConfig()
		tenant, err := c.FindMyTenant(optTenant, true)
		bail(err)

		task, err := c.FindTask(tenant, args[0], !optExact)
		bail(err)

		r := tui.NewReport()
		r.Add("Owner", task.Owner)
		r.Add("Type", task.Type)
		r.Add("Status", task.Status)
		r.Break()

		started := "(pending)"
		stopped := "(not yet started)"
		if task.StartedAt != 0 {
			stopped = "(running)"
			started = strftime(task.StartedAt)
		}
		if task.StoppedAt != 0 {
			stopped = strftime(task.StoppedAt)
		}
		r.Add("Started at", started)
		r.Add("Stopped at", stopped)
		r.Break()

		if job, err := c.GetJob(tenant, task.JobUUID); err == nil {
			r.Add("Job", fmt.Sprintf("%s (%s)", job.Name, task.JobUUID))
		}
		if task.ArchiveUUID != "" {
			r.Add("Archive UUID", task.ArchiveUUID)
		}
		r.Output(os.Stdout)

		if task.StoppedAt != 0 {
			fail(1, "This task cannot be cancelled, as it has already completed.\n")
		}
		if !confirm(optYes, "Cancel this task?") {
			return nil
		}
		rs, err := c.CancelTask(tenant, task)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(rs))
			return nil
		}
		fmt.Printf("%s\n", rs.OK)
		return nil
	},
}

func init() {
	tasksCmd.Flags().StringVarP(&tasksStatus, "status", "s", "", "Filter by status (pending, running, canceled, failed, done, all)")
	tasksCmd.Flags().BoolVar(&tasksActive, "active", false, "Show only active tasks")
	tasksCmd.Flags().BoolVar(&tasksInactive, "inactive", false, "Show only inactive tasks")
	tasksCmd.Flags().BoolVarP(&tasksAll, "all", "a", false, "Show all tasks regardless of status")
	tasksCmd.Flags().StringVar(&tasksTarget, "target", "", "Filter by target data system")
	tasksCmd.Flags().StringVar(&tasksStore, "store", "", "Filter by cloud storage system")
	tasksCmd.Flags().StringVar(&tasksType, "type", "", "Filter by task type")
	tasksCmd.Flags().IntVarP(&tasksLimit, "limit", "l", 30, "Limit number of results")
	tasksCmd.Flags().StringVar(&tasksBefore, "before", "", "Show tasks before this time")

	rootCmd.AddCommand(tasksCmd)
	rootCmd.AddCommand(taskCmd)
	rootCmd.AddCommand(cancelCmd)
}
