package core

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/jhunt/go-log"

	"github.com/starkandwayne/shield/core/bus"
	"github.com/starkandwayne/shield/core/scheduler"
	"github.com/starkandwayne/shield/core/vault"
	"github.com/starkandwayne/shield/db"
	"github.com/starkandwayne/shield/timespec"
)

func (c Core) Main() {
	/* we need a usable database first */
	c.ConnectToDatabase()
	c.ApplyFixups()
	c.ConfigureMessageBus()
	c.WireUpAuthenticationProviders()

	/* startup cleanup tasks */
	c.CreateFailsafeUser()
	c.ExpireInteractiveSessions()
	c.PrecreateTenants()
	c.CleanupLeftoverTasks()

	/* prepare for operation */
	c.ConnectToVault()
	c.Bind()
	c.StartScheduler()

	log.Infof("INITIALIZATION COMPLETE; entering main loop.")

	hype := time.NewTicker(125 * time.Millisecond)
	fast := time.NewTicker(time.Second * time.Duration(c.Config.Scheduler.FastLoop))
	slow := time.NewTicker(time.Second * time.Duration(c.Config.Scheduler.SlowLoop))
	for {
		select {
		case <-hype.C:
			if c.bailout {
				log.Infof("SHIELD BAILOUT triggered; exiting...")
				os.Exit(0)
			}
			c.scheduler.Run()

		case <-fast.C:
			if c.Unlocked() {
				c.ScheduleBackupTasks()
			}
			c.ScheduleAgentStatusCheckTasks(&db.AgentFilter{Status: "pending"})
			c.scheduler.Elevate()
			c.TasksToChores()

		case <-slow.C:
			c.CheckArchiveExpiries()
			c.SchedulePurgeTasks()
			c.MarkIrrelevantTasks()
			c.ScheduleAgentStatusCheckTasks(nil)
			c.AnalyzeStorage()
			c.CleanupOrphanedObjects()

			if c.Unlocked() {
				c.PurgeExpiredAPISessions()
				c.ScheduleStorageTestTasks()
			}
		}
	}
}

func (c *Core) ConnectToDatabase() {
	log.Infof("INITIALIZING: connecting to the database...")
	if c.db != nil {
		log.Alertf("ANOMALY: tried to connect to database, but we're already connected...")
		return
	}

	log.Debugf("connecting to database at %s...", c.DataFile("shield.db"))
	db, err := db.Connect(c.DataFile("shield.db"))
	c.MaybeTerminate(err)
	c.db = db

	log.Infof("INITIALIZING: checking database schema...")
	c.MaybeTerminate(c.db.CheckCurrentSchema())

	log.Debugf("connected successfully to database!")
}

func (c *Core) ApplyFixups() {
	err := c.db.ApplyFixups()
	c.MaybeTerminate(err)
}

func (c *Core) CreateFailsafeUser() {
	log.Infof("INITIALIZING: creating the SHIELD (local auth) failsafe user...")

	if c.Config.API.Failsafe.Username == "" {
		log.Infof("no api.failsafe.username specified; skipping failsafe account creation")
		return
	}

	log.Debugf("checking to see if we should re-instate the '%s' failsafe account", c.Config.API.Failsafe.Username)
	users, err := c.db.GetAllUsers(&db.UserFilter{Backend: "local"})
	c.MaybeTerminate(err)

	if len(users) > 0 {
		log.Debugf("existing users found in database; skipping failsafe account creation...")
		return
	}

	log.Debugf("creating failsafe account '%s'", c.Config.API.Failsafe.Username)
	user := &db.User{
		Name:    "Administrator",
		Account: c.Config.API.Failsafe.Username,
		Backend: "local",
		SysRole: "admin",
	}
	user.SetPassword(c.Config.API.Failsafe.Password)
	_, err = c.db.CreateUser(user)
	c.MaybeTerminate(err)

	tenants, err := c.db.GetAllTenants(&db.TenantFilter{
		Name:       "Default Tenant",
		ExactMatch: true,
	})
	if err == nil && len(tenants) == 1 {
		log.Debugf("adding failsafe account to 'Default Tenant' as an admin")
		tenant := tenants[0]
		err = c.db.AddUserToTenant(user.UUID, tenant.UUID, "admin")
		c.MaybeTerminate(err)
	}
}

