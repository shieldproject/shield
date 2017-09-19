package crypto_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"errors"
	"fmt"
	. "github.com/cloudfoundry/bosh-utils/crypto"
	"io/ioutil"
	"os"
	"strings"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("digestImpl", func() {
	Describe("Verify", func() {
		var digest Digest

		Context("sha1", func() {
			BeforeEach(func() {
				digest = NewDigest(DigestAlgorithmSHA1, "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed")
			})

			It("returns nil when valid reader", func() {
				Expect(digest.Verify(strings.NewReader("hello world"))).To(BeNil())
			})

			It("returns error when invalid sum", func() {
				err := digest.Verify(strings.NewReader("omg"))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Expected stream to have digest '2aae6c35c94fcfb415dbe95f408b9ce91ee846ed' but was 'adccece39a0795801972604c8cf21a22bf45b262'"))
			})
		})

		Context("sha256", func() {
			BeforeEach(func() {
				digest = NewDigest(DigestAlgorithmSHA256, "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9")
			})

			It("returns nil when valid reader", func() {
				Expect(digest.Verify(strings.NewReader("hello world"))).To(BeNil())
			})

			It("returns error when invalid sum", func() {
				err := digest.Verify(strings.NewReader("omg"))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Expected stream to have digest 'sha256:b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9' but was 'sha256:651de3316fa27e74d6a9de619ebac68e3b781e9527fdd00c5ef7143b1fa581b6'"))
			})
		})

		Context("sha512", func() {
			BeforeEach(func() {
				digest = NewDigest(DigestAlgorithmSHA512, "309ecc489c12d6eb4cc40f50c902f2b4d0ed77ee511a7c7a9bcd3ca86d4cd86f989dd35bc5ff499670da34255b45b0cfd830e81f605dcf7dc5542e93ae9cd76f")
			})

			It("returns nil when valid reader", func() {
				Expect(digest.Verify(strings.NewReader("hello world"))).To(BeNil())
			})

			It("returns error when invalid sum", func() {
				err := digest.Verify(strings.NewReader("omg"))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Expected stream to have digest 'sha512:309ecc489c12d6eb4cc40f50c902f2b4d0ed77ee511a7c7a9bcd3ca86d4cd86f989dd35bc5ff499670da34255b45b0cfd830e81f605dcf7dc5542e93ae9cd76f' but was 'sha512:a1eb442d3b6c9680e95b73033968223e6ea5fbff7c3d6ed8f6f9ec38cec74cad307f5b8662291323c65e81cc2ec1d24384e4c1a165aed36d9874efecf976b2c4'"))
			})
		})
	})

	Describe("VerifyFilePath", func() {
		var (
			file   *os.File
			digest Digest
		)

		BeforeEach(func() {
			var err error
			file, err = ioutil.TempFile("", "multiple-digest")
			Expect(err).ToNot(HaveOccurred())
			defer file.Close()
			file.Write([]byte("fake-contents"))

			digest = NewDigest(DigestAlgorithmSHA1, "978ad524a02039f261773fe93d94973ae7de6470")
		})

		It("can read a file and verify its content aginst the digest", func() {
			logger := boshlog.NewLogger(boshlog.LevelNone)
			fileSystem := boshsys.NewOsFileSystem(logger)

			err := digest.VerifyFilePath(file.Name(), fileSystem)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if the file cannot be opened", func() {
			fileSystem := fakesys.NewFakeFileSystem()
			fileSystem.OpenFileErr = errors.New("nope")

			err := digest.VerifyFilePath(file.Name(), fileSystem)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("Calculating digest of '%s': nope", file.Name())))
		})
	})

	Describe("String", func() {
		Context("sha1", func() {
			It("excludes algorithm", func() {
				digest := NewDigest(DigestAlgorithmSHA1, "value")
				Expect(digest.String()).To(Equal("value"))
			})
		})

		Context("sha256", func() {
			It("includes algorithm", func() {
				digest := NewDigest(DigestAlgorithmSHA256, "value")
				Expect(digest.String()).To(Equal("sha256:value"))
			})
		})

		Context("sha512", func() {
			It("includes algorithm", func() {
				digest := NewDigest(DigestAlgorithmSHA512, "value")
				Expect(digest.String()).To(Equal("sha512:value"))
			})
		})
	})
})
