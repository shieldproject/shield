package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/jhunt/go-log"
	"github.com/pborman/uuid"

	"github.com/starkandwayne/shield/crypter"
	"github.com/starkandwayne/shield/db"
	"github.com/starkandwayne/shield/timespec"
)

var Version = "(development)"

const SessionCookieName = "shield7"

var DataDir = "setme"

type Core struct {
	fastloop *time.Ticker
	slowloop *time.Ticker

	timeout int
	agent   *AgentClient

	debug bool //For exposing debug API endpoints

	/* cached for /v2/health */
	ip   string
	fqdn string

	/* poison pill */
	seppuku bool

	/* foreman */
	numWorkers int
	workers    chan *db.Task
	broadcast  Broadcaster
	events     chan Event

	/* monitor */
	agents map[string]chan *db.Agent

	/* data dir */
	dataDir string

	/* janitor */
	purgeAgent string

	/* api */
	webroot string
	listen  string
	auth    map[string]AuthProvider
	env     string
	color   string
	motd    string

	/* vault */
	vault          crypter.Vault
	encryptionType string
	vaultKeyfile   string
	vaultAddress   string
	vaultCACert    string

	/* sessions */
	sessionTimeout int

	failsafe FailsafeConfig

	DB *db.DB
}

func NewCore(file string) (*Core, error) {
	config, err := ReadConfig(file)
	if err != nil {
		return nil, err
	}
	agent, err := NewAgentClient(config.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent key file %s: %s", config.KeyFile, err)
	}

	ip, fqdn := networkIdentity()

	DataDir = config.DataDir

	core := &Core{
		fastloop: time.NewTicker(time.Second * time.Duration(config.FastLoop)),
		slowloop: time.NewTicker(time.Second * time.Duration(config.SlowLoop)),

		timeout: config.Timeout,
		agent:   agent,

		/* poison pill */
		seppuku: false,

		ip:   ip,
		fqdn: fqdn,

		debug: config.Debug,

		/* foreman */
		numWorkers: config.Workers,
		workers:    make(chan *db.Task),
		broadcast:  NewBroadcaster(2048),
		events:     make(chan Event),

		/* monitor */
		agents: make(map[string]chan *db.Agent),

		/* data dir */
		dataDir: config.DataDir,

		/* janitor */
		purgeAgent: config.Purge,

		/* api */
		webroot: config.WebRoot,
		listen:  config.Addr,
		env:     config.Environment,
		color:   config.Color,
		motd:    config.MOTD,

		/* encryption */
		encryptionType: config.EncryptionType,
		vaultKeyfile:   path.Join(config.DataDir, "/vault/config.crypt"),
		vaultAddress:   config.VaultAddress,

		/* session */
		sessionTimeout: config.SessionTimeout,

		failsafe: config.Failsafe,

		/* db */
		DB: &db.DB{
			Driver: "sqlite3",
			DSN:    path.Join(config.DataDir, "/shield.db"),
		},
	}

	if config.VaultCACert != "" {
		b, err := ioutil.ReadFile(config.VaultCACert)
		if err != nil {
			return nil, err
		}
		core.vaultCACert = string(b)
	}

	core.auth = make(map[string]AuthProvider)
	for i, auth := range config.Auth {
		if auth.Identifier == "" {
			return nil, fmt.Errorf("provider #%d lacks the required `identifier' field", i+1)
		}
		if auth.Name == "" {
			return nil, fmt.Errorf("%s provider lacks the required `name' field", auth.Identifier)
		}
		if auth.Backend == "" {
			return nil, fmt.Errorf("%s provider lacks the required `backend' field", auth.Identifier)
		}

		switch auth.Backend {
		case "github":
			core.auth[auth.Identifier] = &GithubAuthProvider{
				AuthProviderBase: AuthProviderBase{
					Name:       auth.Name,
					Identifier: auth.Identifier,
					Type:       auth.Backend,
				},
				core: core,
			}
		case "uaa":
			core.auth[auth.Identifier] = &UAAAuthProvider{
				AuthProviderBase: AuthProviderBase{
					Name:       auth.Name,
					Identifier: auth.Identifier,
					Type:       auth.Backend,
				},
				core: core,
			}
		default:
			return nil, fmt.Errorf("%s provider has an unrecognized `backend' of '%s'; must be one of github or uaa", auth.Identifier, auth.Backend)
		}

		if err := core.auth[auth.Identifier].Configure(auth.Properties); err != nil {
			return nil, fmt.Errorf("failed to configure '%s' auth provider '%s': %s",
				auth.Backend, auth.Identifier, err)
		}
	}

	return core, nil
}

