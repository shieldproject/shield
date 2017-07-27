package cmd_test

import (
	"errors"
	"os"

	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
)

var _ = Describe("FileBytesArg", func() {
	Describe("UnmarshalFlag", func() {
		var (
			fs  *fakesys.FakeFileSystem
			arg FileBytesArg
		)

		BeforeEach(func() {
			fs = fakesys.NewFakeFileSystem()
			arg = FileBytesArg{FS: fs}
		})

		Context("when dash is given as path", func() {
			It("reads bytes from stdin", func() {
				r, w, err := os.Pipe()
				Expect(err).ToNot(HaveOccurred())

				os.Stdin = r

				_, err = w.Write([]byte("content"))
				Expect(err).ToNot(HaveOccurred())

				err = w.Close()
				Expect(err).ToNot(HaveOccurred())

				err = (&arg).UnmarshalFlag("-")
				Expect(err).ToNot(HaveOccurred())
				Expect(arg.Bytes).To(Equal([]byte("content")))
			})

			It("returns error if reading from stdin fails", func() {
				os.Stdin = nil

				err := (&arg).UnmarshalFlag("-")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Reading from stdin"))
			})
		})

		Context("when path is not a dash", func() {
			It("sets bytes from file contents", func() {
				fs.WriteFileString("/some/path", "content")

				err := (&arg).UnmarshalFlag("/some/path")
				Expect(err).ToNot(HaveOccurred())
				Expect(arg.Bytes).To(Equal([]byte("content")))
			})

			It("returns an error if expanding path fails", func() {
				fs.ExpandPathErr = errors.New("fake-err")

				err := (&arg).UnmarshalFlag("/some/path")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("returns an error if reading file fails", func() {
				fs.WriteFileString("/some/path", "content")
				fs.ReadFileError = errors.New("fake-err")

				err := (&arg).UnmarshalFlag("/some/path")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("returns an error when it's empty", func() {
				err := (&arg).UnmarshalFlag("")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Expected file path to be non-empty"))
			})
		})
	})
})
