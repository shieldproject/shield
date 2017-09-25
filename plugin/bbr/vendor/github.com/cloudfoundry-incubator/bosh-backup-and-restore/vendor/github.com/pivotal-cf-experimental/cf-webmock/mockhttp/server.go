package mockhttp

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"runtime/debug"
	"strings"
	"sync"

	. "github.com/onsi/ginkgo"
)

type Server struct {
	expectedAuthorizationHeader string

	*httptest.Server
	*sync.Mutex

	mockHandlers   []MockedResponseBuilder
	currentHandler int

	logger *log.Logger
}

func StartServer(name string, startTestServer func(http.Handler) *httptest.Server) *Server {
	d := &Server{Mutex: new(sync.Mutex)}
	d.Server = startTestServer(d)

	d.logger = log.New(GinkgoWriter, "["+name+"] ", log.LstdFlags)
	return d
}

func (d *Server) ExpectedAuthorizationHeader(header string) {
	d.expectedAuthorizationHeader = header
}
func (d *Server) ExpectedBasicAuth(username, password string) {
	d.ExpectedAuthorizationHeader(basicAuth(username, password))
}

func (d *Server) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	d.Lock()
	defer d.Unlock()
	defer GinkgoRecover()
	if d.currentHandler >= len(d.mockHandlers) {
		Fail(fmt.Sprintf("unmocked call received\n\t%s\nCompleted mocks:\n%s\nPending mocks:\n%s\n", req.Method+" "+req.URL.String(), strings.Join(d.completedMocks(), "\n"), strings.Join(d.pendingMocks(), "\n")))
	}

	d.logger.Printf("%s %s\n", req.Method, req.URL.String())
	d.mockHandlers[d.currentHandler].Verify(req, d)
	d.mockHandlers[d.currentHandler].Respond(writer, d.logger)
	d.currentHandler += 1
}

func (d *Server) completedMocks() []string {
	completedMocks := []string{}
	for i := 0; i < d.currentHandler; i++ {
		completedMocks = append(completedMocks, "\t"+d.mockHandlers[i].Url())
	}
	return completedMocks
}

func (d *Server) pendingMocks() []string {
	pendingMocks := []string{}
	for i := d.currentHandler; i < len(d.mockHandlers); i++ {
		pendingMocks = append(pendingMocks, "\t"+d.mockHandlers[i].Url())
	}
	return pendingMocks
}

func (d *Server) VerifyAndMock(mockedResponses ...MockedResponseBuilder) {
	d.Lock()
	defer d.Unlock()
	d.VerifyMocks()

	d.currentHandler = 0
	d.mockHandlers = mockedResponses
}

func (d *Server) VerifyMocks() {
	if len(d.mockHandlers) != d.currentHandler {
		debug.PrintStack()
		Fail(fmt.Sprintf("Uninvoked mocks\nCompleted mocks:\n%s\nPending mocks:\n%s\n", strings.Join(d.completedMocks(), "\n"), strings.Join(d.pendingMocks(), "\n")))
	}
}

type MockedResponseBuilder interface {
	Verify(req *http.Request, d *Server)
	Respond(writer http.ResponseWriter, logger *log.Logger)
	Url() string
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}