func (core *Core) Run() error {
	var err error
	if err = core.DB.Connect(); err != nil {
		return fmt.Errorf("failed to connect to database: %s", err)
	}
	if err = core.DB.CheckCurrentSchema(); err != nil {
		return fmt.Errorf("database failed schema version check: %s", err)
	}

	if core.failsafe.Username != "" {
		log.Infof("checking to see if we should re-instate the failsafe administrator account '%s'", core.failsafe.Username)
		existing, err := core.DB.GetAllUsers(&db.UserFilter{Backend: "local"})
		if err != nil {
			return fmt.Errorf("Failed to retrieve list of local users: %s", err)
		}
		if len(existing) == 0 {
			log.Infof("no local users detected; creating failsafe administrator account '%s'", core.failsafe.Username)
			user := &db.User{
				Name:    "Administrator",
				Account: core.failsafe.Username,
				Backend: "local",
				SysRole: "admin",
			}

			user.SetPassword(core.failsafe.Password)
			_, err := core.DB.CreateUser(user)
			if err != nil {
				return fmt.Errorf("Failed to create failsafe administative account '%s': %s", core.failsafe.Username, err)
			}
		}
	}

	log.Infof("Purging prior authenticated sessions from previous SHIELD instance.")
	core.DB.Exec("DELETE FROM `sessions`")

	tenants := make(map[string]bool)
	for _, auth := range core.auth {
		for _, tenant := range auth.ReferencedTenants() {
			if tenant != "SYSTEM" {
				tenants[tenant] = true
			}
		}
	}
	for tenant := range tenants {
		if _, err := core.DB.EnsureTenant(tenant); err != nil {
			return fmt.Errorf("unable to pre-create tenant '%s' (referenced in authentication providers): %s", err)
		}
	}

	if err = core.fixups(); err != nil {
		return fmt.Errorf("failed to run (idempotent) fixups against database: %s", err)
	}
	core.cleanup()

	core.vault, err = crypter.NewVault(core.vaultAddress, core.vaultCACert)
	if err != nil {
		log.Errorf("Failed to create core vault instance with error: %s", err)
		os.Exit(2)
	}

	if vault_status, err := core.vault.Status(); err != nil || vault_status != "unsealed" {
		if err != nil {
			return err
		}
		log.Errorf("Vault is currently %s, please initialize or unseal the vault via the WebUI or CLI", vault_status)
	}

	core.api()
	core.runWorkers()
	core.runBroadcast()

	for {
		select {
		case <-core.fastloop.C:
			sealed, err := core.vault.IsSealed()
			initialized, initErr := core.vault.IsInitialized()
			if initialized && !sealed {
				core.scheduleTasks()
				core.runPending()
				core.ShouldIDie()
				core.checkPendingAgents()
			} else {
				if err != nil || initErr != nil {
					log.Errorf("Failed to schedule tasks due to Vault error: %s %s", err, initErr)
				}
			}

		case <-core.slowloop.C:
			core.expireArchives()
			core.markTasks()
			core.checkAllAgents()
			core.updateStorageUsage()
			core.purgeExpiredSessions()

			sealed, _ := core.vault.IsSealed()
			initialized, _ := core.vault.IsInitialized()
			if initialized && !sealed {
				core.purge()
				core.testStorage()
			}
		}
	}
}

func (core *Core) api() {
	http.Handle("/v1/", core)
	http.Handle("/v2/", core.v2API())
	http.Handle("/auth/", core)
	http.Handle("/init.js", core)
	http.Handle("/", http.FileServer(http.Dir(core.webroot)))

	log.Infof("starting up api listener on %s", core.listen)
	go func() {
		err := http.ListenAndServe(core.listen, nil)
		if err != nil {
			log.Errorf("shield core api failed to start up: %s", err)
			os.Exit(2)
		}
		log.Infof("shutting down shield core api")
	}()
}

