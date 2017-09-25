package director_test

import (
	"errors"
	"net/http"
	"os"

	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	semver "github.com/cppforlife/go-semi-semantic/version"
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

	Describe("Stemcells", func() {
		It("returns stemcells", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/stemcells"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[
  {
    "name": "stem1-name",
    "version": "stem1-ver",
    "operating_system": "stem1-os",
    "cid": "stem1-cid",
    "deployments": [ "stem1-dep1", "stem1-dep2" ]
  },
  {
    "name": "stem2-name",
    "version": "stem2-ver",
    "operating_system": "stem2-os",
    "cid": "stem2-cid",
    "deployments": []
  }
]`),
				),
			)

			stems, err := director.Stemcells()
			Expect(err).ToNot(HaveOccurred())
			Expect(stems).To(HaveLen(2))

			Expect(stems[0].Name()).To(Equal("stem1-name"))
			Expect(stems[0].Version()).To(Equal(semver.MustNewVersionFromString("stem1-ver")))
			Expect(stems[0].OSName()).To(Equal("stem1-os"))
			Expect(stems[0].CID()).To(Equal("stem1-cid"))
			Expect(stems[0].VersionMark("*")).To(Equal("*"))

			Expect(stems[1].Name()).To(Equal("stem2-name"))
			Expect(stems[1].Version()).To(Equal(semver.MustNewVersionFromString("stem2-ver")))
			Expect(stems[1].OSName()).To(Equal("stem2-os"))
			Expect(stems[1].CID()).To(Equal("stem2-cid"))
			Expect(stems[1].VersionMark("*")).To(Equal(""))
		})

		It("returns an error for invalid stemcell versions", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/stemcells"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[{"name":"name","version":"-"}]`),
				),
			)

			_, err := director.Stemcells()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Parsing version for stemcell"))
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("GET", "/stemcells"), server)

			_, err := director.Stemcells()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Finding stemcells: Director responded with non-successful status code"))
		})

		It("returns error if response cannot be unmarshalled", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/stemcells"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)

			_, err := director.Stemcells()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Finding stemcells: Unmarshaling Director response"))
		})
	})

	Describe("FindStemcell", func() {
		It("does not return an error", func() {
			_, err := director.FindStemcell(NewStemcellSlug("name", "ver"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if version is invalid", func() {
			_, err := director.FindStemcell(NewStemcellSlug("name", "-"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Parsing version for stemcell"))
		})
	})

	Describe("HasStemcell", func() {
		act := func() (bool, error) { return director.HasStemcell("name", "ver") }

		It("returns true if name and version matches", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/stemcells"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[{"name":"name","version": "ver"}]`),
				),
			)

			found, err := act()
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())
		})

		It("returns false if name and version does not match", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/stemcells"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[
  {"name": "name", "version": "other-ver"},
  {"name": "other-name", "version": "ver"}
]`),
				),
			)

			found, err := act()
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeFalse())
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("GET", "/stemcells"), server)

			_, err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Finding stemcells: Director responded with non-successful status code"))
		})

		It("returns error if response cannot be unmarshalled", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/stemcells"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)

			_, err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Finding stemcells: Unmarshaling Director response"))
		})
	})

	Describe("UploadStemcellURL", func() {
		It("uploads stemcell by URL", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/stemcells", ""),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"application/json"},
					}),
					ghttp.VerifyBody([]byte(`{"location":"url"}`)),
				),
				"",
				server,
			)

			Expect(director.UploadStemcellURL("url", "", false)).ToNot(HaveOccurred())
		})

		It("uploads stemcell by URL with sha1 and fix", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/stemcells", "fix=true"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"application/json"},
					}),
					ghttp.VerifyBody([]byte(`{"location":"url","sha1":"sha1"}`)),
				),
				"",
				server,
			)

			Expect(director.UploadStemcellURL("url", "sha1", true)).ToNot(HaveOccurred())
		})

		It("returns error if URL is empty", func() {
			err := director.UploadStemcellURL("", "", false)
			Expect(err).To(Equal(errors.New("Expected non-empty URL")))
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("POST", "/stemcells"), server)

			err := director.UploadStemcellURL("url", "", false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Uploading remote stemcell 'url': Director responded with non-successful status code"))
		})

		It("returns error if response cannot be unmarshalled", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/stemcells"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)

			err := director.UploadStemcellURL("url", "", false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Uploading remote stemcell 'url': Unmarshaling Director response"))
		})
	})

	Describe("UploadStemcellFile", func() {
		var (
			file UploadFile
		)

		BeforeEach(func() {
			fs := fakesys.NewFakeFileSystem()
			fs.WriteFileString("/file", "content")

			var err error

			file, err = fs.OpenFile("/file", os.O_RDONLY, 0)
			Expect(err).ToNot(HaveOccurred())
		})

		It("uploads stemcell file", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/stemcells", ""),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type":   []string{"application/x-compressed"},
						"Content-Length": []string{"7"},
					}),
					ghttp.VerifyBody([]byte("content")),
				),
				"",
				server,
			)

			Expect(director.UploadStemcellFile(file, false)).ToNot(HaveOccurred())
		})

		It("uploads stemcell file with fix", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/stemcells", "fix=true"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type":   []string{"application/x-compressed"},
						"Content-Length": []string{"7"},
					}),
					ghttp.VerifyBody([]byte("content")),
				),
				"",
				server,
			)

			Expect(director.UploadStemcellFile(file, true)).ToNot(HaveOccurred())
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("POST", "/stemcells"), server)

			err := director.UploadStemcellFile(file, true)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Uploading stemcell file: Director responded with non-successful status code"))
		})

		It("returns error if response cannot be unmarshalled", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/stemcells"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)

			err := director.UploadStemcellFile(file, true)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Uploading stemcell file: Unmarshaling Director response"))
		})
	})
})

var _ = Describe("Stemcell", func() {
	var (
		director Director
		stemcell Stemcell
		server   *ghttp.Server
	)

	BeforeEach(func() {
		director, server = BuildServer()

		var err error

		stemcell, err = director.FindStemcell(NewStemcellSlug("name", "ver"))
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("Name", func() {
		It("returns name", func() {
			Expect(stemcell.Name()).To(Equal("name"))
		})
	})

	Describe("Delete", func() {
		It("succeeds deleting", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("DELETE", "/stemcells/name/ver", ""),
					ghttp.VerifyBasicAuth("username", "password"),
				),
				"",
				server,
			)

			Expect(stemcell.Delete(false)).ToNot(HaveOccurred())
		})

		It("succeeds deleting with force flag", func() {
			ConfigureTaskResult(ghttp.VerifyRequest("DELETE", "/stemcells/name/ver", "force=true"), "", server)

			Expect(stemcell.Delete(true)).ToNot(HaveOccurred())
		})

		It("succeeds even if error occurrs if stemcell no longer exist", func() {
			AppendBadRequest(ghttp.VerifyRequest("DELETE", "/stemcells/name/ver"), server)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/stemcells"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, "[]"),
				),
			)

			Expect(stemcell.Delete(false)).ToNot(HaveOccurred())
		})

		It("returns delete error if listing stemcells fails", func() {
			AppendBadRequest(ghttp.VerifyRequest("DELETE", "/stemcells/name/ver"), server)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/stemcells"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)

			err := stemcell.Delete(false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Deleting stemcell 'name/ver': Director responded with non-successful status code"))
		})

		It("returns delete error if response is non-200 and stemcell still exists", func() {
			AppendBadRequest(ghttp.VerifyRequest("DELETE", "/stemcells/name/ver"), server)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/stemcells"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[{"name": "name", "version": "ver"}]`),
				),
			)

			err := stemcell.Delete(false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Deleting stemcell 'name/ver': Director responded with non-successful status code"))
		})
	})
})