func (c *Core) ExpireInteractiveSessions() {
	if c.Config.API.Session.ClearOnBoot {
		log.Infof("INITIALIZING: expiring all interactive sessions...")
		c.db.ClearExpiredSessions(time.Now())
	}
}

func (c *Core) PrecreateTenants() {
	log.Infof("INITIALIZING: populating SHIELD tenants referenced in auth providers...")

	tenants := make(map[string]bool)
	for _, auth := range c.providers {
		for _, tenant := range auth.ReferencedTenants() {
			if tenant != "SYSTEM" {
				tenants[tenant] = true
			}
		}
	}

	for tenant := range tenants {
		if _, err := c.db.EnsureTenant(tenant); err != nil {
			c.Terminate(fmt.Errorf("unable to pre-create tenant '%s' (referenced in authentication providers): %s", tenant, err))
		}
	}
}

func (c *Core) CleanupLeftoverTasks() {
	log.Infof("INITIALIZING: cleaning up leftover task records...")

	tasks, err := c.db.GetAllTasks(&db.TaskFilter{ForStatus: db.RunningStatus})
	c.MaybeTerminate(err)

	now := time.Now()
	for _, task := range tasks {
		log.Infof("CLEANUP: found task %s in 'running' state at startup; setting to 'failed'", task.UUID)
		if err := c.db.FailTask(task.UUID, now); err != nil {
			log.Errorf("failed to sweep database of running tasks [%s]: %s", task.UUID, err)
			continue
		}

		if task.Op == db.BackupOperation && task.ArchiveUUID != "" {
			archive, err := c.db.GetArchive(task.ArchiveUUID)
			if err != nil {
				log.Warnf("unable to retrieve archive %s (for task %s) from the database: %s",
					task.ArchiveUUID, task.UUID, err)
				continue
			}
			if archive == nil {
				log.Infof("task %s was a backup task, but associated archive (%s) was never created; skipping...", task.UUID, task.ArchiveUUID)
				continue
			}
			log.Infof("task %s was a backup task, associated with archive %s; purging the archive", task.UUID, archive.UUID)
			task, err := c.db.CreatePurgeTask("", archive)
			if err != nil {
				log.Errorf("failed to purge archive %s (for task %s, which was running at boot): %s", archive.UUID, task.UUID, err)
			}
		}
	}

	err = c.db.UnscheduleAllTasks()
	if err != nil {
		log.Errorf("failed to reset previously scheduled tasks into a pending state at boot: %s", err)
	}
}

func (c *Core) ConnectToVault() {
	log.Infof("INITIALIZING: connecting to the local SHIELD vault...")

	v, err := vault.Connect(c.Config.Vault.Address, c.Config.Vault.ca)
	c.MaybeTerminate(err)

	status, err := v.Status()
	c.MaybeTerminate(err)

	if status != "unsealed" {
		log.Errorf("SHIELD's vault is %s; please initialize or unlock this SHIELD core via the web UI or the CLI", status)
	}

	c.vault = v
}

func (c *Core) Bind() {
	pprofMux := http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()

	if c.Config.API.PProf != "" {
		log.Infof("INITIALIZING: binding profiling endpoints to %s", c.Config.API.PProf)
		go func() {
			s := &http.Server{
				Addr:    c.Config.API.PProf,
				Handler: pprofMux,
			}
			if err := s.ListenAndServe(); err != nil {
				log.Alertf("SHIELD Core API/pprof failed: %s", err)
			}
		}()
	}

	log.Infof("INITIALIZING: binding the SHIELD Core API/UI on %s...", c.Config.API.Bind)
	log.Infof("INITIALIZING: serving SHIELD Web UI static assets from %s...", c.Config.WebRoot)

	http.Handle("/v1/", c.v1API())
	http.Handle("/v2/", c.v2API())
	http.Handle("/auth/", c.authAPI())
	http.Handle("/", http.FileServer(http.Dir(c.Config.WebRoot)))

	go func() {
		if err := http.ListenAndServe(c.Config.API.Bind, nil); err != nil {
			log.Alertf("SHIELD Core API/UI failed: %s", err)
			os.Exit(2)
		}
		log.Infof("shutting down SHIELD Core API/UI...")
	}()
}

