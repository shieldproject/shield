package supervisor_test

import (
	"github.com/markbates/goth/gothic"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/starkandwayne/shield/supervisor"
)

var _ = Describe("WebServer", func() {
	var ws *WebServer
	var req *http.Request
	BeforeEach(func() {
		db, err := Database()
		if err != nil {
			panic(err)
		}
		ws = &WebServer{
			Database: db,
			Addr:     ":0", // choose a random one
			Auth: AuthConfig{
				Basic: BasicAuthConfig{
					User:     "testuser",
					Password: "testpassword",
				},
			},
			Supervisor: &Supervisor{
				PrivateKeyFile: "",
			},
			WebRoot: "test/webroot",
		}
		http.DefaultServeMux = http.NewServeMux()
	})
	Describe("When serving requests", func() {
		var res *FakeResponder
		BeforeEach(func() {
			res = NewFakeResponder()
			err := ws.Setup()
			if err != nil {
				panic(err)
			}
		})
		It("Requires auth for protected resources", func() {
			var err error
			req, err = http.NewRequest("GET", "/", nil)
			Expect(err).ShouldNot(HaveOccurred())
			http.DefaultServeMux.ServeHTTP(res, req)
			Expect(res.Status).Should(Equal(401))
		})
		It("Allows access to unauthenticated resources", func() {
			var err error
			req, err = http.NewRequest("GET", "/v1/ping", nil)
			Expect(err).ShouldNot(HaveOccurred())
			http.DefaultServeMux.ServeHTTP(res, req)
			Expect(res.Status).Should(Equal(200))
		})
	})
	Describe("When using OAuth2", func() {
		var err error
		BeforeEach(func() {
			ws.Auth.OAuth = OAuthConfig{
				Provider: "faux",
				Sessions: SessionsConfig{
					Type: "mock",
				},
			}
			req, err = http.NewRequest("GET", "/", nil)
			if err != nil {
				panic(err)
			}
			gothic.Store = &FakeSessionStore{}
			gothic.Store.Get(req, gothic.SessionName)
		})
		It("sets up gothic.GetProviderName to return the configured provider", func() {
			Expect(ws.Setup()).Should(Succeed())
			Expect(gothic.GetProviderName(req)).Should(Equal("faux"))
		})
		It("sets up gothic.SetState to return a unique state value every time", func() {

			Expect(ws.Setup()).Should(Succeed())
			firstState := gothic.SetState(req)
			Expect(firstState).Should(Equal(gothic.Store.(*FakeSessionStore).Session.Values["state"]))
			secondState := gothic.SetState(req)
			Expect(secondState).ShouldNot(Equal(firstState))
			Expect(secondState).Should(Equal(gothic.Store.(*FakeSessionStore).Session.Values["state"]))

		})
	})
})
