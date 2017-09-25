package manifest_test

import (
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshjob "github.com/cloudfoundry/bosh-cli/release/job"
	boshman "github.com/cloudfoundry/bosh-cli/release/manifest"
	fakerel "github.com/cloudfoundry/bosh-cli/release/releasefakes"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
	. "github.com/cloudfoundry/bosh-cli/release/set/manifest"
)

var _ = Describe("Validator", func() {
	var (
		release   *fakerel.FakeRelease
		validator Validator
	)

	BeforeEach(func() {
		release = &fakerel.FakeRelease{}
		release.NameReturns("fake-release-name")
		release.VersionReturns("1.0")
		release.JobsReturns([]*boshjob.Job{
			boshjob.NewJob(NewResource("fake-job-name", "", nil)),
		})
	})

	JustBeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		validator = NewValidator(logger)
	})

	Describe("Validate", func() {
		It("does not error if deployment is valid", func() {
			manifest := Manifest{
				Releases: []boshman.ReleaseRef{
					{
						Name: "fake-release-name",
						URL:  "file://fake-release-path",
					},
				},
			}

			err := validator.Validate(manifest)
			Expect(err).ToNot(HaveOccurred())
		})

		It("validates there is at least one release", func() {
			manifest := Manifest{
				Releases: []boshman.ReleaseRef{},
			}

			err := validator.Validate(manifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("releases must contain at least 1 release"))
		})

		It("validates releases have names", func() {
			manifest := Manifest{
				Releases: []boshman.ReleaseRef{{}},
			}

			err := validator.Validate(manifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("releases[0].name must be provided"))
		})

		It("validates releases have urls", func() {
			manifest := Manifest{
				Releases: []boshman.ReleaseRef{
					{Name: "fake-release-name"},
				},
			}

			err := validator.Validate(manifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("releases[0].url must be provided"))
		})

		It("accepts file://, http://, https:// as valid URLs", func() {
			manifest := Manifest{
				Releases: []boshman.ReleaseRef{
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
				Releases: []boshman.ReleaseRef{
					{Name: "fake-release-name", URL: "http://fake-url"},
				},
			}

			err := validator.Validate(manifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("releases[0].sha1 must be provided for http URL"))
		})

		It("validates releases have valid urls", func() {
			manifest := Manifest{
				Releases: []boshman.ReleaseRef{
					{Name: "fake-release-name", URL: "invalid-url"},
				},
			}

			err := validator.Validate(manifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("releases[0].url must be a valid URL (file:// or http(s)://)"))
		})

		It("validates releases are unique", func() {
			manifest := Manifest{
				Releases: []boshman.ReleaseRef{
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
