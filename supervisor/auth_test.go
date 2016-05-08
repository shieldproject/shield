package supervisor_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/starkandwayne/shield/supervisor"
)

var _ = Describe("Authentication", func() {
	var req *http.Request
	BeforeEach(func() {
		var err error
		req, err = http.NewRequest("GET", "/", nil)
		if err != nil {
			panic(err)
		}
	})
	Describe("BasicAuthenticator", func() {
		var ba BasicAuthenticator
		BeforeEach(func() {
			ba = BasicAuthenticator{
				Cfg: BasicAuthConfig{
					User:     "testuser",
					Password: "testpassword",
				},
			}
		})
		Describe("When seeing if requests are authenticated", func() {
			It("Returns false if Auth Header is not basic", func() {
				req.Header.Set("www-authenticate", "bearer this.bearer.token")
				Expect(ba.IsAuthenticated(req)).Should(BeFalse())
			})
			It("Returns false if there was a failure decoding the credentials", func() {
				req.Header.Set("www-authenticate", "basic this.bearer.token")
				Expect(ba.IsAuthenticated(req)).Should(BeFalse())
			})
			It("Returns false if the credentials were incorrect", func() {
				req.Header.Set("www-authenticate", "basic dGVzdHBhc3N3b3JkOnRlc3R1c2Vy")
				Expect(ba.IsAuthenticated(req)).Should(BeFalse())
			})
			It("Returns true if credentials were correct", func() {
				req.Header.Set("www-authenticate", "basic dGVzdHVzZXI6dGVzdHBhc3N3b3Jk")
				Expect(ba.IsAuthenticated(req)).Should(BeFalse())
			})
		})
		Describe("When requiring authentication on a request", func() {
			It("Sends the client a 401, with a proper www-authenticate header", func() {
				res := NewFakeResponder()
				ba.RequireAuth(res, req)
				Expect(res.Status).Should(Equal(401))
				Expect(res.Headers.Get("Www-authenticate")).Should(Equal("Basic realm=\"shield\""))
				data, err := res.ReadBody()
				Expect(err).ShouldNot(HaveOccurred())
				Expect(data).Should(Equal("Unauthorized"))
			})
		})
	})
	Describe("When handling authentication of http requests", func() {
		var handler http.Handler
		var res *FakeResponder
		BeforeEach(func() {
			UserAuthenticator = BasicAuthenticator{
				Cfg: BasicAuthConfig{
					User:     "testuser",
					Password: "testpassword",
				},
			}
			api_tokens := map[string]string{
				"test-token": "LETMEINFORTESTING",
			}
			handler = Authenticate(api_tokens, FakeResponderHandler())
			res = NewFakeResponder()
		})
		It("Allows the next handler to serve request if request had a valid API token", func() {
			req.Header.Set("X-Shield-Token", "LETMEINFORTESTING")
			handler.ServeHTTP(res, req)
			Expect(res.Status).Should(Equal(200))
			data, err := res.ReadBody()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(data).Should(Equal("Processed request"))
		})
		It("Prevents the next handler from serving request if it had an invalid API token", func() {
			req.Header.Set("X-Shield-Token", "LETMEINPRETTYPLEASE")
			handler.ServeHTTP(res, req)
			Expect(res.Status).Should(Equal(401))
			data, err := res.ReadBody()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(data).Should(Equal("Unauthorized"))
		})
		It("Allows the next handler to serve request if the request was authenticated properly", func() {
			req.Header.Set("Authorization", "basic dGVzdHVzZXI6dGVzdHBhc3N3b3Jk")
			handler.ServeHTTP(res, req)
			Expect(res.Status).Should(Equal(200))
			data, err := res.ReadBody()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(data).Should(Equal("Processed request"))
		})
		It("Prevents the next handler from serving requests if not authenticated properly", func() {
			req.Header.Set("Authorization", "basic dGVzdHBhc3N3b3JkOnRlc3R1c2Vy")
			handler.ServeHTTP(res, req)
			Expect(res.Status).Should(Equal(401))
			data, err := res.ReadBody()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(data).Should(Equal("Unauthorized"))
		})
	})
	Describe("When parsing auth headers", func() {
		It("returns empty strings if there are not 2 values to the authorization header", func() {
			kind, token := AuthHeader(req)
			Expect(kind).Should(Equal(""))
			Expect(token).Should(Equal(""))
		})
		It("Returns auth type, and auth token for proper headers", func() {
			req.Header.Set("Authorization", "basic dGVzdHVzZXI6dGVzdHBhc3N3b3Jk")
			kind, token := AuthHeader(req)
			Expect(kind).Should(Equal("basic"))
			Expect(token).Should(Equal("dGVzdHVzZXI6dGVzdHBhc3N3b3Jk"))
		})
	})
})
