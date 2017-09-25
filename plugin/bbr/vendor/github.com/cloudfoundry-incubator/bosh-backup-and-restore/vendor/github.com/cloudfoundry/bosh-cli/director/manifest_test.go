package director_test

import (
	"errors"

	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/director"
)

var _ = Describe("NewManifestFromPath", func() {
	var (
		fs *fakesys.FakeFileSystem
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
	})

	It("returns a manifest with parsed name", func() {
		fs.WriteFileString("/path", "---\nname: name")

		man, err := NewManifestFromPath("/path", fs)
		Expect(err).ToNot(HaveOccurred())
		Expect(man).To(Equal(Manifest{Name: "name"}))
	})

	It("returns an error if manifest cannot be read", func() {
		fs.WriteFileString("/path", "name: name")
		fs.ReadFileError = errors.New("fake-err")

		_, err := NewManifestFromPath("/path", fs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("fake-err"))
	})

	It("returns an error if parsing yaml manifest", func() {
		fs.WriteFileString("/path", "-")

		_, err := NewManifestFromPath("/path", fs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Unmarshalling manifest"))
	})
})

var _ = Describe("NewManifestFromBytes", func() {
	It("returns a manifest with parsed name", func() {
		man, err := NewManifestFromBytes([]byte("---\nname: name"))
		Expect(err).ToNot(HaveOccurred())
		Expect(man).To(Equal(Manifest{Name: "name"}))
	})

	It("returns an error if parsing yaml manifest", func() {
		_, err := NewManifestFromBytes([]byte("-"))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Unmarshalling manifest"))
	})
})