func (c *Core) StartScheduler() {
	log.Infof("INITIALIZING: starting up the scheduler...")

	c.scheduler = scheduler.New(c.Config.Scheduler.Threads, c.db)
}

func (c *Core) ConfigureMessageBus() {
	log.Infof("INITIALIZING: configuring message bus...")
	c.bus = bus.New(2048)
	c.db.Inform(c.bus)
}

func (c *Core) WireUpAuthenticationProviders() {
	log.Infof("INITIALIZING: wiring up authentication providers...")
	for id := range c.providers {
		c.providers[id].WireUpTo(c)
	}
}

func (c *Core) ScheduleBackupTasks() {
	log.Infof("UPKEEP: scheduling backup tasks...")

	l, err := c.db.GetAllJobs(&db.JobFilter{
		Overdue:    true,
		SkipPaused: true,
	})
	if err != nil {
		log.Errorf("error retrieving all overdue jobs from database: %s", err)
		return
	}

	tasks, err := c.db.GetAllTasks(&db.TaskFilter{
		ForOp:        "backup",
		SkipInactive: true,
	})
	if err != nil {
		log.Errorf("error retrieving in-flight tasks from database: %s", err)
		return
	}

	lookup := make(map[string]*db.Task)
	for _, task := range tasks {
		lookup[task.JobUUID] = task
	}

	for _, job := range l {
		if task, running := lookup[job.UUID]; running {
			log.Infof("skipping next run of job %s [%s]; already running in task [%s] (status %s)...", job.Name, job.UUID, task.UUID, task.Status)
			_, err := c.db.SkipBackupTask("system", job,
				fmt.Sprintf("... skipping this run; task %s is still not finished ...\n", task.UUID))
			if err != nil {
				log.Errorf("failed to insert skipped backup task record: %s", err)
			}

		} else {
			log.Infof("scheduling a run of job %s [%s]", job.Name, job.UUID)
			_, err := c.db.CreateBackupTask("system", job)
			if err != nil {
				log.Errorf("failed to insert backup task record: %s", err)
			}
		}

		if spec, err := timespec.Parse(job.Schedule); err != nil {
			log.Errorf("error re-scheduling job %s [%s]: %s", job.Name, job.UUID, err)
		} else {
			if next, err := spec.Next(time.Now()); err != nil {
				log.Errorf("error re-scheduling job %s [%s]: %s", job.Name, job.UUID, err)
			} else {
				if err = c.db.RescheduleJob(job, next); err != nil {
					log.Errorf("error re-scheduling job %s [%s]: %s", job.Name, job.UUID, err)
				}
			}
		}
	}
}

func (c *Core) ScheduleAgentStatusCheckTasks(f *db.AgentFilter) {
	log.Infof("UPKEEP: scheduling immediate agent status check tasks for newly registered agents...")

	agents, err := c.db.GetAllAgents(f)
	if err != nil {
		log.Errorf("error retrieving agent registration records from database: %s", err)
		return
	}

	for _, agent := range agents {
		if _, err := c.db.CreateAgentStatusTask("system", agent); err != nil {
			log.Errorf("error scheduling status check of agent %s: %s", agent.Name, err)
			continue
		}
		if agent.Status == "pending" {
			agent.Status = "checking"
			if err := c.db.UpdateAgent(agent); err != nil {
				log.Errorf("error update agent '%s' status to 'checking': %s", err)
				continue
			}
		}
	}
}

