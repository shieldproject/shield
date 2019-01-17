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

		SomeJob       *Job
		SomeJob2      *Job
		SomeTarget    *Target
		SomeTarget2   *Target
		SomeStore     *Store
		SomeStore2    *Store
		SomeRetention *RetentionPolicy
		SomeArchive   *Archive
		SomeArchive2  *Archive
		SomeTask      *Task
		PurgeTask     *Task
		Ten3Task      *Task
		Tenant2       *Tenant
		Tenant3       *Tenant
		AdminUser     *User
		OtherUser     *User
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
		SomeJob = &Job{UUID: uuid.Parse("8dcb45ef-581a-415e-abc0-0234bc70c7a9")}
		SomeJob2 = &Job{UUID: uuid.Parse("018fc927-569e-4d64-968a-9add3d363ed5")}
		SomeTarget = &Target{UUID: uuid.Parse("e36850ff-2392-4018-a130-84db28862650")}
		SomeTarget2 = &Target{UUID: uuid.Parse("3da946be-f17d-4a2a-a9d2-1d6e5bf7e37b")}
		SomeStore = &Store{UUID: uuid.NewRandom()}
		SomeStore2 = &Store{UUID: uuid.NewRandom()}
		SomeRetention = &RetentionPolicy{UUID: uuid.Parse("565a53c2-18a0-45ad-b45b-41e92b2e152c")}
		SomeArchive = &Archive{UUID: uuid.NewRandom()}
		SomeArchive2 = &Archive{UUID: uuid.Parse("863d1c48-e7f4-45fc-9047-4c28be4caaad")}
		SomeTask = &Task{UUID: uuid.NewRandom()}
		PurgeTask = &Task{UUID: uuid.Parse("bbae5dc2-7839-4dc1-9d9d-69bb057254e1")}
		Ten3Task = &Task{UUID: uuid.Parse("3874908e-e4c6-48b9-8700-ed814688abaa")}
		Tenant2 = &Tenant{UUID: uuid.Parse("3f950780-120f-4e00-b46b-28f35e5882df")}
		Tenant3 = &Tenant{UUID: uuid.Parse("b1d6eeeb-1235-4c93-8800-f5c44ee50f1b")}
		AdminUser = &User{UUID: uuid.Parse("4cedd497-9af4-484d-a0b2-b79bdb46223f")}
		OtherUser = &User{UUID: uuid.Parse("e0122b8b-0ca7-480e-a5d8-40aab7e5e8cb")}
		AdminUser = AdminUser
		OtherUser = OtherUser

		db, err = Database(
			// need a target1
			`INSERT INTO targets (tenant_uuid, uuid, name, summary, plugin, endpoint, agent)
			   VALUES ("`+Tenant2.UUID.String()+`", "`+SomeTarget.UUID.String()+`", "name", "a summary", "plugin", "endpoint", "127.0.0.1:5444")`,
			// need a target2
			`INSERT INTO targets (tenant_uuid, uuid, name, summary, plugin, endpoint, agent)
			VALUES ("`+Tenant3.UUID.String()+`", "`+SomeTarget2.UUID.String()+`", "name2", "a summary2", "plugin2", "endpoint2", "127.0.0.2:5444")`,
			// need a store1
			`INSERT INTO stores (tenant_uuid, uuid, name, summary, plugin, endpoint)
			   VALUES ("`+Tenant2.UUID.String()+`", "`+SomeStore.UUID.String()+`", "name", "", "plugin", "endpoint")`,
			//need a store2
			`INSERT INTO stores (tenant_uuid, uuid, name, summary, plugin, endpoint)
			VALUES ("`+Tenant3.UUID.String()+`", "`+SomeStore2.UUID.String()+`", "name", "", "plugin", "endpoint")`,
			//need a retention policy
			`INSERT INTO retention (uuid, name, summary, expiry)
			VALUES ("`+SomeRetention.UUID.String()+`", "Some Retention", "", 3600)`,
			// need a job1
			`INSERT INTO jobs (uuid, name, summary, paused, target_uuid, store_uuid, retention_uuid, schedule, tenant_uuid)
			  VALUES ("`+SomeJob.UUID.String()+`", "Some Job", "A summary", 0, "`+SomeTarget.UUID.String()+`", "`+SomeStore.UUID.String()+`", "`+SomeRetention.UUID.String()+`", "daily 3am", "`+Tenant2.UUID.String()+`")`,
			// need a job2
			`INSERT INTO jobs (uuid, name, summary, paused, target_uuid, store_uuid, retention_uuid, schedule, tenant_uuid)
			  VALUES ("`+SomeJob2.UUID.String()+`", "Some Job2", "A summary2", 0, "`+SomeTarget.UUID.String()+`", "`+SomeStore.UUID.String()+`", "`+SomeRetention.UUID.String()+`", "daily 3am", "`+Tenant3.UUID.String()+`")`,
			// need an archive1
			`INSERT INTO archives (tenant_uuid, uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, notes, status, purge_reason)
			    VALUES ("`+Tenant2.UUID.String()+`", "`+SomeArchive.UUID.String()+`", "`+SomeTarget.UUID.String()+`", "`+SomeStore.UUID.String()+`", "key", 0, 0, "(no notes)", "valid", "")`,
			////need an archive2
			`INSERT INTO archives (tenant_uuid, uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, notes, status, purge_reason)
			    VALUES ("`+Tenant3.UUID.String()+`", "`+SomeArchive2.UUID.String()+`", "`+SomeTarget2.UUID.String()+`", "`+SomeStore2.UUID.String()+`", "key", 0, 0, "(no notes)", "valid", "")`,
			// need a purge task for tenant1
			`INSERT INTO tasks (tenant_uuid, uuid, owner, op, requested_at, status, stopped_at)
			    VALUES ("`+Tenant2.UUID.String()+`", "`+PurgeTask.UUID.String()+`", "some owner", "purge", 0, "done", 0)`,
			//need a archive tasks for tenant1
			`INSERT INTO tasks (uuid, owner, op, requested_at, status, tenant_uuid)
			VALUES ("`+SomeTask.UUID.String()+`", "some owner", "backup", 0, "done", "`+Tenant2.UUID.String()+`")`,
			//need archive tasks for tenant2
			`INSERT INTO tasks (tenant_uuid, uuid, op, owner, requested_at, status)
			VALUES ("`+Tenant3.UUID.String()+`", "`+Ten3Task.UUID.String()+`", "backup", "different owner", 0, "done")`,
			//need a tenant
			`INSERT INTO tenants (uuid, name)
			VALUES ("`+Tenant2.UUID.String()+`", "tenant2")`,
			//need a second tenant
			`INSERT INTO tenants (uuid, name)
			VALUES ("`+Tenant3.UUID.String()+`", "tenant3")`,
			//admin user to tenant 1
			`INSERT INTO memberships (user_uuid, tenant_uuid, role)
			VALUES ("`+AdminUser.UUID.String()+`","`+Tenant2.UUID.String()+`",5)`,
			//other user to tenant 2
			`INSERT INTO memberships (user_uuid, tenant_uuid, role)
			VALUES ("`+AdminUser.UUID.String()+`","`+Tenant3.UUID.String()+`",6)`,
			//other user to tenant 1
			`INSERT INTO memberships (user_uuid, tenant_uuid, role)
			VALUES ("`+OtherUser.UUID.String()+`","`+Tenant2.UUID.String()+`",6)`,
		)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(db).ShouldNot(BeNil())

		SomeJob, err = db.GetJob(SomeJob.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(SomeJob).ShouldNot(BeNil())

		SomeJob2, err = db.GetJob(SomeJob2.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(SomeJob2).ShouldNot(BeNil())

		SomeTarget, err = db.GetTarget(SomeTarget.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(SomeTarget).ShouldNot(BeNil())

		SomeTarget2, err = db.GetTarget(SomeTarget2.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(SomeTarget2).ShouldNot(BeNil())

		SomeStore, err = db.GetStore(SomeStore.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(SomeStore).ShouldNot(BeNil())

		SomeArchive, err = db.GetArchive(SomeArchive.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(SomeArchive).ShouldNot(BeNil())

		SomeArchive2, err = db.GetArchive(SomeArchive2.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(SomeArchive2).ShouldNot(BeNil())

		SomeTask, err = db.GetTask(SomeTask.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(SomeTask).ShouldNot(BeNil())

		PurgeTask, err = db.GetTask(PurgeTask.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(PurgeTask).ShouldNot(BeNil())

		tasks, err := db.GetAllTasks(&TaskFilter{ForTenant: Tenant2.UUID.String()})
		Ω(err).ShouldNot(HaveOccurred())
		Ω(len(tasks)).Should(Equal(2))

		Ten3Task, err = db.GetTask(Ten3Task.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(Ten3Task).ShouldNot(BeNil())

		Tenant2, err = db.GetTenantByName("tenant2")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(Tenant2).ShouldNot(BeNil())
		Ω(Tenant2.UUID.String()).Should(Equal("3f950780-120f-4e00-b46b-28f35e5882df"))

		Tenant3, err = db.GetTenantByName("tenant3")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(Tenant3).ShouldNot(BeNil())
		Ω(Tenant3.UUID.String()).Should(Equal("b1d6eeeb-1235-4c93-8800-f5c44ee50f1b"))

		Members, err := db.GetMembershipsForUser(AdminUser.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(len(Members)).Should(Equal(2))

		Members, err = db.GetMembershipsForUser(OtherUser.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(len(Members)).Should(Equal(1))

	})

	It("Will fail non recursive with jobs", func() {
		err := db.DeleteTenant(Tenant2, false)
		Expect(err).Should(HaveOccurred())
	})

	It("Will fail non recursive with stores", func() {
		_, err := db.DeleteJob(SomeJob.UUID)
		Expect(err).ShouldNot(HaveOccurred())
		err = db.DeleteTenant(Tenant2, false)
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

	It("Will pass a recursive delete", func() {
		err := db.DeleteTenant(Tenant2, true)
		Expect(err).ShouldNot(HaveOccurred())

		//only the purge task remains afters
		tasks, err := db.GetAllTasks(&TaskFilter{ForTenant: Tenant2.UUID.String()})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(len(tasks)).Should(Equal(1))

		//only the purge task and the other tennant tasks remains
		tasks, err = db.GetAllTasks(nil)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(len(tasks)).Should(Equal(2))

		//no stores remain from the deleted tenant
		stores, err := db.GetAllStores(&StoreFilter{ForTenant: Tenant2.UUID.String()})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(len(stores)).Should(Equal(0))

		//no jobs remain from the deleted tenant
		jobs, err := db.GetAllJobs(&JobFilter{ForTenant: Tenant2.UUID.String()})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(len(jobs)).Should(Equal(0))

		//no targets remain from the deleted tenant
		targets, err := db.GetAllTargets(&TargetFilter{ForTenant: Tenant2.UUID.String()})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(len(targets)).Should(Equal(0))

		//no jobs remain from the deleted tenant
		archives, err := db.GetAllArchives(&ArchiveFilter{ForTenant: Tenant2.UUID.String()})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(len(archives)).Should(Equal(0))

		//only one job was marked as expired
		archives, err = db.GetAllArchives(&ArchiveFilter{WithStatus: []string{"expired"}})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(len(archives)).Should(Equal(1))

		//only delete the membership for the deleted tenant for
		Members, err := db.GetMembershipsForUser(AdminUser.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(len(Members)).Should(Equal(1))

		//only delete the membership for the deleted tenant for
		Members, err = db.GetMembershipsForUser(OtherUser.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(len(Members)).Should(Equal(0))
	})

})
