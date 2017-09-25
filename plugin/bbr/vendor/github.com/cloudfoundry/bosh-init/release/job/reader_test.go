package job_test

import (
	. "github.com/cloudfoundry/bosh-init/release/job"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	fakecmd "github.com/cloudfoundry/bosh-utils/fileutil/fakes"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("Reader", func() {
	var (
		compressor *fakecmd.FakeCompressor
		fakeFs     *fakesys.FakeFileSystem
		reader     Reader
	)
	BeforeEach(func() {
		compressor = fakecmd.NewFakeCompressor()
		fakeFs = fakesys.NewFakeFileSystem()
		reader = NewReader("/some/job/archive", "/extracted/job", compressor, fakeFs)
	})

	Context("when the job archive is a valid tar", func() {
		Context("when the job manifest is valid", func() {
			BeforeEach(func() {
				fakeFs.WriteFileString(
					"/extracted/job/job.MF",
					`---
name: fake-job
templates:
  some_template: some_file
packages:
- fake-package
properties:
  fake-property:
    description: "Fake description"
    default: "fake-default"
`,
				)
			})

			It("returns a job with the details from the manifest", func() {
				job, err := reader.Read()
				Expect(err).NotTo(HaveOccurred())
				Expect(job).To(Equal(
					Job{
						Name:          "fake-job",
						Templates:     map[string]string{"some_template": "some_file"},
						PackageNames:  []string{"fake-package"},
						ExtractedPath: "/extracted/job",
						Properties: map[string]PropertyDefinition{
							"fake-property": PropertyDefinition{
								Description: "Fake description",
								Default:     biproperty.Property("fake-default"),
							},
						},
					},
				))
			})
		})

		Context("when the job manifest is invalid", func() {
			It("returns an error when the job manifest is missing", func() {
				_, err := reader.Read()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Reading job manifest"))
			})

			It("returns an error when the job manifest is invalid", func() {
				fakeFs.WriteFileString("/extracted/job/job.MF", "{")
				_, err := reader.Read()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Parsing job manifest"))
			})
		})
	})

	Context("when the job archive is not a valid tar", func() {
		BeforeEach(func() {
			compressor.DecompressFileToDirErr = bosherr.Error("fake-error")
		})

		It("returns error", func() {
			_, err := reader.Read()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Extracting job archive '/some/job/archive'"))
		})
	})
})
