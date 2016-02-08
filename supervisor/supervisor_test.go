package supervisor_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	"github.com/starkandwayne/shield/db"
	. "github.com/starkandwayne/shield/supervisor"
)

func Database(sqls ...string) (*db.DB, error) {
	database := &db.DB{
		Driver: "sqlite3",
		DSN:    ":memory:",
	}

	if err := database.Connect(); err != nil {
		return nil, err
	}

	if err := database.Setup(); err != nil {
		database.Disconnect()
		return nil, err
	}

	for _, s := range sqls {
		err := database.Exec(s)
		if err != nil {
			database.Disconnect()
			return nil, err
		}
	}

	return database, nil
}

func JSONValidated(h http.Handler, method string, uri string) {
	req, _ := http.NewRequest(method, uri, strings.NewReader("}"))
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)
	Ω(res.Code).Should(Equal(400),
		fmt.Sprintf("%s %s should elicit HTTP 400 (Bad Request) response...", method, uri))
	Ω(res.Body.String()).Should(
		MatchJSON(`{"error":"bad JSON payload: invalid character '}' looking for beginning of value"}`),
		fmt.Sprintf("%s %s should have a JSON error in the Response Body...", method, uri))
}

func NotImplemented(h http.Handler, method string, uri string, body io.Reader) {
	req, _ := http.NewRequest(method, uri, body)
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)
	Ω(res.Code).Should(Equal(501),
		fmt.Sprintf("%s %s should elicit HTTP 501 (Not Implemented) response...", method, uri))
	Ω(res.Body.String()).Should(Equal(""),
		fmt.Sprintf("%s %s should have no HTTP Response Body...", method, uri))
}

func NotFound(h http.Handler, method string, uri string, body io.Reader) {
	req, _ := http.NewRequest(method, uri, body)
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)
	Ω(res.Code).Should(Equal(404))
	Ω(res.Body.String()).Should(Equal(""))
}

func GET(h http.Handler, uri string) *httptest.ResponseRecorder {
	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", uri, nil)

	h.ServeHTTP(res, req)
	return res
}

func WithJSON(s string) string {
	var data interface{}
	Ω(json.Unmarshal([]byte(s), &data)).Should(Succeed(),
		fmt.Sprintf("this is not JSON:\n%s\n", s))
	return s
}

func POST(h http.Handler, uri string, body string) *httptest.ResponseRecorder {
	res := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", uri, strings.NewReader(body))

	h.ServeHTTP(res, req)
	return res
}

func PUT(h http.Handler, uri string, body string) *httptest.ResponseRecorder {
	res := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", uri, strings.NewReader(body))

	h.ServeHTTP(res, req)
	return res
}

func DELETE(h http.Handler, uri string) *httptest.ResponseRecorder {
	res := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", uri, nil)

	h.ServeHTTP(res, req)
	return res
}

