package db_test

import (
	"fmt"
	"github.com/starkandwayne/shield/timestamp"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	. "github.com/starkandwayne/shield/db"
)

var T0 = time.Date(1997, 8, 29, 2, 14, 0, 0, time.UTC)

func at(seconds int) time.Time {
	return T0.Add(time.Duration(seconds) * time.Second)
}

var _ = Describe("Task Management", func() {
	var (
		db *DB

		SomeJob       *Job
		SomeTarget    *Target
		SomeStore     *Store
		SomeRetention *RetentionPolicy
		SomeSchedule  *Schedule
		SomeArchive   *Archive
	)

	shouldExist := func(q string, params ...interface{}) {
		n, err := db.Count(q, params...)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(n).Should(BeNumerically(">", 0))
	}

	shouldNotExist := func(q string, params ...interface{}) {
		n, err := db.Count(q, params...)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(n).Should(BeNumerically("==", 0))
	}

	BeforeEach(func() {
		var err error
		SomeJob = &Job{UUID: uuid.NewRandom()}
		SomeTarget = &Target{UUID: uuid.NewRandom()}
		SomeStore = &Store{UUID: uuid.NewRandom()}
		SomeRetention = &RetentionPolicy{UUID: uuid.NewRandom()}
		SomeSchedule = &Schedule{UUID: uuid.NewRandom()}
		SomeArchive = &Archive{UUID: uuid.NewRandom()}

		db, err = Database(
			// need a target
			`INSERT INTO targets (uuid, name, summary, plugin, endpoint, agent)
			   VALUES ("`+SomeTarget.UUID.String()+`", "Some Target", "", "plugin", "endpoint", "127.0.0.1:5444")`,

			// need a store
			`INSERT INTO stores (uuid, name, summary, plugin, endpoint)
			   VALUES ("`+SomeStore.UUID.String()+`", "Some Store", "", "plugin", "endpoint")`,

			// need a retention policy
			`INSERT INTO retention (uuid, name, summary, expiry)
			   VALUES ("`+SomeRetention.UUID.String()+`", "Some Retention", "", 3600)`,

			// need a schedule
			`INSERT INTO schedules (uuid, name, summary, timespec)
			   VALUES ("`+SomeSchedule.UUID.String()+`", "Some Schedule", "", "daily 4am")`,

			// need a job
			`INSERT INTO jobs (uuid, name, summary, paused,
			                   target_uuid, store_uuid, retention_uuid, schedule_uuid)
			   VALUES ("`+SomeJob.UUID.String()+`", "Some Job", "just a job...", 0,
			           "`+SomeTarget.UUID.String()+`", "`+SomeStore.UUID.String()+`",
			           "`+SomeRetention.UUID.String()+`", "`+SomeSchedule.UUID.String()+`")`,

			// need an archive
			`INSERT INTO archives (uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, notes, status, purge_reason)
			    VALUES("`+SomeArchive.UUID.String()+`", "`+SomeTarget.UUID.String()+`",
			           "`+SomeStore.UUID.String()+`", "key", 0, 0, "(no notes)", "valid", "")`,
		)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(db).ShouldNot(BeNil())

		SomeJob, err = db.GetJob(SomeJob.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(SomeJob).ShouldNot(BeNil())

		SomeTarget, err = db.GetTarget(SomeTarget.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(SomeTarget).ShouldNot(BeNil())

		SomeStore, err = db.GetStore(SomeStore.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(SomeStore).ShouldNot(BeNil())

		SomeRetention, err = db.GetRetentionPolicy(SomeRetention.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(SomeRetention).ShouldNot(BeNil())

		SomeSchedule, err = db.GetSchedule(SomeSchedule.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(SomeSchedule).ShouldNot(BeNil())

		SomeArchive, err = db.GetArchive(SomeArchive.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(SomeArchive).ShouldNot(BeNil())
	})

	It("Can create a new backup task", func() {
		task, err := db.CreateBackupTask("owner-name", SomeJob)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(task).ShouldNot(BeNil())

		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE uuid = ?`, task.UUID.String())
		shouldExist(`SELECT * FROM tasks WHERE owner = ?`, "owner-name")
		shouldExist(`SELECT * FROM tasks WHERE op = ?`, BackupOperation)
		shouldExist(`SELECT * FROM tasks WHERE job_uuid = ?`, SomeJob.UUID.String())
		shouldExist(`SELECT * FROM tasks WHERE archive_uuid IS NULL`)
		shouldExist(`SELECT * from tasks WHERE store_uuid = ?`, SomeStore.UUID.String())
		shouldExist(`SELECT * FROM tasks WHERE target_uuid = ?`, SomeTarget.UUID.String())
		shouldExist(`SELECT * FROM tasks WHERE status = ?`, PendingStatus)
		shouldExist(`SELECT * FROM tasks WHERE requested_at IS NOT NULL`)
		shouldExist(`SELECT * FROM tasks WHERE started_at IS NULL`)
		shouldExist(`SELECT * FROM tasks WHERE stopped_at IS NULL`)
	})

	It("Can create a new purge task", func() {
		archive, err := db.GetArchive(SomeArchive.UUID)
		Expect(err).ShouldNot(HaveOccurred())

		task, err := db.CreatePurgeTask("owner-name", archive, "127.0.0.1:9938")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(task).ShouldNot(BeNil())

		shouldExist(`SELECT * from tasks`)
		shouldExist(`SELECT * FROM tasks WHERE uuid = ?`, task.UUID.String())
		shouldExist(`SELECT * FROM tasks WHERE owner = ?`, "owner-name")
		shouldExist(`SELECT * FROM tasks WHERE op = ?`, PurgeOperation)
		shouldExist(`SELECT * FROM tasks WHERE archive_uuid = ?`, SomeArchive.UUID.String())
		shouldExist(`SELECT * FROM tasks WHERE target_uuid IS NULL`)
		shouldExist(`SELECT * FROM tasks WHERE store_uuid = ?`, SomeStore.UUID.String())
		shouldExist(`SELECT * FROM tasks WHERE job_uuid IS NULL`)
		shouldExist(`SELECT * FROM tasks WHERE status = ?`, PendingStatus)
		shouldExist(`SELECT * FROM tasks WHERE requested_at IS NOT NULL`)
		shouldExist(`SELECT * FROM tasks WHERE started_at IS NULL`)
		shouldExist(`SELECT * FROM tasks WHERE stopped_at IS NULL`)
		shouldExist(`SELECT * FROM tasks WHERE agent = ?`, "127.0.0.1:9938")
	})

	It("Can create a new restore task", func() {
		task, err := db.CreateRestoreTask("owner-name", SomeArchive, SomeTarget)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(task).ShouldNot(BeNil())

		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE uuid = ?`, task.UUID.String())
		shouldExist(`SELECT * FROM tasks WHERE owner = ?`, "owner-name")
		shouldExist(`SELECT * FROM tasks WHERE op = ?`, RestoreOperation)
		shouldExist(`SELECT * FROM tasks WHERE archive_uuid = ?`, SomeArchive.UUID.String())
		shouldExist(`SELECT * FROM tasks WHERE target_uuid = ?`, SomeTarget.UUID.String())
		shouldExist(`SELECT * from tasks WHERE store_uuid = ?`, SomeStore.UUID.String())
		shouldExist(`SELECT * FROM tasks WHERE job_uuid IS NULL`)
		shouldExist(`SELECT * FROM tasks WHERE status = ?`, PendingStatus)
		shouldExist(`SELECT * FROM tasks WHERE requested_at IS NOT NULL`)
		shouldExist(`SELECT * FROM tasks WHERE started_at IS NULL`)
		shouldExist(`SELECT * FROM tasks WHERE stopped_at IS NULL`)
	})

	It("Can start an existing task", func() {
		task, err := db.CreateBackupTask("bob", SomeJob)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(task).ShouldNot(BeNil())

		Ω(db.StartTask(task.UUID, time.Now())).Should(Succeed())

		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE status = ?`, RunningStatus)
		shouldExist(`SELECT * FROM tasks WHERE started_at IS NOT NULL`)
		shouldExist(`SELECT * FROM tasks WHERE stopped_at IS NULL`)
	})

	It("Can cancel a running task", func() {
		task, err := db.CreateBackupTask("bob", SomeJob)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(task).ShouldNot(BeNil())

		Ω(db.StartTask(task.UUID, time.Now())).Should(Succeed())
		Ω(db.CancelTask(task.UUID, time.Now())).Should(Succeed())

		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE status = ?`, CanceledStatus)
		shouldExist(`SELECT * FROM tasks WHERE started_at IS NOT NULL`)
		shouldExist(`SELECT * FROM tasks WHERE stopped_at IS NOT NULL`)
	})

	It("Can complete a running task", func() {
		task, err := db.CreateBackupTask("bob", SomeJob)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(task).ShouldNot(BeNil())

		Ω(db.StartTask(task.UUID, time.Now())).Should(Succeed())
		Ω(db.CompleteTask(task.UUID, time.Now())).Should(Succeed())

		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE status = ?`, DoneStatus)
		shouldExist(`SELECT * FROM tasks WHERE started_at IS NOT NULL`)
		shouldExist(`SELECT * FROM tasks WHERE stopped_at IS NOT NULL`)
	})

	It("Can update the task log piecemeal", func() {
		task, err := db.CreateBackupTask("bob", SomeJob)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(task).ShouldNot(BeNil())

		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE log = ?`, "")

		Ω(db.UpdateTaskLog(task.UUID, "line 1\n")).Should(Succeed())
		shouldExist(`SELECT * FROM tasks WHERE log = ?`, "line 1\n")

		Ω(db.UpdateTaskLog(task.UUID, "\n")).Should(Succeed())
		shouldExist(`SELECT * FROM tasks WHERE log = ?`, "line 1\n\n")

		Ω(db.UpdateTaskLog(task.UUID, "line ")).Should(Succeed())
		shouldExist(`SELECT * FROM tasks WHERE log = ?`, "line 1\n\nline ")

		Ω(db.UpdateTaskLog(task.UUID, "2\n")).Should(Succeed())
		shouldExist(`SELECT * FROM tasks WHERE log = ?`, "line 1\n\nline 2\n")
	})

	It("Can associate archives with the task", func() {
		task, err := db.CreateBackupTask("bob", SomeJob)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(task).ShouldNot(BeNil())

		Ω(db.StartTask(task.UUID, time.Now())).Should(Succeed())
		Ω(db.CompleteTask(task.UUID, time.Now())).Should(Succeed())
		archive_id, err := db.CreateTaskArchive(task.UUID, "SOME-KEY", time.Now())
		Expect(err).ShouldNot(HaveOccurred())
		Expect(archive_id).ShouldNot(BeNil())

		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE archive_uuid IS NOT NULL`)

		shouldExist(`SELECT * FROM archives`)
		shouldExist(`SELECT * FROM archives WHERE uuid = ?`, archive_id.String())
		shouldExist(`SELECT * FROM archives WHERE target_uuid = ?`, SomeTarget.UUID.String())
		shouldExist(`SELECT * FROM archives WHERE store_uuid = ?`, SomeStore.UUID.String())
		shouldExist(`SELECT * FROM archives WHERE store_key = ?`, "SOME-KEY")
		shouldExist(`SELECT * FROM archives WHERE taken_at IS NOT NULL`)
		shouldExist(`SELECT * FROM archives WHERE expires_at IS NOT NULL`)
	})
	It("Fails to associate archives with a task, when no restore key is present", func() {
		task, err := db.CreateBackupTask("bob", SomeJob)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(task).ShouldNot(BeNil())

		Expect(db.StartTask(task.UUID, time.Now())).Should(Succeed())
		Expect(db.CompleteTask(task.UUID, time.Now())).Should(Succeed())
		archive_id, err := db.CreateTaskArchive(task.UUID, "", time.Now())
		Expect(err).Should(HaveOccurred())
		Expect(archive_id).Should(BeNil())

		shouldNotExist(`SELECT * from archives where store_key = ''`)
	})
	It("Can limit the number of tasks returned", func() {
		task1, err1 := db.CreateBackupTask("first", SomeJob)
		task2, err2 := db.CreateBackupTask("second", SomeJob)
		task3, err3 := db.CreateBackupTask("third", SomeJob)
		task4, err4 := db.CreateBackupTask("fourth", SomeJob)
		Ω(err1).ShouldNot(HaveOccurred())
		Ω(task1).ShouldNot(BeNil())
		Ω(err2).ShouldNot(HaveOccurred())
		Ω(task2).ShouldNot(BeNil())
		Ω(err3).ShouldNot(HaveOccurred())
		Ω(task3).ShouldNot(BeNil())
		Ω(err4).ShouldNot(HaveOccurred())
		Ω(task4).ShouldNot(BeNil())

		Ω(db.StartTask(task1.UUID, at(0))).Should(Succeed())
		Ω(db.CompleteTask(task1.UUID, at(2))).Should(Succeed())
		Ω(db.StartTask(task2.UUID, at(4))).Should(Succeed())
		Ω(db.CompleteTask(task2.UUID, at(6))).Should(Succeed())
		Ω(db.StartTask(task3.UUID, at(8))).Should(Succeed())
		Ω(db.StartTask(task4.UUID, at(12))).Should(Succeed())
		Ω(db.CompleteTask(task4.UUID, at(14))).Should(Succeed())
		shouldExist(`SELECT * FROM tasks`)

		filter := TaskFilter{
			Limit: "2",
		}
		tasks, err := db.GetAllTasks(&filter)
		Ω(err).ShouldNot(HaveOccurred(), "does not error")
		Ω(len(tasks)).Should(Equal(2), "returns two tasks")
		Ω(tasks[0].Owner).Should(Equal("fourth"))
		Ω(tasks[1].Owner).Should(Equal("third"))

		filter = TaskFilter{
			ForStatus: DoneStatus,
			Limit:     "2",
		}
		tasks, err = db.GetAllTasks(&filter)
		Ω(err).ShouldNot(HaveOccurred(), "does not error")
		Ω(len(tasks)).Should(Equal(2), "returns two tasks")
		Ω(tasks[0].Owner).Should(Equal("fourth"))
		Ω(tasks[1].Owner).Should(Equal("second"))

		// Negative values return all tasks, these are prevented in the API
		filter = TaskFilter{
			Limit: "-1",
		}
		tasks, err = db.GetAllTasks(&filter)
		Ω(err).ShouldNot(HaveOccurred(), "does not error")
		Ω(len(tasks)).Should(Equal(4), "returns four tasks")
	})

	Describe("GetTask", func() {
		TASK1_UUID := uuid.NewRandom()
		TASK2_UUID := uuid.NewRandom()

		BeforeEach(func() {
			err := db.Exec(fmt.Sprintf(`INSERT INTO tasks (uuid, owner, op, status, requested_at)`+
				`VALUES('%s', '%s', '%s', '%s', %d)`,
				TASK1_UUID.String(), "system", BackupOperation, PendingStatus, 0))
			Expect(err).ShouldNot(HaveOccurred())

			err = db.Exec(
				fmt.Sprintf(`INSERT INTO tasks (uuid, owner, op, status, requested_at, archive_uuid, job_uuid)`+
					`VALUES('%s', '%s', '%s', '%s', %d, '%s', '%s')`,
					TASK2_UUID.String(), "system", RestoreOperation, PendingStatus, 2,
					SomeArchive.UUID.String(), SomeJob.UUID.String()))
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("Returns an individual task even when not associated with anything", func() {
			task, err := db.GetTask(TASK1_UUID)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(task).Should(BeEquivalentTo(&Task{
				UUID:        TASK1_UUID,
				Owner:       "system",
				Op:          BackupOperation,
				JobUUID:     nil,
				ArchiveUUID: nil,
				Status:      PendingStatus,
				StartedAt:   timestamp.Timestamp{},
				StoppedAt:   timestamp.Timestamp{},
				Log:         "",
			}))
		})
		It("Returns an individual task when associated with job/archive", func() {
			task, err := db.GetTask(TASK2_UUID)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(task).Should(BeEquivalentTo(&Task{
				UUID:        TASK2_UUID,
				Owner:       "system",
				Op:          RestoreOperation,
				JobUUID:     SomeJob.UUID,
				ArchiveUUID: SomeArchive.UUID,
				Status:      PendingStatus,
				StartedAt:   timestamp.Timestamp{},
				StoppedAt:   timestamp.Timestamp{},
				Log:         "",
			}))
		})
	})

	Describe("IsTaskRunnable", func() {
		notRunningTaskTargetUUID := uuid.NewRandom()
		runningTaskTargetUUID := uuid.NewRandom()

		BeforeEach(func() {
			err := db.Exec(fmt.Sprintf(`INSERT INTO tasks (uuid, op, status, requested_at, target_uuid)`+
				`VALUES('%s', '%s', '%s', %d, '%s')`,
				uuid.NewRandom().String(), BackupOperation, PendingStatus, 0, notRunningTaskTargetUUID.String()))
			Expect(err).ShouldNot(HaveOccurred())
			err = db.Exec(fmt.Sprintf(`INSERT INTO tasks (uuid, op, status, requested_at, target_uuid)`+
				`VALUES('%s', '%s', '%s', %d, '%s')`,
				uuid.NewRandom().String(), BackupOperation, RunningStatus, 0, runningTaskTargetUUID.String()))
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("Returns true if no other task with same target uuid is running", func() {
			runnable, err := db.IsTaskRunnable(&Task{TargetUUID: notRunningTaskTargetUUID})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(runnable).To(BeTrue())
		})
		It("Returns false if another task with same target uuid is running", func() {
			runnable, err := db.IsTaskRunnable(&Task{TargetUUID: runningTaskTargetUUID})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(runnable).To(BeFalse())
		})
	})
})
