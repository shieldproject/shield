package director_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/director"
)

var _ = Describe("NewReleaseOrSeriesSlug", func() {
	It("returns slug if name is provided", func() {
		slug := NewReleaseOrSeriesSlug("name", "")
		Expect(slug.Name()).To(Equal("name"))
		Expect(slug.Version()).To(Equal(""))
	})

	It("returns slug if name and id are provided", func() {
		slug := NewReleaseOrSeriesSlug("name", "ver")
		Expect(slug.Name()).To(Equal("name"))
		Expect(slug.Version()).To(Equal("ver"))
	})

	It("returns false if name is not specified", func() {
		Expect(func() { NewReleaseOrSeriesSlug("", "ver") }).To(Panic())
	})
})

var _ = Describe("ReleaseOrSeriesSlug", func() {
	Describe("ReleaseSlug", func() {
		It("returns slug if name and id are provided", func() {
			slug, ok := NewReleaseOrSeriesSlug("name", "ver").ReleaseSlug()
			Expect(slug).To(Equal(NewReleaseSlug("name", "ver")))
			Expect(ok).To(BeTrue())
		})

		It("returns false if version is not specified", func() {
			_, ok := NewReleaseOrSeriesSlug("name", "").ReleaseSlug()
			Expect(ok).To(BeFalse())
		})
	})

	Describe("SeriesSlug", func() {
		It("returns slug if with name", func() {
			slug := NewReleaseOrSeriesSlug("name", "").SeriesSlug()
			Expect(slug).To(Equal(NewReleaseSeriesSlug("name")))
		})
	})

	Describe("UnmarshalFlag", func() {
		var (
			slug *ReleaseOrSeriesSlug
		)

		BeforeEach(func() {
			slug = &ReleaseOrSeriesSlug{}
		})

		It("populates slug when name is just given", func() {
			err := slug.UnmarshalFlag("name")
			Expect(err).ToNot(HaveOccurred())
			Expect(*slug).To(Equal(NewReleaseOrSeriesSlug("name", "")))
		})

		It("populates slug when name and version is given", func() {
			err := slug.UnmarshalFlag("name/ver")
			Expect(err).ToNot(HaveOccurred())
			Expect(*slug).To(Equal(NewReleaseOrSeriesSlug("name", "ver")))
		})

		It("returns an error if string doesnt have 1 or 2 pieces", func() {
			err := slug.UnmarshalFlag("")
			Expect(err).To(Equal(errors.New("Expected release '' to specify non-empty name")))

			err = slug.UnmarshalFlag("name/1/2")
			Expect(err).To(Equal(errors.New("Expected release or series 'name/1/2' to be in format 'name' or 'name/version'")))
		})

		It("returns an error if name is empty", func() {
			err := slug.UnmarshalFlag("/")
			Expect(err).To(Equal(errors.New("Expected release '/' to specify non-empty name")))
		})

		It("returns an error if version is empty", func() {
			err := slug.UnmarshalFlag("name/")
			Expect(err).To(Equal(errors.New("Expected release 'name/' to specify non-empty version")))
		})
	})
})
