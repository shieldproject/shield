package supervisor_test

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/faux"
	"github.com/starkandwayne/goutils/log"
	"io/ioutil"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/starkandwayne/shield/supervisor"
)

var _ = Describe("OAuthenticator", func() {
	var oa OAuthenticator
	var req *http.Request
	BeforeEach(func() {
		log.SetupLogging(log.LogConfig{Type: "file", File: "/dev/null", Level: "error"})
		data, err := ioutil.ReadFile("test/etc/jwt/valid.pem")
		if err != nil {
			panic(err)
		}
		sk, err := jwt.ParseRSAPrivateKeyFromPEM(data)
		if err != nil {
			panic(err)
		}
		oa = OAuthenticator{
			Cfg: OAuthConfig{
				Key:           "mykey",
				Secret:        "mysecret",
				JWTPrivateKey: sk,
				JWTPublicKey:  &sk.PublicKey,
			},
		}
		gothic.Store = &FakeSessionStore{}

		req, err = http.NewRequest("GET", "/", nil)
		if err != nil {
			panic(err)
		}
		gothic.Store.Get(req, gothic.SessionName)
		goth.UseProviders(&faux.Provider{})
		gothic.GetProviderName = func(req *http.Request) (string, error) {
			return "faux", nil
		}
	})

	Describe("When seeing if requests are authenticated", func() {
		It("Returns false if no bearer token, or authenticated session", func() {
			Expect(oa.IsAuthenticated(req)).Should(BeFalse())
		})
		It("Returns false if no bearer token, and session couldn't be retrieved", func() {
			gothic.Store.(*FakeSessionStore).Error = true
			Expect(oa.IsAuthenticated(req)).Should(BeFalse())
		})
		It("Returns false if bearer token is set and invalid", func() {
			req.Header.Set("Authorization", "bearer my.invalid.token")
			Expect(oa.IsAuthenticated(req)).Should(BeFalse())
		})
		It("Returns false if bearer token is set, valid, but expired", func() {
			jc := JWTCreator{SigningKey: oa.Cfg.JWTPrivateKey}
			expiredToken, err := jc.GenToken("not-currently-used", 0)
			Expect(err).ShouldNot(HaveOccurred())
			req.Header.Set("Authorization", "Bearer "+expiredToken)
			Expect(oa.IsAuthenticated(req)).Should(BeFalse())
		})
		It("Returns true if session is valid", func() {
			gothic.Store.(*FakeSessionStore).Session.Values["User"] = "validuser"
			Expect(oa.IsAuthenticated(req)).Should(BeTrue())
		})
		It("Returns true if bearer token is valid/unexpired", func() {
			jc := JWTCreator{SigningKey: oa.Cfg.JWTPrivateKey}
			validToken, err := jc.GenToken("not-currently-used", int(time.Now().Unix())+120)
			Expect(err).ShouldNot(HaveOccurred())
			req.Header.Set("Authorization", "Bearer "+validToken)
			Expect(oa.IsAuthenticated(req)).Should(BeTrue())
		})
	})
	Describe("When requiring authentication on a request", func() {
		It("Sends the client a 500 and error if error retrieving the session", func() {
			gothic.Store.(*FakeSessionStore).Error = true
			res := NewFakeResponder()
			oa.RequireAuth(res, req)
			Expect(res.Status).Should(Equal(500))
			data, err := res.ReadBody()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(data).Should(Equal("Unexpected error retrieving session data"))
		})
		It("Creates a new session for a client if received a securecookie error", func() {
			gothic.Store.(*FakeSessionStore).CookieError = true
			res := NewFakeResponder()
			req.URL.Path = "/from-whence-i-came"
			oa.RequireAuth(res, req)
			Expect(res.Status).Should(Equal(307))
			_, err := res.ReadBody()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(res.Header().Get("Location")).Should(Equal("http://example.com/auth/"))
			Expect(gothic.Store.(*FakeSessionStore).Session.Flashes()).Should(Equal([]interface{}{"/from-whence-i-came"}))
		})
		It("307s to the oauth provider if Should OAuth redirect", func() {
			res := NewFakeResponder()
			req.URL.Path = "/from-whence-i-came"
			oa.RequireAuth(res, req)
			Expect(res.Status).Should(Equal(307))
			_, err := res.ReadBody()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(res.Header().Get("Location")).Should(Equal("http://example.com/auth/"))
			Expect(gothic.Store.(*FakeSessionStore).Session.Flashes()).Should(Equal([]interface{}{"/from-whence-i-came"}))
		})
		It("401s with a bearer authenticate message if shouldn't oauth redirect", func() {
			res := NewFakeResponder()
			req.URL.Path = "/v1/i/am/an/api"
			oa.RequireAuth(res, req)
			Expect(res.Status).Should(Equal(401))
			data, err := res.ReadBody()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(res.Header().Get("WWW-authenticate")).Should(Equal("Bearer"))
			Expect(res.Header().Get("Location")).Should(Equal(""))
			Expect(data).Should(Equal("Unauthorized"))
		})
	})
	Describe("When determining if a request should get an oauth redirect", func() {
		It("Returns true if a request is not an API call", func() {
			Expect(ShouldOAuthRedirect("/non-api")).Should(BeTrue())
		})
		It("Returns true if a request is an auth call", func() {
			Expect(ShouldOAuthRedirect("/v1/auth/stuff")).Should(BeTrue())
		})
		It("Returns false if a request is an api call, and is not an auth call", func() {
			Expect(ShouldOAuthRedirect("/v1/beepboop")).Should(BeFalse())
		})
	})
	Describe("When processing oauth callbacks", func() {
		var res *FakeResponder
		BeforeEach(func() {
			res = NewFakeResponder()
			var err error
			req, err = http.NewRequest("GET", "/?state=csrf-detection&code=success", nil)
			if err != nil {
				panic(err)
			}
		})
		It("Returns a 500 if session state could not be retrieved", func() {
			gothic.Store.(*FakeSessionStore).Error = true
			OAuthCallback.ServeHTTP(res, req)
			Expect(res.Status).Should(Equal(500))
		})
		It("Returns a 403 if CSRF checking failed", func() {
			gothic.Store.(*FakeSessionStore).Session.Values["state"] = "willfail"
			OAuthCallback.ServeHTTP(res, req)
			Expect(res.Status).Should(Equal(403))
			data, err := res.ReadBody()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(data).Should(Equal("Unauthorized"))
		})
		It("Returns an 403 if gothic couldn't complete auth", func() {
			gothic.Store.(*FakeSessionStore).Session.Values["state"] = "csrf-detection"
			gothic.Store.(*FakeSessionStore).Session.Values[gothic.SessionName] = nil
			OAuthCallback.ServeHTTP(res, req)
			Expect(res.Status).Should(Equal(403))
			data, err := res.ReadBody()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(data).Should(Equal("UnOAuthorized"))
		})
		It("Returns a 403 if no code was specified in the callback from the provider", func() {
			var err error
			gothic.Store.(*FakeSessionStore).Session.Values["state"] = "csrf-detection"
			req, err = http.NewRequest("GET", "/?state=csrf-detection", nil)
			Expect(err).ShouldNot(HaveOccurred())
			OAuthCallback.ServeHTTP(res, req)
			Expect(res.Status).Should(Equal(403))
			data, err := res.ReadBody()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(data).Should(Equal("No oauth code issued from provider"))

		})
		It("Returns a 302 back to / if no flash was set in the session", func() {
			gothic.Store.(*FakeSessionStore).Session.Values["state"] = "csrf-detection"
			OAuthCallback.ServeHTTP(res, req)
			Expect(res.Status).Should(Equal(302))
			_, err := res.ReadBody()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(res.Header().Get("Location")).Should(Equal("/"))
		})
		It("Returns a 302 back to the flash url if it was present", func() {
			gothic.Store.(*FakeSessionStore).Session.Values["state"] = "csrf-detection"
			gothic.Store.(*FakeSessionStore).Session.AddFlash("/send-me-here")
			OAuthCallback.ServeHTTP(res, req)
			Expect(res.Status).Should(Equal(302))
			_, err := res.ReadBody()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(res.Header().Get("Location")).Should(Equal("/send-me-here"))
		})
	})
})
