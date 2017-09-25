package uaa_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "github.com/cloudfoundry/bosh-cli/uaa"
)

var _ = Describe("UAA", func() {
	var (
		uaa    UAA
		server *ghttp.Server
	)

	BeforeEach(func() {
		uaa, server = BuildServer()
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("NewStaleAccessToken", func() {
		It("returns a new access token that can only be refreshed", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.VerifyBody([]byte("grant_type=refresh_token&refresh_token=refresh-token")),
					ghttp.VerifyHeader(http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}}),
					ghttp.RespondWith(http.StatusOK, `{
                 		"token_type": "new-bearer",
                 		"access_token": "new-access-token",
                 		"refresh_token": "new-refresh-token"
	                }`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.VerifyBody([]byte("grant_type=refresh_token&refresh_token=new-refresh-token")),
					ghttp.RespondWith(http.StatusOK, `{
                 		"token_type": "newer-bearer",
                 		"access_token": "newer-access-token",
                 		"refresh_token": "newer-refresh-token"
	                }`),
				),
			)

			newToken, err := uaa.NewStaleAccessToken("refresh-token").Refresh()
			Expect(err).ToNot(HaveOccurred())
			Expect(newToken.Type()).To(Equal("new-bearer"))
			Expect(newToken.Value()).To(Equal("new-access-token"))
			Expect(newToken.RefreshToken().Type()).To(Equal("new-bearer"))
			Expect(newToken.RefreshToken().Value()).To(Equal("new-refresh-token"))

			newerToken, err := newToken.Refresh()
			Expect(err).ToNot(HaveOccurred())
			Expect(newerToken.Type()).To(Equal("newer-bearer"))
			Expect(newerToken.Value()).To(Equal("newer-access-token"))
			Expect(newerToken.RefreshToken().Type()).To(Equal("newer-bearer"))
			Expect(newerToken.RefreshToken().Value()).To(Equal("newer-refresh-token"))
		})

		It("returns error if token response in non-200", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.RespondWith(http.StatusBadRequest, ``),
				),
			)

			_, err := uaa.NewStaleAccessToken("refresh-token").Refresh()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("UAA responded with non-successful status code"))
		})

		It("returns error if token cannot be unmarshalled", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.RespondWith(http.StatusOK, ``),
				),
			)

			_, err := uaa.NewStaleAccessToken("refresh-token").Refresh()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unmarshaling UAA response"))
		})

		It("panics if refresh value is empty", func() {
			Expect(func() { uaa.NewStaleAccessToken("") }).To(Panic())
		})
	})

	Describe("ClientCredentialsGrant", func() {
		It("obtains client token", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.VerifyBody([]byte("grant_type=client_credentials")),
					ghttp.VerifyHeader(http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}}),
					ghttp.VerifyBasicAuth("client", "client-secret"),
					ghttp.VerifyHeader(http.Header{
						"Accept": []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusOK, `{
                 		"token_type": "bearer",
                 		"access_token": "access-token"
	                }`),
				),
			)

			token, err := uaa.ClientCredentialsGrant()
			Expect(err).ToNot(HaveOccurred())
			Expect(token.Type()).To(Equal("bearer"))
			Expect(token.Value()).To(Equal("access-token"))
		})

		It("returns error if prompts response in non-200", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.RespondWith(http.StatusBadRequest, ``),
				),
			)

			_, err := uaa.ClientCredentialsGrant()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("UAA responded with non-successful status code"))
		})

		It("returns error if prompts cannot be unmarshalled", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.RespondWith(http.StatusOK, ``),
				),
			)

			_, err := uaa.ClientCredentialsGrant()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unmarshaling UAA response"))
		})
	})

	Describe("OwnerPasswordCredentialsGrant", func() {
		It("obtains access token based on prompt answers", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.VerifyBody([]byte("grant_type=password&key1=ans1&key2=ans2")),
					ghttp.VerifyHeader(http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}}),
					ghttp.VerifyBasicAuth("client", "client-secret"),
					ghttp.VerifyHeader(http.Header{
						"Accept": []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusOK, `{
                 		"token_type": "bearer",
                 		"access_token": "access-token",
                 		"refresh_token": "refresh-token"
	                }`),
				),
			)

			answers := []PromptAnswer{
				{Key: "key1", Value: "ans1"},
				{Key: "key2", Value: "ans2"},
			}

			token, err := uaa.OwnerPasswordCredentialsGrant(answers)
			Expect(err).ToNot(HaveOccurred())
			Expect(token.Type()).To(Equal("bearer"))
			Expect(token.Value()).To(Equal("access-token"))
			Expect(token.RefreshToken().Type()).To(Equal("bearer"))
			Expect(token.RefreshToken().Value()).To(Equal("refresh-token"))
		})

		It("returns error if prompts response in non-200", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.RespondWith(http.StatusBadRequest, ``),
				),
			)

			_, err := uaa.OwnerPasswordCredentialsGrant(nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("UAA responded with non-successful status code"))
		})

		It("returns error if prompts cannot be unmarshalled", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.RespondWith(http.StatusOK, ``),
				),
			)

			_, err := uaa.OwnerPasswordCredentialsGrant(nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unmarshaling UAA response"))
		})
	})
})
