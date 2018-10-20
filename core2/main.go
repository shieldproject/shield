package core2

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jhunt/go-log"

	"github.com/starkandwayne/shield/core/bus"
	"github.com/starkandwayne/shield/core/fabric"
	"github.com/starkandwayne/shield/core/scheduler"
	"github.com/starkandwayne/shield/core/vault"
	"github.com/starkandwayne/shield/db"
	"github.com/starkandwayne/shield/timespec"
)

func (c Core) Main() {
	/* we need a usable database first */
	c.ConnectToDatabase()

	/* startup cleanup tasks */
	c.CreateFailsafeUser()
	c.ExpireInteractiveSessions()
	c.PrecreateTenants()
	c.CleanupLeftoverTasks()

	/* prepare for operation */
	c.ConnectToVault()
	c.BindAPI()
	c.StartScheduler()
	c.ConfigureMessageBus()

	log.Infof("INITIALIZATION COMPLETE; entering main loop.")
	fast := time.NewTicker(time.Second * time.Duration(c.Config.Scheduler.FastLoop))
	slow := time.NewTicker(time.Second * time.Duration(c.Config.Scheduler.SlowLoop))
	for {
		select {
		case <-fast.C:
			c.ScheduleBackupTasks()
			c.ScheduleAgentStatusCheckTasks(&db.AgentFilter{Status: "pending"})
			c.RunScheduler()

		case <-slow.C:
			c.CheckArchiveExpiries()
			c.SchedulePurgeTasks()
			c.MarkIrrelevantTasks()
			c.ScheduleAgentStatusCheckTasks(nil)
			c.UpdateGlobalStorageUsage()
			c.UpdateTenantStorageUsage()

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
}

func (c *Core) ExpireInteractiveSessions() {
	log.Infof("INITIALIZING: expiring all interactive sessions...")
	c.db.ClearExpiredSessions(time.Now())
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
			c.Terminate(fmt.Errorf("unable to pre-create tnant '%s' (referenced in authentication providers): %s", tenant, err))
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

		if task.Op == db.BackupOperation && task.ArchiveUUID != nil {
			archive, err := c.db.GetArchive(task.ArchiveUUID)
			if err != nil {
				log.Warnf("unable to retrieve archive %s (for task %s) from the database: %s",
					task.ArchiveUUID, task.UUID, err)
				continue
			}
			log.Infof("task %s was a backup task, associated with archive %s; purging the archive", task.UUID, archive.UUID)
			task, err := c.db.CreatePurgeTask("", archive)
			if err != nil {
				log.Errorf("failed to purge archive %s (for task %s, which was running at boot): %s", archive.UUID, task.UUID, err)
			}
		}
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

func (c *Core) BindAPI() {
	log.Infof("INITIALIZING: binding the SHIELD Core API on %s...", c.Config.API.Bind)

	http.Handle("/v2/", c.v2API())

	go func() {
		if err := http.ListenAndServe(c.Config.API.Bind, nil); err != nil {
			log.Alertf("SHIELD Core API failed: %s", err)
			os.Exit(2)
		}
		log.Infof("shutting down SHIELD Core API...")
	}()
}

func (c *Core) StartScheduler() {
	log.Infof("INITIALIZING: starting up the scheduler...")

	c.scheduler = scheduler.New(c.Config.Scheduler.Threads)
}

func (c *Core) ConfigureMessageBus() {
	log.Infof("INITIALIZING: configuring message bus...")
	c.bus = bus.New(2048)
	c.db.Inform(c.bus)
}

func (c *Core) ScheduleBackupTasks() {
	log.Infof("UPKEEP: scheduling backup tasks...")

	l, err := c.db.GetAllJobs(&db.JobFilter{Overdue: true})
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

	lookup := make(map[string]string)
	for _, task := range tasks {
		lookup[task.JobUUID.String()] = task.UUID.String()
	}

	for _, job := range l {
		if tid, running := lookup[job.UUID.String()]; running {
			log.Infof("skipping next run of job %s [%s]; already running in task [%s]...", job.Name, job.UUID, tid)
			_, err := c.db.SkipBackupTask("system", job,
				fmt.Sprintf("... skipping this run; task %s is still not finished ...\n", tid))
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
		c.scheduler.Schedule(0, fabric.Legacy(agent.Address, "FIXME/ssh/key/missing").Status())
	}
}

func (c *Core) RunScheduler() {
	log.Infof("SCHEDULER: initiating a run of the scheduler")

	tasks, err := c.db.GetAllTasks(&db.TaskFilter{ForStatus: "pending"})
	if err != nil {
		log.Errorf("unable to retrieve pending tasks from database, in order to schedule them: %s", err)
		return
	}

	for _, task := range tasks {
		log.Infof("SCHEDULER: scheduling [%s] task %s", task.Op, task.UUID)

		// FIXME: fabric, err := c.FabricFor(task)
		if err != nil {
			log.Errorf("unable to find a fabric to facilitate execution of [%s] task %s: %s", task.Op, task.UUID, err)
			log.Errorf("marking [%s] task %s as errored", task.Op, task.UUID)

			c.db.UpdateTaskLog(task.UUID, "unable to find a fabric to facilitate execution of this task\n")
			c.db.FailTask(task.UUID, time.Now())
			continue
		}
		// FIXME c.scheduler.Schedule(task)

		if err := c.db.ScheduledTask(task.UUID); err != nil {
			log.Errorf("unable to mark task %s as 'scheduled' in the database: %s", err)
			log.Errorf("THIS TASK MAY BE INADVERTANTLY RE-SCHEDULED!!!")
		}
	}

	log.Infof("SCHEDULER: running the scheduling algorithm")
	c.scheduler.Run()
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
		if err := c.vault.Delete(fmt.Sprintf("secret/archives/%s", archive.UUID.String())); err != nil {
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

func (c *Core) UpdateGlobalStorageUsage() {
	log.Infof("UPKEEP: updating global storage usage statistics...")

	base := time.Now()
	threshold := base.Add(0 - time.Duration(24)*time.Hour)

	stores, err := c.db.GetAllStores(nil)
	if err != nil {
		log.Errorf("Failed to get stores for daily storage statistics: %s", err)
		return
	}

	for _, store := range stores {
		delta, err := c.DeltaIncrease(
			&db.ArchiveFilter{
				ForStore:      store.UUID.String(),
				Before:        &base,
				After:         &threshold,
				ExpiresBefore: &base,
				ExpiresAfter:  &threshold,
			},
		)
		if err != nil {
			log.Errorf("Failed to get archive stats for daily storage statistics: %s", err)
			continue
		}

		total_size, err := c.db.ArchiveStorageFootprint(
			&db.ArchiveFilter{
				ForStore:   store.UUID.String(),
				WithStatus: []string{"valid"},
			},
		)
		if err != nil {
			log.Errorf("Failed to get archive stats for daily storage statistics: %s", err)
			continue
		}

		total_count, err := c.db.CountArchives(
			&db.ArchiveFilter{
				ForStore:   store.UUID.String(),
				WithStatus: []string{"valid"},
			},
		)
		if err != nil {
			log.Errorf("Failed to get archive stats for daily storage statistics: %s", err)
			continue
		}

		store.DailyIncrease = delta
		store.StorageUsed = total_size
		store.ArchiveCount = total_count
		log.Debugf("updating store '%s' (%s) %d archives, %db storage used, %db increase",
			store.Name, store.UUID.String(), store.ArchiveCount, store.StorageUsed, store.DailyIncrease)
		err = c.db.UpdateStore(store)
		if err != nil {
			log.Errorf("Failed to update stores with daily storage statistics: %s", err)
		}
	}
}

func (c *Core) UpdateTenantStorageUsage() {
	log.Infof("UPKEEP: updating per-tenant storage usage statistics...")

	base := time.Now()
	threshold := base.Add(0 - time.Duration(24)*time.Hour)

	tenants, err := c.db.GetAllTenants(nil)
	if err != nil {
		log.Errorf("Failed to get tenants for daily storage statistics: %s", err)
		return
	}

	for _, tenant := range tenants {
		delta, err := c.DeltaIncrease(
			&db.ArchiveFilter{
				ForTenant:     tenant.UUID.String(),
				Before:        &base,
				After:         &threshold,
				ExpiresBefore: &base,
				ExpiresAfter:  &threshold,
			},
		)
		if err != nil {
			log.Errorf("Failed to get archive stats for daily storage statistics: %s", err)
			continue
		}

		total_size, err := c.db.ArchiveStorageFootprint(
			&db.ArchiveFilter{
				ForTenant:  tenant.UUID.String(),
				WithStatus: []string{"valid"},
			},
		)
		if err != nil {
			log.Errorf("Failed to get archive stats for daily storage statistics: %s", err)
			continue
		}

		total_count, err := c.db.CountArchives(
			&db.ArchiveFilter{
				ForTenant:  tenant.UUID.String(),
				WithStatus: []string{"valid"},
			},
		)
		if err != nil {
			log.Errorf("Failed to get archive stats for daily storage statistics: %s", err)
			continue
		}

		tenant.StorageUsed = total_size
		tenant.ArchiveCount = total_count
		tenant.DailyIncrease = delta

		log.Debugf("updating tenant '%s' (%s) %d archives, %db storage used, %db increase",
			tenant.Name, tenant.UUID.String(), tenant.ArchiveCount, tenant.StorageUsed, tenant.DailyIncrease)
		if _, err = c.db.UpdateTenant(tenant); err != nil {
			log.Errorf("Failed to update tenant with daily storage statistics: %s", err)
		}
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
		lookup[task.StoreUUID.String()] = true
	}

	for _, store := range stores {
		if _, inqueue := lookup[store.UUID.String()]; inqueue {
			continue
		}

		if _, err := c.db.CreateTestStoreTask("system", store); err != nil {
			log.Errorf("failed to create test store task: %s", err)
		}
	}
}
