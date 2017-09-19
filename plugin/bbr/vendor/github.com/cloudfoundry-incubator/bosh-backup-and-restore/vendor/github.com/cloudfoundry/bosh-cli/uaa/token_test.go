package uaa_test

import (
	"errors"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "github.com/cloudfoundry/bosh-cli/uaa"
)

var _ = Describe("AccessToken", func() {
	var (
		uaa    UAA
		token  AccessToken
		server *ghttp.Server
	)

	BeforeEach(func() {
		uaa, server = BuildServer()
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("Refresh", func() {
		BeforeEach(func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.RespondWith(http.StatusOK, `{
                 		"token_type": "bearer",
                 		"access_token": "access-token",
                 		"refresh_token": "refresh-token"
	                }`),
				),
			)

			var err error

			token, err = uaa.OwnerPasswordCredentialsGrant(nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns a new access token by using refresh token", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.VerifyBody([]byte("grant_type=refresh_token&refresh_token=refresh-token")),
					ghttp.RespondWith(http.StatusOK, `{
                 		"token_type": "new-bearer",
                 		"access_token": "new-access-token",
                 		"refresh_token": "new-refresh-token"
	                }`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.VerifyBody([]byte("grant_type=refresh_token&refresh_token=new-refresh-token")),
					ghttp.RespondWith(http.StatusOK, `{
                 		"token_type": "newer-bearer",
                 		"access_token": "newer-access-token",
                 		"refresh_token": "newer-refresh-token"
	                }`),
				),
			)

			newToken, err := token.Refresh()
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

			_, err := token.Refresh()
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

			_, err := token.Refresh()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unmarshaling UAA response"))
		})
	})
})

var _ = Describe("NewTokenInfoFromValue", func() {
	It("returns parsed token", func() {
		info, err := NewTokenInfoFromValue("seg.eyJ1c2VyX25hbWUiOiJhZG1pbiIsInNjb3BlIjpbIm9wZW5pZCIsImJvc2guYWRtaW4iXSwiZXhwIjoxMjN9.seg")
		Expect(err).ToNot(HaveOccurred())
		Expect(info).To(Equal(TokenInfo{
			Username:  "admin",
			Scopes:    []string{"openid", "bosh.admin"},
			ExpiredAt: 123,
		}))
	})

	It("returns an error if token doesnt have 3 segments", func() {
		_, err := NewTokenInfoFromValue("seg")
		Expect(err).To(Equal(errors.New("Expected token value to have 3 segments")))

		_, err = NewTokenInfoFromValue("seg.seg")
		Expect(err).To(Equal(errors.New("Expected token value to have 3 segments")))

		_, err = NewTokenInfoFromValue("seg.seg.seg.seg")
		Expect(err).To(Equal(errors.New("Expected token value to have 3 segments")))
	})

	It("returns an error if token's 2nd segment cannot be decoded", func() {
		_, err := NewTokenInfoFromValue("seg.eyJrZXkiOiJ2YWx1ZXoifQ==.seg")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Decoding token info"))
	})

	It("returns an error if token's 2nd segment cannot unmarshaled", func() {
		_, err := NewTokenInfoFromValue("seg.a2V5.seg")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Unmarshaling token info"))
	})
})
