package manifest_test

import (
	bireljob "github.com/cloudfoundry/bosh-init/release/job"
	birelmanifest "github.com/cloudfoundry/bosh-init/release/manifest"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	fakebirel "github.com/cloudfoundry/bosh-init/release/fakes"

	. "github.com/cloudfoundry/bosh-init/release/set/manifest"
)

var _ = Describe("Validator", func() {
	var (
		logger    boshlog.Logger
		validator Validator

		validManifest Manifest
		fakeRelease   *fakebirel.FakeRelease
	)

	BeforeEach(func() {
		logger = boshlog.NewLogger(boshlog.LevelNone)

		validManifest = Manifest{
			Releases: []birelmanifest.ReleaseRef{
				{
					Name: "fake-release-name",
					URL:  "file://fake-release-path",
				},
			},
		}

		fakeRelease = fakebirel.New("fake-release-name", "1.0")
		fakeRelease.ReleaseJobs = []bireljob.Job{{Name: "fake-job-name"}}
	})

	JustBeforeEach(func() {
		validator = NewValidator(logger)
	})

	Describe("Validate", func() {
		It("does not error if deployment is valid", func() {
			manifest := validManifest

			err := validator.Validate(manifest)
			Expect(err).ToNot(HaveOccurred())
		})

		It("validates there is at least one release", func() {
			manifest := Manifest{
				Releases: []birelmanifest.ReleaseRef{},
			}

			err := validator.Validate(manifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("releases must contain at least 1 release"))
		})

		It("validates releases have names", func() {
			manifest := Manifest{
				Releases: []birelmanifest.ReleaseRef{{}},
			}

			err := validator.Validate(manifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("releases[0].name must be provided"))
		})

		It("validates releases have urls", func() {
			manifest := Manifest{
				Releases: []birelmanifest.ReleaseRef{
					{Name: "fake-release-name"},
				},
			}

			err := validator.Validate(manifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("releases[0].url must be provided"))
		})

		It("accepts file://, http://, https:// as valid URLs", func() {
			manifest := Manifest{
				Releases: []birelmanifest.ReleaseRef{
					{Name: "fake-release-name-1", URL: "file://fake-file"},
					{Name: "fake-release-name-2", URL: "http://fake-http", SHA1: "fake-sha1"},
					{Name: "fake-release-name-3", URL: "https://fake-https", SHA1: "fake-sha2"},
				},
			}

			err := validator.Validate(manifest)
			Expect(err).ToNot(HaveOccurred())
		})

		It("validates releases with http urls have sha1", func() {
			manifest := Manifest{
				Releases: []birelmanifest.ReleaseRef{
					{Name: "fake-release-name", URL: "http://fake-url"},
				},
			}

			err := validator.Validate(manifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("releases[0].sha1 must be provided for http URL"))
		})

		It("validates releases have valid urls", func() {
			manifest := Manifest{
				Releases: []birelmanifest.ReleaseRef{
					{Name: "fake-release-name", URL: "invalid-url"},
				},
			}

			err := validator.Validate(manifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("releases[0].url must be a valid URL (file:// or http(s)://)"))
		})

		It("validates releases are unique", func() {
			manifest := Manifest{
				Releases: []birelmanifest.ReleaseRef{
					{Name: "fake-release-name"},
					{Name: "fake-release-name"},
				},
			}

			err := validator.Validate(manifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("releases[1].name 'fake-release-name' must be unique"))
		})
	})
})
