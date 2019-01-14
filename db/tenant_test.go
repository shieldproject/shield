package db_test

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	. "github.com/starkandwayne/shield/db"
)

var T00 = time.Date(1997, 8, 29, 2, 14, 0, 0, time.UTC)

func att(seconds int) time.Time {
	return T0.Add(time.Duration(seconds) * time.Second)
}

var _ = Describe("tenant Management", func() {
	var (
		db *DB

		//SomeJob       *Job
		SomeTarget *Target
		SomeStore  *Store
		//SomeRetention *RetentionPolicy
		SomeArchive *Archive
		SomeTask    *Task
		Tenant2     *Tenant
	)

	//shouldExist := func(q string, params ...interface{}) {
	//	n, err := db.Count(q, params...)
	//	Ω(err).ShouldNot(HaveOccurred())
	//	Ω(n).Should(BeNumerically(">", 0))
	//}

	//shouldNotExist := func(q string, params ...interface{}) {
	//	n, err := db.Count(q, params...)
	//	Expect(err).ShouldNot(HaveOccurred())
	//	Expect(n).Should(BeNumerically("==", 0))
	//}

	BeforeEach(func() {
		var err error
		//SomeJob = &Job{UUID: uuid.NewRandom()}
		SomeTarget = &Target{UUID: uuid.NewRandom()}
		SomeStore = &Store{UUID: uuid.NewRandom()}
		//SomeRetention = &RetentionPolicy{UUID: uuid.NewRandom()}
		SomeArchive = &Archive{UUID: uuid.NewRandom()}
		SomeTask = &Task{UUID: uuid.NewRandom()}
		Tenant2 = &Tenant{UUID: uuid.Parse("3f950780-120f-4e00-b46b-28f35e5882df")}

		db, err = Database(
			// need a target1
			`INSERT INTO targets (tenant_uuid, uuid, name, summary, plugin, endpoint, agent)
			   VALUES ("tenant1", "`+SomeTarget.UUID.String()+`", "name", "a summary", "plugin", "endpoint", "127.0.0.1:5444")`,
			// need a target2
			`INSERT INTO targets (tenant_uuid, uuid, name, summary, plugin, endpoint, agent)
			VALUES ("tenant2", "`+SomeTarget.UUID.String()+`2", "name", "a summary", "plugin", "endpoint", "127.0.0.1:5444")`,
			// need a store1
			`INSERT INTO stores (tenant_uuid, uuid, name, summary, plugin, endpoint)
			   VALUES ("tenant1", "`+SomeStore.UUID.String()+`", "name", "", "plugin", "endpoint")`,
			//need a store2
			`INSERT INTO stores (tenant_uuid, uuid, name, summary, plugin, endpoint)
			VALUES ("tenant2", "`+SomeStore.UUID.String()+`2", "name", "", "plugin", "endpoint")`,
			//// need a job1
			//`INSERT INTO jobs (tenant_uuid, uuid, name, summary, paused, target_uuid, store_uuid, retention_uuid, schedule)
			//  VALUES ("tenant1", "`+SomeJob.UUID.String()+`", "Some Job", "just a job...", 0, "`+SomeTarget.UUID.String()+`", "`+SomeStore.UUID.String()+`", "`+SomeRetention.UUID.String()+`", "daily 3am")`,
			//// need a job2
			//`INSERT INTO jobs (tenant_uuid, uuid, name, summary, target_uuid, store_uuid, retention_uuid, schedule)
			//	VALUES ("tenant2", "`+SomeJob.UUID.String()+`2", "Some Job", "just a job...", "`+SomeTarget.UUID.String()+`2", "`+SomeStore.UUID.String()+`2", "`+SomeRetention.UUID.String()+`2", "daily 3am")`,
			// need an archive1
			`INSERT INTO archives (tenant_uuid, uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, notes, status, purge_reason)
			    VALUES("`+Tenant2.UUID.String()+`", "`+SomeArchive.UUID.String()+`", "`+SomeTarget.UUID.String()+`", "`+SomeStore.UUID.String()+`", "key", 0, 0, "(no notes)", "valid", "")`,
			////need an archive2
			`INSERT INTO archives (tenant_uuid, uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, notes, status, purge_reason)
			    VALUES("`+Tenant2.UUID.String()+`", "`+SomeArchive.UUID.String()+`2", "`+SomeTarget.UUID.String()+`2", "`+SomeStore.UUID.String()+`2", "key", 0, 0, "(no notes)", "valid", "")`,
			//// need a purge task for tenant1
			//`INSERT INTO tasks (tenant_uuid, uuid, op, requested_at, status)
			//VALUES ("tenant1", "purgetask1", "purge", 0, "done")`,
			//need a archive tasks for tenant1
			`INSERT INTO tasks (uuid, owner, op, requested_at, status, tenant_uuid)
			VALUES ("`+SomeTask.UUID.String()+`", "some owner", "backup", 0, "done", "`+Tenant2.UUID.String()+`")`,
			//need archive tasks for tenant2
			//`INSERT INTO tasks (tenant_uuid, uuid, op, requested_at, status)
			//VALUES ("tenant2", "backuptask2", "backup", 0, "done")`,
			//need a tenant
			`INSERT INTO tenants (uuid, name)
			VALUES ("`+Tenant2.UUID.String()+`", "tenant2")`,
		)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(db).ShouldNot(BeNil())

		//SomeJob, err = db.GetJob(SomeJob.UUID)
		//Ω(err).ShouldNot(HaveOccurred())
		//Ω(SomeJob).ShouldNot(BeNil())

		SomeTarget, err = db.GetTarget(SomeTarget.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(SomeTarget).ShouldNot(BeNil())

		SomeStore, err = db.GetStore(SomeStore.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(SomeStore).ShouldNot(BeNil())

		SomeArchive, err = db.GetArchive(SomeArchive.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(SomeArchive).ShouldNot(BeNil())

		SomeTask, err = db.GetTask(SomeTask.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(SomeTask).ShouldNot(BeNil())

		tasks, err := db.GetAllTasks(&TaskFilter{ForTenant: Tenant2.UUID.String()})
		Ω(err).ShouldNot(HaveOccurred())
		Ω(len(tasks)).Should(Equal(1))

		Tenant2, err = db.GetTenantByName("tenant2")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(Tenant2).ShouldNot(BeNil())
		Ω(Tenant2.UUID.String()).Should(Equal("3f950780-120f-4e00-b46b-28f35e5882df"))
	})

	It("Will fail non recursive with stores", func() {
		err := db.DeleteTenant(Tenant2, false)
		Expect(err).Should(HaveOccurred())
	})

	It("Will fail non recursive with targets", func() {
		_, err := db.DeleteStore(SomeStore.UUID)
		Expect(err).ShouldNot(HaveOccurred())
		err = db.DeleteTenant(Tenant2, false)
		Expect(err).Should(HaveOccurred())
	})

	It("Will fail non recursive with archives", func() {
		_, err := db.DeleteStore(SomeStore.UUID)
		Expect(err).ShouldNot(HaveOccurred())
		_, err = db.DeleteTarget(SomeTarget.UUID)
		Expect(err).ShouldNot(HaveOccurred())
		err = db.DeleteTenant(Tenant2, false)
		Expect(err).Should(HaveOccurred())
	})

	It("Will fail non recursive with tasks", func() {
		fmt.Fprintf(os.Stdout, "\ntenant uuid is %s \n", Tenant2.UUID.String())
		fmt.Fprintf(os.Stdout, "\ntenant uuid from task %s\n", SomeTask.TenantUUID.String())
		err := db.DeleteTenant(Tenant2, false)
		Expect(err).Should(HaveOccurred())
	})

	//It("Fails to associate archives with a task, when no restore key is present", func() {
	//	task, err := db.CreateBackupTask("bob", SomeJob)
	//	Expect(err).ShouldNot(HaveOccurred())
	//	Expect(task).ShouldNot(BeNil())
	//
	//	Expect(db.StartTask(task.UUID, time.Now())).Should(Succeed())
	//	Expect(db.CompleteTask(task.UUID, time.Now())).Should(Succeed())
	//	archive_id, err := db.CreateTaskArchive(task.UUID, uuid.NewRandom(), "", time.Now(), "aes-256-ctr", "gz", 0, task.TenantUUID)
	//	Expect(err).Should(HaveOccurred())
	//	Expect(archive_id).Should(BeNil())
	//
	//	shouldNotExist(`SELECT * from archives where store_key = ''`)
	//	shouldExist("0")
	//})

})