func (c *Core) TasksToChores() {
	log.Infof("SCHEDULER: converting tasks (database) into chores (scheduler)")

	inflight := make(map[string]*db.Task)
	if active, err := c.db.GetAllTasks(&db.TaskFilter{SkipInactive: true}); err != nil {
		log.Errorf("unable to retrieve active tasks from database, in order to avoid scheduling conflicts: %s", err)
		return
	} else {
		for _, task := range active {
			if task.Status == "pending" {
				continue
			}
			if task.Op == db.BackupOperation || task.Op == db.RestoreOperation {
				inflight[task.TargetUUID] = task
			}
		}
	}

	tasks, err := c.db.GetAllTasks(&db.TaskFilter{ForStatus: "pending"})
	if err != nil {
		log.Errorf("unable to retrieve pending tasks from database, in order to schedule them: %s", err)
		return
	}

	for _, task := range tasks {
		log.Infof("SCHEDULER: scheduling [%s] task %s", task.Op, task.UUID)

		fabric, err := c.FabricFor(task)
		if err != nil {
			log.Errorf("unable to find a fabric to facilitate execution of [%s] task %s: %s", task.Op, task.UUID, err)
			log.Errorf("marking [%s] task %s as errored", task.Op, task.UUID)

			c.db.UpdateTaskLog(task.UUID, "unable to find a fabric to facilitate execution of this task\n")
			c.db.FailTask(task.UUID, time.Now())
			continue
		}

		switch task.Op {
		default:
			c.TaskErrored(task, "unrecognized task type '%s'\n", task.Op)
			continue

		case db.BackupOperation:
			if other, ok := inflight[task.TargetUUID]; ok {
				log.Infof("SCHEDULER: SKIPPING [%s] task %s, another %s task [%s] is already in-flight for target [%s]", task.Op, task.UUID, other.Op, other.UUID, task.TargetUUID)
				continue
			}
			encryption, err := c.vault.NewParameters(task.ArchiveUUID, c.Config.Cipher, task.FixedKey)
			if err != nil {
				c.TaskErrored(task, "unable to generate encryption parameters:\n%s\n", err)
				continue
			}
			c.scheduler.Schedule(20, fabric.Backup(task, encryption))
			inflight[task.TargetUUID] = task

		case db.RestoreOperation:
			if op, ok := inflight[task.TargetUUID]; ok {
				log.Infof("SCHEDULER: SKIPPING [%s] task %s, another %s operation is already in-flight for target [%s]", task.Op, task.UUID, op, task.TargetUUID)
				continue
			}
			encryption, err := c.vault.Retrieve(task.ArchiveUUID)
			if err != nil {
				c.TaskErrored(task, "unable to retrieve encryption parameters:\n%s\n", err)
				continue
			}
			if encryption.Type == "" {
				c.TaskErrored(task, "unable to retrieve encryption parameters:\nencryption parameters for archive '%s' not found in vault\n", task.ArchiveUUID)
				continue
			}
			c.scheduler.Schedule(20, fabric.Restore(task, encryption))
			inflight[task.TargetUUID] = task

		case db.PurgeOperation:
			c.scheduler.Schedule(50, fabric.Purge(task))

		case db.AgentStatusOperation:
			c.scheduler.Schedule(30, fabric.Status(task))

		case db.TestStoreOperation:
			c.scheduler.Schedule(40, fabric.TestStore(task))
		}

		if err := c.db.ScheduledTask(task.UUID); err != nil {
			log.Errorf("unable to mark task %s as 'scheduled' in the database: %s", err)
			log.Errorf("THIS TASK MAY BE INADVERTANTLY RE-SCHEDULED!!!")
		}
	}
}

func (c *Core) CheckArchiveExpiries() {
	log.Infof("UPKEEP: checking archive expiries...")

	l, err := c.db.GetExpiredArchives()
	if err != nil {
		log.Errorf("error retrieving archives that have outlived their retention policy: %s", err)
		return
	}

	for _, archive := range l {
		log.Infof("archive %s has expiration %s, marking as expired", archive.UUID, archive.ExpiresAt)
		if err := c.db.ExpireArchive(archive.UUID); err != nil {
			log.Errorf("error marking archive %s as expired: %s", archive.UUID, err)
			continue
		}

		log.Infof("deleting encryption parameters for archive %s", archive.UUID)
		if err := c.vault.Delete(fmt.Sprintf("secret/archives/%s", archive.UUID)); err != nil {
			log.Errorf("failed to delete encryption parameters for archive %s: %s", archive.UUID, err)
		}
	}
}