func (core *Core) runBroadcast() {
	go func() {
		for ev := range core.events {
			core.broadcast.Broadcast(ev)
		}
	}()
}

func (core *Core) runWorkers() {
	log.Infof("shield core spinning %d worker threads", core.numWorkers)
	for id := 1; id <= core.numWorkers; id++ {
		log.Debugf("spawning worker %d", id)
		go core.worker(id)
	}
}

/* it's necessary to restart SHIELD after self-restore, however we cannot
call os.Exit() from the API, as the HTTP response will never be sent.
it is then necessary to create a function within the fastloop that
checks the 'seppuku' flag, and if true, SHIELD core calls os.Exit() */
func (core *Core) ShouldIDie() {
	if core.seppuku {
		os.Exit(0)
	}
}

func (core *Core) cleanup() {
	tasks, err := core.DB.GetAllTasks(&db.TaskFilter{ForStatus: db.RunningStatus})
	if err != nil {
		log.Errorf("failed to cleanup leftover running tasks: %s", err)
		return
	}

	now := time.Now()
	for _, task := range tasks {
		log.Warnf("found task %s in 'running' state at startup; setting to 'failed'", task.UUID)
		if err := core.DB.FailTask(task.UUID, now); err != nil {
			log.Errorf("failed to sweep database of running tasks [%s]: %s", task.UUID, err)
			continue
		}

		if task.Op == db.BackupOperation && task.ArchiveUUID != nil {
			archive, err := core.DB.GetArchive(task.ArchiveUUID)
			if err != nil {
				log.Warnf("unable to retrieve archive %s (for task %s) from the database: %s",
					task.ArchiveUUID, task.UUID, err)
				continue
			}
			log.Warnf("found archive %s for task %s, purging", archive.UUID, task.UUID)
			task, err := core.DB.CreatePurgeTask("", archive, core.purgeAgent)
			if err != nil {
				log.Errorf("failed to purge archive %s (for task %s, which was running at boot): %s",
					archive.UUID, task.UUID, err)
			}
		}
	}
}

func (core *Core) scheduleTasks() {
	l, err := core.DB.GetAllJobs(&db.JobFilter{Overdue: true})
	if err != nil {
		log.Errorf("error retrieving all overdue jobs from database: %s", err)
		return
	}

	for _, job := range l {
		log.Infof("scheduling a run of job %s [%s]", job.Name, job.UUID)
		core.DB.CreateBackupTask("system", job)

		if spec, err := timespec.Parse(job.Schedule); err != nil {
			log.Errorf("error re-scheduling job %s [%s]: %s", job.Name, job.UUID, err)
		} else {
			if next, err := spec.Next(time.Now()); err != nil {
				log.Errorf("error re-scheduling job %s [%s]: %s", job.Name, job.UUID, err)
			} else {
				if err = core.DB.RescheduleJob(job, next); err != nil {
					log.Errorf("error re-scheduling job %s [%s]: %s", job.Name, job.UUID, err)
				}
			}
		}
	}
}

func (core *Core) runPending() {
	l, err := core.DB.GetAllTasks(&db.TaskFilter{ForStatus: "pending"})
	if err != nil {
		log.Errorf("error retrieving pending tasks from database: %s", err)
		return
	}

	for _, task := range l {
		/* set up the deadline for execution */
		task.TimeoutAt = time.Now().Unix() + (int64)(core.timeout)
		log.Infof("schedule task %s with deadline %v", task.UUID, task.TimeoutAt)

		/* mark the task as scheduled, so we don't pick it up again */
		core.DB.ScheduledTask(task.UUID)

		/* spin up a goroutine so that we can block in the write
		   to the workers channel, yet return immediately to here,
		   and 'queue up' the remaining pending tasks */
		go func(task db.Task) {
			core.workers <- &task
			log.Debugf("dispatched task %s to a worker goroutine", task.UUID)
		}(*task)
	}
}

