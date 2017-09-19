package util_test

import (
	"path/filepath"

	"github.com/cloudfoundry/bosh-cli/common/util"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AbsolutifyPath", func() {
	var realfs boshsys.FileSystem
	var fakeManifestPath, fakeFilePath string

	BeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		realfs = boshsys.NewOsFileSystem(logger)
		fakeManifestPath = "/fake/manifest/path/manifest.yml"
	})

	Context("File path is not a url", func() {
		Context("File path is relative", func() {
			Context("File path begins with a series of ../", func() {
				It("joins file path to the manifest directory", func() {
					fakeFilePath = "../fake/relative/path/file.tgz"
					Expect(util.AbsolutifyPath(fakeManifestPath, fakeFilePath, realfs)).To(
						Equal(filepath.Join("/", "fake", "manifest", "fake", "relative", "path", "file.tgz")))
				})
			})
			Context("File is located in same directory as manifest or subdirectory", func() {
				It("makes the file path relative to the manifest directory", func() {
					fakeFilePath = "fake/relative/path/file.tgz"
					result, err := util.AbsolutifyPath(fakeManifestPath, fakeFilePath, realfs)
					Expect(result).To(Equal(filepath.Join("/", "fake", "manifest", "path", "fake", "relative", "path", "file.tgz")))
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})

		Context("File path is absolute", func() {
			Context("file path starts with ~", func() {
				It("expands the file path", func() {
					fakeFilePath = "~/fake/absolute/path/file.tgz"
					currentUserHome, _ := realfs.HomeDir("")

					result, err := util.AbsolutifyPath(fakeManifestPath, fakeFilePath, realfs)
					Expect(result).To(Equal(currentUserHome + filepath.Join("/", "fake", "absolute", "path", "file.tgz")))
					Expect(err).ToNot(HaveOccurred())
				})
			})

			Context("file path starts with /", func() {
				It("passes the file path it received", func() {
					fakeFilePath = "/fake/absolute/path/file.tgz"
					result, err := util.AbsolutifyPath(fakeManifestPath, fakeFilePath, realfs)
					Expect(result).To(Equal("/fake/absolute/path/file.tgz"))
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})
	})

	Context("File path is a url", func() {
		Context("file path begins with http", func() {
			It("passes the file path it recieved", func() {
				fakeFilePath = "http://fake/absolute/path/file.tgz"
				result, err := util.AbsolutifyPath(fakeManifestPath, fakeFilePath, realfs)
				Expect(result).To(Equal("http://fake/absolute/path/file.tgz"))
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("file path begins with file://", func() {
			Context("file path is relative to manifest", func() {
				It("joins file path to the manifest directory", func() {
					fakeFilePath = "file://fake/relative/path/file.tgz"
					result, err := util.AbsolutifyPath(fakeManifestPath, fakeFilePath, realfs)
					Expect(result).To(Equal("file:///fake/manifest/path/fake/relative/path/file.tgz"))
					Expect(err).ToNot(HaveOccurred())
				})
			})

			Context("file path is absolute", func() {
				Context("file path begins with 'file://~'", func() {
					It("passes the file path it received", func() {
						fakeFilePath = "file://~fake/absolute/path/file.tgz"
						result, err := util.AbsolutifyPath(fakeManifestPath, fakeFilePath, realfs)
						Expect(result).To(Equal("file://~fake/absolute/path/file.tgz"))
						Expect(err).ToNot(HaveOccurred())
					})
				})

				Context("file path begins with 'file:///'", func() {
					It("passes the file path it recieved", func() {
						fakeFilePath = "file:///fake/absolute/path/file.tgz"
						result, err := util.AbsolutifyPath(fakeManifestPath, fakeFilePath, realfs)
						Expect(result).To(Equal("file:///fake/absolute/path/file.tgz"))
						Expect(err).ToNot(HaveOccurred())
					})
				})
			})
		})

	})
})
