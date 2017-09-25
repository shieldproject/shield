package fakes_test

import (
	"errors"
	"os"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("FakeFileSystem", func() {
	var (
		fs *FakeFileSystem
	)

	BeforeEach(func() {
		fs = NewFakeFileSystem()
	})

	Describe("RemoveAll", func() {
		It("removes the specified file", func() {
			fs.WriteFileString("foobar", "asdfghjk")
			fs.WriteFileString("foobarbaz", "qwertyuio")

			err := fs.RemoveAll("foobar")
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.FileExists("foobar")).To(BeFalse())
			Expect(fs.FileExists("foobarbaz")).To(BeTrue())

			err = fs.RemoveAll("foobarbaz")
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.FileExists("foobarbaz")).To(BeFalse())
		})

		It("works with windows drives", func() {
			fs.WriteFileString("D:/env1", "fake-content1")
			Expect(fs.FileExists("D:/env1")).To(BeTrue())

			fs.WriteFileString("C:/env2", "fake-content2")
			Expect(fs.FileExists("C:/env2")).To(BeTrue())
		})

		It("removes the specified dir and the files under it", func() {
			err := fs.MkdirAll("foobarbaz", os.ModePerm)
			Expect(err).ToNot(HaveOccurred())
			err = fs.WriteFileString("foobarbaz/stuff.txt", "asdfghjk")
			Expect(err).ToNot(HaveOccurred())
			err = fs.MkdirAll("foobar", os.ModePerm)
			Expect(err).ToNot(HaveOccurred())
			err = fs.WriteFileString("foobar/stuff.txt", "qwertyuio")
			Expect(err).ToNot(HaveOccurred())

			err = fs.RemoveAll("foobar")
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.FileExists("foobar")).To(BeFalse())
			Expect(fs.FileExists("foobar/stuff.txt")).To(BeFalse())
			Expect(fs.FileExists("foobarbaz")).To(BeTrue())
			Expect(fs.FileExists("foobarbaz/stuff.txt")).To(BeTrue())

			err = fs.RemoveAll("foobarbaz")
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.FileExists("foobarbaz")).To(BeFalse())
			Expect(fs.FileExists("foobarbaz/stuff.txt")).To(BeFalse())
		})

		It("removes the specified symlink (but not the file it links to)", func() {
			err := fs.WriteFileString("foobarbaz", "asdfghjk")
			Expect(err).ToNot(HaveOccurred())
			err = fs.Symlink("foobarbaz", "foobar")
			Expect(err).ToNot(HaveOccurred())

			err = fs.RemoveAll("foobarbaz")
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.FileExists("foobarbaz")).To(BeFalse())
			Expect(fs.FileExists("foobar")).To(BeTrue())

			err = fs.RemoveAll("foobar")
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.FileExists("foobar")).To(BeFalse())
		})

		Context("RemoveAllStub", func() {
			It("calls it and performs its normal behavior as well", func() {
				called := false
				fs.RemoveAllStub = func(path string) error {
					called = true
					return nil
				}
				fs.WriteFileString("foobar", "asdfghjk")

				err := fs.RemoveAll("foobar")
				Expect(err).ToNot(HaveOccurred())

				Expect(fs.FileExists("foobar")).To(BeFalse())
				Expect(called).To(BeTrue())
			})

			It("supports returning an error", func() {
				fs.RemoveAllStub = func(path string) error {
					return errors.New("ERR")
				}
				err := fs.RemoveAll("foobar")
				Expect(err).To(MatchError("ERR"))
			})
		})
	})

	Describe("CopyDir", func() {
		var fixtureFiles = map[string]string{
			"foo.txt":         "asdfghjkl",
			"bar/bar.txt":     "qwertyuio",
			"bar/baz/bar.txt": "zxcvbnm,\nafawg",
		}

		var (
			fixtureDirPath = "fixtures"
		)

		BeforeEach(func() {
			for fixtureFile, contents := range fixtureFiles {
				fs.WriteFileString(path.Join(fixtureDirPath, fixtureFile), contents)
			}
		})

		It("recursively copies directory contents", func() {
			srcPath := fixtureDirPath
			dstPath, err := fs.TempDir("CopyDirTestDir")
			Expect(err).ToNot(HaveOccurred())
			defer fs.RemoveAll(dstPath)

			err = fs.CopyDir(srcPath, dstPath)
			Expect(err).ToNot(HaveOccurred())

			for fixtureFile := range fixtureFiles {
				srcContents, err := fs.ReadFile(path.Join(srcPath, fixtureFile))
				Expect(err).ToNot(HaveOccurred())

				dstContents, err := fs.ReadFile(path.Join(dstPath, fixtureFile))
				Expect(err).ToNot(HaveOccurred())

				Expect(srcContents).To(Equal(dstContents), "Copied file does not match source file: '%s", fixtureFile)
			}

			err = fs.RemoveAll(dstPath)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("GlobStub", func() {
		It("should allow glob to be replaced with a custom callback", func() {
			fs.GlobStub = func(pattern string) ([]string, error) {
				fs.GlobStub = nil
				return []string{}, errors.New("Oh noes!")
			}
			fs.SetGlob("glob/pattern", []string{"matchingFile1", "matchingFile2"})

			matches, err := fs.Glob("glob/pattern")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Oh noes!"))
			Expect(matches).To(BeEmpty())

			matches, err = fs.Glob("glob/pattern")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(matches)).To(Equal(2))
		})
	})
})
