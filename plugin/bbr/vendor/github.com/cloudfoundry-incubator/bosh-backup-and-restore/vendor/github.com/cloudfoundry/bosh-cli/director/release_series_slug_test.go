package director_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/director"
)

var _ = Describe("ReleaseSeriesSlug", func() {
	Describe("Name", func() {
		It("returns name", func() {
			Expect(NewReleaseSeriesSlug("name").Name()).To(Equal("name"))
		})
	})

	Describe("String", func() {
		It("returns name", func() {
			Expect(NewReleaseSeriesSlug("name").String()).To(Equal("name"))
		})
	})

	Describe("UnmarshalFlag", func() {
		var (
			slug *ReleaseSeriesSlug
		)

		BeforeEach(func() {
			slug = &ReleaseSeriesSlug{}
		})

		It("populates slug", func() {
			err := slug.UnmarshalFlag("name")
			Expect(err).ToNot(HaveOccurred())
			Expect(*slug).To(Equal(NewReleaseSeriesSlug("name")))
		})

		It("returns an error if name is empty", func() {
			err := slug.UnmarshalFlag("")
			Expect(err).To(Equal(errors.New("Expected non-empty release series name")))
		})
	})
})
