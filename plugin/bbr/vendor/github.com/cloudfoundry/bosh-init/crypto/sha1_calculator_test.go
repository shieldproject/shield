package crypto_test

import (
	"errors"
	"os"

	. "github.com/cloudfoundry/bosh-init/crypto"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sha1Calculator", func() {
	var (
		fs             *fakesys.FakeFileSystem
		sha1Calculator SHA1Calculator
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		sha1Calculator = NewSha1Calculator(fs)
	})

	Describe("Calculate", func() {
		Context("when path is directory", func() {
			BeforeEach(func() {
				fs.RegisterOpenFile("/fake-templates-dir", &fakesys.FakeFile{
					Stats: &fakesys.FakeFileStats{FileType: fakesys.FakeFileTypeDir},
				})

				fs.RegisterOpenFile("/fake-templates-dir/file-1", &fakesys.FakeFile{
					Contents: []byte("fake-file-1-contents"),
				})

				fs.WriteFileString("/fake-templates-dir/file-1", "fake-file-1-contents")

				fs.RegisterOpenFile("/fake-templates-dir/config/file-2", &fakesys.FakeFile{
					Contents: []byte("fake-file-2-contents"),
				})
				fs.MkdirAll("/fake-templates-dir/config", os.ModePerm)
				fs.WriteFileString("/fake-templates-dir/config/file-2", "fake-file-2-contents")
			})

			It("returns sha1 of the all files in the directory", func() {
				sha1, err := sha1Calculator.Calculate("/fake-templates-dir")
				Expect(err).ToNot(HaveOccurred())
				Expect(sha1).To(Equal("bc0646cd41b98cd6c878db7a0573eca345f78200"))
			})
		})

		Context("when path is a file", func() {
			BeforeEach(func() {
				fs.RegisterOpenFile("/fake-archived-templates-path", &fakesys.FakeFile{
					Contents: []byte("fake-archive-contents"),
					Stats:    &fakesys.FakeFileStats{FileType: fakesys.FakeFileTypeFile},
				})
			})

			It("returns sha1 of the file", func() {
				sha1, err := sha1Calculator.Calculate("/fake-archived-templates-path")
				Expect(err).ToNot(HaveOccurred())
				Expect(sha1).To(Equal("4603db250d7b5b78dfe17869649784353177b549"))
			})
		})

		Context("when reading the file fails", func() {
			BeforeEach(func() {
				fs.OpenFileErr = errors.New("fake-open-file-error")
			})

			It("returns an error", func() {
				_, err := sha1Calculator.Calculate("/fake-archived-templates-path")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-open-file-error"))
			})
		})
	})
})
