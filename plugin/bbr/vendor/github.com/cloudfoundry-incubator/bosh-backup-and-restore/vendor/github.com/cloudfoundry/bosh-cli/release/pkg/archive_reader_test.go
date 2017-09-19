package pkg_test

import (
	"errors"
	"os"

	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	fakecmd "github.com/cloudfoundry/bosh-utils/fileutil/fakes"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshman "github.com/cloudfoundry/bosh-cli/release/manifest"
	. "github.com/cloudfoundry/bosh-cli/release/pkg"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
)

var _ = Describe("ArchiveReaderImpl", func() {
	var (
		compressor *fakecmd.FakeCompressor
		fs         *fakesys.FakeFileSystem
		ref        boshman.PackageRef
		reader     ArchiveReaderImpl
	)

	BeforeEach(func() {
		ref = boshman.PackageRef{
			Name:         "name",
			Fingerprint:  "fp",
			SHA1:         "archive-sha1",
			Dependencies: []string{"pkg1"},
		}
		compressor = fakecmd.NewFakeCompressor()
		fs = fakesys.NewFakeFileSystem()
	})

	Context("when planning to extract", func() {
		BeforeEach(func() {
			reader = NewArchiveReaderImpl(true, compressor, fs)
			fs.TempDirDir = "/extracted/pkg"
		})

		It("returns a package", func() {
			pkg, err := reader.Read(ref, "archive-path")
			Expect(err).NotTo(HaveOccurred())

			Expect(pkg.Name()).To(Equal("name"))
			Expect(pkg.Fingerprint()).To(Equal("fp"))
			Expect(pkg.ArchivePath()).To(Equal("archive-path"))
			Expect(pkg.ArchiveSHA1()).To(Equal("archive-sha1"))
			Expect(pkg.DependencyNames()).To(Equal([]string{"pkg1"}))
			Expect(pkg.ExtractedPath()).To(Equal("/extracted/pkg"))

			Expect(compressor.DecompressFileToDirTarballPaths).To(Equal([]string{"archive-path"}))
			Expect(compressor.DecompressFileToDirDirs).To(Equal([]string{"/extracted/pkg"}))
			Expect(compressor.DecompressFileToDirOptions).To(Equal([]boshcmd.CompressorOptions{{}}))
		})

		It("returns error when the package archive is not a valid tar", func() {
			compressor.DecompressFileToDirErr = errors.New("fake-err")

			_, err := reader.Read(ref, "archive-path")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("returns a package that can be cleaned up", func() {
			fs.MkdirAll("/extracted/pkg", os.ModeDir)

			pkg, err := reader.Read(ref, "archive-path")
			Expect(err).NotTo(HaveOccurred())

			Expect(pkg.CleanUp()).ToNot(HaveOccurred())
			Expect(fs.FileExists("/extracted/pkg")).To(BeFalse())
		})

		It("returns error when cleaning up fails", func() {
			fs.RemoveAllStub = func(_ string) error { return errors.New("fake-err") }

			pkg, err := reader.Read(ref, "archive-path")
			Expect(err).NotTo(HaveOccurred())

			Expect(pkg.CleanUp()).To(Equal(errors.New("fake-err")))
		})
	})

	Context("when planning to avoid extraction", func() {
		It("returns a package", func() {
			reader = NewArchiveReaderImpl(false, compressor, fs)

			pkg, err := reader.Read(ref, "archive-path")
			Expect(err).ToNot(HaveOccurred())
			Expect(pkg).To(Equal(NewPackage(NewResourceWithBuiltArchive(
				"name", "fp", "archive-path", "archive-sha1"), []string{"pkg1"})))
		})
	})
})
