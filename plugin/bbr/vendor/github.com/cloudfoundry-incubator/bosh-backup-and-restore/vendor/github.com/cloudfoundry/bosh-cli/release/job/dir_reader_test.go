package job_test

import (
	"errors"
	"path/filepath"

	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/release/job"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
	fakeres "github.com/cloudfoundry/bosh-cli/release/resource/resourcefakes"
)

var _ = Describe("DirReaderImpl", func() {
	var (
		collectedFiles     []File
		collectedPrepFiles []File
		collectedChunks    []string
		archive            *fakeres.FakeArchive
		fs                 *fakesys.FakeFileSystem
		reader             DirReaderImpl
	)

	BeforeEach(func() {
		archive = &fakeres.FakeArchive{}
		archiveFactory := func(files, prepFiles []File, chunks []string) Archive {
			collectedFiles = files
			collectedPrepFiles = prepFiles
			collectedChunks = chunks
			return archive
		}
		fs = fakesys.NewFakeFileSystem()
		reader = NewDirReaderImpl(archiveFactory, fs)
	})

	Describe("Read", func() {
		It("returns a job with the details collected from job directory", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "spec"), `---
name: name
templates: {src: dst}
packages: [pkg]
properties:
  prop:
    description: prop-desc
    default: prop-default
`)

			fs.WriteFileString(filepath.Join("/", "dir", "monit"), "monit-content")
			fs.WriteFileString(filepath.Join("/", "dir", "templates", "src"), "tpl-content")

			archive.FingerprintReturns("fp", nil)

			expectedJob := NewJob(NewResource("name", "fp", archive))
			expectedJob.PackageNames = []string{"pkg"} // only expect pkg names

			job, err := reader.Read(filepath.Join("/", "dir"))
			Expect(err).NotTo(HaveOccurred())
			Expect(job).To(Equal(expectedJob))

			Expect(collectedFiles).To(Equal([]File{
				File{Path: filepath.Join("/", "dir", "spec"), DirPath: filepath.Join("/", "dir"), RelativePath: "job.MF"},
				File{Path: filepath.Join("/", "dir", "monit"), DirPath: filepath.Join("/", "dir"), RelativePath: "monit"},
				File{Path: filepath.Join("/", "dir", "templates", "src"), DirPath: filepath.Join("/", "dir"), RelativePath: filepath.Join("templates", "src")},
			}))

			Expect(collectedPrepFiles).To(BeEmpty())
			Expect(collectedChunks).To(BeEmpty())
		})

		It("returns a job with the details without monit file", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "spec"), "---\nname: name")

			archive.FingerprintReturns("fp", nil)

			job, err := reader.Read(filepath.Join("/", "dir"))
			Expect(err).NotTo(HaveOccurred())
			Expect(job).To(Equal(NewJob(NewResource("name", "fp", archive))))

			Expect(collectedFiles).To(Equal([]File{
				File{Path: filepath.Join("/", "dir", "spec"), DirPath: filepath.Join("/", "dir"), RelativePath: "job.MF"},
			}))

			Expect(collectedPrepFiles).To(BeEmpty())
			Expect(collectedChunks).To(BeEmpty())
		})

		It("returns error if spec file is not valid", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "spec"), `-`)

			_, err := reader.Read(filepath.Join("/", "dir"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Collecting job files"))
		})

		It("returns error if fingerprinting fails", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "spec"), "")

			archive.FingerprintReturns("", errors.New("fake-err"))

			_, err := reader.Read(filepath.Join("/", "dir"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
