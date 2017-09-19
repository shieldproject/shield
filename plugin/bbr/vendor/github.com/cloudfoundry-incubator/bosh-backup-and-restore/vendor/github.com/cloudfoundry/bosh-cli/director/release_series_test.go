package director_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "github.com/cloudfoundry/bosh-cli/director"
)

var _ = Describe("Director", func() {
	Describe("FindReleaseSeries", func() {
		It("does not return an error", func() {
			director, server := BuildServer()
			defer server.Close()

			_, err := director.FindReleaseSeries(NewReleaseSeriesSlug("name"))
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

var _ = Describe("ReleaseSeries", func() {
	var (
		director Director
		series   ReleaseSeries
		server   *ghttp.Server
	)

	BeforeEach(func() {
		director, server = BuildServer()

		var err error

		series, err = director.FindReleaseSeries(NewReleaseSeriesSlug("name"))
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("Name", func() {
		It("returns name", func() {
			Expect(series.Name()).To(Equal("name"))
		})
	})

	Describe("Delete", func() {
		It("succeeds deleting", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("DELETE", "/releases/name", ""),
					ghttp.VerifyBasicAuth("username", "password"),
				),
				"",
				server,
			)

			Expect(series.Delete(false)).ToNot(HaveOccurred())
		})

		It("succeeds deleting with force flag", func() {
			ConfigureTaskResult(ghttp.VerifyRequest("DELETE", "/releases/name", "force=true"), "", server)

			Expect(series.Delete(true)).ToNot(HaveOccurred())
		})

		It("succeeds even if error occurrs if release series no longer exist", func() {
			AppendBadRequest(ghttp.VerifyRequest("DELETE", "/releases/name"), server)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/releases"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, "[]"),
				),
			)

			Expect(series.Delete(false)).ToNot(HaveOccurred())
		})

		It("returns delete error if listing releases fails", func() {
			AppendBadRequest(ghttp.VerifyRequest("DELETE", "/releases/name"), server)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/releases"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)

			err := series.Delete(false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Deleting release or series 'name[/]': Director responded with non-successful status code"))
		})

		It("returns delete error if response is non-200 and release still exists", func() {
			AppendBadRequest(ghttp.VerifyRequest("DELETE", "/releases/name"), server)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/releases"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[{"name": "name"}]`),
				),
			)

			err := series.Delete(false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Deleting release or series 'name[/]': Director responded with non-successful status code"))
		})
	})
})
