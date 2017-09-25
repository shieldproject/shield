package release_test

import (
	. "github.com/cloudfoundry/bosh-init/release"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	bireljob "github.com/cloudfoundry/bosh-init/release/job"
	birelpkg "github.com/cloudfoundry/bosh-init/release/pkg"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	fakecmd "github.com/cloudfoundry/bosh-utils/fileutil/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"

	fakebirel "github.com/cloudfoundry/bosh-init/release/fakes"
)

var _ = Describe("Extractor", func() {

	var (
		fakeFS               *fakesys.FakeFileSystem
		compressor           *fakecmd.FakeCompressor
		fakeReleaseValidator *fakebirel.FakeValidator
		releaseExtractor     Extractor
	)

	BeforeEach(func() {
		fakeFS = fakesys.NewFakeFileSystem()
		compressor = fakecmd.NewFakeCompressor()
		fakeReleaseValidator = fakebirel.NewFakeValidator()
		logger := boshlog.NewLogger(boshlog.LevelNone)
		releaseExtractor = NewExtractor(fakeFS, compressor, fakeReleaseValidator, logger)
	})

	Describe("Extract", func() {
		var (
			releaseTarballPath string
		)
		BeforeEach(func() {
			releaseTarballPath = "/fake/release.tgz"
			fakeFS.WriteFileString(releaseTarballPath, "fake-tgz-contents")
		})

		Context("when an extracted release directory can be created", func() {
			BeforeEach(func() {
				fakeFS.TempDirDirs = []string{"/extracted-release-path"}
				releaseContents := `---
name: fake-release-name
version: fake-release-version

packages:
- name: fake-release-package-name
  version: fake-release-package-version
  fingerprint: fake-release-package-fingerprint
  sha1: fake-release-package-sha1
  dependencies: []
jobs:
- name: cpi
  version: fake-release-job-version
  fingerprint: fake-release-job-fingerprint
  sha1: fake-release-job-sha1
`
				fakeFS.WriteFileString("/extracted-release-path/release.MF", releaseContents)
				jobManifestContents := `---
name: cpi
templates:
  cpi.erb: bin/cpi
  cpi.yml.erb: config/cpi.yml

packages:
- fake-release-package-name

properties: {}
`
				fakeFS.WriteFileString("/extracted-release-path/extracted_jobs/cpi/job.MF", jobManifestContents)
			})

			Context("and the tarball is a valid BOSH release", func() {
				It("extracts the release to the ExtractedPath", func() {
					release, err := releaseExtractor.Extract(releaseTarballPath)
					Expect(err).NotTo(HaveOccurred())

					expectedPackage := &birelpkg.Package{
						Name:          "fake-release-package-name",
						Fingerprint:   "fake-release-package-fingerprint",
						SHA1:          "fake-release-package-sha1",
						ExtractedPath: "/extracted-release-path/extracted_packages/fake-release-package-name",
						ArchivePath:   "/extracted-release-path/packages/fake-release-package-name.tgz",
						Dependencies:  []*birelpkg.Package{},
					}
					expectedRelease := NewRelease(
						"fake-release-name",
						"fake-release-version",
						[]bireljob.Job{
							{
								Name:          "cpi",
								Fingerprint:   "fake-release-job-fingerprint",
								SHA1:          "fake-release-job-sha1",
								ExtractedPath: "/extracted-release-path/extracted_jobs/cpi",
								Templates: map[string]string{
									"cpi.erb":     "bin/cpi",
									"cpi.yml.erb": "config/cpi.yml",
								},
								PackageNames: []string{
									"fake-release-package-name",
								},
								Packages:   []*birelpkg.Package{expectedPackage},
								Properties: map[string]bireljob.PropertyDefinition{},
							},
						},
						[]*birelpkg.Package{expectedPackage},
						"/extracted-release-path",
						fakeFS,
						false,
					)

					Expect(release).To(Equal(expectedRelease))

					Expect(fakeFS.FileExists("/extracted-release-path")).To(BeTrue())
					Expect(fakeFS.FileExists("/extracted-release-path/extracted_packages/fake-release-package-name")).To(BeTrue())
					Expect(fakeFS.FileExists("/extracted-release-path/extracted_jobs/cpi")).To(BeTrue())
				})
			})

			Context("and the tarball is not a valid BOSH release", func() {
				BeforeEach(func() {
					fakeFS.WriteFileString("/extracted-release-path/release.MF", `{}`)
					fakeReleaseValidator.ValidateError = bosherr.Error("fake-error")
				})

				It("returns an error", func() {
					_, err := releaseExtractor.Extract(releaseTarballPath)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-error"))
				})

				It("deletes the destination file path", func() {
					releaseExtractor.Extract(releaseTarballPath)
					Expect(fakeFS.FileExists("/extracted-release-path")).To(BeFalse())
				})
			})

			Context("and the tarball cannot be read", func() {
				BeforeEach(func() {
					compressor.DecompressFileToDirErr = bosherr.Error("fake-error")
				})

				It("returns an error", func() {
					_, err := releaseExtractor.Extract(releaseTarballPath)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Reading release from '/fake/release.tgz'"))
				})

				It("deletes the destination file path", func() {
					releaseExtractor.Extract(releaseTarballPath)
					Expect(fakeFS.FileExists("/extracted-release-path")).To(BeFalse())
				})
			})
		})

		Context("when an extracted release path cannot be created", func() {
			BeforeEach(func() {
				fakeFS.TempDirError = bosherr.Error("fake-tmp-dir-error")
			})

			It("returns an error", func() {
				_, err := releaseExtractor.Extract(releaseTarballPath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-tmp-dir-error"))
				Expect(err.Error()).To(ContainSubstring("Creating temp directory to extract release '/fake/release.tgz'"))
			})
		})
	})
})
