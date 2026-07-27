package db

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	// sql drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// These specs run the real schema migrations and the real queries against a
// live PostgreSQL and MySQL, which is the only way to know that the multi-
// database support actually works -- unit tests over hand-written error
// strings and DDL cannot tell us anything about what a server will accept.
//
// They are skipped unless the corresponding DSN is exported:
//
//	SHIELD_TEST_PG_DSN='postgres://shield:shield@127.0.0.1:15432/shield?sslmode=disable'
//	SHIELD_TEST_MYSQL_DSN='shield:shield@tcp(127.0.0.1:13306)/shield?parseTime=true'
//
// `make test-databases` brings up both servers and exports these for you.

type backend struct {
	name   string
	driver string
	env    string
	// listTables names every table in the working database.
	listTables string
}

var backends = []backend{
	{
		name:   "PostgreSQL",
		driver: "pgx",
		env:    "SHIELD_TEST_PG_DSN",
		listTables: `SELECT tablename FROM pg_tables
		              WHERE schemaname = current_schema()`,
	},
	{
		name:   "MySQL",
		driver: "mysql",
		env:    "SHIELD_TEST_MYSQL_DSN",
		listTables: `SELECT table_name FROM information_schema.tables
		              WHERE table_schema = DATABASE()
		                AND table_type = 'BASE TABLE'`,
	},
}

// connect dials the backend and resets it to an empty database, or skips the
// running spec if no DSN was supplied for it.
func (b backend) connect() *DB {
	GinkgoHelper()

	dsn := os.Getenv(b.env)
	if dsn == "" {
		Skip(b.name + " not configured; export " + b.env + " to run these specs")
	}

	db, err := Connect(b.driver, dsn)
	Ω(err).ShouldNot(HaveOccurred())

	b.wipe(db)
	return db
}

// wipe drops every table, leaving an empty database for Setup() to populate.
//
// DDL goes over the connection directly rather than through db.Exec: MySQL
// will not accept several of these statements over the prepared-statement
// protocol. Drops are retried because the tables carry foreign keys and we do
// not care to work out a safe ordering.
func (b backend) wipe(db *DB) {
	GinkgoHelper()

	for {
		rows, err := db.connection.Query(b.listTables)
		Ω(err).ShouldNot(HaveOccurred())

		tables := []string{}
		for rows.Next() {
			var t string
			Ω(rows.Scan(&t)).Should(Succeed())
			tables = append(tables, t)
		}
		Ω(rows.Err()).ShouldNot(HaveOccurred())
		Ω(rows.Close()).Should(Succeed())

		if len(tables) == 0 {
			return
		}

		dropped := 0
		var lastErr error
		for _, t := range tables {
			if _, err := db.connection.Exec(`DROP TABLE ` + t); err != nil {
				lastErr = err
				continue
			}
			dropped++
		}
		Ω(dropped).Should(BeNumerically(">", 0),
			"made no progress dropping %v: %v", tables, lastErr)
	}
}

