package db_test

import (
	. "github.com/starkandwayne/shield/db"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
	"time"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"
)

var _ = Describe("Task Management", func() {
	JOB_UUID := uuid.NewRandom()
	TARGET_UUID := uuid.NewRandom()
	STORE_UUID := uuid.NewRandom()
	RETENTION_UUID := uuid.NewRandom()

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
			`INSERT INTO targets (uuid) VALUES ("` + TARGET_UUID.String() +`")`,

			// need a store
			`INSERT INTO stores (uuid) VALUES ("` + STORE_UUID.String() +`")`,

			// need a retention policy
			`INSERT INTO retention (uuid, expiry) VALUES ("` + RETENTION_UUID.String() +`", 3600)`,

			// need a job
			`INSERT INTO jobs (uuid, target_uuid, store_uuid, retention_uuid)
				VALUES ("` + JOB_UUID.String() + `", "` + TARGET_UUID.String() + `",
				        "` + STORE_UUID.String() + `", "` + RETENTION_UUID.String() + `")`,

		)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(db).ShouldNot(BeNil())
	})

	It("Can create a new task", func() {
		id, err := db.CreateTask("owner-name", "backup", `{"args":"test"}`, JOB_UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(id).ShouldNot(BeNil())

		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE uuid = ?`, id.String())
		shouldExist(`SELECT * FROM tasks WHERE owner = ?`, "owner-name")
		shouldExist(`SELECT * FROM tasks WHERE op = ?`, "backup")
		shouldExist(`SELECT * FROM tasks WHERE args = ?`, `{"args":"test"}`)
		shouldExist(`SELECT * FROM tasks WHERE job_uuid = ?`, JOB_UUID.String())
		shouldExist(`SELECT * FROM tasks WHERE archive_uuid IS NULL`)
		shouldExist(`SELECT * FROM tasks WHERE status = ?`, "pending")
		shouldExist(`SELECT * FROM tasks WHERE started_at IS NULL`)
		shouldExist(`SELECT * FROM tasks WHERE stopped_at IS NULL`)
	})

	It("Can start an existing task", func() {
		id, err := db.CreateTask("bob", "backup", `ARGS`, JOB_UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(id).ShouldNot(BeNil())

		Ω(db.StartTask(id, time.Now())).Should(Succeed())

		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE status = ?`, "running")
		shouldExist(`SELECT * FROM tasks WHERE started_at IS NOT NULL`)
		shouldExist(`SELECT * FROM tasks WHERE stopped_at IS NULL`)
	})

	It("Can cancel a running task", func() {
		id, err := db.CreateTask("bob", "backup", `ARGS`, JOB_UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(id).ShouldNot(BeNil())

		Ω(db.StartTask(id, time.Now())).Should(Succeed())
		Ω(db.CancelTask(id, time.Now())).Should(Succeed())

		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE status = ?`, "canceled")
		shouldExist(`SELECT * FROM tasks WHERE started_at IS NOT NULL`)
		shouldExist(`SELECT * FROM tasks WHERE stopped_at IS NOT NULL`)
	})

	It("Can complete a running task", func() {
		id, err := db.CreateTask("bob", "backup", `ARGS`, JOB_UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(id).ShouldNot(BeNil())

		Ω(db.StartTask(id, time.Now())).Should(Succeed())
		Ω(db.CompleteTask(id, time.Now())).Should(Succeed())

		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE status = ?`, "done")
		shouldExist(`SELECT * FROM tasks WHERE started_at IS NOT NULL`)
		shouldExist(`SELECT * FROM tasks WHERE stopped_at IS NOT NULL`)
	})

	It("Can update the task log piecemeal", func() {
		id, err := db.CreateTask("bob", "backup", `ARGS`, JOB_UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(id).ShouldNot(BeNil())

		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE log = ?`, "")

		Ω(db.UpdateTaskLog(id, "line 1\n")).Should(Succeed())
		shouldExist(`SELECT * FROM tasks WHERE log = ?`, "line 1\n")

		Ω(db.UpdateTaskLog(id, "\n")).Should(Succeed())
		shouldExist(`SELECT * FROM tasks WHERE log = ?`, "line 1\n\n")

		Ω(db.UpdateTaskLog(id, "line ")).Should(Succeed())
		shouldExist(`SELECT * FROM tasks WHERE log = ?`, "line 1\n\nline ")

		Ω(db.UpdateTaskLog(id, "2\n")).Should(Succeed())
		shouldExist(`SELECT * FROM tasks WHERE log = ?`, "line 1\n\nline 2\n")
	})

	It("Can associate archives with the task", func() {
		id, err := db.CreateTask("bob", "backup", `ARGS`, JOB_UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(id).ShouldNot(BeNil())

		Ω(db.StartTask(id, time.Now())).Should(Succeed())
		Ω(db.CompleteTask(id, time.Now())).Should(Succeed())
		Ω(db.CreateTaskArchive(id, "SOME-KEY", time.Now())).Should(Succeed())

		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE archive_uuid IS NOT NULL`)

		shouldExist(`SELECT * FROM archives`)
		shouldExist(`SELECT * FROM archives WHERE target_uuid = ?`, TARGET_UUID.String())
		shouldExist(`SELECT * FROM archives WHERE store_uuid = ?`, STORE_UUID.String())
		shouldExist(`SELECT * FROM archives WHERE store_key = ?`, "SOME-KEY")
		shouldExist(`SELECT * FROM archives WHERE taken_at IS NOT NULL`)
		shouldExist(`SELECT * FROM archives WHERE expires_at IS NOT NULL`)
	})
})
