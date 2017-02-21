package supervisor_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/starkandwayne/goutils/log"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	"github.com/starkandwayne/shield/db"
	. "github.com/starkandwayne/shield/supervisor"
)

var _ = Describe("HTTP API /v1/schedule", func() {
	var API http.Handler
	var resyncChan chan int

	WEEKLY := `51e69607-eb48-4679-afd2-bc3b4c92e691`
	DAILY := `647bc775-b07b-4f87-bb67-d84cccac34a7`

	NIL := `00000000-0000-0000-0000-000000000000`

	databaseEntries := []string{
		`INSERT INTO schedules (uuid, name, summary, timespec) VALUES
				("` + WEEKLY + `",
				 "Weekly Backups",
				 "A schedule for weekly bosh-blobs, during normal maintenance windows",
				 "sundays at 3:15am")`,

		`INSERT INTO schedules (uuid, name, summary, timespec) VALUES
				("` + DAILY + `",
				 "Daily Backups",
				 "Use for daily (11-something-at-night) bosh-blobs",
				 "daily at 11:24pm")`,

		`INSERT INTO jobs (uuid, store_uuid, target_uuid, schedule_uuid, retention_uuid)
				VALUES ("abc-def", "` + NIL + `", "` + NIL + `", "` + WEEKLY + `", "` + NIL + `")`,
	}

	var data *db.DB

	BeforeEach(func() {
		log.SetupLogging(log.LogConfig{Type: "file", Level: "EMERG", File: "/dev/null"})
		var err error
		data, err = Database(databaseEntries...)
		Ω(err).ShouldNot(HaveOccurred())
		resyncChan = make(chan int, 1)
	})

	JustBeforeEach(func() {
		API = ScheduleAPI{
			Data:       data,
			ResyncChan: resyncChan,
		}

	})

	AfterEach(func() {
		close(resyncChan)
		resyncChan = nil
	})

	It("Fails to update schedules with bad timespecs", func() {
		res := PUT(API, "/v1/schedule/"+DAILY, WithJSON(`{
			"name"    : "Daily Backup Schedule",
			"summary" : "UPDATED?",
			"when"    : "this should fail"
		}`))
		Expect(res.Code).Should(Equal(500))
		Expect(res.Body.String()).Should(Equal(""))
	})
})
