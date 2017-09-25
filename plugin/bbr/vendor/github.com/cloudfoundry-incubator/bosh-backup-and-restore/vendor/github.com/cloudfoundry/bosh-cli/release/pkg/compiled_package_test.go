package pkg_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/release/pkg"

	"errors"
	"github.com/cloudfoundry/bosh-cli/crypto/fakes"
	"github.com/cloudfoundry/bosh-utils/crypto/cryptofakes"
	fakes2 "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("NewCompiledPackageWithoutArchive", func() {
	var (
		compiledPkg *CompiledPackage
	)

	BeforeEach(func() {
		compiledPkg = NewCompiledPackageWithoutArchive(
			"name", "fp", "os-slug", "sha1", []string{"pkg1", "pkg2"})
	})

	Describe("common methods", func() {
		It("returns values", func() {
			Expect(compiledPkg.Name()).To(Equal("name"))
			Expect(compiledPkg.Fingerprint()).To(Equal("fp"))
			Expect(compiledPkg.OSVersionSlug()).To(Equal("os-slug"))

			Expect(func() { compiledPkg.ArchivePath() }).To(Panic())
			Expect(compiledPkg.ArchiveSHA1()).To(Equal("sha1"))

			Expect(compiledPkg.DependencyNames()).To(Equal([]string{"pkg1", "pkg2"}))
		})
	})

	Describe("AttachDependencies", func() {
		It("attaches dependencies based on their names", func() {
			pkg1 := NewCompiledPackageWithoutArchive("pkg1", "fp", "os-slug", "sha1", nil)
			pkg2 := NewCompiledPackageWithoutArchive("pkg2", "fp", "os-slug", "sha1", nil)
			unusedPkg := NewCompiledPackageWithoutArchive("unused", "fp", "os-slug", "sha1", nil)

			err := compiledPkg.AttachDependencies([]*CompiledPackage{pkg1, unusedPkg, pkg2})
			Expect(err).ToNot(HaveOccurred())

			Expect(compiledPkg.Dependencies).To(Equal([]*CompiledPackage{pkg1, pkg2}))
		})

		It("returns error if dependency cannot be found", func() {
			pkg2 := NewCompiledPackageWithoutArchive("pkg2", "fp", "os-slug", "sha1", nil)

			err := compiledPkg.AttachDependencies([]*CompiledPackage{pkg2})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected to find compiled package 'pkg1' since it's a dependency of compiled package 'name'"))
		})
	})
})

var _ = Describe("NewCompiledPackageWithArchive", func() {
	var (
		compiledPkg          *CompiledPackage
		fakeDigestCalculator *fakes.FakeDigestCalculator
		fakeArchiveReader    *cryptofakes.FakeArchiveDigestFilePathReader
		fakeFile             *fakes2.FakeFile
		fakeFileContentSha1  string
	)

	BeforeEach(func() {
		compiledPkg = NewCompiledPackageWithArchive(
			"name", "fp", "os-slug", "path", "sha1", []string{"pkg1", "pkg2"})
	})

	Describe("common methods", func() {
		It("returns values", func() {
			Expect(compiledPkg.Name()).To(Equal("name"))
			Expect(compiledPkg.Fingerprint()).To(Equal("fp"))
			Expect(compiledPkg.OSVersionSlug()).To(Equal("os-slug"))

			Expect(compiledPkg.ArchivePath()).To(Equal("path"))
			Expect(compiledPkg.ArchiveSHA1()).To(Equal("sha1"))

			Expect(compiledPkg.DependencyNames()).To(Equal([]string{"pkg1", "pkg2"}))
		})
	})

	Describe("AttachDependencies", func() {
		It("attaches dependencies based on their names", func() {
			pkg1 := NewCompiledPackageWithArchive("pkg1", "fp", "os-slug", "path", "sha1", nil)
			pkg2 := NewCompiledPackageWithArchive("pkg2", "fp", "os-slug", "path", "sha1", nil)
			unusedPkg := NewCompiledPackageWithArchive("unused", "fp", "os-slug", "path", "sha1", nil)

			err := compiledPkg.AttachDependencies([]*CompiledPackage{pkg1, unusedPkg, pkg2})
			Expect(err).ToNot(HaveOccurred())

			Expect(compiledPkg.Dependencies).To(Equal([]*CompiledPackage{pkg1, pkg2}))
		})

		It("returns error if dependency cannot be found", func() {
			pkg2 := NewCompiledPackageWithArchive("pkg2", "fp", "os-slug", "path", "sha1", nil)

			err := compiledPkg.AttachDependencies([]*CompiledPackage{pkg2})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected to find compiled package 'pkg1' since it's a dependency of compiled package 'name'"))
		})
	})

	Describe("RehashWithCalculator", func() {
		BeforeEach(func() {
			fakeDigestCalculator = fakes.NewFakeDigestCalculator()
			fakeArchiveReader = &cryptofakes.FakeArchiveDigestFilePathReader{}
			fakeFile = &fakes2.FakeFile{Contents: []byte("hello world")}
		})

		Context("When compiled package can be rehashed", func() {
			BeforeEach(func() {
				fakeDigestCalculator.SetCalculateBehavior(map[string]fakes.CalculateInput{
					"path": {DigestStr: "sha256:compiledpkgsha256"},
				})

				fakeFileContentSha1 = "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed"

				fakeArchiveReader.OpenFileReturns(fakeFile, nil)

				compiledPkg = NewCompiledPackageWithArchive(
					"name", "fp", "os-slug", "path", fakeFileContentSha1, []string{"pkg1", "pkg2"})
			})

			It("returns new compiled package with sha 256 digest", func() {
				newCompiledPkg, err := compiledPkg.RehashWithCalculator(fakeDigestCalculator, fakeArchiveReader)
				Expect(err).ToNot(HaveOccurred())
				Expect(newCompiledPkg.ArchiveSHA1()).To(Equal("sha256:compiledpkgsha256"))
			})
		})

		Context("When archive is invalid", func() {
			BeforeEach(func() {
				fakeArchiveReader.OpenFileReturns(nil, errors.New("fake-file-open-error"))
			})

			Context("When archive cannot be opened", func() {
				It("returns an error opening file", func() {
					_, err := compiledPkg.RehashWithCalculator(fakeDigestCalculator, fakeArchiveReader)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-file-open-error"))
				})
			})

			Context("When package file fails digest verification", func() {
				BeforeEach(func() {
					fakeArchiveReader.OpenFileReturns(fakeFile, nil)
				})

				It("returns an error verifying", func() {
					_, err := compiledPkg.RehashWithCalculator(fakeDigestCalculator, fakeArchiveReader)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Expected stream to have digest 'sha1'"))
				})
			})

			Context("When digest calculator fails to calculate digest", func() {
				BeforeEach(func() {
					compiledPkg = NewCompiledPackageWithArchive(
						"name", "fp", "os-slug", "path", fakeFileContentSha1, []string{"pkg1", "pkg2"})

					fakeArchiveReader.OpenFileReturns(fakeFile, nil)
					fakeDigestCalculator.SetCalculateBehavior(map[string]fakes.CalculateInput{
						"path": {Err: errors.New("fake-digest-calculator-error")},
					})
				})

				It("returns an error calculating the sha 256 digest", func() {
					_, err := compiledPkg.RehashWithCalculator(fakeDigestCalculator, fakeArchiveReader)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-digest-calculator-error"))
				})
			})
		})
	})
})