func (c *Core) SchedulePurgeTasks() {
	log.Infof("UPKEEP: schedule purge tasks for all expired archives...")

	l, err := c.db.GetArchivesNeedingPurge()
	if err != nil {
		log.Errorf("error retrieving archives to purge: %s", err)
		return
	}

	for _, archive := range l {
		log.Infof("scheduling purge of archive %s due to status '%s'", archive.UUID, archive.Status)
		_, err := c.db.CreatePurgeTask("system", archive)
		if err != nil {
			log.Errorf("error scheduling purge of archive %s: %s", archive.UUID, err)
			continue
		}
	}
}

func (c *Core) MarkIrrelevantTasks() {
	log.Infof("UPKEEP: marking irrelevant tasks that have been superseded...")

	c.db.MarkTasksIrrelevant()
}

func (c *Core) AnalyzeStorage() {
	task, err := c.db.CreateInternalTask("system", db.AnalyzeStorageOperation, db.GlobalTenantUUID)
	if err != nil {
		log.Errorf("failed to schedule internal task to analyze cloud storage: %s", err)
		return
	}

	c.scheduler.Schedule(50, scheduler.NewChore(
		task.UUID,
		func(chore scheduler.Chore) {
			delta := func(filter db.ArchiveFilter) (int64, int64, error) {
				filter.WithStatus = []string{"valid"}
				increase, err := c.db.ArchiveStorageFootprint(&filter)
				if err != nil {
					return 0, 0, err
				}

				filter.WithStatus = []string{"purged"}
				purged, err := c.db.ArchiveStorageFootprint(&filter)
				if err != nil {
					return 0, 0, err
				}

				return increase, purged, nil
			}

			base := time.Now()
			threshold := base.Add(0 - time.Duration(24)*time.Hour)

			chore.Errorf("GLOBAL CLOUD STORAGE")
			chore.Errorf("====================")

			stores, err := c.db.GetAllStores(nil)
			if err != nil {
				chore.Errorf("FAILED to get stores for daily storage statistics: %s", err)
				return
			}

			for _, store := range stores {
				chore.Errorf("")
				chore.Errorf(">> analyzing usage of store '%s' (uuid %s)...", store.Name, store.UUID)
				up, down, err := delta(db.ArchiveFilter{
					ForStore:      store.UUID,
					Before:        &base,
					After:         &threshold,
					ExpiresBefore: &base,
					ExpiresAfter:  &threshold,
				})
				if err != nil {
					chore.Errorf("      ERROR: failed to calculate daily usage increase:")
					chore.Errorf("      ERROR: %s", err)
					continue
				}

				total, err := c.db.ArchiveStorageFootprint(&db.ArchiveFilter{
					ForStore:   store.UUID,
					WithStatus: []string{"valid"},
				})
				if err != nil {
					chore.Errorf("      ERROR: failed to calculate total usage:")
					chore.Errorf("      ERROR: %s", err)
					continue
				}

				count, err := c.db.CountArchives(&db.ArchiveFilter{
					ForStore:   store.UUID,
					WithStatus: []string{"valid"},
				})
				if err != nil {
					chore.Errorf("      ERROR: failed to calculate total archive count:")
					chore.Errorf("      ERROR: %s", err)
					continue
				}

				store.DailyIncrease = up - down
				store.StorageUsed = total
				store.ArchiveCount = count
				chore.Errorf("      new archives in the last 24h used %d bytes", up)
				chore.Errorf("      expired archives purged reclaimed %d bytes", down)
				chore.Errorf("")
				chore.Errorf("      net daily increase: %d bytes", store.DailyIncrease)
				chore.Errorf("      total storage used: %d bytes", store.StorageUsed)
				chore.Errorf("      # archives present: %d", store.ArchiveCount)

				err = c.db.UpdateStore(store)
				if err != nil {
					chore.Errorf("      ERROR: failed to update store '%s':", store.UUID)
					chore.Errorf("      ERROR: %s", err)
					continue
				}
			}

			chore.Errorf("")
			chore.Errorf("TENANT CLOUD STORAGE")
			chore.Errorf("====================")

			tenants, err := c.db.GetAllTenants(nil)
			if err != nil {
				chore.Errorf("FAILED to get tenants for daily storage statistics: %s", err)
				return
			}

			for _, tenant := range tenants {
				chore.Errorf("")
				chore.Errorf(">> analyzing usage of tenant '%s' (uuid %s)...", tenant.Name, tenant.UUID)
				up, down, err := delta(db.ArchiveFilter{
					ForTenant:     tenant.UUID,
					Before:        &base,
					After:         &threshold,
					ExpiresBefore: &base,
					ExpiresAfter:  &threshold,
				})
				if err != nil {
					chore.Errorf("      ERROR: failed to calculate daily usage increase:")
					chore.Errorf("      ERROR: %s", err)
					continue
				}

				total, err := c.db.ArchiveStorageFootprint(&db.ArchiveFilter{
					ForTenant:  tenant.UUID,
					WithStatus: []string{"valid"},
				})
				if err != nil {
					chore.Errorf("      ERROR: failed to calculate total usage:")
					chore.Errorf("      ERROR: %s", err)
					continue
				}

				count, err := c.db.CountArchives(&db.ArchiveFilter{
					ForTenant:  tenant.UUID,
					WithStatus: []string{"valid"},
				})
				if err != nil {
					chore.Errorf("      ERROR: failed to calculate total archive count:")
					chore.Errorf("      ERROR: %s", err)
					continue
				}

				tenant.DailyIncrease = up - down
				tenant.StorageUsed = total
				tenant.ArchiveCount = count
				chore.Errorf("      new archives in the last 24h used %d bytes", up)
				chore.Errorf("      expired archives purged reclaimed %d bytes", down)
				chore.Errorf("")
				chore.Errorf("      net daily increase: %d bytes", tenant.DailyIncrease)
				chore.Errorf("      total storage used: %d bytes", tenant.StorageUsed)
				chore.Errorf("      # archives present: %d", tenant.ArchiveCount)

				_, err = c.db.UpdateTenant(tenant)
				if err != nil {
					chore.Errorf("      ERROR: failed to update tenant '%s':", tenant.UUID)
					chore.Errorf("      ERROR: %s", err)
					continue
				}
			}

			chore.Errorf("")
			chore.Errorf("COMPLETE")
		}))
}

