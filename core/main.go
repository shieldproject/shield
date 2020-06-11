package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"regexp"
	"time"

	"github.com/jhunt/go-log"

	"github.com/shieldproject/shield/core/bus"
	"github.com/shieldproject/shield/core/metrics"
	"github.com/shieldproject/shield/core/scheduler"
	"github.com/shieldproject/shield/db"
	"github.com/shieldproject/shield/timespec"
)

const StorageGatewayPlugin = "ssg"

func (c Core) Main() {
	/* print out our configuration */
	c.PrintConfiguration()

	/* we need a usable database first */
	c.ConnectToDatabase()
	c.InitializePrometheus() //Initialize metric values
	c.ApplyFixups()
	c.ConfigureMessageBus()
	c.WireUpAuthenticationProviders()

	/* startup cleanup tasks */
	c.CreateFailsafeUser()
	c.ExpireInteractiveSessions()
	c.PrecreateTenants()
	c.CleanupLeftoverTasks()

	/* prepare for operation */
	c.Bind()
	c.StartScheduler()
	go c.metrics.Watch("*")

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
			c.ScheduleBackupTasks()
			c.ScheduleAgentStatusCheckTasks(&db.AgentFilter{Status: "pending"})
			c.scheduler.Elevate()
			c.TasksToChores()

		case <-slow.C:
			c.CheckArchiveExpiries()
			c.MarkIrrelevantTasks()
			c.ScheduleAgentStatusCheckTasks(nil)
			c.TruncateOldTaskLogs()
			c.DeleteOldPurgedArchives()
			c.CleanupOrphanedObjects()
			c.PurgeExpiredAPISessions()
		}
	}
}

func (c *Core) PrintConfiguration() {
	log.Infof("CONFIG | web root:          '%s'", c.Config.WebRoot)
	if len(c.Config.PluginPaths) == 0 {
		log.Infof("CONFIG | plugin paths:      (none)")
	} else {
		log.Infof("CONFIG | plugin paths:")
		for _, path := range c.Config.PluginPaths {
			log.Infof("CONFIG |  - '%s'", path)
		}
	}

	log.Infof("CONFIG | scheduler loop:    fast=%ds slow=%ds", c.Config.Scheduler.FastLoop, c.Config.Scheduler.SlowLoop)
	log.Infof("CONFIG | scheduler threads: %d", c.Config.Scheduler.Threads)
	log.Infof("CONFIG | scheduler timeout: %ds", c.Config.Scheduler.Timeout)
	log.Infof("CONFIG | api bind:          '%s'", c.Config.API.Bind)
	log.Infof("CONFIG | session timeout:   %ds", c.Config.API.Session.Timeout)
	log.Infof("CONFIG | failsafe username: '%s'", c.Config.API.Failsafe.Username)
	log.Infof("CONFIG | websocket timeout: %ds", c.Config.API.Websocket.WriteTimeout)
	log.Infof("CONFIG | websocket ping:    %ds", c.Config.API.Websocket.PingInterval)
	log.Infof("CONFIG | mbus max clients:  %d", c.Config.Mbus.MaxSlots)
	log.Infof("CONFIG | mbus backlog:      %d connections", c.Config.Mbus.Backlog)
	log.Infof("")
	log.Infof("CONFIG | backup archives must be kept for at least %s", (duration)(c.Config.Limit.Retention.Min))
	log.Infof("CONFIG | backup archives are kept for no more than %s", (duration)(c.Config.Limit.Retention.Max))
	log.Infof("CONFIG | task logs will be truncated after %s", c.Config.Metadata.Retention.TaskLogs)
	log.Infof("CONFIG | purged archives will be deleted after %s", c.Config.Metadata.Retention.PurgedArchives)
	log.Infof("")
}

func sanitize(s string) string {
	re := regexp.MustCompile(`(.*:\/\/.*?:)(.*?)(@.*)`)
	if m := re.FindStringSubmatch(s); m != nil {
		replace := m[1]
		for range m[2] {
			replace += "*"
		}
		replace += m[3]
		return replace
	}

	re = regexp.MustCompile(`(.*\bpassword=)(.*?)(\s.+)?$`)
	if m := re.FindStringSubmatch(s); m != nil {
		replace := m[1]
		for range m[2] {
			replace += "*"
		}
		replace += m[3]
		return replace
	}

	return s
}

func (c *Core) ConnectToDatabase() {
	log.Infof("INITIALIZING: connecting to the database...")
	if c.db != nil {
		log.Alertf("ANOMALY: tried to connect to database, but we're already connected...")
		return
	}

	log.Debugf("connecting to database at %s...", sanitize(c.Config.Database))
	db, err := db.Connect(c.Config.Database)
	c.MaybeTerminate(err)
	c.db = db

	log.Infof("INITIALIZING: checking database schema...")
	c.MaybeTerminate(c.db.CheckCurrentSchema())

	log.Debugf("connected successfully to database!")
}

