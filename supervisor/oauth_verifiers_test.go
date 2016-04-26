package supervisor_test

import (
	"github.com/google/go-github/github"
	"github.com/markbates/goth"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "github.com/starkandwayne/shield/supervisor"
)

var _ = Describe("GithubVerifier", func() {
	var client *http.Client
	var proxy *FakeProxy
	var gv *GithubVerifier
	var fakeSvr *ghttp.Server

	BeforeEach(func() {
		fakeSvr = ghttp.NewServer()
		proxy = &FakeProxy{Backend: fakeSvr, ResponseCode: http.StatusOK}
		client = &http.Client{Transport: proxy}
		gv = &GithubVerifier{Orgs: []string{"no-such-group"}}
	})
	AfterEach(func() {
		proxy.Backend.Close()
	})
	Context("When Verifying a user's access", func() {
		BeforeEach(func() {
			proxy.Backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/user/orgs", "page=1"),
					ghttp.RespondWithJSONEncoded(
						proxy.ResponseCode,
						[]github.Organization{
							{Login: github.String("test-org-1")},
							{Login: github.String("test-org-2")},
						},
						http.Header{
							"Link": []string{
								`<https://github.example.com/user/orgs?page=2>; rel="next"`,
								`<https://github.example.com/user/orgs?page=2>; rel="last"`,
							},
						},
					),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/user/orgs", "page=2"),
					ghttp.RespondWithJSONEncoded(
						proxy.ResponseCode,
						[]github.Organization{
							{Login: github.String("test-org-3")},
						},
						http.Header{
							"Link": []string{
								`<https://github.example.com/user/orgs?page=1>; rel="first"`,
								`<https://github.example.com/user/orgs?page=1>; rel="prev"`,
							},
						},
					),
				),
			)
		})
		It("Denies access if there was an error communicating with github", func() {
			proxy.ResponseCode = http.StatusInternalServerError
			Expect(gv.Verify(goth.User{}, client)).To(BeFalse())
		})
		It("Denies access if the user was not in an org in the list of allowed orgs", func() {
			Expect(gv.Verify(goth.User{}, client)).To(BeFalse())
		})
		It("Grants access if the user was in an org in the list of allowed orgs", func() {
			gv.Orgs = []string{"test-org-3"}
			Expect(gv.Verify(goth.User{}, client)).To(BeTrue())
		})
		It("Denies access if there are no orgs specified to allow access", func() {
			gv.Orgs = []string{}
			Expect(gv.Verify(goth.User{}, client)).To(BeFalse())
		})
	})
})