func (c *Core) CleanupOrphanedObjects() {
	log.Infof("UPKEEP: cleaning up orphaned objects leftover from tenant removal...")

	if err := c.db.CleanTargets(); err != nil {
		log.Errorf("Failed to clean up orphaned targets: %s", err)
	}

	if err := c.db.CleanStores(); err != nil {
		log.Errorf("Failed to clean up orphaned stores: %s", err)
	}
}

func (c *Core) PurgeExpiredAPISessions() {
	log.Infof("UPKEEP: purging expired API sessions...")

	if err := c.db.ClearExpiredSessions(time.Now().Add(0 - (time.Duration(c.Config.API.Session.Timeout) * time.Hour))); err != nil {
		log.Errorf("Failed to purge expired API sessions: %s", err)
	}
}

func (c *Core) ScheduleStorageTestTasks() {
	log.Infof("UPKEEP: scheduling cloud storage test tasks...")

	stores, err := c.db.GetAllStores(nil)
	if err != nil {
		log.Errorf("failed to get stores for health tests: %s", err)
		return
	}

	inflight, err := c.db.GetAllTasks(&db.TaskFilter{
		ForOp:        db.TestStoreOperation,
		SkipInactive: true,
	})
	if err != nil {
		log.Errorf("failed to get in-flight health test tasks: %s", err)
		return
	}

	lookup := make(map[string]bool)
	for _, task := range inflight {
		lookup[task.StoreUUID] = true
	}

	for _, store := range stores {
		if _, inqueue := lookup[store.UUID]; inqueue {
			continue
		}

		if _, err := c.db.CreateTestStoreTask("system", store); err != nil {
			log.Errorf("failed to create test store task: %s", err)
		}
	}
}
