package db_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
	. "github.com/starkandwayne/shield/db"
	"time"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"
)

var _ = Describe("Task Management", func() {
	JOB_UUID := uuid.NewRandom()
	TARGET_UUID := uuid.NewRandom()
	STORE_UUID := uuid.NewRandom()
	RETENTION_UUID := uuid.NewRandom()
	ARCHIVE_UUID := uuid.NewRandom()

	var db *DB

	shouldExist := func(q string, params ...interface{}) {
		n, err := db.Count(q, params...)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(n).Should(BeNumerically(">", 0))
	}

	BeforeEach(func() {
		var err error
		db, err = Database(
			// need a target
			`INSERT INTO targets (uuid, plugin, endpoint, agent) VALUES ("`+TARGET_UUID.String()+`", "plugin", "endpoint", "127.0.0.1:5444")`,

			// need a store
			`INSERT INTO stores (uuid, plugin, endpoint) VALUES ("`+STORE_UUID.String()+`", "plugin", "endpoint")`,

			// need a retention policy
			`INSERT INTO retention (uuid, expiry) VALUES ("`+RETENTION_UUID.String()+`", 3600)`,

			// need a job
			`INSERT INTO jobs (uuid, target_uuid, store_uuid, retention_uuid, schedule_uuid)
				VALUES ("`+JOB_UUID.String()+`", "`+TARGET_UUID.String()+`",
				        "`+STORE_UUID.String()+`", "`+RETENTION_UUID.String()+`", "ec3e4fe4-99b4-411a-a91e-4887b5929399")`,

			// need an archive
			`INSERT INTO archives (uuid, target_uuid, store_uuid, store_key, taken_at, expires_at)
				VALUES("`+ARCHIVE_UUID.String()+`", "`+TARGET_UUID.String()+`",
						"`+STORE_UUID.String()+`", "key", 0, 0)`,
		)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(db).ShouldNot(BeNil())
	})

	It("Can create a new backup task", func() {
		id, err := db.CreateBackupTask("owner-name", JOB_UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(id).ShouldNot(BeNil())

		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE uuid = $1`, id.String())
		shouldExist(`SELECT * FROM tasks WHERE owner = $1`, "owner-name")
		shouldExist(`SELECT * FROM tasks WHERE op = $1`, "backup")
		shouldExist(`SELECT * FROM tasks WHERE job_uuid = $1`, JOB_UUID.String())
		shouldExist(`SELECT * FROM tasks WHERE archive_uuid IS NULL`)
		shouldExist(`SELECT * from tasks WHERE store_uuid IS NULL`)
		shouldExist(`SELECT * FROM tasks WHERE target_uuid IS NULL`)
		shouldExist(`SELECT * FROM tasks WHERE status = $1`, "pending")
		shouldExist(`SELECT * FROM tasks WHERE requested_at IS NOT NULL`)
		shouldExist(`SELECT * FROM tasks WHERE started_at IS NULL`)
		shouldExist(`SELECT * FROM tasks WHERE stopped_at IS NULL`)
	})

	It("Can create a new purge task", func() {
		archive, err := db.GetAnnotatedArchive(ARCHIVE_UUID)
		Expect(err).ShouldNot(HaveOccurred())
		id, err := db.CreatePurgeTask("owner-name", archive)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(id).ShouldNot(BeNil())

		shouldExist(`SELECT * from tasks`)
		shouldExist(`SELECT * FROM tasks WHERE uuid = $1`, id.String())
		shouldExist(`SELECT * FROM tasks WHERE owner = $1`, "owner-name")
		shouldExist(`SELECT * FROM tasks WHERE op = $1`, "purge")
		shouldExist(`SELECT * FROM tasks WHERE archive_uuid = $1`, ARCHIVE_UUID.String())
		shouldExist(`SELECT * FROM tasks WHERE target_uuid IS NULL`)
		shouldExist(`SELECT * FROM tasks WHERE store_uuid = $1`, STORE_UUID.String())
		shouldExist(`SELECT * FROM tasks WHERE job_uuid IS NULL`)
		shouldExist(`SELECT * FROM tasks WHERE status = $1`, "pending")
		shouldExist(`SELECT * FROM tasks WHERE requested_at IS NOT NULL`)
		shouldExist(`SELECT * FROM tasks WHERE started_at IS NULL`)
		shouldExist(`SELECT * FROM tasks WHERE stopped_at IS NULL`)
	})

	It("Can create a new restore task", func() {
		id, err := db.CreateRestoreTask("owner-name", ARCHIVE_UUID, TARGET_UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(id).ShouldNot(BeNil())

		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE uuid = $1`, id.String())
		shouldExist(`SELECT * FROM tasks WHERE owner = $1`, "owner-name")
		shouldExist(`SELECT * FROM tasks WHERE op = $1`, "restore")
		shouldExist(`SELECT * FROM tasks WHERE archive_uuid = $1`, ARCHIVE_UUID.String())
		shouldExist(`SELECT * FROM tasks WHERE target_uuid = $1`, TARGET_UUID.String())
		shouldExist(`SELECT * from tasks WHERE store_uuid IS NULL`)
		shouldExist(`SELECT * FROM tasks WHERE job_uuid IS NULL`)
		shouldExist(`SELECT * FROM tasks WHERE status = $1`, "pending")
		shouldExist(`SELECT * FROM tasks WHERE requested_at IS NOT NULL`)
		shouldExist(`SELECT * FROM tasks WHERE started_at IS NULL`)
		shouldExist(`SELECT * FROM tasks WHERE stopped_at IS NULL`)
	})

	It("Can start an existing task", func() {
		id, err := db.CreateBackupTask("bob", JOB_UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(id).ShouldNot(BeNil())

		Ω(db.StartTask(id, time.Now())).Should(Succeed())

		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE status = $1`, "running")
		shouldExist(`SELECT * FROM tasks WHERE started_at IS NOT NULL`)
		shouldExist(`SELECT * FROM tasks WHERE stopped_at IS NULL`)
	})

	It("Can cancel a running task", func() {
		id, err := db.CreateBackupTask("bob", JOB_UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(id).ShouldNot(BeNil())

		Ω(db.StartTask(id, time.Now())).Should(Succeed())
		Ω(db.CancelTask(id, time.Now())).Should(Succeed())

		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE status = $1`, "canceled")
		shouldExist(`SELECT * FROM tasks WHERE started_at IS NOT NULL`)
		shouldExist(`SELECT * FROM tasks WHERE stopped_at IS NOT NULL`)
	})

	It("Can complete a running task", func() {
		id, err := db.CreateBackupTask("bob", JOB_UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(id).ShouldNot(BeNil())

		Ω(db.StartTask(id, time.Now())).Should(Succeed())
		Ω(db.CompleteTask(id, time.Now())).Should(Succeed())

		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE status = $1`, "done")
		shouldExist(`SELECT * FROM tasks WHERE started_at IS NOT NULL`)
		shouldExist(`SELECT * FROM tasks WHERE stopped_at IS NOT NULL`)
	})

	It("Can update the task log piecemeal", func() {
		id, err := db.CreateBackupTask("bob", JOB_UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(id).ShouldNot(BeNil())

		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE log = $1`, "")

		Ω(db.UpdateTaskLog(id, "line 1\n")).Should(Succeed())
		shouldExist(`SELECT * FROM tasks WHERE log = $1`, "line 1\n")

		Ω(db.UpdateTaskLog(id, "\n")).Should(Succeed())
		shouldExist(`SELECT * FROM tasks WHERE log = $1`, "line 1\n\n")

		Ω(db.UpdateTaskLog(id, "line ")).Should(Succeed())
		shouldExist(`SELECT * FROM tasks WHERE log = $1`, "line 1\n\nline ")

		Ω(db.UpdateTaskLog(id, "2\n")).Should(Succeed())
		shouldExist(`SELECT * FROM tasks WHERE log = $1`, "line 1\n\nline 2\n")
	})

	It("Can associate archives with the task", func() {
		id, err := db.CreateBackupTask("bob", JOB_UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(id).ShouldNot(BeNil())

		Ω(db.StartTask(id, time.Now())).Should(Succeed())
		Ω(db.CompleteTask(id, time.Now())).Should(Succeed())
		archive_id, err := db.CreateTaskArchive(id, "SOME-KEY", time.Now())
		Expect(err).ShouldNot(HaveOccurred())
		Expect(id).ShouldNot(BeNil())

		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE archive_uuid IS NOT NULL`)

		shouldExist(`SELECT * FROM archives`)
		shouldExist(`SELECT * FROM archives WHERE uuid = $1`, archive_id.String())
		shouldExist(`SELECT * FROM archives WHERE target_uuid = $1`, TARGET_UUID.String())
		shouldExist(`SELECT * FROM archives WHERE store_uuid = $1`, STORE_UUID.String())
		shouldExist(`SELECT * FROM archives WHERE store_key = $1`, "SOME-KEY")
		shouldExist(`SELECT * FROM archives WHERE taken_at IS NOT NULL`)
		shouldExist(`SELECT * FROM archives WHERE expires_at IS NOT NULL`)
	})
})
