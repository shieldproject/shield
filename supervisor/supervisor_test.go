package supervisor_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

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

type FakeSessionStore struct {
	Error       bool
	SaveError   bool
	Saved       int
	CookieError bool
	Session     *sessions.Session
}

func (fs *FakeSessionStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	if fs.Error {
		return nil, fmt.Errorf("Error getting session")
	}
	if fs.CookieError {
		fs.CookieError = false
		return nil, securecookie.MultiError{}
	}

	if fs.Session == nil {
		s, err := fs.New(r, name)
		if err != nil {
			return nil, err
		}
		fs.Session = s
	}
	return fs.Session, nil
}

func (fs *FakeSessionStore) New(r *http.Request, name string) (*sessions.Session, error) {
	if fs.Error {
		return nil, fmt.Errorf("Error creating session")
	}
	s := sessions.NewSession(gothic.Store, name)
	s.Values[gothic.SessionName] = `{"session":"exists"}`
	s.Values["User"] = "fakeUser"
	s.Values["Membership"] = map[string]interface{}{"Groups": []interface{}{"fakeGroup1", "fakeGroup2"}}
	s.Options = &sessions.Options{MaxAge: 60}
	return s, nil
}

func (fs *FakeSessionStore) Save(r *http.Request, w http.ResponseWriter, s *sessions.Session) error {
	if fs.SaveError {
		return fmt.Errorf("Error saving session")
	}
	fs.Session = s
	fs.Saved++
	return nil
}

type FakeResponder struct {
	Status  int
	Headers http.Header
	Body    *bytes.Buffer
}

func NewFakeResponder() *FakeResponder {
	return &FakeResponder{Body: bytes.NewBuffer([]byte{}), Headers: http.Header{}}
}

func (fr *FakeResponder) Header() http.Header {
	return fr.Headers
}

func (fr *FakeResponder) WriteHeader(i int) {
	fr.Status = i
}

func (fr *FakeResponder) Write(data []byte) (int, error) {
	if fr.Status == 0 {
		fr.Status = 200
	}
	return fr.Body.Write(data)
}

func (fr *FakeResponder) ReadBody() (string, error) {
	data, err := ioutil.ReadAll(fr.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func FakeResponderHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Processed request"))
	})
}

type FakeVerifier struct {
	Allow           bool
	MembershipError bool
}

func (fv *FakeVerifier) Verify(u string, m map[string]interface{}) bool {
	return fv.Allow
}

func (fv *FakeVerifier) Membership(u goth.User, c *http.Client) (map[string]interface{}, error) {
	if fv.MembershipError {
		return nil, fmt.Errorf("Mock error")
	}
	return map[string]interface{}{}, nil
}

type FakeProxy struct {
	Backend      *ghttp.Server
	ResponseCode int
}

func (fp *FakeProxy) RoundTrip(r *http.Request) (*http.Response, error) {
	r.URL.Host = fp.Backend.Addr()
	r.URL.Scheme = "http"

	return (&http.Transport{}).RoundTrip(r)
}
