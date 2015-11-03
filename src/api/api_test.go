package api_test

import (
	. "api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	"db"

	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
)

func setupORM(sqls ...string) (*db.ORM, error) {
	database := &db.DB{
		Driver: "sqlite3",
		DSN:    ":memory:",
	}

	if err := database.Connect(); err != nil {
		return nil, err
	}

	orm, err := db.NewORM(database)
	if err != nil {
		database.Disconnect()
		return nil, err
	}

	if err := orm.Setup(); err != nil {
		database.Disconnect()
		return nil, err
	}

	for _, s := range sqls {
		err := database.ExecOnce(s)
		if err != nil {
			database.Disconnect()
			return nil, err
		}
	}

	return orm, nil
}

func notImplemented(h http.Handler, method string, uri string, body io.Reader) {
	req, _ := http.NewRequest(method, uri, body)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)
	Ω(w.Code).Should(Equal(415),
		fmt.Sprintf("%s %s should elicit HTTP 415 (Not Implemented) response...", method, uri))
	Ω(w.Body.String()).Should(Equal(""),
		fmt.Sprintf("%s %s should have no HTTP Response Body...", method, uri))
}

func notFound(h http.Handler, method string, uri string, body io.Reader) {
	req, _ := http.NewRequest(method, uri, body)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)
	Ω(w.Code).Should(Equal(404))
	Ω(w.Body.String()).Should(Equal(""))
}

var _ = Describe("HTTP Rest API", func() {
	Describe("/v1/ping API", func() {
		It("handles GET requests", func() {
			handler := PingAPI{}
			req, _ := http.NewRequest("GET", "/v1/ping", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)
			Ω(w.Code).Should(Equal(200))
			Ω(w.Body.String()).Should(Equal(`{"ok":"pong"}`))
		})

		It("ignores other HTTP methods", func() {
			for _, method := range []string{
				"PUT", "POST", "DELETE", "PATCH", "OPTIONS", "TRACE",
			} {
				notImplemented(PingAPI{}, method, "/v1/ping", nil)
			}
		})

		It("ignores requests not to /v1/ping (sub-URIs)", func() {
			notFound(PingAPI{}, "GET", "/v1/ping/stuff", nil)
			notFound(PingAPI{}, "OPTIONS", "/v1/ping/OPTIONAL/STUFF", nil)
		})
	})
})
