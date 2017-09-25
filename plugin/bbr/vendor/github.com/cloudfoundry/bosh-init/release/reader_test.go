package release_test

import (
	"errors"

	. "github.com/cloudfoundry/bosh-init/release"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	bireljob "github.com/cloudfoundry/bosh-init/release/job"
	birelpkg "github.com/cloudfoundry/bosh-init/release/pkg"
	fakecmd "github.com/cloudfoundry/bosh-utils/fileutil/fakes"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("tarReader", func() {
	var (
		reader     Reader
		fakeFs     *fakesys.FakeFileSystem
		compressor *fakecmd.FakeCompressor
	)

	BeforeEach(func() {
		fakeFs = fakesys.NewFakeFileSystem()
		compressor = fakecmd.NewFakeCompressor()
		reader = NewReader("/some/release.tgz", "/extracted/release", fakeFs, compressor)
	})

	Describe("Read", func() {
		Context("when the given release archive is a valid tar", func() {
			Context("when the release manifest is valid", func() {
				BeforeEach(func() {
					fakeFs.WriteFileString(
						"/extracted/release/release.MF",
						`---
name: fake-release
version: fake-version

commit_hash: abc123
uncommitted_changes: true

jobs:
- name: fake-job
  version: fake-job-version
  fingerprint: fake-job-fingerprint
  sha1: fake-job-sha

packages:
- name: fake-package
  version: fake-package-version
  fingerprint: fake-package-fingerprint
  sha1: fake-package-sha
  dependencies:
  - fake-package-1
`,
					)
				})

				Context("when the jobs and packages in the release are valid", func() {
					BeforeEach(func() {
						fakeFs.WriteFileString(
							"/extracted/release/extracted_jobs/fake-job/job.MF",
							`---
name: fake-job
templates:
  some_template: some_file
packages:
- fake-package
`,
						)
					})

					Context("when the packages in the release are valid", func() {
						It("returns a release from the given tar file", func() {
							release, err := reader.Read()
							Expect(err).NotTo(HaveOccurred())

							expectedPackage := &birelpkg.Package{
								Name:          "fake-package",
								Fingerprint:   "fake-package-fingerprint",
								SHA1:          "fake-package-sha",
								Dependencies:  []*birelpkg.Package{&birelpkg.Package{Name: "fake-package-1"}},
								ExtractedPath: "/extracted/release/extracted_packages/fake-package",
								ArchivePath:   "/extracted/release/packages/fake-package.tgz",
							}
							Expect(release.Name()).To(Equal("fake-release"))
							Expect(release.Version()).To(Equal("fake-version"))
							Expect(release.Jobs()).To(Equal([]bireljob.Job{
								{
									Name:          "fake-job",
									Fingerprint:   "fake-job-fingerprint",
									SHA1:          "fake-job-sha",
									ExtractedPath: "/extracted/release/extracted_jobs/fake-job",
									Templates:     map[string]string{"some_template": "some_file"},
									PackageNames:  []string{"fake-package"},
									Packages:      []*birelpkg.Package{expectedPackage},
									Properties:    map[string]bireljob.PropertyDefinition{},
								},
							}))
							Expect(release.Packages()).To(Equal([]*birelpkg.Package{expectedPackage}))
						})
					})

					Context("when the package cannot be extracted", func() {
						BeforeEach(func() {
							compressor.DecompressFileToDirErr = errors.New("Extracting package 'fake-package'")
						})

						It("returns errors for each invalid package", func() {
							_, err := reader.Read()
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(ContainSubstring("Extracting package 'fake-package'"))
						})
					})
				})

				Context("when the jobs in the release are not valid", func() {
					BeforeEach(func() {
						fakeFs.WriteFileString(
							"/extracted/release/release.MF",
							`---
name: fake-release
version: fake-version

jobs:
- name: fake-job
  version: fake-job-version
  fingerprint: fake-job-fingerprint
  sha1: fake-job-sha
- name: fake-job-2
  version: fake-job-2-version
  fingerprint: fake-job-2-fingerprint
  sha1: fake-job-2-sha

packages:
- name: fake-package
  version: fake-package-version
  fingerprint: fake-package-fingerprint
  sha1: fake-package-sha
  dependencies:
  - fake-package-1
`,
						)
					})

					It("returns errors for each invalid job", func() {
						_, err := reader.Read()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("Reading job 'fake-job' from archive"))
						Expect(err.Error()).To(ContainSubstring("Reading job 'fake-job-2' from archive"))
					})
				})

				Context("when an extracted job path cannot be created", func() {
					BeforeEach(func() {
						fakeFs.MkdirAllError = errors.New("")
					})

					It("returns err", func() {
						_, err := reader.Read()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("Creating extracted job path"))
					})
				})
			})

			Context("when the CPI release manifest is invalid", func() {
				BeforeEach(func() {
					fakeFs.WriteFileString("/extracted/release/release.MF", "{")
				})

				It("returns an error when the YAML in unparseable", func() {
					_, err := reader.Read()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Parsing release manifest"))
				})

				It("returns an error when the release manifest is missing", func() {
					fakeFs.RemoveAll("/extracted/release/release.MF")
					_, err := reader.Read()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Reading release manifest"))
				})
			})

			Context("when the job refers to a package that does not exist", func() {
				It("returns error", func() {
					releaseMFContents :=
						`---
name: fake-release
version: fake-version

commit_hash: abc123
uncommitted_changes: true

jobs:
- name: fake-job
version: fake-job-version
fingerprint: fake-job-fingerprint
sha1: fake-job-sha
`
					fakeFs.WriteFileString("/extracted/release/release.MF", releaseMFContents)
					jobMFContents :=
						`---
name: fake-job
templates:
  some_template: some_file
packages:
- not_there
`
					fakeFs.WriteFileString("/extracted/release/extracted_jobs/fake-job/job.MF", jobMFContents)
					_, err := reader.Read()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Package not found"))
				})
			})
		})

		Context("when the given compiled release archive is a valid tar", func() {
			Context("when the compiled release manifest is valid", func() {
				BeforeEach(func() {
					fakeFs.WriteFileString(
						"/extracted/release/release.MF",
						`---
name: fake-release
version: fake-version

commit_hash: abc123
uncommitted_changes: true

jobs:
- name: fake-job
  version: fake-job-version
  fingerprint: fake-job-fingerprint
  sha1: fake-job-sha

compiled_packages:
- name: fake-package
  version: fake-package-version
  fingerprint: fake-package-fingerprint
  sha1: fake-package-sha
  stemcell: centos/8547
  dependencies:
  - fake-package-1
`,
					)
				})

				Context("when the jobs and packages in the release are valid", func() {
					BeforeEach(func() {
						fakeFs.WriteFileString(
							"/extracted/release/extracted_jobs/fake-job/job.MF",
							`---
name: fake-job
templates:
  some_template: some_file
packages:
- fake-package
`,
						)
					})

					Context("when the compiled packages in the release are valid", func() {
						It("returns a release from the given tar file", func() {
							release, err := reader.Read()
							Expect(err).NotTo(HaveOccurred())

							expectedPackage := &birelpkg.Package{
								Name:          "fake-package",
								Fingerprint:   "fake-package-fingerprint",
								SHA1:          "fake-package-sha",
								Stemcell:      "centos/8547",
								Dependencies:  []*birelpkg.Package{&birelpkg.Package{Name: "fake-package-1"}},
								ExtractedPath: "/extracted/release/extracted_packages/fake-package",
								ArchivePath:   "/extracted/release/compiled_packages/fake-package.tgz",
							}
							Expect(release.Name()).To(Equal("fake-release"))
							Expect(release.Version()).To(Equal("fake-version"))
							Expect(release.Jobs()).To(Equal([]bireljob.Job{
								{
									Name:          "fake-job",
									Fingerprint:   "fake-job-fingerprint",
									SHA1:          "fake-job-sha",
									ExtractedPath: "/extracted/release/extracted_jobs/fake-job",
									Templates:     map[string]string{"some_template": "some_file"},
									PackageNames:  []string{"fake-package"},
									Packages:      []*birelpkg.Package{expectedPackage},
									Properties:    map[string]bireljob.PropertyDefinition{},
								},
							}))
							Expect(release.Packages()).To(Equal([]*birelpkg.Package{expectedPackage}))
						})
					})

					Context("when the package cannot be extracted", func() {
						BeforeEach(func() {
							compressor.DecompressFileToDirErr = errors.New("Extracting package 'fake-package'")
						})

						It("returns errors for each invalid package", func() {
							_, err := reader.Read()
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(ContainSubstring("Extracting package 'fake-package'"))
						})
					})
				})

				Context("when the jobs in the release are not valid", func() {
					BeforeEach(func() {
						fakeFs.WriteFileString(
							"/extracted/release/release.MF",
							`---
name: fake-release
version: fake-version

jobs:
- name: fake-job
  version: fake-job-version
  fingerprint: fake-job-fingerprint
  sha1: fake-job-sha
- name: fake-job-2
  version: fake-job-2-version
  fingerprint: fake-job-2-fingerprint
  sha1: fake-job-2-sha

compiled_packages:
- name: fake-package
  version: fake-package-version
  fingerprint: fake-package-fingerprint
  sha1: fake-package-sha
  dependencies:
  - fake-package-1
`,
						)
					})

					It("returns errors for each invalid job", func() {
						_, err := reader.Read()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("Reading job 'fake-job' from archive"))
						Expect(err.Error()).To(ContainSubstring("Reading job 'fake-job-2' from archive"))
					})
				})

				Context("when an extracted job path cannot be created", func() {
					BeforeEach(func() {
						fakeFs.MkdirAllError = errors.New("")
					})

					It("returns err", func() {
						_, err := reader.Read()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("Creating extracted job path"))
					})
				})
			})

			Context("when the compiled release manifest is invalid", func() {
				BeforeEach(func() {
					fakeFs.WriteFileString(
						"/extracted/release/release.MF",
						`---
name: fake-release
version: fake-version

commit_hash: abc123
uncommitted_changes: true

jobs:
- name: fake-job
  version: fake-job-version
  fingerprint: fake-job-fingerprint
  sha1: fake-job-sha

compiled_packages:
- name: fake-compiled-package
  version: fake-compiled-package-version
  fingerprint: fake-compiled-package-fingerprint
  sha1: fake-compiled-package-sha
  stemcell: centos/8547
  dependencies:
  - fake-package-1
packages:
- name: fake-package
  version: fake-package-version
  fingerprint: fake-package-fingerprint
  sha1: fake-package-sha
  dependencies:
  - fake-package-1
`,
					)
				})

				It("returns an error when release contains compiled and non compiled packages", func() {
					_, err := reader.Read()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Release 'fake-release' contains compiled and non-compiled pacakges"))
				})

			})
		})

		Context("when the CPI release is not a valid tar", func() {
			BeforeEach(func() {
				compressor.DecompressFileToDirErr = errors.New("fake-error")
			})

			It("returns err", func() {
				_, err := reader.Read()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Extracting release"))
			})
		})
	})
})
