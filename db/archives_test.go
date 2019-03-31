package db

import (
	"time"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Archive Management", func() {
	TARGET_UUID := RandomID()
	STORE_UUID := RandomID()
	ARCHIVE_UUID := RandomID()
	TENANT_UUID := RandomID()

	var db *DB

	shouldHaveArchiveStatus := func(id, status string) {
		a, err := db.GetArchive(id)
		Ω(err).ShouldNot(HaveOccurred(), "Retrieving the archive should not have failed")
		Expect(a).ShouldNot(BeNil(), "An archive should be returned")
		Expect(a.Status).Should(Equal(status), "the archive should have correct status")
	}

	shouldHavePurgeReason := func(id, reason string) {
		a, err := db.GetArchive(id)
		Ω(err).ShouldNot(HaveOccurred(), "Retrieving the archive should not have failed")
		Expect(a).ShouldNot(BeNil(), "An archive should be returned")
		Expect(a.PurgeReason).Should(Equal(reason), "the archive should have correct purge_reason")
	}

	BeforeEach(func() {
		var err error
		db, err = Database(
			// need a target
			`INSERT INTO targets (uuid, plugin, endpoint, agent, name) VALUES ("`+TARGET_UUID+`", "target_plugin", "target_endpoint", "127.0.0.1:5444", "target_name")`,
			// need a store
			`INSERT INTO stores (uuid, plugin, endpoint, name) VALUES ("`+STORE_UUID+`", "store_plugin", "store_endpoint", "store_name")`,
			// need an ARCHIVE
			`INSERT INTO archives (uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, status, notes, purge_reason, tenant_uuid)
				VALUES ("`+ARCHIVE_UUID+`", "`+TARGET_UUID+`",
				        "`+STORE_UUID+`", "key", 0, 0, "valid", "my_notes", "", "`+TENANT_UUID+`")`,
		)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(db).ShouldNot(BeNil())

		shouldHaveArchiveStatus(ARCHIVE_UUID, "valid")
		shouldHavePurgeReason(ARCHIVE_UUID, "")
	})

	It("Archives can be invalidated", func() {
		err := db.InvalidateArchive(ARCHIVE_UUID)
		Expect(err).ShouldNot(HaveOccurred())

		shouldHaveArchiveStatus(ARCHIVE_UUID, "invalid")
	})

	It("Archives can be expired", func() {
		err := db.ExpireArchive(ARCHIVE_UUID)
		Expect(err).ShouldNot(HaveOccurred())

		shouldHaveArchiveStatus(ARCHIVE_UUID, "expired")
	})

	Describe("Purging archives", func() {
		It("with an archive whose status is 'valid'", func() {
			err := db.PurgeArchive(ARCHIVE_UUID)
			Expect(err).Should(HaveOccurred(), "should generate an error")

			shouldHaveArchiveStatus(ARCHIVE_UUID, "valid")
		})

		It("with an archive whose status is 'invalid'", func() {
			err := db.InvalidateArchive(ARCHIVE_UUID)
			Expect(err).ShouldNot(HaveOccurred(), "Invalidating archive should not have generated an error")

			err = db.PurgeArchive(ARCHIVE_UUID)
			Expect(err).ShouldNot(HaveOccurred(), "Purging archive should not have generated an error")

			shouldHaveArchiveStatus(ARCHIVE_UUID, "purged")
			shouldHavePurgeReason(ARCHIVE_UUID, "invalid")
		})

		It("If they are 'expired'", func() {
			err := db.ExpireArchive(ARCHIVE_UUID)
			Expect(err).ShouldNot(HaveOccurred(), "should not generate an error")

			err = db.PurgeArchive(ARCHIVE_UUID)
			Expect(err).ShouldNot(HaveOccurred(), "Purging archive should not have generated an error")

			shouldHaveArchiveStatus(ARCHIVE_UUID, "purged")
			shouldHavePurgeReason(ARCHIVE_UUID, "expired")
		})
	})

	Describe("Archive Retrieval", func() {
		TARGET2_UUID := RandomID()
		STORE2_UUID := RandomID()
		ARCHIVE_PURGED := RandomID()
		ARCHIVE_INVALID := RandomID()
		ARCHIVE_EXPIRED := RandomID()
		ARCHIVE_TARGET2 := RandomID()
		ARCHIVE_STORE2 := RandomID()
		BeforeEach(func() {
			var err error
			db.Exec(`INSERT INTO targets (uuid, plugin, endpoint, agent, name) VALUES("` + TARGET2_UUID + `","target_plugin2", "target_endpoint2", "127.0.0.1:5444", "target_name2")`)
			err = db.Exec(`INSERT INTO stores (uuid, plugin, endpoint, name) VALUES("` + STORE2_UUID + `","store_plugin2", "store_endpoint2", "store_name2")`)
			Expect(err).ShouldNot(HaveOccurred())
			err = db.Exec(`INSERT INTO archives (uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, status) VALUES("` +
				ARCHIVE_PURGED + `","` + TARGET_UUID + `", "` + STORE_UUID +
				`", "key", 10, 10, "purged")`)
			Expect(err).ShouldNot(HaveOccurred())
			err = db.Exec(`INSERT INTO archives (uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, status) VALUES("` +
				ARCHIVE_INVALID + `","` + TARGET_UUID + `", "` + STORE_UUID +
				`", "key", 10, 10, "invalid")`)
			Expect(err).ShouldNot(HaveOccurred())
			err = db.Exec(`INSERT INTO archives (uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, status) VALUES("` +
				ARCHIVE_EXPIRED + `","` + TARGET_UUID + `", "` + STORE_UUID +
				`", "key", 20, 20, "expired")`)
			Expect(err).ShouldNot(HaveOccurred())
			err = db.Exec(`INSERT INTO archives (uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, status) VALUES("` +
				ARCHIVE_TARGET2 + `","` + TARGET2_UUID + `", "` + STORE_UUID +
				`", "key", 20, 20, "valid")`)
			Expect(err).ShouldNot(HaveOccurred())
			err = db.Exec(`INSERT INTO archives (uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, status) VALUES("` +
				ARCHIVE_STORE2 + `","` + TARGET_UUID + `", "` + STORE2_UUID +
				`", "key", 20, 20, "invalid")`)
			Expect(err).ShouldNot(HaveOccurred())
		})
		Describe("Of Individual archives", func() {
			It("Should return the requested archive", func() {
				a, err := db.GetArchive(ARCHIVE_UUID)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(a).ShouldNot(BeNil())
				Expect(a).Should(BeEquivalentTo(&Archive{
					UUID:           ARCHIVE_UUID,
					TenantUUID:     TENANT_UUID,
					StoreKey:       "key",
					TakenAt:        0,
					ExpiresAt:      0,
					Notes:          "my_notes",
					Status:         "valid",
					PurgeReason:    "",
					Compression:    "none",
					TargetUUID:     TARGET_UUID,
					TargetName:     "target_name",
					TargetPlugin:   "target_plugin",
					TargetEndpoint: "target_endpoint",
					StoreUUID:      STORE_UUID,
					StoreName:      "store_name",
					StoreEndpoint:  "store_endpoint",
					StorePlugin:    "store_plugin",
				}))
			})
			It("Should return error nil/nil if no records exist", func() {
				a, err := db.GetArchive(RandomID())
				Expect(err).ShouldNot(HaveOccurred())
				Expect(a).Should(BeNil())
			})
		})

		Describe("Of multiple archives", func() {
			It("When filtering by Status", func() {
				filter := ArchiveFilter{
					WithStatus: []string{"purged"},
				}
				archives, err := db.GetAllArchives(&filter)
				Expect(err).ShouldNot(HaveOccurred(), "does not error")
				Expect(len(archives)).Should(Equal(1), "returns the correct number of archives")
				Expect(archives[0].UUID).Should(Equal(ARCHIVE_PURGED), "returns the correct archive")
			})
			It("When filtering by Target", func() {
				filter := ArchiveFilter{
					ForTarget: TARGET2_UUID,
				}
				archives, err := db.GetAllArchives(&filter)
				Expect(err).ShouldNot(HaveOccurred(), "does not error")
				Expect(len(archives)).Should(Equal(1), "returns the correct number of archives")
				Expect(archives[0].UUID).Should(Equal(ARCHIVE_TARGET2), "returns the correct archive")
			})
			It("When filtering by Store", func() {
				filter := ArchiveFilter{
					ForStore: STORE2_UUID,
				}
				archives, err := db.GetAllArchives(&filter)
				Expect(err).ShouldNot(HaveOccurred(), "does not error")
				Expect(len(archives)).Should(Equal(1), "returns the correct number of archives")
				Expect(archives[0].UUID).Should(Equal(ARCHIVE_STORE2), "returns the correct archive")
			})
			It("When filtering with After", func() {
				t := time.Unix(15, 0).UTC()
				filter := ArchiveFilter{
					After: &t,
				}
				archives, err := db.GetAllArchives(&filter)
				Expect(err).ShouldNot(HaveOccurred(), "does not error")
				Expect(len(archives)).Should(Equal(3), "returns the correct number of archives")

				var uuids []string
				for _, e := range archives {
					uuids = append(uuids, e.UUID)
				}
				Expect(uuids).Should(ConsistOf([]string{ARCHIVE_EXPIRED, ARCHIVE_TARGET2, ARCHIVE_STORE2}),
					"returns the correct archives")
			})
			It("When filtering with Before", func() {
				t := time.Unix(5, 0).UTC()
				filter := ArchiveFilter{
					Before: &t,
				}
				archives, err := db.GetAllArchives(&filter)
				Expect(err).ShouldNot(HaveOccurred(), "does not error")
				Expect(len(archives)).Should(Equal(1), "returns the correct number of archives")
				Expect(archives[0].UUID).Should(Equal(ARCHIVE_UUID), "returns the correct archive in the first result")
			})
			It("When filtering via a combination of values", func() {
				t := time.Unix(15, 0).UTC()
				filter := ArchiveFilter{
					WithStatus: []string{"invalid"},
					After:      &t,
				}
				archives, err := db.GetAllArchives(&filter)
				Expect(err).ShouldNot(HaveOccurred(), "does not error")
				Expect(len(archives)).Should(Equal(1), "returns the correct number of archives")
				Expect(archives[0].UUID).Should(Equal(ARCHIVE_STORE2), "returns the correct archive")

			})
			It("When filtering by WithoutStatus", func() {
				filter := ArchiveFilter{
					WithOutStatus: []string{"valid"},
				}
				archives, err := db.GetAllArchives(&filter)
				Expect(err).ShouldNot(HaveOccurred(), "does not error")
				Expect(len(archives)).Should(Equal(4), "returns the correct number of archives")

				var uuids []string
				for _, e := range archives {
					uuids = append(uuids, e.UUID)
				}
				Expect(uuids).Should(ConsistOf([]string{ARCHIVE_EXPIRED, ARCHIVE_PURGED, ARCHIVE_INVALID, ARCHIVE_STORE2}),
					"returns the correct archives")
			})
			It("limits the number of results returned with valid limit", func() {
				filter := ArchiveFilter{
					Limit: 3,
				}
				archives, err := db.GetAllArchives(&filter)
				Ω(err).ShouldNot(HaveOccurred(), "does not error")
				Ω(len(archives)).Should(Equal(3), "returns three archives")
			})
			It("correctly uses the limit in conjunction with other filters", func() {
				//This is prevented in the supervisor layer.
				filter := ArchiveFilter{
					WithOutStatus: []string{"valid"},
					Limit:         2,
				}
				archives, err := db.GetAllArchives(&filter)
				Ω(err).ShouldNot(HaveOccurred(), "does not err")
				Ω(len(archives)).Should(Equal(2), "returns two archives")
			})
			It("returns all entries when limit is higher than matching rows", func() {
				//This is prevented in the supervisor layer.
				filter := ArchiveFilter{
					Limit: 27,
				}
				archives, err := db.GetAllArchives(&filter)
				Ω(err).ShouldNot(HaveOccurred(), "does not err")
				Ω(len(archives)).Should(Equal(6), "returns six archives")
			})

		})

		Describe("GetArchivesNeedingPurge", func() {
			var expectedArchiveCount int

			BeforeEach(func() {
				all, err := db.GetAllArchives(nil)
				Expect(err).ShouldNot(HaveOccurred())
				valid, err := db.GetAllArchives(&ArchiveFilter{WithStatus: []string{"valid"}})
				Expect(err).ShouldNot(HaveOccurred())
				purged, err := db.GetAllArchives(&ArchiveFilter{WithStatus: []string{"purged"}})
				Expect(err).ShouldNot(HaveOccurred())
				expectedArchiveCount = len(all) - len(valid) - len(purged)
			})

			It("returns all jobs whose status is not 'purged' or 'valid'", func() {
				archives, err := db.GetArchivesNeedingPurge()
				Expect(err).ShouldNot(HaveOccurred(), "does not error")
				for _, archive := range archives {
					Expect(archive.Status).ShouldNot(Equal("purged"), "does not return 'purged' archives")
					Expect(archive.Status).ShouldNot(Equal("valid"), "does not return 'valid' archives")
				}
				Expect(len(archives)).Should(Equal(expectedArchiveCount), "returns the correct number of archives")
			})
		})

		Describe("GetExpiredArchives()", func() {
			UNEXPIRED_ARCHIVE := RandomID()
			UNEXPIRED_ARCHIVE2 := RandomID()
			EXPIRABLE_ARCHIVE := RandomID()

			var expectedArchiveCount int
			BeforeEach(func() {
				// get us a clean slate for these tests
				err := db.Exec(`DELETE FROM archives`)
				Expect(err).ShouldNot(HaveOccurred())

				// insert an archive that should be expired
				err = db.Exec(`INSERT INTO archives (uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, status) VALUES("`+
					EXPIRABLE_ARCHIVE+`","`+TARGET_UUID+`", "`+STORE2_UUID+
					`", "key", 20, ?, "valid")`, time.Now().Add(-30*time.Second).Unix())
				Expect(err).ShouldNot(HaveOccurred())

				// insert archive expiring in a day
				err = db.Exec(`INSERT INTO archives (uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, status) VALUES("`+
					UNEXPIRED_ARCHIVE+`","`+TARGET_UUID+`", "`+STORE2_UUID+
					`", "key", 20, ?, "valid")`, time.Now().Unix())

				Expect(err).ShouldNot(HaveOccurred())

				// insert an expired but invalid archive
				err = db.Exec(`INSERT INTO archives (uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, status) VALUES("` +
					UNEXPIRED_ARCHIVE2 + `","` + TARGET_UUID + `", "` + STORE2_UUID +
					`", "key", 20, 20, "invalid")`)
				Expect(err).ShouldNot(HaveOccurred())
				// get expeted count of expired archives
				all, err := db.GetAllArchives(nil)
				Expect(err).ShouldNot(HaveOccurred())

				expectedArchiveCount = len(all) - 2 // two un-expirable results in the db currently
			})
			It("returns all jobs who have expired", func() {
				archives, err := db.GetExpiredArchives()
				Expect(err).ShouldNot(HaveOccurred(), "does not error")
				for _, archive := range archives {
					Expect(archive.UUID).ShouldNot(Equal(UNEXPIRED_ARCHIVE), "does not return the unexpired archive")
					Expect(archive.UUID).ShouldNot(Equal(UNEXPIRED_ARCHIVE2), "does not return the expired but not 'valid' archive")
					Expect(archive.ExpiresAt).Should(BeNumerically("<", time.Now().Unix()), "does not return archives that have not expired yet")
					Expect(archive.Status).Should(Equal("valid"), "does not return archives that aren't valid")
				}
				Expect(len(archives)).Should(Equal(expectedArchiveCount), "returns the correct number of archives")
			})

		})
	})
})