func (c *Core) InitializePrometheus() error {
	tenants, err := c.db.GetAllTenants(nil)
	c.MaybeTerminate(err)

	agents, err := c.db.GetAllAgents(nil)
	c.MaybeTerminate(err)

	targets, err := c.db.GetAllTargets(nil)
	c.MaybeTerminate(err)

	stores, err := c.db.GetAllStores(nil)
	c.MaybeTerminate(err)

	jobs, err := c.db.GetAllJobs(nil)
	c.MaybeTerminate(err)

	tasks, err := c.db.GetAllTasks(nil)
	c.MaybeTerminate(err)

	archives, err := c.db.GetAllArchives(nil)
	c.MaybeTerminate(err)

	storageBytesUsed, err := c.db.ArchiveStorageFootprint(&db.ArchiveFilter{
		WithStatus: []string{"valid"},
	})
	c.MaybeTerminate(err)

	c.metrics = metrics.New(&metrics.Exporter{
		Username:  c.Config.Prometheus.Username,
		Password:  c.Config.Prometheus.Password,
		Realm:     c.Config.Prometheus.Realm,
		Namespace: c.Config.Prometheus.Namespace,

		TenantCount:      len(tenants),
		AgentCount:       len(agents),
		TargetCount:      len(targets),
		StoreCount:       len(stores),
		JobCount:         len(jobs),
		TaskCount:        len(tasks),
		ArchiveCount:     len(archives),
		StorageUsedCount: storageBytesUsed,
	})

	return nil
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
			err = c.db.PurgeArchive(task.ArchiveUUID)
			if err != nil {
				panic(fmt.Errorf("failed to purge archive %s from the database: %s", archive.UUID, err))
			}

		}
	}

	err = c.db.UnscheduleAllTasks()
	if err != nil {
		log.Errorf("failed to reset previously scheduled tasks into a pending state at boot: %s", err)
	}
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
	http.Handle("/metrics/", c.metrics.Handler())
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
	log.Infof("INITIALIZING: configuring message bus with %d slots and %d backlog per slot...", c.Config.Mbus.MaxSlots, c.Config.Mbus.Backlog)
	c.bus = bus.New(c.Config.Mbus.MaxSlots, c.Config.Mbus.Backlog)
	c.metrics.Inform(c.bus)
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
			if task.StorePlugin == StorageGatewayPlugin {
				uploadInfo, err := c.GatedUpload(task.ArchiveUUID, 3)
				if err != nil {
					c.TaskErrored(task, "unable to set up storage gateway upload:\n%s\n", err)
					continue
				}
				ep := struct {
					URL           string `json:"url"`
					Path          string `json:"path"`
					UploadID      string `json:"upload_id"`
					UploadToken   string `json:"upload_token"`
					DownloadID    string `json:"download_id"`
					DownloadToken string `json:"download_token"`
				}{
					URL:         uploadInfo.url,
					Path:        uploadInfo.path,
					UploadID:    uploadInfo.uploadInfo.ID,
					UploadToken: uploadInfo.uploadInfo.Token,
				}
				b, err := json.Marshal(ep)
				if err != nil {
					c.TaskErrored(task, "unable to set up storage gateway upload:\n%s\n", err)
					continue
				}
				task.StoreEndpoint = string(b)
			}

			c.scheduler.Schedule(20, fabric.Backup(task))
			inflight[task.TargetUUID] = task

		case db.RestoreOperation:
			if op, ok := inflight[task.TargetUUID]; ok {
				log.Infof("SCHEDULER: SKIPPING [%s] task %s, another %s operation is already in-flight for target [%s]", task.Op, task.UUID, op, task.TargetUUID)
				continue
			}

			if task.StorePlugin == StorageGatewayPlugin {
				downloadInfo, err := c.GatedDownload(task.RestoreKey, 3)
				if err != nil {
					c.TaskErrored(task, "unable to set up storage gateway download:\n%s\n", err)
					continue
				}
				ep := struct {
					URL           string `json:"url"`
					Path          string `json:"path"`
					UploadID      string `json:"upload_id"`
					UploadToken   string `json:"upload_token"`
					DownloadID    string `json:"download_id"`
					DownloadToken string `json:"download_token"`
				}{
					URL:           downloadInfo.url,
					DownloadID:    downloadInfo.downloadInfo.ID,
					DownloadToken: downloadInfo.downloadInfo.Token,
				}
				b, err := json.Marshal(ep)
				if err != nil {
					c.TaskErrored(task, "unable to set up storage gateway download:\n%s\n", err)
					continue
				}
				task.StoreEndpoint = string(b)
			}

			c.scheduler.Schedule(20, fabric.Restore(task))
			inflight[task.TargetUUID] = task

		case db.AgentStatusOperation:
			c.scheduler.Schedule(30, fabric.Status(task))
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
	}
}

func (c *Core) MarkIrrelevantTasks() {
	log.Infof("UPKEEP: marking irrelevant tasks that have been superseded...")

	c.db.MarkTasksIrrelevant()
}

func (c *Core) TruncateOldTaskLogs() {
	when := c.Config.Metadata.Retention.TaskLogs
	log.Infof("UPKEEP: truncating logs for tasks older than %s...", when)

	if err := c.db.TruncateTaskLogs((int)(when)); err != nil {
		log.Errorf("Failed to truncate task logs from %s ago (or more): %s", when, err)
	}
}

func (c *Core) DeleteOldPurgedArchives() {
	when := c.Config.Metadata.Retention.PurgedArchives
	log.Infof("UPKEEP: deleting archives purged more than %s ago...", when)

	if err := c.db.CleanupArchives((int)(when)); err != nil {
		log.Errorf("Failed to deleted archives purged more then %s ago: %s", when, err)
	}
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

	if err := c.db.ClearExpiredSessions(time.Now().Add(0 - (time.Duration(c.Config.API.Session.Timeout) * time.Second))); err != nil {
		log.Errorf("Failed to purge expired API sessions: %s", err)
	}
}
