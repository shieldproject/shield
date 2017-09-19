package director_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/director"
)

var _ = Describe("InstanceSlug", func() {
	Describe("Name", func() {
		It("returns name", func() {
			Expect(NewInstanceSlug("name", "id").Name()).To(Equal("name"))
		})
	})

	Describe("IndexOrID", func() {
		It("returns index-or-id", func() {
			Expect(NewInstanceSlug("name", "id").IndexOrID()).To(Equal("id"))
		})
	})

	Describe("IsProvided", func() {
		It("returns true if name and id are specified", func() {
			Expect(NewInstanceSlug("name", "id").IsProvided()).To(BeTrue())
		})

		It("returns false if it's empty", func() {
			Expect(InstanceSlug{}.IsProvided()).To(BeFalse())
		})
	})

	Describe("String", func() {
		It("returns nice string", func() {
			Expect(NewInstanceSlug("name", "id").String()).To(Equal("name/id"))
		})
	})

	Describe("UnmarshalFlag", func() {
		var (
			slug *InstanceSlug
		)

		BeforeEach(func() {
			slug = &InstanceSlug{}
		})

		It("populates slug", func() {
			err := slug.UnmarshalFlag("name/id")
			Expect(err).ToNot(HaveOccurred())
			Expect(*slug).To(Equal(NewInstanceSlug("name", "id")))
		})

		It("returns an error if string doesnt have 2 pieces", func() {
			err := slug.UnmarshalFlag("1")
			Expect(err).To(Equal(errors.New("Expected instance '1' to be in format 'name/index-or-id'")))

			err = slug.UnmarshalFlag("1.2.3")
			Expect(err).To(Equal(errors.New("Expected instance '1.2.3' to be in format 'name/index-or-id'")))
		})

		It("returns an error if name is empty", func() {
			err := slug.UnmarshalFlag("/id")
			Expect(err).To(Equal(errors.New("Expected instance '/id' to specify non-empty name")))
		})

		It("returns an error if index-or-id is empty", func() {
			err := slug.UnmarshalFlag("name/")
			Expect(err).To(Equal(errors.New("Expected instance 'name/' to specify non-empty index or ID")))
		})
	})
})
