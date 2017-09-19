package stemcell_test

import (
	"errors"
	. "github.com/cloudfoundry/bosh-cli/stemcell"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	fakecmd "github.com/cloudfoundry/bosh-utils/fileutil/fakes"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("Reader", func() {
	var (
		compressor     *fakecmd.FakeCompressor
		stemcellReader Reader
		fs             *fakesys.FakeFileSystem
	)

	BeforeEach(func() {
		compressor = fakecmd.NewFakeCompressor()
		fs = fakesys.NewFakeFileSystem()
		stemcellReader = NewReader(compressor, fs)

		manifestContents := `
---
name: fake-stemcell-name
version: '2690'
operating_system: ubuntu-trusty
sha1: sha
bosh_protocol: 1
cloud_properties:
  infrastructure: aws
  ami:
    us-east-1: fake-ami-version
    `
		fs.WriteFileString("fake-extracted-path/stemcell.MF", manifestContents)
	})

	It("extracts the stemcells from a stemcell path", func() {
		_, err := stemcellReader.Read("fake-stemcell-path", "fake-extracted-path")
		Expect(err).ToNot(HaveOccurred())
		Expect(compressor.DecompressFileToDirTarballPaths).To(ContainElement("fake-stemcell-path"))
		Expect(compressor.DecompressFileToDirDirs).To(ContainElement("fake-extracted-path"))
	})

	It("generates correct stemcell", func() {
		stemcell, err := stemcellReader.Read("fake-stemcell-path", "fake-extracted-path")
		Expect(err).ToNot(HaveOccurred())
		expectedStemcell := NewExtractedStemcell(
			Manifest{
				Name:         "fake-stemcell-name",
				Version:      "2690",
				OS:           "ubuntu-trusty",
				SHA1:         "sha",
				BoshProtocol: "1",
				CloudProperties: biproperty.Map{
					"infrastructure": "aws",
					"ami": biproperty.Map{
						"us-east-1": "fake-ami-version",
					},
				},
			},
			"fake-extracted-path",
			compressor,
			fs,
		)
		Expect(stemcell).To(Equal(expectedStemcell))
	})

	Context("when extracting stemcell fails", func() {
		BeforeEach(func() {
			compressor.DecompressFileToDirErr = errors.New("fake-decompress-error")
		})

		It("returns an error", func() {
			_, err := stemcellReader.Read("fake-stemcell-path", "fake-extracted-path")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-decompress-error"))
		})
	})

	Context("when reading stemcell manifest fails", func() {
		BeforeEach(func() {
			fs.ReadFileError = errors.New("fake-read-error")
		})

		It("returns an error", func() {
			_, err := stemcellReader.Read("fake-stemcell-path", "fake-extracted-path")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-read-error"))
		})
	})

	Context("when parsing stemcell manifest fails", func() {
		BeforeEach(func() {
			fs.WriteFileString("fake-extracted-path/stemcell.MF", "<not-a-yaml>")
		})

		It("returns an error", func() {
			_, err := stemcellReader.Read("fake-stemcell-path", "fake-extracted-path")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Parsing stemcell manifest"))
		})
	})
})
