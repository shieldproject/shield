package api_test

import (
	. "api"
	"db"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func NotImplemented(h http.Handler, method string, uri string, body io.Reader) {
	req, _ := http.NewRequest(method, uri, body)
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)
	Ω(res.Code).Should(Equal(415),
		fmt.Sprintf("%s %s should elicit HTTP 415 (Not Implemented) response...", method, uri))
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
			NotFound(PingAPI{}, "GET", "/v1/ping/stuff", nil)
			NotFound(PingAPI{}, "OPTIONS", "/v1/ping/OPTIONAL/STUFF", nil)
		})
	})
})