func (core *Core) expireArchives() {
	log.Debugf("scanning for archives that outlived their retention policy")
	l, err := core.DB.GetExpiredArchives()
	if err != nil {
		log.Errorf("error retrieving archives that have outlived their retention policy: %s", err)
		return

	}
	for _, archive := range l {
		log.Infof("marking archive %s has expiration date %s, marking as expired", archive.UUID, archive.ExpiresAt)
		err := core.DB.ExpireArchive(archive.UUID)
		if err != nil {
			log.Errorf("error marking archive %s as expired: %s", archive.UUID, err)
			continue
		}
	}
}

func (core *Core) purge() {
	log.Debugf("scanning for archvies that need purged")
	l, err := core.DB.GetArchivesNeedingPurge()
	if err != nil {
		log.Errorf("error retrieving archives to purge: %s", err)
		return
	}

	for _, archive := range l {
		log.Infof("requesting purge of archive %s due to status '%s'", archive.UUID, archive.Status)
		_, err := core.DB.CreatePurgeTask("system", archive, core.purgeAgent)
		if err != nil {
			log.Errorf("error scheduling purge of archive %s: %s", archive.UUID, err)
			continue
		}
	}
}

func (core *Core) markTasks() {
	core.DB.MarkTasksIrrelevant()
}

func (core *Core) checkAgents(agents []*db.Agent) {
	for _, a := range agents {
		if c, ok := core.agents[a.Address]; ok {
			select {
			case c <- a:
				log.Infof("monitor: dispatched agent health check for '%s' to a monitor thread", a.Address)

			default:
				log.Infof("monitor: dropped agent health check for '%s'; there is already an operation in-flight",
					a.Address)
			}
			return
		}

		/* spin up a new goroutine to this and future
		   health checks of this SHIELD agent */
		core.agents[a.Address] = make(chan *db.Agent)
		go func(in chan *db.Agent) {
			for a := range in {
				func() {
					stdout := make(chan string, 1)
					stderr := make(chan string)
					go func() {
						for s := range stderr {
							log.Debugf("  [monitor] %s> %s", a.Address, strings.Trim(s, "\n"))
						}
					}()

					if err := core.agent.Run(a.Address, stdout, stderr, &AgentCommand{Op: "status"}); err != nil {
						log.Errorf("  [monitor] %s: !! failed to run status op: %s", a.Address, err)

						a.Status = "failing"
						a.LastError = fmt.Sprintf("failed to run status op: %s", err)

						log.Debugf("  [monitor] %s> updating (agent=%s) with status '%s'...", a.Address, a.UUID, a.Status)
						if err := core.DB.UpdateAgent(a); err != nil {
							log.Errorf("  [monitor] %s: !! failed to update database: %s", a.Address, err)
						}
						return
					}

					response := <-stdout

					var x struct {
						Name    string `json:"name"`
						Version string `json:"version"`
						Health  string `json:"health"`
					}
					if err := json.Unmarshal([]byte(response), &x); err != nil {
						log.Errorf("  [monitor] %s: !! failed to parse status op response: %s", a.Address, err)

						a.Status = "failing"
						a.LastError = fmt.Sprintf("failed to parse status op response: %s", err)

						log.Debugf("  [monitor] %s> updating (agent=%s) with status '%s'...", a.Address, a.UUID, a.Status)
						if err := core.DB.UpdateAgent(a); err != nil {
							log.Errorf("  [monitor] %s: !! failed to update database: %s", a.Address, err)
						}
						return
					}

					if a.Name != x.Name {
						log.Errorf("  [monitor] %s: !! got response for agent '%s' (not '%s')", a.Address, x.Name, a.Name)

						a.Status = "degraded"
						a.LastError = fmt.Sprintf("got response for agent '%s' (not '%s')", x.Name, a.Name)

						log.Debugf("  [monitor] %s> updating (agent=%s) with status '%s'...", a.Address, a.UUID, a.Status)
						if err := core.DB.UpdateAgent(a); err != nil {
							log.Errorf("  [monitor] %s: !! failed to update database: %s", a.Address, err)
						}
						return
					}

					a.Status = x.Health
					a.Version = x.Version
					a.RawMeta = response

					log.Debugf("  [monitor] %s> updating (agent=%s) with status '%s'...", a.Address, a.UUID, a.Status)
					if err := core.DB.UpdateAgent(a); err != nil {
						log.Errorf("  [monitor] %s: !! failed to update database: %s", a.Address, err)
					}
				}()
			}
		}(core.agents[a.Address])
		core.agents[a.Address] <- a
	}
}

