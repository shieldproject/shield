package director_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/director"
)

var _ = Describe("NewAllOrInstanceGroupOrInstanceSlugFromString", func() {
	It("populates slug with empty name and index-or-id", func() {
		slug, err := NewAllOrInstanceGroupOrInstanceSlugFromString("")
		Expect(err).ToNot(HaveOccurred())
		Expect(slug.Name()).To(Equal(""))
		Expect(slug.IndexOrID()).To(Equal(""))
	})

	It("populates slug when name is just given", func() {
		slug, err := NewAllOrInstanceGroupOrInstanceSlugFromString("name")
		Expect(err).ToNot(HaveOccurred())
		Expect(slug.Name()).To(Equal("name"))
		Expect(slug.IndexOrID()).To(Equal(""))
	})

	It("populates slug when name and index-or-id is given", func() {
		slug, err := NewAllOrInstanceGroupOrInstanceSlugFromString("name/id")
		Expect(err).ToNot(HaveOccurred())
		Expect(slug.Name()).To(Equal("name"))
		Expect(slug.IndexOrID()).To(Equal("id"))
	})

	It("returns an error if string doesnt have 1 or 2 pieces", func() {
		_, err := NewAllOrInstanceGroupOrInstanceSlugFromString("1/2/3")
		Expect(err).To(Equal(errors.New("Expected pool or instance '1/2/3' to be in format 'name' or 'name/id-or-index'")))
	})

	It("returns an error if name is empty", func() {
		_, err := NewAllOrInstanceGroupOrInstanceSlugFromString("/")
		Expect(err).To(Equal(errors.New("Expected pool or instance '/' to specify non-empty name")))
	})

	It("returns an error if index-or-id is empty", func() {
		_, err := NewAllOrInstanceGroupOrInstanceSlugFromString("name/")
		Expect(err).To(Equal(errors.New("Expected instance 'name/' to specify non-empty ID or index")))
	})
})

var _ = Describe("AllInstanceGroupOrInstanceSlug", func() {
	Describe("InstanceSlug", func() {
		It("returns true and slug if name and id is set", func() {
			slug, ok := NewAllOrInstanceGroupOrInstanceSlug("name", "id").InstanceSlug()
			Expect(slug).To(Equal(NewInstanceSlug("name", "id")))
			Expect(ok).To(BeTrue())
		})

		It("returns false if name or id is not set", func() {
			slug, ok := NewAllOrInstanceGroupOrInstanceSlug("", "").InstanceSlug()
			Expect(slug).To(Equal(InstanceSlug{}))
			Expect(ok).To(BeFalse())

			slug, ok = NewAllOrInstanceGroupOrInstanceSlug("name", "").InstanceSlug()
			Expect(slug).To(Equal(InstanceSlug{}))
			Expect(ok).To(BeFalse())

			slug, ok = NewAllOrInstanceGroupOrInstanceSlug("", "id").InstanceSlug()
			Expect(slug).To(Equal(InstanceSlug{}))
			Expect(ok).To(BeFalse())
		})
	})

	Describe("String", func() {
		It("returns empty if name or id is not set", func() {
			Expect(NewAllOrInstanceGroupOrInstanceSlug("", "").String()).To(Equal(""))
		})

		It("returns name string if id is not set", func() {
			Expect(NewAllOrInstanceGroupOrInstanceSlug("name", "").String()).To(Equal("name"))
		})

		It("returns name/id string if id is set", func() {
			Expect(NewAllOrInstanceGroupOrInstanceSlug("name", "id").String()).To(Equal("name/id"))
		})
	})

	Describe("UnmarshalFlag", func() {
		var (
			slug *AllOrInstanceGroupOrInstanceSlug
		)

		BeforeEach(func() {
			slug = &AllOrInstanceGroupOrInstanceSlug{}
		})

		It("populates slug with empty name and index-or-id", func() {
			err := slug.UnmarshalFlag("")
			Expect(err).ToNot(HaveOccurred())
			Expect(*slug).To(Equal(NewAllOrInstanceGroupOrInstanceSlug("", "")))
		})

		It("populates slug when name is just given", func() {
			err := slug.UnmarshalFlag("name")
			Expect(err).ToNot(HaveOccurred())
			Expect(*slug).To(Equal(NewAllOrInstanceGroupOrInstanceSlug("name", "")))
		})

		It("populates slug when name and index-or-id is given", func() {
			err := slug.UnmarshalFlag("name/id")
			Expect(err).ToNot(HaveOccurred())
			Expect(*slug).To(Equal(NewAllOrInstanceGroupOrInstanceSlug("name", "id")))
		})

		It("returns an error if string doesnt have 1 or 2 pieces", func() {
			err := slug.UnmarshalFlag("1/2/3")
			Expect(err).To(Equal(errors.New("Expected pool or instance '1/2/3' to be in format 'name' or 'name/id-or-index'")))
		})

		It("returns an error if name is empty", func() {
			err := slug.UnmarshalFlag("/")
			Expect(err).To(Equal(errors.New("Expected pool or instance '/' to specify non-empty name")))
		})

		It("returns an error if index-or-id is empty", func() {
			err := slug.UnmarshalFlag("name/")
			Expect(err).To(Equal(errors.New("Expected instance 'name/' to specify non-empty ID or index")))
		})
	})
})
