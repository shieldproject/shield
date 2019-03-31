package db

import (
	"fmt"
	"time"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var T0 = time.Date(1997, 8, 29, 2, 14, 0, 0, time.UTC)

func at(seconds int) time.Time {
	return T0.Add(time.Duration(seconds) * time.Second)
}

var _ = Describe("Task Management", func() {
	var (
		db *DB

		SomeTenant  *Tenant
		SomeJob     *Job
		SomeTarget  *Target
		SomeStore   *Store
		SomeArchive *Archive
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
		SomeTenant = &Tenant{UUID: RandomID()}
		SomeJob = &Job{UUID: RandomID()}
		SomeTarget = &Target{UUID: RandomID()}
		SomeStore = &Store{UUID: RandomID()}
		SomeArchive = &Archive{UUID: RandomID()}

		db, err = Database(
			// need a tenant
			`INSERT INTO tenants (uuid, name)
			   VALUES ("`+SomeTenant.UUID+`", "Some Tenant")`,

			// need a target
			`INSERT INTO targets (uuid, tenant_uuid, name, summary, plugin, endpoint, agent)
			   VALUES ("`+SomeTarget.UUID+`", "`+SomeTenant.UUID+`", "Some Target", "", "plugin", '{"end":"point"}', "127.0.0.1:5444")`,

			// need a store
			`INSERT INTO stores (uuid, tenant_uuid, name, summary, plugin, endpoint, agent)
			   VALUES ("`+SomeStore.UUID+`", "`+SomeTenant.UUID+`", "Some Store", "", "plugin", '{"end":"point"}', "127.0.0.1:9938")`,

			// need a job
			`INSERT INTO jobs (uuid, tenant_uuid, name, summary, paused,
			                   target_uuid, store_uuid, schedule, keep_days)
			   VALUES ("`+SomeJob.UUID+`", "`+SomeTenant.UUID+`", "Some Job", "just a job...", 0,
			           "`+SomeTarget.UUID+`", "`+SomeStore.UUID+`", "daily 3am", 7)`,

			// need an archive
			`INSERT INTO archives (uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, notes, status, purge_reason)
			    VALUES("`+SomeArchive.UUID+`", "`+SomeTarget.UUID+`",
			           "`+SomeStore.UUID+`", "key", 0, 0, "(no notes)", "valid", "")`,
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

		SomeArchive, err = db.GetArchive(SomeArchive.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(SomeArchive).ShouldNot(BeNil())
	})

	It("Can create a new backup task", func() {
		task, err := db.CreateBackupTask("owner-name", SomeJob)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(task).ShouldNot(BeNil())

		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE uuid = ?`, task.UUID)
		shouldExist(`SELECT * FROM tasks WHERE owner = ?`, "owner-name")
		shouldExist(`SELECT * FROM tasks WHERE op = ?`, BackupOperation)
		shouldExist(`SELECT * FROM tasks WHERE job_uuid = ?`, SomeJob.UUID)
		shouldExist(`SELECT * from tasks WHERE store_uuid = ?`, SomeStore.UUID)
		shouldExist(`SELECT * FROM tasks WHERE target_uuid = ?`, SomeTarget.UUID)
		shouldExist(`SELECT * FROM tasks WHERE status = ?`, PendingStatus)
		shouldExist(`SELECT * FROM tasks WHERE requested_at IS NOT NULL`)
		shouldExist(`SELECT * FROM tasks WHERE started_at IS NULL`)
		shouldExist(`SELECT * FROM tasks WHERE stopped_at IS NULL`)
	})

	It("Can create a new purge task", func() {
		archive, err := db.GetArchive(SomeArchive.UUID)
		Expect(err).ShouldNot(HaveOccurred())

		task, err := db.CreatePurgeTask("owner-name", archive)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(task).ShouldNot(BeNil())

		shouldExist(`SELECT * from tasks`)
		shouldExist(`SELECT * FROM tasks WHERE uuid = ?`, task.UUID)
		shouldExist(`SELECT * FROM tasks WHERE owner = ?`, "owner-name")
		shouldExist(`SELECT * FROM tasks WHERE op = ?`, PurgeOperation)
		shouldExist(`SELECT * FROM tasks WHERE archive_uuid = ?`, SomeArchive.UUID)
		shouldExist(`SELECT * FROM tasks WHERE target_uuid IS NULL`)
		shouldExist(`SELECT * FROM tasks WHERE store_uuid = ?`, SomeStore.UUID)
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
		shouldExist(`SELECT * FROM tasks WHERE uuid = ?`, task.UUID)
		shouldExist(`SELECT * FROM tasks WHERE owner = ?`, "owner-name")
		shouldExist(`SELECT * FROM tasks WHERE op = ?`, RestoreOperation)
		shouldExist(`SELECT * FROM tasks WHERE archive_uuid = ?`, SomeArchive.UUID)
		shouldExist(`SELECT * FROM tasks WHERE target_uuid = ?`, SomeTarget.UUID)
		shouldExist(`SELECT * from tasks WHERE store_uuid = ?`, SomeStore.UUID)
		shouldExist(`SELECT * FROM tasks WHERE job_uuid IS NULL`)
		shouldExist(`SELECT * FROM tasks WHERE status = ?`, PendingStatus)
		shouldExist(`SELECT * FROM tasks WHERE requested_at IS NOT NULL`)
		shouldExist(`SELECT * FROM tasks WHERE started_at IS NULL`)
		shouldExist(`SELECT * FROM tasks WHERE stopped_at IS NULL`)
	})

	It("Can create a new TestStore task", func() {
		//`json:"config,omitempty"`
		SomeStore.Config = map[string]interface{}{"argkey": "fake"}
		task, err := db.CreateTestStoreTask("owner-name", SomeStore)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(task).ShouldNot(BeNil())
		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE uuid = ?`, task.UUID)
		shouldExist(`SELECT * FROM tasks WHERE owner = ?`, "owner-name")
		shouldExist(`SELECT * FROM tasks WHERE op = ?`, TestStoreOperation)
		shouldExist(`SELECT * FROM tasks WHERE store_plugin = ?`, SomeStore.Plugin)
		shouldExist(`SELECT * from tasks WHERE store_uuid = ?`, SomeStore.UUID)
		shouldExist(`SELECT * FROM tasks WHERE store_endpoint IS NOT NULL`)
		shouldExist(`SELECT * FROM tasks WHERE status = ?`, PendingStatus)
		shouldExist(`SELECT * FROM tasks WHERE requested_at IS NOT NULL`)
		shouldExist(`SELECT * FROM tasks WHERE agent = ?`, SomeStore.Agent)
		shouldExist(`SELECT * FROM tasks WHERE attempts >= 0`)
		shouldExist(`SELECT * FROM tasks WHERE tenant_uuid = ?`, SomeStore.TenantUUID)

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
		archive_id, err := db.CreateTaskArchive(task.UUID, RandomID(), "SOME-KEY", time.Now(), "aes-256-ctr", "gz", 0, task.TenantUUID)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(archive_id).ShouldNot(BeNil())

		shouldExist(`SELECT * FROM tasks`)
		shouldExist(`SELECT * FROM tasks WHERE archive_uuid IS NOT NULL`)

		shouldExist(`SELECT * FROM archives`)
		shouldExist(`SELECT * FROM archives WHERE uuid = ?`, archive_id)
		shouldExist(`SELECT * FROM archives WHERE target_uuid = ?`, SomeTarget.UUID)
		shouldExist(`SELECT * FROM archives WHERE store_uuid = ?`, SomeStore.UUID)
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
		archive_id, err := db.CreateTaskArchive(task.UUID, RandomID(), "", time.Now(), "aes-256-ctr", "gz", 0, task.TenantUUID)
		Expect(err).Should(HaveOccurred())
		Expect(archive_id).Should(BeEmpty())

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
			Limit: 2,
		}
		tasks, err := db.GetAllTasks(&filter)
		Ω(err).ShouldNot(HaveOccurred(), "does not error")
		Ω(len(tasks)).Should(Equal(2), "returns two tasks")
		Ω(tasks[0].Owner).Should(Equal("fourth"))
		Ω(tasks[1].Owner).Should(Equal("third"))

		filter = TaskFilter{
			ForStatus: DoneStatus,
			Limit:     2,
		}
		tasks, err = db.GetAllTasks(&filter)
		Ω(err).ShouldNot(HaveOccurred(), "does not error")
		Ω(len(tasks)).Should(Equal(2), "returns two tasks")
		Ω(tasks[0].Owner).Should(Equal("fourth"))
		Ω(tasks[1].Owner).Should(Equal("second"))

		// Negative values return all tasks, these are prevented in the API
		filter = TaskFilter{
			Limit: -1,
		}
		tasks, err = db.GetAllTasks(&filter)
		Ω(err).ShouldNot(HaveOccurred(), "does not error")
		Ω(len(tasks)).Should(Equal(4), "returns four tasks")
	})

	Describe("GetTask", func() {
		TASK1_UUID := RandomID()
		TASK2_UUID := RandomID()
		TENANT1_UUID := RandomID()
		TENANT2_UUID := RandomID()

		BeforeEach(func() {
			err := db.Exec(fmt.Sprintf(`INSERT INTO tasks (uuid, owner, op, status, requested_at, tenant_uuid)`+
				`VALUES('%s', '%s', '%s', '%s', %d, '%s')`,
				TASK1_UUID, "system", BackupOperation, PendingStatus, 0, TENANT1_UUID))
			Expect(err).ShouldNot(HaveOccurred())

			err = db.Exec(
				fmt.Sprintf(`INSERT INTO tasks (uuid, owner, op, status, requested_at, archive_uuid, job_uuid, tenant_uuid)`+
					`VALUES('%s', '%s', '%s', '%s', %d, '%s', '%s', '%s')`,
					TASK2_UUID, "system", RestoreOperation, PendingStatus, 2,
					SomeArchive.UUID, SomeJob.UUID, TENANT2_UUID))
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("Returns an individual task even when not associated with anything", func() {
			task, err := db.GetTask(TASK1_UUID)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(task).Should(BeEquivalentTo(&Task{
				UUID:        TASK1_UUID,
				TenantUUID:  TENANT1_UUID,
				Owner:       "system",
				Op:          BackupOperation,
				JobUUID:     "",
				ArchiveUUID: "",
				Status:      PendingStatus,
				RequestedAt: task.RequestedAt,
				Log:         "",
				OK:          true,
				Notes:       "",
				Clear:       "normal",
			}))
		})
		It("Returns an individual task when associated with job/archive", func() {
			task, err := db.GetTask(TASK2_UUID)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(task).Should(BeEquivalentTo(&Task{
				UUID:        TASK2_UUID,
				TenantUUID:  TENANT2_UUID,
				Owner:       "system",
				Op:          RestoreOperation,
				JobUUID:     SomeJob.UUID,
				ArchiveUUID: SomeArchive.UUID,
				RequestedAt: task.RequestedAt,
				Status:      PendingStatus,
				Log:         "",
				OK:          true,
				Notes:       "",
				Clear:       "normal",
			}))
		})
	})

	Describe("IsTaskRunnable", func() {
		notRunningTaskTargetUUID := RandomID()
		runningTaskTargetUUID := RandomID()

		BeforeEach(func() {
			err := db.Exec(fmt.Sprintf(`INSERT INTO tasks (uuid, op, status, requested_at, target_uuid)`+
				`VALUES('%s', '%s', '%s', %d, '%s')`,
				RandomID(), BackupOperation, PendingStatus, 0, notRunningTaskTargetUUID))
			Expect(err).ShouldNot(HaveOccurred())
			err = db.Exec(fmt.Sprintf(`INSERT INTO tasks (uuid, op, status, requested_at, target_uuid)`+
				`VALUES('%s', '%s', '%s', %d, '%s')`,
				RandomID(), BackupOperation, RunningStatus, 0, runningTaskTargetUUID))
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
