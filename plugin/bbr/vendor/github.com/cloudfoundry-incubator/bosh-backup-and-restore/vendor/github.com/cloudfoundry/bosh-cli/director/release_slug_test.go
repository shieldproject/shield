package director_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/director"
)

var _ = Describe("ReleaseSlug", func() {
	Describe("Name", func() {
		It("returns name", func() {
			Expect(NewReleaseSlug("name", "ver").Name()).To(Equal("name"))
		})
	})

	Describe("Version", func() {
		It("returns version", func() {
			Expect(NewReleaseSlug("name", "ver").Version()).To(Equal("ver"))
		})
	})

	Describe("String", func() {
		It("returns nice string", func() {
			Expect(NewReleaseSlug("name", "ver").String()).To(Equal("name/ver"))
		})
	})

	Describe("UnmarshalFlag", func() {
		var (
			slug *ReleaseSlug
		)

		BeforeEach(func() {
			slug = &ReleaseSlug{}
		})

		It("populates slug", func() {
			err := slug.UnmarshalFlag("name/ver")
			Expect(err).ToNot(HaveOccurred())
			Expect(*slug).To(Equal(NewReleaseSlug("name", "ver")))
		})

		It("returns an error if string doesnt have 2 pieces", func() {
			err := slug.UnmarshalFlag("name")
			Expect(err).To(Equal(errors.New("Expected release 'name' to be in format 'name/version'")))

			err = slug.UnmarshalFlag("name/2/3")
			Expect(err).To(Equal(errors.New("Expected release 'name/2/3' to be in format 'name/version'")))
		})

		It("returns an error if name is empty", func() {
			err := slug.UnmarshalFlag("/ver")
			Expect(err).To(Equal(errors.New("Expected release '/ver' to specify non-empty name")))
		})

		It("returns an error if version is empty", func() {
			err := slug.UnmarshalFlag("name/")
			Expect(err).To(Equal(errors.New("Expected release 'name/' to specify non-empty version")))
		})
	})
})
