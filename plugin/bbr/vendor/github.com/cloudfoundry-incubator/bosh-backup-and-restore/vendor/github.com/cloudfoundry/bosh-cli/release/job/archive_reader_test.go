package job_test

import (
	"errors"
	"os"

	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	fakecmd "github.com/cloudfoundry/bosh-utils/fileutil/fakes"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/release/job"
	boshman "github.com/cloudfoundry/bosh-cli/release/manifest"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
)

var _ = Describe("ArchiveReaderImpl", func() {
	var (
		compressor *fakecmd.FakeCompressor
		fs         *fakesys.FakeFileSystem
		ref        boshman.JobRef
		reader     ArchiveReaderImpl
	)

	BeforeEach(func() {
		ref = boshman.JobRef{
			Name:        "name",
			Fingerprint: "fp",
			SHA1:        "archive-sha1",
		}
		compressor = fakecmd.NewFakeCompressor()
		fs = fakesys.NewFakeFileSystem()
	})

	Context("when planning to extract", func() {
		BeforeEach(func() {
			reader = NewArchiveReaderImpl(true, compressor, fs)
			fs.TempDirDir = "/extracted/job"
		})

		It("returns a job with the details from the manifest", func() {
			fs.WriteFileString("/extracted/job/job.MF", `---
name: name
templates: {src: dst}
packages: [pkg]
properties:
  prop:
    description: prop-desc
    default: prop-default
`)

			job, err := reader.Read(ref, "archive-path")
			Expect(err).NotTo(HaveOccurred())

			Expect(job.Name()).To(Equal("name"))
			Expect(job.Fingerprint()).To(Equal("fp"))
			Expect(job.ArchivePath()).To(Equal("archive-path"))
			Expect(job.ArchiveSHA1()).To(Equal("archive-sha1"))

			Expect(job.Templates).To(Equal(map[string]string{"src": "dst"}))
			Expect(job.PackageNames).To(Equal([]string{"pkg"}))
			Expect(job.Properties).To(Equal(map[string]PropertyDefinition{
				"prop": PropertyDefinition{
					Description: "prop-desc",
					Default:     biproperty.Property("prop-default"),
				},
			}))

			Expect(job.ExtractedPath()).To(Equal("/extracted/job"))

			Expect(compressor.DecompressFileToDirTarballPaths).To(Equal([]string{"archive-path"}))
			Expect(compressor.DecompressFileToDirDirs).To(Equal([]string{"/extracted/job"}))
			Expect(compressor.DecompressFileToDirOptions).To(Equal([]boshcmd.CompressorOptions{{}}))
		})

		It("returns an error when the job manifest is invalid", func() {
			fs.WriteFileString("/extracted/job/job.MF", "-")

			_, err := reader.Read(ref, "archive-path")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unmarshalling job spec"))
		})

		It("returns error when the job archive is not a valid tar", func() {
			compressor.DecompressFileToDirErr = errors.New("fake-err")

			_, err := reader.Read(ref, "archive-path")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("returns a job that can be cleaned up", func() {
			fs.WriteFileString("/extracted/job/job.MF", "")
			fs.MkdirAll("/extracted/job", os.ModeDir)

			job, err := reader.Read(ref, "archive-path")
			Expect(err).NotTo(HaveOccurred())

			Expect(job.CleanUp()).ToNot(HaveOccurred())
			Expect(fs.FileExists("/extracted/job")).To(BeFalse())
		})

		It("returns error when cleaning up fails", func() {
			fs.WriteFileString("/extracted/job/job.MF", "")
			fs.RemoveAllStub = func(_ string) error { return errors.New("fake-err") }

			job, err := reader.Read(ref, "archive-path")
			Expect(err).NotTo(HaveOccurred())

			Expect(job.CleanUp()).To(Equal(errors.New("fake-err")))
		})
	})

	Context("when planning to avoid extraction", func() {
		It("returns a job without details of the manifest", func() {
			reader = NewArchiveReaderImpl(false, compressor, fs)

			job, err := reader.Read(ref, "archive-path")
			Expect(err).ToNot(HaveOccurred())
			Expect(job).To(Equal(NewJob(NewResourceWithBuiltArchive(
				"name", "fp", "archive-path", "archive-sha1"))))
		})
	})
})
