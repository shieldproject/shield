package route_test

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/shieldproject/shield/route"
)

var _ = Describe("Session Cookie Security", func() {
	var (
		w   *httptest.ResponseRecorder
		req *http.Request
		r   *route.Request
	)

	BeforeEach(func() {
		var err error
		req, err = http.NewRequest("GET", "/", nil)
		Ω(err).ShouldNot(HaveOccurred())
		w = httptest.NewRecorder()
		r = route.NewRequest(w, req, false)
	})

	Describe("SetSession", func() {
		BeforeEach(func() {
			r.SetSession("test-session-id")
		})

		It("sets a cookie named 'shield7'", func() {
			cookies := w.Result().Cookies()
			Ω(len(cookies)).Should(BeNumerically(">=", 1))
			Ω(cookies[0].Name).Should(Equal("shield7"))
		})

		It("sets the cookie value to the session id", func() {
			cookies := w.Result().Cookies()
			Ω(cookies[0].Value).Should(Equal("test-session-id"))
		})

		It("sets the HttpOnly flag", func() {
			cookies := w.Result().Cookies()
			Ω(cookies[0].HttpOnly).Should(BeTrue())
		})

		// shieldd always speaks plain HTTP; a Secure cookie handed back over
		// an unencrypted connection is dropped by the browser on the spot.
		It("omits the Secure flag over a plain connection", func() {
			cookies := w.Result().Cookies()
			Ω(cookies[0].Secure).Should(BeFalse())
		})

		// Strict would withhold this cookie on the top-level cross-site
		// navigation back from an oauth provider, breaking oauth login.
		It("sets SameSite=Lax", func() {
			cookies := w.Result().Cookies()
			Ω(cookies[0].SameSite).Should(Equal(http.SameSiteLaxMode))
		})
	})

	Describe("SetSession over TLS", func() {
		It("sets the Secure flag when the request arrived over TLS", func() {
			req.TLS = &tls.ConnectionState{}
			route.NewRequest(w, req, false).SetSession("test-session-id")

			cookies := w.Result().Cookies()
			Ω(len(cookies)).Should(BeNumerically(">=", 1))
			Ω(cookies[0].Secure).Should(BeTrue())
		})

		// The daemon is normally fronted by a TLS-terminating proxy, so the
		// only evidence it has of the browser's scheme is this header.
		It("sets the Secure flag when a proxy reports an https scheme", func() {
			req.Header.Set("X-Forwarded-Proto", "https")
			route.NewRequest(w, req, false).SetSession("test-session-id")

			cookies := w.Result().Cookies()
			Ω(len(cookies)).Should(BeNumerically(">=", 1))
			Ω(cookies[0].Secure).Should(BeTrue())
		})

		It("ignores a proxy reporting a plain scheme", func() {
			req.Header.Set("X-Forwarded-Proto", "http")
			route.NewRequest(w, req, false).SetSession("test-session-id")

			cookies := w.Result().Cookies()
			Ω(len(cookies)).Should(BeNumerically(">=", 1))
			Ω(cookies[0].Secure).Should(BeFalse())
		})
	})

	Describe("SetCookie", func() {
		BeforeEach(func() {
			r.SetCookie("via", "cli", "/auth")
		})

		It("scopes the cookie to the given path", func() {
			cookies := w.Result().Cookies()
			Ω(len(cookies)).Should(BeNumerically(">=", 1))
			Ω(cookies[0].Name).Should(Equal("via"))
			Ω(cookies[0].Value).Should(Equal("cli"))
			Ω(cookies[0].Path).Should(Equal("/auth"))
		})

		// The oauth handler reads this cookie at /auth/:provider/redir, which
		// the provider reaches by redirecting the browser cross-site.
		It("sets SameSite=Lax so it survives the oauth redirect", func() {
			cookies := w.Result().Cookies()
			Ω(cookies[0].SameSite).Should(Equal(http.SameSiteLaxMode))
		})
	})

	Describe("ClearSession", func() {
		BeforeEach(func() {
			r.ClearSession()
		})

		It("sets MaxAge to 0 or negative", func() {
			cookies := w.Result().Cookies()
			Ω(len(cookies)).Should(BeNumerically(">=", 1))
			Ω(cookies[0].MaxAge).Should(BeNumerically("<=", 0))
		})

		It("sets the HttpOnly flag", func() {
			cookies := w.Result().Cookies()
			Ω(cookies[0].HttpOnly).Should(BeTrue())
		})

		It("omits the Secure flag over a plain connection", func() {
			cookies := w.Result().Cookies()
			Ω(cookies[0].Secure).Should(BeFalse())
		})

		It("sets SameSite=Lax", func() {
			cookies := w.Result().Cookies()
			Ω(cookies[0].SameSite).Should(Equal(http.SameSiteLaxMode))
		})
	})
})