var _ = Describe("Retrieving Jobs", func() {
	var s Supervisor
	BeforeEach(func() {
		var err error
		s.Database, err = Database()
		Ω(err).ShouldNot(HaveOccurred())
		s.Timeout = 1 * time.Second
	})
	Context("With an empty database", func() {
		It("should return an empty list of jobs", func() {
			jobs, err := s.GetAllJobs()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(len(jobs)).Should(Equal(0))
		})
	})
	Context("With a non-empty database", func() {
		BeforeEach(func() {
			var err error
			JOB_ONE_UUID := `6809c55a-8250-11e5-8bcf-feff819cdc9f`
			TARGET_ONE_UUID := `c957554e-6fe0-4ae9-816d-307e20f155cb`
			STORE_ONE_UUID := `32d810b6-073b-4296-8c68-6544a91760f9`
			SCHED_ONE_UUID := `b36eeea3-9f5c-46f2-a337-6de5344e8d0f`
			RETEN_ONE_UUID := `52c6f512-5c90-4364-998c-f849fa416243`
			JOB_TWO_UUID := `b20ca8b6-8250-11e5-8bcf-feff819cdc9f`
			TARGET_TWO_UUID := `cf65a73e-79c1-48e8-b706-23ec7644c721`
			STORE_TWO_UUID := `9eb022c4-227f-44f1-b11b-b6d8bcfc3c4f`
			SCHED_TWO_UUID := `4b8432b9-b5e9-46d5-b23e-ba70983a2acc`
			RETEN_TWO_UUID := `f7dee6c2-59c4-439d-9d92-04046b8beb68`
			s.Database, err = Database(
				`INSERT INTO jobs (uuid, target_uuid, store_uuid, schedule_uuid, retention_uuid, paused, name, summary) VALUES
					("`+JOB_ONE_UUID+`",
					 "`+TARGET_ONE_UUID+`",
					 "`+STORE_ONE_UUID+`",
					 "`+SCHED_ONE_UUID+`",
					 "`+RETEN_ONE_UUID+`",
					 "t",
					 "job 1",
					 "First test job in queue")`,
				`INSERT INTO targets (uuid, name, summary, plugin, endpoint, agent) VALUES
					 ("`+TARGET_ONE_UUID+`",
					 "redis-shared",
					 "Shared Redis services for CF",
					 "redis",
					 "<<redis-configuration>>",
					 "127.0.0.1:5444")`,
				`INSERT INTO stores (uuid, name, summary, plugin, endpoint) VALUES
					("`+STORE_ONE_UUID+`",
					 "redis-shared",
					 "Shared Redis services for CF",
					 "redis",
					 "<<redis-configuration>>")`,
				`INSERT INTO schedules (uuid, name, summary, timespec) VALUES
					("`+SCHED_ONE_UUID+`",
					 "Weekly Backups",
					 "A schedule for weekly bosh-blobs, during normal maintenance windows",
					 "sundays at 3:15am")`,
				`INSERT INTO retention (uuid, name, summary, expiry) VALUES
					("`+RETEN_ONE_UUID+`",
					 "Hourly Retention",
					 "Keep backups for 1 hour",
					 3600)`,
				`INSERT INTO jobs (uuid, target_uuid, store_uuid, schedule_uuid, retention_uuid, paused, name, summary) VALUES
					("`+JOB_TWO_UUID+`",
					 "`+TARGET_TWO_UUID+`",
					 "`+STORE_TWO_UUID+`",
					 "`+SCHED_TWO_UUID+`",
					 "`+RETEN_TWO_UUID+`",
					 "f",
					 "job 2",
					 "Second test job in queue")`,
				`INSERT INTO targets (uuid, name, summary, plugin, endpoint, agent) VALUES
					("`+TARGET_TWO_UUID+`",
					 "s3",
					 "Amazon S3 Blobstore",
					 "s3",
					 "<<s3-configuration>>",
					 "127.0.0.1:5444")`,
				`INSERT INTO stores (uuid, name, summary, plugin, endpoint) VALUES
					("`+STORE_TWO_UUID+`",
					 "s3",
					 "Amazon S3 Blobstore",
					 "s3",
					 "<<s3-configuration>>")`,
				`INSERT INTO schedules (uuid, name, summary, timespec) VALUES
					("`+SCHED_TWO_UUID+`",
					 "Daily Backups",
					 "Use for daily (11-something-at-night) bosh-blobs",
					 "daily at 11:24pm")`,
				`INSERT INTO retention (uuid, name, summary, expiry) VALUES
					("`+RETEN_TWO_UUID+`",
					 "Yearly Retention",
					 "Keep backups for 1 year",
					 31536000)`,
			)
			Ω(err).ShouldNot(HaveOccurred())
		})
		It("should return a complete list of jobs", func() {
			jobs, err := s.GetAllJobs()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(len(jobs)).Should(Equal(2))
		})
		It("should return failed job in error list", func() {
			var err error
			JOB_ERR_UUID := `36bbb985-9d1a-4086-b154-c66f70501522`
			TARGET_ERR_UUID := `40f73cb2-1f81-4203-bb50-0eea2da3fb47`
			STORE_ERR_UUID := `c0db87f2-629f-4fe6-ab8f-29e7f2831fbb`
			SCHED_ERR_UUID := `29d381b8-021e-4049-82c0-52a5d3e52794`
			RETEN_ERR_UUID := `01c0b2e2-8e4e-4039-a0b1-74f429180c4c`
			s.Database, err = Database(
				`INSERT INTO jobs (uuid, target_uuid, store_uuid, schedule_uuid, retention_uuid, paused, name, summary) VALUES
					("`+JOB_ERR_UUID+`",
					 "`+TARGET_ERR_UUID+`",
					 "`+STORE_ERR_UUID+`",
					 "`+SCHED_ERR_UUID+`",
					 "`+RETEN_ERR_UUID+`",
					 "t",
					 "job err",
					 "Job with malformed sched")`,
				`INSERT INTO targets (uuid, name, summary, plugin, endpoint, agent) VALUES
					 ("`+TARGET_ERR_UUID+`",
					 "redis-shared",
					 "Shared Redis services for CF",
					 "redis",
					 "<<redis-configuration>>",
					 "127.0.0.1:5444")`,
				`INSERT INTO stores (uuid, name, summary, plugin, endpoint) VALUES
					("`+STORE_ERR_UUID+`",
					 "redis-shared",
					 "Shared Redis services for CF",
					 "redis",
					 "<<redis-configuration>>")`,
				`INSERT INTO schedules (uuid, name, summary, timespec) VALUES
					("`+SCHED_ERR_UUID+`",
					 "Weekly Backups",
					 "A schedule for weekly bosh-blobs, during normal maintenance windows",
					 "yearly at 3:15am")`,
				`INSERT INTO retention (uuid, name, summary, expiry) VALUES
					("`+RETEN_ERR_UUID+`",
					 "Hourly Retention",
					 "Keep backups for 1 hour",
					 3600)`,
			)
			_, err = s.GetAllJobs()
			Ω(err.Error()).Should(MatchRegexp(`the following job\(s\) failed: job err \(` + JOB_ERR_UUID + `\)`))
			Ω(err).Should(HaveOccurred())
		})
	})
})
