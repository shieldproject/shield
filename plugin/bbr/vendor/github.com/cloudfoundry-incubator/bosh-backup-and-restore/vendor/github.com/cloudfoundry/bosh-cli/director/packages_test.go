package director_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "github.com/cloudfoundry/bosh-cli/director"
)

var _ = Describe("Director", func() {
	var (
		director Director
		server   *ghttp.Server
	)

	BeforeEach(func() {
		director, server = BuildServer()
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("MatchPackages", func() {
		Context("when checking for non-compiled packages", func() {
			act := func() ([]string, error) {
				return director.MatchPackages(map[string]bool{"manifest": true}, false)
			}

			It("returns fingerprint matches", func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/packages/matches"),
						ghttp.VerifyBasicAuth("username", "password"),
						ghttp.VerifyHeader(http.Header{
							"Content-Type": []string{"text/yaml"},
						}),
						ghttp.VerifyBody([]byte("manifest: true\n")),
						ghttp.RespondWith(http.StatusOK, `["match1","match2"]`),
					),
				)

				matches, err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(matches).To(Equal([]string{"match1", "match2"}))
			})

			It("does not return error if response is non-200", func() {
				AppendBadRequest(ghttp.VerifyRequest("POST", "/packages/matches"), server)

				matches, err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(matches).To(BeEmpty())
			})

			It("does not return error if response cannot be unmarshalled", func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/packages/matches"),
						ghttp.RespondWith(http.StatusOK, ``),
					),
				)

				matches, err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(matches).To(BeEmpty())
			})
		})

		Context("when checking for compiled packages", func() {
			act := func() ([]string, error) {
				return director.MatchPackages(map[string]bool{"manifest": true}, true)
			}

			It("returns fingerprint matches", func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/packages/matches_compiled"),
						ghttp.VerifyBasicAuth("username", "password"),
						ghttp.VerifyHeader(http.Header{
							"Content-Type": []string{"text/yaml"},
						}),
						ghttp.VerifyBody([]byte("manifest: true\n")),
						ghttp.RespondWith(http.StatusOK, `["match1","match2"]`),
					),
				)

				matches, err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(matches).To(Equal([]string{"match1", "match2"}))
			})

			It("does not return error if response is non-200", func() {
				AppendBadRequest(ghttp.VerifyRequest("POST", "/packages/matches_compiled"), server)

				matches, err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(matches).To(BeEmpty())
			})

			It("does not return error if response cannot be unmarshalled", func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/packages/matches_compiled"),
						ghttp.RespondWith(http.StatusOK, ``),
					),
				)

				matches, err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(matches).To(BeEmpty())
			})
		})
	})
})
