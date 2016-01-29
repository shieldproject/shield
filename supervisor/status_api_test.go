package supervisor_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	. "github.com/starkandwayne/shield/supervisor"
)

var _ = Describe("HTTP Rest API", func() {
	Describe("/v1/status API", func() {
		It("handles GET requests", func() {
			r := GET(StatusAPI{}, "/v1/status")
			Ω(r.Code).Should(Equal(200))
		})

		It("ignores other HTTP methods", func() {
			for _, method := range []string{
				"PUT", "POST", "DELETE", "PATCH", "OPTIONS", "TRACE",
			} {
				NotImplemented(StatusAPI{}, method, "/v1/status", nil)
			}
		})

		It("ignores requests not to /v1/status (sub-URIs)", func() {
			NotImplemented(StatusAPI{}, "GET", "/v1/status/stuff", nil)
			NotImplemented(StatusAPI{}, "OPTIONS", "/v1/status/OPTIONAL/STUFF", nil)
		})
	})
})
