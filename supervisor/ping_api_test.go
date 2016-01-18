package supervisor_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/starkandwayne/shield/supervisor"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"
)

var _ = Describe("HTTP Rest API", func() {
	Describe("/v1/ping API", func() {
		It("handles GET requests", func() {
			r := GET(PingAPI{}, "/v1/ping")
			Ω(r.Code).Should(Equal(200))
			Ω(r.Body.String()).Should(Equal(`{"ok":"pong"}`))
		})

		It("ignores other HTTP methods", func() {
			for _, method := range []string{
				"PUT", "POST", "DELETE", "PATCH", "OPTIONS", "TRACE",
			} {
				NotImplemented(PingAPI{}, method, "/v1/ping", nil)
			}
		})

		It("ignores requests not to /v1/ping (sub-URIs)", func() {
			NotImplemented(PingAPI{}, "GET", "/v1/ping/stuff", nil)
			NotImplemented(PingAPI{}, "OPTIONS", "/v1/ping/OPTIONAL/STUFF", nil)
		})
	})
})
