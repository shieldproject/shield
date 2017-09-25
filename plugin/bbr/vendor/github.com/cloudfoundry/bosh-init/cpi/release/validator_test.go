package release_test

import (
	. "github.com/cloudfoundry/bosh-init/cpi/release"
	birel "github.com/cloudfoundry/bosh-init/release"
	bireljob "github.com/cloudfoundry/bosh-init/release/job"
	birelpkg "github.com/cloudfoundry/bosh-init/release/pkg"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validator", func() {
	var (
		fakeFs *fakesys.FakeFileSystem

		cpiReleaseJobName = "fake-cpi-release-job-name"
	)

	BeforeEach(func() {
		fakeFs = fakesys.NewFakeFileSystem()
	})

	It("validates a valid release without error", func() {
		release := birel.NewRelease(
			"fake-release-name",
			"fake-release-version",
			[]bireljob.Job{
				{
					Name:        "fake-cpi-release-job-name",
					Fingerprint: "fake-job-1-fingerprint",
					SHA1:        "fake-job-1-sha",
					Templates: map[string]string{
						"cpi.erb": "bin/cpi",
					},
				},
			},
			[]*birelpkg.Package{},
			"/some/release/path",
			fakeFs,
			false,
		)
		validator := NewValidator()

		err := validator.Validate(release, cpiReleaseJobName)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("when the cpi job is not present", func() {
		var validator Validator
		var release birel.Release

		BeforeEach(func() {
			release = birel.NewRelease(
				"fake-release-name",
				"fake-release-version",
				[]bireljob.Job{
					{
						Name:        "non-cpi-job",
						Fingerprint: "fake-job-1-fingerprint",
						SHA1:        "fake-job-1-sha",
						Templates: map[string]string{
							"cpi.erb": "bin/cpi",
						},
					},
				},
				[]*birelpkg.Package{},
				"/some/release/path",
				fakeFs,
				false,
			)
			validator = NewValidator()
		})

		It("returns an error that the cpi job is not present", func() {
			err := validator.Validate(release, cpiReleaseJobName)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("CPI release must contain specified job 'fake-cpi-release-job-name'"))
		})
	})

	Context("when the templates are missing a bin/cpi target", func() {
		var validator Validator
		var release birel.Release

		BeforeEach(func() {
			release = birel.NewRelease(
				"fake-release-name",
				"fake-release-version",
				[]bireljob.Job{
					{
						Name:        "fake-cpi-release-job-name",
						Fingerprint: "fake-job-1-fingerprint",
						SHA1:        "fake-job-1-sha",
						Templates: map[string]string{
							"cpi.erb": "nonsense",
						},
					},
				},
				[]*birelpkg.Package{},
				"/some/release/path",
				fakeFs,
				false,
			)
			validator = NewValidator()
		})

		It("returns an error that the bin/cpi template target is missing", func() {
			err := validator.Validate(release, cpiReleaseJobName)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Specified CPI release job 'fake-cpi-release-job-name' must contain a template that renders to target 'bin/cpi'"))
		})
	})
})
