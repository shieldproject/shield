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
				Client:        http.DefaultClient,
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
		OAuthVerifier = &FakeVerifier{Allow: true}
	})

	Describe("When seeing if requests are authenticated", func() {
		Context("For bearer token authentication", func() {
			It("Returns false if no bearer token, or authenticated session", func() {
				delete(gothic.Store.(*FakeSessionStore).Session.Values, "User")
				delete(gothic.Store.(*FakeSessionStore).Session.Values, "Membership")
				delete(gothic.Store.(*FakeSessionStore).Session.Values, gothic.SessionName)

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
				expiredToken, err := jc.GenToken("user1", map[string]interface{}{}, 0)
				Expect(err).ShouldNot(HaveOccurred())
				req.Header.Set("Authorization", "Bearer "+expiredToken)
				Expect(oa.IsAuthenticated(req)).Should(BeFalse())
			})
			It("Returns false if bearer token user is not a string", func() {
				token := jwt.New(jwt.SigningMethodRS256)
				token.Claims["user"] = 1234
				token.Claims["expiration"] = int(time.Now().Unix()) + 120
				token.Claims["membership"] = map[string]interface{}{}
				badUserToken, err := token.SignedString(oa.Cfg.JWTPrivateKey)
				Expect(err).ShouldNot(HaveOccurred())
				req.Header.Set("Authorization", "Bearer "+badUserToken)
				Expect(oa.IsAuthenticated(req)).Should(BeFalse())
			})
			It("Returns false if bearer token membership is not a map[string]interface{}", func() {
				jc := JWTCreator{SigningKey: oa.Cfg.JWTPrivateKey}
				badMembershipToken, err := jc.GenToken("user1", []string{"group1"}, int(time.Now().Unix())+120)
				Expect(err).ShouldNot(HaveOccurred())
				req.Header.Set("Authorization", "Bearer "+badMembershipToken)
				Expect(oa.IsAuthenticated(req)).Should(BeFalse())
			})
			It("Returns false if bearer token's OAuthVerifier.Verify() failed", func() {
				jc := JWTCreator{SigningKey: oa.Cfg.JWTPrivateKey}
				validToken, err := jc.GenToken("user1", map[string]interface{}{}, int(time.Now().Unix())+120)
				Expect(err).ShouldNot(HaveOccurred())
				req.Header.Set("Authorization", "Bearer "+validToken)
				OAuthVerifier.(*FakeVerifier).Allow = false
				Expect(oa.IsAuthenticated(req)).Should(BeFalse())
			})
			It("Returns true if OAuthVerifier.Verify() succeeds", func() {
				jc := JWTCreator{SigningKey: oa.Cfg.JWTPrivateKey}
				validToken, err := jc.GenToken("user1", map[string]interface{}{}, int(time.Now().Unix())+120)
				Expect(err).ShouldNot(HaveOccurred())
				req.Header.Set("Authorization", "Bearer "+validToken)
				Expect(oa.IsAuthenticated(req)).Should(BeTrue())
			})
		})
		Context("For session based authentication", func() {
			It("Returns false when retrieving session data fails", func() {
				gothic.Store.(*FakeSessionStore).Error = true
				gothic.Store.(*FakeSessionStore).Session.Values["User"] = "validuser"
				gothic.Store.(*FakeSessionStore).Session.Values["Membership"] = map[string]interface{}{}
				Expect(oa.IsAuthenticated(req)).Should(BeFalse())
			})
			It("Returns false when session value for 'User' is not string", func() {
				gothic.Store.(*FakeSessionStore).Session.Values["User"] = 1234
				gothic.Store.(*FakeSessionStore).Session.Values["Membership"] = map[string]interface{}{}
				Expect(oa.IsAuthenticated(req)).Should(BeFalse())
			})
			It("Returns false when 'Membership' value for user is not map[string]interface{}", func() {
				gothic.Store.(*FakeSessionStore).Session.Values["User"] = "validuser"
				gothic.Store.(*FakeSessionStore).Session.Values["Membership"] = "oops"
				Expect(oa.IsAuthenticated(req)).Should(BeFalse())
			})
			It("Returns false when OAuthVerify fails", func() {
				gothic.Store.(*FakeSessionStore).Session.Values["User"] = "validuser"
				gothic.Store.(*FakeSessionStore).Session.Values["Membership"] = map[string]interface{}{}
				OAuthVerifier.(*FakeVerifier).Allow = false
				Expect(oa.IsAuthenticated(req)).Should(BeFalse())
			})
			It("Returns true if OAuthVerifier.Verify() succeeds", func() {
				gothic.Store.(*FakeSessionStore).Session.Values["User"] = "validuser"
				gothic.Store.(*FakeSessionStore).Session.Values["Membership"] = map[string]interface{}{}
				Expect(oa.IsAuthenticated(req)).Should(BeTrue())
			})
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
			oa.OAuthCallback().ServeHTTP(res, req)
			Expect(res.Status).Should(Equal(500))
		})
		It("Returns a 403 if CSRF checking failed", func() {
			gothic.Store.(*FakeSessionStore).Session.Values["state"] = "willfail"
			oa.OAuthCallback().ServeHTTP(res, req)
			Expect(res.Status).Should(Equal(403))
			data, err := res.ReadBody()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(data).Should(Equal("Unauthorized"))
		})
		It("Returns an 403 if gothic couldn't complete auth", func() {
			gothic.Store.(*FakeSessionStore).Session.Values["state"] = "csrf-detection"
			gothic.Store.(*FakeSessionStore).Session.Values[gothic.SessionName] = nil
			oa.OAuthCallback().ServeHTTP(res, req)
			Expect(res.Status).Should(Equal(403))
			data, err := res.ReadBody()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(data).Should(Equal("UnOAuthorized"))
		})
		It("Returns a 403 if no code was specified in the callback from the provider", func() {
			var err error
			gothic.Store.(*FakeSessionStore).Session.Values["state"] = "csrf-detection"
			req, err = http.NewRequest("GET", "?state=csrf-detection", nil) // valid state, no code
			Expect(err).ShouldNot(HaveOccurred())
			oa.OAuthCallback().ServeHTTP(res, req)
			Expect(res.Status).Should(Equal(403))
			data, err := res.ReadBody()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(data).Should(Equal("No oauth code issued from provider"))
		})
		It("Returns a 403 error when UserAuthenticator.Membership() errors", func() {
			gothic.Store.(*FakeSessionStore).Session.Values["state"] = "csrf-detection"
			OAuthVerifier.(*FakeVerifier).MembershipError = true
			oa.OAuthCallback().ServeHTTP(res, req)
			Expect(res.Status).Should(Equal(403))
			data, err := res.ReadBody()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(data).Should(Equal("Unable to verify your membership"))
		})
		It("Returns a 403 if the user could not be authorized with the OAuthVerifier", func() {
			var err error
			gothic.Store.(*FakeSessionStore).Session.Values["state"] = "csrf-detection"
			OAuthVerifier.(*FakeVerifier).Allow = false
			oa.OAuthCallback().ServeHTTP(res, req)
			Expect(res.Status).Should(Equal(403))
			data, err := res.ReadBody()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(data).Should(Equal("You are not authorized to view this content"))
		})
		It("Returns a 500 error when failing to save session data", func() {
			gothic.Store.(*FakeSessionStore).Session.Values["state"] = "csrf-detection"
			gothic.Store.(*FakeSessionStore).SaveError = true
			oa.OAuthCallback().ServeHTTP(res, req)
			Expect(res.Status).Should(Equal(500))
			data, err := res.ReadBody()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(data).Should(Equal("Unable to save authentication data. Check the SHIELD logs for more info."))
		})
		It("Returns a 302 back to / if no flash was set in the session", func() {
			gothic.Store.(*FakeSessionStore).Session.Values["state"] = "csrf-detection"
			oa.OAuthCallback().ServeHTTP(res, req)
			Expect(res.Status).Should(Equal(302))
			_, err := res.ReadBody()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(res.Header().Get("Location")).Should(Equal("/"))
		})
		It("Returns a 302 back to the flash url if it was present", func() {
			gothic.Store.(*FakeSessionStore).Session.Values["state"] = "csrf-detection"
			gothic.Store.(*FakeSessionStore).Session.AddFlash("/send-me-here")
			oa.OAuthCallback().ServeHTTP(res, req)
			Expect(res.Status).Should(Equal(302))
			_, err := res.ReadBody()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(res.Header().Get("Location")).Should(Equal("/send-me-here"))
		})
		It("Saves 'User' and 'Membership' session values appropriately", func() {
			gothic.Store.(*FakeSessionStore).Session.Values["state"] = "csrf-detection"
			oa.OAuthCallback().ServeHTTP(res, req)
			Expect(res.Status).Should(Equal(302))
			Expect(gothic.Store.(*FakeSessionStore).Saved).Should(Equal(1))
		})
	})
})
