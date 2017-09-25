package cmd_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"

	"errors"
	sysfakes "github.com/cloudfoundry/bosh-utils/system/fakes"
	"os"
)

var _ = Describe("FileArg", func() {
	Describe("UnmarshalFlag", func() {
		var (
			arg *FileArg
			fs  *sysfakes.FakeFileSystem
		)

		BeforeEach(func() {
			fs = sysfakes.NewFakeFileSystem()
			arg = &FileArg{FS: fs}

			fs.MkdirAll("/some/empty/dir", os.ModeDir)
			fs.MkdirAll("/some/dir", os.ModeDir)
			fs.WriteFileString("stuff", "/some/dir/contents")
		})

		Context("when the given path cannot be expanded in the file system", func() {
			It("reports this as an error", func() {
				fs.ExpandPathErr = errors.New("can't expand")
				err := arg.UnmarshalFlag("kaboom")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Checking file path: can't expand"))
			})
		})

		Context("when the given path can be expanded in the file system", func() {
			Context("when there is no file at the given path", func() {
				It("returns with ExpandedPath set", func() {
					Expect(arg.UnmarshalFlag("/some/dir/newball")).To(Succeed())
					Expect(arg.ExpandedPath).To(Equal("/some/dir/newball"))
				})

				It("expands the path before setting", func() {
					fs.ExpandPathExpanded = "expanded/path"
					Expect(arg.UnmarshalFlag("newball")).To(Succeed())
					Expect(arg.ExpandedPath).To(Equal("expanded/path"))
					Expect(fs.ExpandPathPath).To(Equal("newball"))
				})

				It("propagates an empty string input path through to ExpandedPath", func() {
					Expect(arg.UnmarshalFlag("")).To(Succeed())
					Expect(arg.ExpandedPath).To(Equal(""))
				})
			})

			Context("when the filesystem errors while stat'ing the file", func() {
				It("returns an error", func() {
					fs.WriteFileString("/some/tarball/path.tgz", "it exists")
					file := sysfakes.NewFakeFile("/some/tarball/path.tgz", fs)
					file.StatErr = errors.New("can't stat me!")
					fs.RegisterOpenFile("/some/tarball/path.tgz", file)

					err := arg.UnmarshalFlag("/some/tarball/path.tgz")
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("Checking file path: can't stat me!"))
				})
			})

			Context("when there is already a file at that location", func() {
				It("allows paths to existing files", func() {
					Expect(arg.UnmarshalFlag("/some/dir/contents")).To(Succeed())
				})

				It("rejects paths to directories", func() {
					err := arg.UnmarshalFlag("/some/dir")
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("Path must not be directory"))
				})
			})
		})
	})
})
