package director_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/director"
)

var _ = Describe("StemcellSlug", func() {
	Describe("Name", func() {
		It("returns name", func() {
			Expect(NewStemcellSlug("name", "ver").Name()).To(Equal("name"))
		})
	})

	Describe("Version", func() {
		It("returns version", func() {
			Expect(NewStemcellSlug("name", "ver").Version()).To(Equal("ver"))
		})
	})

	Describe("String", func() {
		It("returns nice string", func() {
			Expect(NewStemcellSlug("name", "ver").String()).To(Equal("name/ver"))
		})
	})

	Describe("UnmarshalFlag", func() {
		var (
			slug *StemcellSlug
		)

		BeforeEach(func() {
			slug = &StemcellSlug{}
		})

		It("populates slug", func() {
			err := slug.UnmarshalFlag("name/ver")
			Expect(err).ToNot(HaveOccurred())
			Expect(*slug).To(Equal(NewStemcellSlug("name", "ver")))
		})

		It("returns an error if string doesnt have 2 pieces", func() {
			err := slug.UnmarshalFlag("name")
			Expect(err).To(Equal(errors.New("Expected stemcell 'name' to be in format 'name/version'")))

			err = slug.UnmarshalFlag("name/2/3")
			Expect(err).To(Equal(errors.New("Expected stemcell 'name/2/3' to be in format 'name/version'")))
		})

		It("returns an error if name is empty", func() {
			err := slug.UnmarshalFlag("/ver")
			Expect(err).To(Equal(errors.New("Expected stemcell '/ver' to specify non-empty name")))
		})

		It("returns an error if ver is empty", func() {
			err := slug.UnmarshalFlag("name/")
			Expect(err).To(Equal(errors.New("Expected stemcell 'name/' to specify non-empty version")))
		})
	})

	Describe("UnmarshalJSON", func() {
		var (
			slug *StemcellSlug
		)

		BeforeEach(func() {
			slug = &StemcellSlug{}
		})

		It("populates slug", func() {
			err := slug.UnmarshalJSON([]byte(`"name/ver"`))
			Expect(err).ToNot(HaveOccurred())
			Expect(*slug).To(Equal(NewStemcellSlug("name", "ver")))
		})

		It("returns an error if string doesnt have 2 pieces", func() {
			err := slug.UnmarshalJSON([]byte(`"name"`))
			Expect(err).To(Equal(errors.New("Expected stemcell 'name' to be in format 'name/version'")))

			err = slug.UnmarshalJSON([]byte(`"name/2/3"`))
			Expect(err).To(Equal(errors.New("Expected stemcell 'name/2/3' to be in format 'name/version'")))
		})

		It("returns an error if name is empty", func() {
			err := slug.UnmarshalJSON([]byte(`"/ver"`))
			Expect(err).To(Equal(errors.New("Expected stemcell '/ver' to specify non-empty name")))
		})

		It("returns an error if version is empty", func() {
			err := slug.UnmarshalJSON([]byte(`"name/"`))
			Expect(err).To(Equal(errors.New("Expected stemcell 'name/' to specify non-empty version")))
		})
	})
})
