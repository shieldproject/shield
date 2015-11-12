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
	ARCHIVE_UUID := uuid.NewRandom()

	var db *DB

	shouldExist := func(q string, params ...interface{}) {
		n, err := db.Count(q, params...)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(n).Should(BeNumerically(">", 0))
	}

	BeforeEach(func() {
		var err error
		db, err = Database()
		Ω(err).ShouldNot(HaveOccurred())
		Ω(db).ShouldNot(BeNil())
	})

	It("Can create a new task", func() {
		id, err := db.CreateTask("owner-name", "backup", `{"args":"test"}`, JOB_UUID, ARCHIVE_UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(id).ShouldNot(BeNil())

		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE uuid = ?`, id.String())
		shouldExist(`SELECT * FROM tasks WHERE owner = ?`, "owner-name")
		shouldExist(`SELECT * FROM tasks WHERE op = ?`, "backup")
		shouldExist(`SELECT * FROM tasks WHERE args = ?`, `{"args":"test"}`)
		shouldExist(`SELECT * FROM tasks WHERE job_uuid = ?`, JOB_UUID.String())
		shouldExist(`SELECT * FROM tasks WHERE archive_uuid = ?`, ARCHIVE_UUID.String())
		shouldExist(`SELECT * FROM tasks WHERE status = ?`, "pending")
		shouldExist(`SELECT * FROM tasks WHERE started_at IS NULL`)
		shouldExist(`SELECT * FROM tasks WHERE stopped_at IS NULL`)
	})

	It("Can start an existing task", func() {
		id, err := db.CreateTask("bob", "backup", `ARGS`, JOB_UUID, ARCHIVE_UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(id).ShouldNot(BeNil())

		Ω(db.StartTask(id, time.Now())).Should(Succeed())

		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE status = ?`, "running")
		shouldExist(`SELECT * FROM tasks WHERE started_at IS NOT NULL`)
		shouldExist(`SELECT * FROM tasks WHERE stopped_at IS NULL`)
	})

	It("Can cancel a running task", func() {
		id, err := db.CreateTask("bob", "backup", `ARGS`, JOB_UUID, ARCHIVE_UUID)
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
		id, err := db.CreateTask("bob", "backup", `ARGS`, JOB_UUID, ARCHIVE_UUID)
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
		id, err := db.CreateTask("bob", "backup", `ARGS`, JOB_UUID, ARCHIVE_UUID)
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
})