var _ = Describe("Multi-Database Support", func() {
	for _, b := range backends {
		b := b

		Describe(b.name, func() {
			var db *DB

			BeforeEach(func() {
				db = b.connect()
			})

			AfterEach(func() {
				if db != nil {
					db.Disconnect()
				}
			})

			It("reports schema version 0 for an empty database", func() {
				v, err := db.SchemaVersion()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(v).Should(Equal(0))
			})

			It("recognizes the driver's missing-table error", func() {
				db.exclusive.Lock()
				_, err := db.query(`SELECT version FROM schema_info LIMIT 1`)
				db.exclusive.Unlock()

				Ω(err).Should(HaveOccurred())
				Ω(IsNoSuchTable(err, "schema_info")).Should(BeTrue(),
					"IsNoSuchTable must recognize %T: %s", err, err)
				Ω(IsNoSuchTable(err, "some_other_table")).Should(BeFalse())
			})

			It("deploys every schema migration", func() {
				v, err := db.Setup(0)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(v).Should(Equal(CurrentSchema))

				Ω(db.CheckCurrentSchema()).Should(Succeed())
			})

			Context("with the schema deployed", func() {
				BeforeEach(func() {
					_, err := db.Setup(0)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("round-trips a tenant", func() {
					tenant, err := db.CreateTenant(&Tenant{Name: "acme"})
					Ω(err).ShouldNot(HaveOccurred())
					Ω(tenant.Name).Should(Equal("acme"))

					got, err := db.GetTenant(tenant.UUID)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(got).ShouldNot(BeNil())
					Ω(got.Name).Should(Equal("acme"))
				})

				It("round-trips a store, target, and job", func() {
					tenant, err := db.CreateTenant(&Tenant{Name: "acme"})
					Ω(err).ShouldNot(HaveOccurred())

					store, err := db.CreateStore(&Store{
						TenantUUID: tenant.UUID,
						Name:       "s3",
						Plugin:     "s3",
						Agent:      "127.0.0.1:5444",
						Config:     map[string]interface{}{"bucket": "backups"},
					})
					Ω(err).ShouldNot(HaveOccurred())

					target, err := db.CreateTarget(&Target{
						TenantUUID: tenant.UUID,
						Name:       "pg",
						Plugin:     "postgres",
						Agent:      "127.0.0.1:5444",
						Config:     map[string]interface{}{"host": "localhost"},
					})
					Ω(err).ShouldNot(HaveOccurred())

					job, err := db.CreateJob(&Job{
						TenantUUID: tenant.UUID,
						Name:       "nightly",
						StoreUUID:  store.UUID,
						TargetUUID: target.UUID,
						Schedule:   "daily 4am",
						KeepN:      10,
						KeepDays:   10,
					})
					Ω(err).ShouldNot(HaveOccurred())

					got, err := db.GetJob(job.UUID)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(got).ShouldNot(BeNil())
					Ω(got.Name).Should(Equal("nightly"))
					Ω(got.Paused).Should(BeFalse())
				})

				// Exercises the boolean round-trip, which is where a schema
				// that declares BOOLEAN but defaults it to an integer breaks.
				It("round-trips boolean columns", func() {
					tenant, err := db.CreateTenant(&Tenant{Name: "acme"})
					Ω(err).ShouldNot(HaveOccurred())

					store, err := db.CreateStore(&Store{
						TenantUUID: tenant.UUID,
						Name:       "s3",
						Plugin:     "s3",
						Agent:      "127.0.0.1:5444",
						Config:     map[string]interface{}{},
					})
					Ω(err).ShouldNot(HaveOccurred())
					Ω(store.Healthy).Should(BeFalse())

					store.Healthy = true
					Ω(db.UpdateStoreHealth(store)).Should(Succeed())

					got, err := db.GetStore(store.UUID)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(got.Healthy).Should(BeTrue())
				})

				// The list queries GROUP BY a single column while selecting
				// many, which PostgreSQL rejects outright and MySQL rejects
				// under ONLY_FULL_GROUP_BY.
				It("runs the aggregate list queries", func() {
					_, err := db.GetAllTargets(&TargetFilter{})
					Ω(err).ShouldNot(HaveOccurred())

					_, err = db.GetAllStores(&StoreFilter{})
					Ω(err).ShouldNot(HaveOccurred())

					_, err = db.GetAllJobs(&JobFilter{})
					Ω(err).ShouldNot(HaveOccurred())

					_, err = db.GetAllTasks(&TaskFilter{})
					Ω(err).ShouldNot(HaveOccurred())

					_, err = db.GetAllArchives(&ArchiveFilter{})
					Ω(err).ShouldNot(HaveOccurred())

					_, err = db.GetAllAgents(&AgentFilter{})
					Ω(err).ShouldNot(HaveOccurred())

					_, err = db.GetAllUsers(&UserFilter{})
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("runs a task through its whole lifecycle", func() {
					job := scratchJob(db)

					task, err := db.CreateBackupTask("system", job)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(task).ShouldNot(BeNil())

					Ω(db.StartTask(task.UUID, time.Now())).Should(Succeed())
					Ω(db.UpdateTaskLog(task.UUID, "working...\n")).Should(Succeed())

					_, err = db.CreateTaskArchive(task.UUID, RandomID(), "some-key",
						time.Now(), "aes256-ctr", "none", 1024, job.TenantUUID)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(db.CompleteTask(task.UUID, time.Now())).Should(Succeed())

					got, err := db.GetTask(task.UUID)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(got).ShouldNot(BeNil())
					Ω(got.OK).Should(BeTrue())
					Ω(got.Log).Should(ContainSubstring("working..."))

					Ω(db.AnnotateTargetTask(job.TargetUUID, task.UUID, &TaskAnnotation{
						Disposition: "ok",
						Notes:       "nothing to see here",
						Clear:       "manual",
					})).Should(Succeed())

					// exercises the boolean `relevant` column
					Ω(db.MarkTasksIrrelevant()).Should(Succeed())
				})

				It("fails a task", func() {
					task, err := db.CreateBackupTask("system", scratchJob(db))
					Ω(err).ShouldNot(HaveOccurred())

					Ω(db.StartTask(task.UUID, time.Now())).Should(Succeed())
					Ω(db.FailTask(task.UUID, time.Now())).Should(Succeed())

					got, err := db.GetTask(task.UUID)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(got.OK).Should(BeFalse())
				})

				It("round-trips a user and a session", func() {
					user, err := db.CreateUser(&User{
						Name:    "Some One",
						Account: "someone",
						Backend: "local",
						SysRole: "admin",
					})
					Ω(err).ShouldNot(HaveOccurred())

					session, err := db.CreateSession(&Session{
						UserUUID:  user.UUID,
						IP:        "10.0.0.1",
						UserAgent: "shield/v10",
					})
					Ω(err).ShouldNot(HaveOccurred())
					Ω(session).ShouldNot(BeNil())

					got, err := db.GetSession(session.UUID)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(got).ShouldNot(BeNil())
				})
			})
		})
	}
})

// scratchJob creates the tenant, store, and target a job needs, and returns
// the job, so that task specs can get to the interesting part.
func scratchJob(db *DB) *Job {
	GinkgoHelper()

	tenant, err := db.CreateTenant(&Tenant{Name: "acme"})
	Ω(err).ShouldNot(HaveOccurred())

	store, err := db.CreateStore(&Store{
		TenantUUID: tenant.UUID,
		Name:       "s3",
		Plugin:     "s3",
		Agent:      "127.0.0.1:5444",
		Config:     map[string]interface{}{"bucket": "backups"},
	})
	Ω(err).ShouldNot(HaveOccurred())

	target, err := db.CreateTarget(&Target{
		TenantUUID: tenant.UUID,
		Name:       "pg",
		Plugin:     "postgres",
		Agent:      "127.0.0.1:5444",
		Config:     map[string]interface{}{"host": "localhost"},
	})
	Ω(err).ShouldNot(HaveOccurred())

	job, err := db.CreateJob(&Job{
		TenantUUID: tenant.UUID,
		Name:       "nightly",
		StoreUUID:  store.UUID,
		TargetUUID: target.UUID,
		Schedule:   "daily 4am",
		KeepN:      10,
		KeepDays:   10,
	})
	Ω(err).ShouldNot(HaveOccurred())

	return job
}
