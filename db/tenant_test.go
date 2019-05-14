package db_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

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

		SomeJob      *Job
		SomeJob2     *Job
		SomeTarget   *Target
		SomeTarget2  *Target
		SomeStore    *Store
		SomeStore2   *Store
		SomeArchive  *Archive
		SomeArchive2 *Archive
		SomeTask     *Task
		PurgeTask    *Task
		Ten3Task     *Task
		Tenant2      *Tenant
		Tenant3      *Tenant
		AdminUser    *User
		OtherUser    *User
	)

	BeforeEach(func() {
		var err error
		SomeJob = &Job{UUID: "8dcb45ef-581a-415e-abc0-0234bc70c7a9"}
		SomeJob2 = &Job{UUID: "018fc927-569e-4d64-968a-9add3d363ed5"}
		SomeTarget = &Target{UUID: "e36850ff-2392-4018-a130-84db28862650"}
		SomeTarget2 = &Target{UUID: "3da946be-f17d-4a2a-a9d2-1d6e5bf7e37b"}
		SomeStore = &Store{UUID: "fe948c51-cc84-4deb-bc1d-9f0afa1cbb63"}
		SomeStore2 = &Store{UUID: "f5fe2174-0310-4ba1-9067-1a7d828386d5"}
		SomeArchive = &Archive{UUID: "ecc69879-ad06-4396-a523-da1086716b68"}
		SomeArchive2 = &Archive{UUID: "863d1c48-e7f4-45fc-9047-4c28be4caaad"}
		SomeTask = &Task{UUID: "9fde7a6d-8fa7-47f5-b858-b40987496372"}
		PurgeTask = &Task{UUID: "bbae5dc2-7839-4dc1-9d9d-69bb057254e1"}
		Ten3Task = &Task{UUID: "3874908e-e4c6-48b9-8700-ed814688abaa"}
		Tenant2 = &Tenant{UUID: "3f950780-120f-4e00-b46b-28f35e5882df"}
		Tenant3 = &Tenant{UUID: "b1d6eeeb-1235-4c93-8800-f5c44ee50f1b"}
		AdminUser = &User{UUID: "4cedd497-9af4-484d-a0b2-b79bdb46223f"}
		OtherUser = &User{UUID: "e0122b8b-0ca7-480e-a5d8-40aab7e5e8cb"}
		AdminUser = AdminUser
		OtherUser = OtherUser

		db, err = Database(
			// need a target1
			`INSERT INTO targets (uuid, name, summary, plugin, endpoint, agent, tenant_uuid)
			   VALUES ("`+SomeTarget.UUID+`", "name", "a summary", "plugin", "endpoint", "127.0.0.1:5444", "`+Tenant2.UUID+`")`,
			// need a target2
			`INSERT INTO targets (tenant_uuid, uuid, name, summary, plugin, endpoint, agent)
			VALUES ("`+Tenant3.UUID+`", "`+SomeTarget2.UUID+`", "name2", "a summary2", "plugin2", "endpoint2", "127.0.0.2:5444")`,
			// need a store1
			`INSERT INTO stores (tenant_uuid, uuid, name, summary, plugin, endpoint)
			   VALUES ("`+Tenant2.UUID+`", "`+SomeStore.UUID+`", "name", "", "plugin", "endpoint")`,
			//need a store2
			`INSERT INTO stores (tenant_uuid, uuid, name, summary, plugin, endpoint)
			VALUES ("`+Tenant3.UUID+`", "`+SomeStore2.UUID+`", "name", "", "plugin", "endpoint")`,
			// need a job1
			`INSERT INTO jobs (uuid, name, summary, paused, target_uuid, store_uuid, keep_n, keep_days, schedule, tenant_uuid)
			  VALUES ("`+SomeJob.UUID+`", "Some Job", "A summary", 0, "`+SomeTarget.UUID+`", "`+SomeStore.UUID+`", 4, 4, "daily 3am", "`+Tenant2.UUID+`")`,
			// need a job2
			`INSERT INTO jobs (uuid, name, summary, paused, target_uuid, store_uuid, keep_n, keep_days, schedule, tenant_uuid)
			  VALUES ("`+SomeJob2.UUID+`", "Some Job2", "A summary2", 0, "`+SomeTarget.UUID+`", "`+SomeStore.UUID+`", 4, 4, "daily 3am", "`+Tenant3.UUID+`")`,
			// need an archive1
			`INSERT INTO archives (tenant_uuid, uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, notes, status, purge_reason)
			    VALUES ("`+Tenant2.UUID+`", "`+SomeArchive.UUID+`", "`+SomeTarget.UUID+`", "`+SomeStore.UUID+`", "key", 0, 0, "(no notes)", "valid", "")`,
			////need an archive2
			`INSERT INTO archives (tenant_uuid, uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, notes, status, purge_reason)
			    VALUES ("`+Tenant3.UUID+`", "`+SomeArchive2.UUID+`", "`+SomeTarget2.UUID+`", "`+SomeStore2.UUID+`", "key", 0, 0, "(no notes)", "valid", "")`,
			// need a purge task for tenant1
			`INSERT INTO tasks (tenant_uuid, uuid, owner, op, requested_at, status, stopped_at)
			    VALUES ("`+Tenant2.UUID+`", "`+PurgeTask.UUID+`", "some owner", "valid", 0, "done", 0)`,
			//need a archive tasks for tenant1
			`INSERT INTO tasks (uuid, owner, op, requested_at, status, tenant_uuid)
			VALUES ("`+SomeTask.UUID+`", "some owner", "backup", 0, "done", "`+Tenant2.UUID+`")`,
			//need archive tasks for tenant2
			`INSERT INTO tasks (tenant_uuid, uuid, op, owner, requested_at, status)
			VALUES ("`+Tenant3.UUID+`", "`+Ten3Task.UUID+`", "backup", "different owner", 0, "done")`,
			//need a tenant
			`INSERT INTO tenants (uuid, name)
			VALUES ("`+Tenant2.UUID+`", "tenant2")`,
			//need a second tenant
			`INSERT INTO tenants (uuid, name)
			VALUES ("`+Tenant3.UUID+`", "tenant3")`,
			//admin user to tenant 1
			`INSERT INTO memberships (user_uuid, tenant_uuid, role)
			VALUES ("`+AdminUser.UUID+`","`+Tenant2.UUID+`",5)`,
			//other user to tenant 2
			`INSERT INTO memberships (user_uuid, tenant_uuid, role)
			VALUES ("`+AdminUser.UUID+`","`+Tenant3.UUID+`",6)`,
			//other user to tenant 1
			`INSERT INTO memberships (user_uuid, tenant_uuid, role)
			VALUES ("`+OtherUser.UUID+`","`+Tenant2.UUID+`",6)`,
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

		tasks, err := db.GetAllTasks(&TaskFilter{ForTenant: Tenant2.UUID})
		Ω(err).ShouldNot(HaveOccurred())
		Ω(len(tasks)).Should(Equal(2))

		Ten3Task, err = db.GetTask(Ten3Task.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(Ten3Task).ShouldNot(BeNil())

		Members, err := db.GetMembershipsForUser(AdminUser.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(len(Members)).Should(Equal(2))

		Members, err = db.GetMembershipsForUser(OtherUser.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(len(Members)).Should(Equal(1))

		num, err := db.CountTargets(&TargetFilter{ForTenant: Tenant2.UUID})
		Ω(err).ShouldNot(HaveOccurred())
		Ω(num).Should(Equal(1))
	})

	It("Will fail non recursive with jobs", func() {
		err := db.DeleteTenant(Tenant2, false)
		Expect(err.Error()).Should(Equal("unable to delete tenant: tenant has outstanding jobs"))
	})

	It("Will fail non recursive with stores", func() {
		_, err := db.DeleteJob(SomeJob.UUID)
		err = db.DeleteTenant(Tenant2, false)
		Expect(err.Error()).Should(Equal("unable to delete tenant: tenant has outstanding stores"))
	})

	It("Will fail non recursive with targets", func() {
		_, err := db.DeleteJob(SomeJob.UUID)
		Expect(err).ShouldNot(HaveOccurred())
		err = db.Exec(`DELETE from stores where tenant_uuid=?`, Tenant2.UUID)
		Expect(err).ShouldNot(HaveOccurred())
		err = db.DeleteTenant(Tenant2, false)
		Expect(err.Error()).Should(Equal("unable to delete tenant: tenant has outstanding targets"))
	})

	It("Will fail non recursive with archives", func() {
		_, err := db.DeleteJob(SomeJob.UUID)
		Expect(err).ShouldNot(HaveOccurred())
		err = db.Exec(`DELETE from stores where tenant_uuid=?`, Tenant2.UUID)
		Expect(err).ShouldNot(HaveOccurred())
		err = db.Exec(`DELETE from targets where uuid=?`, SomeTarget.UUID)
		Expect(err).ShouldNot(HaveOccurred())
		err = db.DeleteTenant(Tenant2, false)
		Expect(err.Error()).Should(Equal("unable to delete tenant: tenant has outstanding archives"))
	})

	It("Will fail non recursive with tasks", func() {
		_, err := db.DeleteJob(SomeJob.UUID)
		Expect(err).ShouldNot(HaveOccurred())
		err = db.Exec(`DELETE from stores where tenant_uuid=?`, Tenant2.UUID)
		Expect(err).ShouldNot(HaveOccurred())
		err = db.Exec(`DELETE from targets where uuid=?`, SomeTarget.UUID)
		Expect(err).ShouldNot(HaveOccurred())
		err = db.Exec(`UPDATE archives SET status= "purged" where uuid=?`, SomeArchive.UUID)
		Expect(err).ShouldNot(HaveOccurred())
		err = db.DeleteTenant(Tenant2, false)
		Expect(err.Error()).Should(Equal("unable to delete tenant: tenant has outstanding tasks"))
	})

	It("Will pass a recursive delete", func() {
		err := db.DeleteTenant(Tenant2, true)
		Expect(err).ShouldNot(HaveOccurred())

		//only the purge task remains afters
		tasks, err := db.GetAllTasks(&TaskFilter{ForTenant: Tenant2.UUID})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(len(tasks)).Should(Equal(1))

		//only the purge task and the other tennant tasks remains
		tasks, err = db.GetAllTasks(nil)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(len(tasks)).Should(Equal(2))

		//no stores remain from the deleted tenant
		stores, err := db.GetAllStores(&StoreFilter{ForTenant: Tenant2.UUID})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(len(stores)).Should(Equal(0))

		//no jobs remain from the deleted tenant
		jobs, err := db.GetAllJobs(&JobFilter{ForTenant: Tenant2.UUID})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(len(jobs)).Should(Equal(0))

		//no targets remain from the deleted tenant
		targets, err := db.GetAllTargets(&TargetFilter{ForTenant: Tenant2.UUID})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(len(targets)).Should(Equal(0))

		//no jobs remain from the deleted tenant
		archives, err := db.GetAllArchives(&ArchiveFilter{ForTenant: Tenant2.UUID})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(len(archives)).Should(Equal(0))

		//only one job was marked as 'tenant deleted'
		archives, err = db.GetAllArchives(&ArchiveFilter{WithStatus: []string{"tenant deleted"}})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(len(archives)).Should(Equal(1))

		//only delete the membership for the deleted tenant for admin user
		Members, err := db.GetMembershipsForUser(AdminUser.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(len(Members)).Should(Equal(1))

		//deleted the membership for the deleted tenant for other user
		Members, err = db.GetMembershipsForUser(OtherUser.UUID)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(len(Members)).Should(Equal(0))
	})

	It("Targets are cleaned properly after tenant delete", func() {
		err := db.DeleteTenant(Tenant2, true)
		Ω(err).ShouldNot(HaveOccurred())

		targets, err := db.GetTarget(SomeTarget.UUID)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(targets).ShouldNot(BeNil())

		//show archive in 'tenant deleted' but pre purged status
		archives, err := db.GetAllArchives(&ArchiveFilter{WithStatus: []string{"tenant deleted"}})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(len(archives)).Should(Equal(1))

		err = db.Exec(`UPDATE archives SET status = 'purged' WHERE status = 'tenant deleted'`)
		archives, err = db.GetAllArchives(&ArchiveFilter{WithStatus: []string{"purged"}})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(len(archives)).Should(Equal(1))

		err = db.CleanTargets()
		Ω(err).ShouldNot(HaveOccurred())

		targets, err = db.GetTarget(SomeTarget.UUID)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(targets).Should(BeNil())
	})

	It("Stores are cleaned properly after tenant delete", func() {
		err := db.DeleteTenant(Tenant2, true)
		Ω(err).ShouldNot(HaveOccurred())

		store, err := db.GetStore(SomeStore.UUID)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(store).ShouldNot(BeNil())

		//show archive in 'tenant deleted' but pre purged status
		archives, err := db.GetAllArchives(&ArchiveFilter{WithStatus: []string{"tenant deleted"}})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(len(archives)).Should(Equal(1))

		//"purge" archive
		err = db.Exec(`UPDATE archives SET status = 'purged' WHERE status = 'tenant deleted'`)
		archives, err = db.GetAllArchives(&ArchiveFilter{WithStatus: []string{"purged"}})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(len(archives)).Should(Equal(1))

		err = db.CleanStores()
		Ω(err).ShouldNot(HaveOccurred())

		store, err = db.GetStore(SomeStore.UUID)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(store).Should(BeNil())
	})

})
