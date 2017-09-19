package crypto_test

import (
	"errors"
	"path/filepath"

	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/crypto"

	boshcrypto "github.com/cloudfoundry/bosh-utils/crypto"
)

var _ = Describe("Sha1Calculator", func() {
	var (
		fs               *fakesys.FakeFileSystem
		digestCalculator DigestCalculator
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		digestCalculator = NewDigestCalculator(fs, []boshcrypto.Algorithm{boshcrypto.DigestAlgorithmSHA1})
	})

	Describe("Calculate", func() {
		Context("when path is a file", func() {
			BeforeEach(func() {
				fs.RegisterOpenFile(filepath.Join("/", "fake-archived-templates-path"), &fakesys.FakeFile{
					Contents: []byte("fake-archive-contents"),
					Stats:    &fakesys.FakeFileStats{FileType: fakesys.FakeFileTypeFile},
				})
			})

			It("returns sha1 of the file", func() {
				sha1, err := digestCalculator.Calculate(filepath.Join("/", "fake-archived-templates-path"))
				Expect(err).ToNot(HaveOccurred())
				Expect(sha1).To(Equal("4603db250d7b5b78dfe17869649784353177b549"))
			})

			It("returns a multiple digest string when multiple algorithms are provided", func() {

				digestCalculator = NewDigestCalculator(fs, []boshcrypto.Algorithm{
					boshcrypto.DigestAlgorithmSHA1,
					boshcrypto.DigestAlgorithmSHA256,
				})

				multipleDigestStr, err := digestCalculator.Calculate(filepath.Join("/", "fake-archived-templates-path"))
				Expect(err).ToNot(HaveOccurred())
				Expect(multipleDigestStr).To(Equal("4603db250d7b5b78dfe17869649784353177b549;sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"))
			})
		})

		Context("when reading the file fails", func() {
			BeforeEach(func() {
				fs.OpenFileErr = errors.New("fake-open-file-error")
			})

			It("returns an error", func() {
				_, err := digestCalculator.Calculate(filepath.Join("/", "fake-archived-templates-path"))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-open-file-error"))
			})
		})
	})

	Describe("CalculateString", func() {
		It("returns sha1 of data", func() {
			Expect(digestCalculator.CalculateString("data")).To(Equal("a17c9aaa61e80a1bf71d0d850af4e5baa9800bbd"))
		})

		It("returns a multiple digest string when multiple algorithms are provided", func() {
			digestCalculator = NewDigestCalculator(fs, []boshcrypto.Algorithm{
				boshcrypto.DigestAlgorithmSHA1,
				boshcrypto.DigestAlgorithmSHA256,
			})
			Expect(digestCalculator.CalculateString("data")).To(Equal("a17c9aaa61e80a1bf71d0d850af4e5baa9800bbd;sha256:3a6eb0790f39ac87c94f3856b2dd2c5d110e6811602261a9a923d3bb23adc8b7"))
		})
	})
})
