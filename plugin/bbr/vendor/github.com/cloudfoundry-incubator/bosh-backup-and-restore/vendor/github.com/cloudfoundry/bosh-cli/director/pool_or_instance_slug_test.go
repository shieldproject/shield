package director_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/director"
)

var _ = Describe("NewInstanceGroupOrInstanceSlug", func() {
	It("populates slug when name is just given", func() {
		slug := NewInstanceGroupOrInstanceSlug("name", "")
		Expect(slug.Name()).To(Equal("name"))
		Expect(slug.IndexOrID()).To(Equal(""))
	})

	It("populates slug when name and index-or-id is given", func() {
		slug := NewInstanceGroupOrInstanceSlug("name", "id")
		Expect(slug.Name()).To(Equal("name"))
		Expect(slug.IndexOrID()).To(Equal("id"))
	})

	It("panics if name is empty", func() {
		Expect(func() { NewInstanceGroupOrInstanceSlug("", "") }).To(Panic())
	})
})

var _ = Describe("NewInstanceGroupOrInstanceSlugFromString", func() {
	It("populates slug when name is just given", func() {
		slug, err := NewInstanceGroupOrInstanceSlugFromString("name")
		Expect(err).ToNot(HaveOccurred())
		Expect(slug).To(Equal(NewInstanceGroupOrInstanceSlug("name", "")))
	})

	It("populates slug when name and index-or-id is given", func() {
		slug, err := NewInstanceGroupOrInstanceSlugFromString("name/id")
		Expect(err).ToNot(HaveOccurred())
		Expect(slug).To(Equal(NewInstanceGroupOrInstanceSlug("name", "id")))
	})

	It("returns an error if string doesnt have 1 or 2 pieces", func() {
		_, err := NewInstanceGroupOrInstanceSlugFromString("")
		Expect(err).To(Equal(errors.New("Expected pool or instance '' to specify non-empty name")))

		_, err = NewInstanceGroupOrInstanceSlugFromString("1/2/3")
		Expect(err).To(Equal(errors.New("Expected pool or instance '1/2/3' to be in format 'name' or 'name/id-or-index'")))
	})

	It("returns an error if name is empty", func() {
		_, err := NewInstanceGroupOrInstanceSlugFromString("/")
		Expect(err).To(Equal(errors.New("Expected pool or instance '/' to specify non-empty name")))
	})

	It("returns an error if index-or-id is empty", func() {
		_, err := NewInstanceGroupOrInstanceSlugFromString("name/")
		Expect(err).To(Equal(errors.New("Expected instance 'name/' to specify non-empty ID or index")))
	})
})

var _ = Describe("InstanceGroupOrInstanceSlug", func() {
	Describe("String", func() {
		It("returns name string if id is not set", func() {
			Expect(NewInstanceGroupOrInstanceSlug("name", "").String()).To(Equal("name"))
		})

		It("returns name/id string if id is set", func() {
			Expect(NewInstanceGroupOrInstanceSlug("name", "id").String()).To(Equal("name/id"))
		})
	})
})