func (core *Core) checkAllAgents() {
	log.Debugf("scanning for agents that need to be checked")

	agents, err := core.DB.GetAllAgents(nil)
	if err != nil {
		log.Errorf("error retrieving agent registration records from database: %s", err)
		return
	}
	core.checkAgents(agents)
}

func (core *Core) checkPendingAgents() {
	agents, err := core.DB.GetAllAgents(&db.AgentFilter{Status: "pending"})
	if err != nil {
		log.Errorf("error retrieving agent registration records from database: %s", err)
		return
	}
	core.checkAgents(agents)
}

func (core *Core) worker(id int) {
	/* read a task from the core */
	for task := range core.workers {
		log.Debugf("worker %d starting to execute task %s", id, task.UUID)

		core.startTask(task)
		if task.Agent == "" {
			core.failTask(task, "no remote agent specified for task %s", task.UUID)
			continue
		}

		stdout := make(chan string, 1)
		stderr := make(chan string)
		go func() {
			for s := range stderr {
				core.logToTask(task, s)
			}
		}()

		if task.Op == db.BackupOperation {
			task.ArchiveUUID = uuid.NewRandom()
			var enc_key, enc_iv string
			if task.FixedKey {
				data, exists, err := core.vault.Get("secret/archives/fixed_key")
				if err != nil || !exists {
					core.failTask(task, "shield worker %d unable retrieve fixed-key encryption parameters: %s\n", id, err)
					continue
				}
				enc_key = core.vault.ASCIIHexDecode(data["key"].(string))
				enc_iv = core.vault.ASCIIHexDecode(data["iv"].(string))
			} else {
				var err error
				enc_key, enc_iv, err = core.vault.CreateBackupEncryptionConfig(core.encryptionType)
				if err != nil {
					core.failTask(task, "shield worker %d failed to generate encryption parameters: %s\n", id, err)
					continue
				}
			}

			err := core.vault.Put("secret/archives/"+task.ArchiveUUID.String(), map[string]interface{}{
				"key":  core.vault.ASCIIHexEncode(enc_key, 4),
				"iv":   core.vault.ASCIIHexEncode(enc_iv, 4),
				"type": core.encryptionType,
				"uuid": task.ArchiveUUID.String(),
			})

			if err != nil {
				core.failTask(task, "shield worker %d failed to set encryption parameters: %s\n", id, err)
				continue
			}
		}

		var encType, encKey, encIV string
		if task.Op == db.BackupOperation || task.Op == db.RestoreOperation {
			data, exists, err := core.vault.Get("secret/archives/" + task.ArchiveUUID.String())
			if err != nil {
				core.failTask(task, "shield worker %d unable retrieve encryption parameters: %s\n", id, err)
				continue
			}

			if exists {
				encType = data["type"].(string)
				encKey = data["key"].(string)
				encIV = data["iv"].(string)
			}
		}
		/* connect to the remote SSH agent for this specific request
		   (a worker may connect to lots of different agents in its
		    lifetime; these connections endure long enough to submit
		    the agent command and gather the exit code + output) */
		err := core.agent.Run(task.Agent, stdout, stderr, &AgentCommand{
			Op:             task.Op,
			TargetPlugin:   task.TargetPlugin,
			TargetEndpoint: task.TargetEndpoint,
			StorePlugin:    task.StorePlugin,
			StoreEndpoint:  task.StoreEndpoint,
			RestoreKey:     task.RestoreKey,
			EncryptType:    encType,
			EncryptKey:     encKey,
			EncryptIV:      encIV,
		})

		if err != nil {
			/* N.B.: there is a temptation to print the error here, but in all the
			   time we've run this code in production, the error message is
			   ALWAYS 'process exited X, reason was ()', which is useless. */
			core.failTask(task, "shield worker %d unable to run command against %s\n", id, task.Agent)
			if task.Op == db.TestStoreOperation {
				store, err := core.DB.GetStore(task.StoreUUID)
				if err != nil {
					log.Errorf("error retrieving store %s from task %s:  %s", task.StoreUUID, task.UUID, err)
				}
				if store == nil {
					core.failTask(task, "shield worker %d unable to retrieve store object from database", id)
					continue
				}
				log.Infof("marking store %s [%s] as unhealthy", store.Name, store.Healthy)
				store.Healthy = false
				err = core.DB.UpdateStore(store)
				if err != nil {
					log.Errorf("error updating store: %s", err)
				}
			}
			continue
		}

		response := <-stdout
		if task.Op == db.BackupOperation {
			var v struct {
				Key  string `json:"key"`
				Size int64  `json:"archive_size"`
			}
			if err := json.Unmarshal([]byte(response), &v); err != nil {
				core.failTask(task, "shield worker %d failed to parse JSON response from remote agent %s: %s\n", id, task.Agent, err)
				continue

			} else {
				if v.Key != "" {
					log.Infof("  %s: restore key is %s", task.UUID, v.Key)
					if _, err := core.DB.CreateTaskArchive(task.UUID, task.ArchiveUUID, v.Key, time.Now(), core.encryptionType, v.Size, task.TenantUUID); err != nil {
						log.Errorf("  %s: !! failed to update database: %s", task.UUID, err)
					}

				} else {
					core.failTask(task, "shield worker %d did not detect a restore key in the store plugin output\nCowardly refusing to create an archive record\n", id)
					continue
				}
			}
		}

		if task.Op == db.PurgeOperation {
			log.Infof("  %s: archive %s purged from storage", task.UUID, task.ArchiveUUID)
			if err := core.DB.PurgeArchive(task.ArchiveUUID); err != nil {
				log.Errorf("  %s: !! failed to update database: %s", task.UUID, err)
			}
		}

		if task.Op == db.TestStoreOperation {
			var v struct {
				Healthy bool `json:"healthy"`
			}

			if err := json.Unmarshal([]byte(response), &v); err != nil {
				core.failTask(task, "shield worker %d failed to parse JSON response from remote agent %s: %s\n", id, task.Agent, err)
				continue

			} else {
				store, err := core.DB.GetStore(task.StoreUUID)
				if err != nil {
					log.Errorf("error retrieving store %s from task %s:  %s", task.StoreUUID, task.UUID, err)
				}
				if store == nil {
					core.failTask(task, "shield worker %d unable to retrieve store object from database", id)
					continue
				}
				store.Healthy = v.Healthy
				err = core.DB.UpdateStore(store)
				if err != nil {
					log.Errorf("error updating store: %s", err)
				}
			}
			log.Infof("  %s: cloud storage %s tested", task.UUID, task.StoreUUID)
		}

		log.Infof("  %s: job completed successfully", task.UUID)
		core.finishTask(task)
		if task.Op == db.BackupOperation {
			log.Infof("  %s: triggering an update to storage analytics", task.UUID)
			core.updateStorageUsage()
		}
	}
}

func networkIdentity() (string, string) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "(unknown)", ""
	}

	var v4ip, v6ip, host string

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var (
				found bool
				ip    net.IP
			)

			switch addr.(type) {
			case *net.IPNet:
				ip = addr.(*net.IPNet).IP
				found = !ip.IsLoopback()
			case *net.IPAddr:
				ip = addr.(*net.IPAddr).IP
				found = !ip.IsLoopback()
			}
			log.Debugf("net: found interface with address %s", ip.String())
			isv4 := ip.To4() != nil
			log.Debugf("net: (found=%v, isv4=%v, v4ip=%s, v6ip=%s)",
				found, isv4, v4ip, v6ip)
			if !found || (!isv4 && v6ip != "") || (isv4 && v4ip != "") {
				log.Debugf("net: SKIPPING")
				continue
			}

			if isv4 {
				v4ip = ip.String()
			} else {
				v6ip = ip.String()
			}

			names, err := net.LookupAddr(ip.String())
			if err != nil {
				continue
			}
			if len(names) != 0 {
				host = names[0]
			}
		}
	}

	if v4ip != "" {
		return v4ip, host
	}
	if v6ip != "" {
		return v6ip, host
	}
	return "(unknown)", ""
}
