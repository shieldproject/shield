//+build !windows

package system_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"os"
	"path/filepath"

	"runtime"
	"syscall"
)

var _ = Describe("OS FileSystem", func() {
	Describe("chown", func() {
		var testPath string
		BeforeEach(func() {
			testPath = filepath.Join(os.TempDir(), "ChownTestDir")

			err := os.Mkdir(testPath, os.FileMode(0700))
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			defer os.RemoveAll(testPath)
		})

		if runtime.GOOS == "linux" {
			It("should chown file with owner:group syntax", func() {
				osFs := createOsFs()

				err := os.Chown(testPath, 1000, 1000)
				Expect(err).ToNot(HaveOccurred())

				err = osFs.Chown(testPath, "root:root")
				Expect(err).ToNot(HaveOccurred())
				testPathStat, err := osFs.Stat(testPath)
				Expect(err).ToNot(HaveOccurred())

				Expect(testPathStat.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(0)))
				Expect(testPathStat.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(0)))
			})

			It("should chown file with owner syntax", func() {
				osFs := createOsFs()

				err := os.Chown(testPath, 1000, 1000)
				Expect(err).ToNot(HaveOccurred())

				err = osFs.Chown(testPath, "root")
				Expect(err).ToNot(HaveOccurred())
				testPathStat, err := osFs.Stat(testPath)
				Expect(err).ToNot(HaveOccurred())

				Expect(testPathStat.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(0)))
				Expect(testPathStat.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(0)))
			})
		}

		Context("given an empty owner", func() {
			It("should return an error", func() {
				osFs := createOsFs()

				err := osFs.Chown(testPath, "")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Failed to lookup user ''"))

			})
		})

		Context("given a path that does not exist", func() {
			It("should return an error", func() {
				osFs := createOsFs()

				err := osFs.Chown("/path-that-does-not-exist", "root")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("given a user that does not exist", func() {
			It("should return error", func() {
				osFs := createOsFs()

				err := osFs.Chown(testPath, "garbage-foo")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Failed to lookup user 'garbage-foo'"))
			})
		})

		Context("given a group that does not exist", func() {
			It("should return error", func() {
				osFs := createOsFs()

				err := osFs.Chown(testPath, "root:not-a-group")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Failed to chown"))
			})
		})
	})
})
