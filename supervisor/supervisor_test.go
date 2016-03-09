package supervisor_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/gomega"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	"github.com/starkandwayne/shield/db"
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
