package db

import (
	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Target Deletion and Archive Orphaning", func() {
	var (
		testDB      *DB
		targetUUID  string
		storeUUID   string
		archiveUUID string
		tenantUUID  string
	)

	BeforeEach(func() {
		var err error

		tenantUUID = RandomID()
		targetUUID = RandomID()
		storeUUID = RandomID()
		archiveUUID = RandomID()

		testDB, err = Database(
			`INSERT INTO tenants (uuid, name)
			   VALUES ("`+tenantUUID+`", "Test Tenant")`,

			`INSERT INTO targets (uuid, tenant_uuid, name, summary, plugin, endpoint, agent)
			   VALUES ("`+targetUUID+`", "`+tenantUUID+`", "Test Target", "", "fs", '{"base":"/tmp"}', "127.0.0.1:5444")`,

			`INSERT INTO stores (uuid, tenant_uuid, name, summary, plugin, endpoint, agent)
			   VALUES ("`+storeUUID+`", "`+tenantUUID+`", "Test Store", "", "s3", '{"bucket":"test"}', "127.0.0.1:9938")`,

			`INSERT INTO archives (uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, notes, status, purge_reason, tenant_uuid)
			   VALUES ("`+archiveUUID+`", "`+targetUUID+`", "`+storeUUID+`", "s3://test/backup.tgz", 0, 9999999999, "(no notes)", "valid", "", "`+tenantUUID+`")`,
		)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(testDB).ShouldNot(BeNil())
	})

	AfterEach(func() {
		testDB.Disconnect()
	})

	Context("when deleting a target that has associated archives but no jobs", func() {
		It("succeeds without error", func() {
			deleted, err := testDB.DeleteTarget(targetUUID)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(deleted).Should(BeTrue())
		})

		It("leaves the archive row in the database (GetArchive still returns it)", func() {
			_, err := testDB.DeleteTarget(targetUUID)
			Ω(err).ShouldNot(HaveOccurred())

			// The archive row survives; GetArchive uses LEFT JOIN so the row
			// is still retrievable via the store join even after target deletion.
			archive, err := testDB.GetArchive(archiveUUID)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(archive).ShouldNot(BeNil())
			// TargetUUID comes back empty: LEFT JOIN yields NULL for t.uuid
			// and NullString.Valid == false leaves the field at its zero value.
			Ω(archive.TargetUUID).Should(Equal(""))
		})

		It("returns empty target metadata on the orphaned archive", func() {
			_, err := testDB.DeleteTarget(targetUUID)
			Ω(err).ShouldNot(HaveOccurred())

			archive, err := testDB.GetArchive(archiveUUID)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(archive).ShouldNot(BeNil())
			// LEFT JOIN returns NULL for target fields after deletion;
			// NullString.Valid == false means the field stays as zero value "".
			Ω(archive.TargetName).Should(Equal(""))
			Ω(archive.TargetPlugin).Should(Equal(""))
		})
	})

	Context("when deleting a target that has a job referencing it", func() {
		var jobUUID string

		BeforeEach(func() {
			jobUUID = RandomID()
			err := testDB.Exec(`
				INSERT INTO jobs (uuid, tenant_uuid, name, summary, paused,
				                 target_uuid, store_uuid, schedule, keep_days, retries)
				   VALUES ("` + jobUUID + `", "` + tenantUUID + `", "Some Job", "", 0,
				           "` + targetUUID + `", "` + storeUUID + `", "daily 3am", 7, 7)`)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns false without deleting the target", func() {
			deleted, err := testDB.DeleteTarget(targetUUID)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(deleted).Should(BeFalse())

			// target must still exist
			target, err := testDB.GetTarget(targetUUID)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(target).ShouldNot(BeNil())
		})
	})

	Context("when deleting a target that does not exist", func() {
		It("returns true (idempotent) without error", func() {
			nonexistent := RandomID()
			deleted, err := testDB.DeleteTarget(nonexistent)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(deleted).Should(BeTrue())
		})
	})
})
