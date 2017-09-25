package drain_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshas "github.com/cloudfoundry/bosh-agent/agent/applier/applyspec"
	. "github.com/cloudfoundry/bosh-agent/agent/script/drain"
)

var _ = Describe("NewShutdownParams", func() {
	var (
		oldSpec, newSpec boshas.V1ApplySpec
	)

	BeforeEach(func() {
		oldSpec = boshas.V1ApplySpec{PersistentDisk: 200}
		newSpec = boshas.V1ApplySpec{PersistentDisk: 301}
	})

	Describe("JobState", func() {
		It("returns JSON serialized current spec that only includes persistent disk", func() {
			state, err := NewShutdownParams(oldSpec, &newSpec).JobState()
			Expect(err).ToNot(HaveOccurred())
			Expect(state).To(Equal(`{"persistent_disk":200}`))
		})
	})

	Describe("JobNextState", func() {
		It("returns JSON serialized future spec that only includes persistent disk", func() {
			state, err := NewShutdownParams(oldSpec, &newSpec).JobNextState()
			Expect(err).ToNot(HaveOccurred())
			Expect(state).To(Equal(`{"persistent_disk":301}`))
		})

		It("returns empty string if next state is not available", func() {
			state, err := NewShutdownParams(oldSpec, nil).JobNextState()
			Expect(err).ToNot(HaveOccurred())
			Expect(state).To(Equal(""))
		})
	})
})

var _ = Describe("ToStatusParams", func() {
	var (
		oldSpec, newSpec boshas.V1ApplySpec
	)

	BeforeEach(func() {
		oldSpec = boshas.V1ApplySpec{PersistentDisk: 200}
		newSpec = boshas.V1ApplySpec{PersistentDisk: 301}
	})

	Describe("JobState", func() {
		It("returns JSON serialized current spec that only includes persistent disk", func() {
			state, err := NewUpdateParams(oldSpec, newSpec).ToStatusParams().JobState()
			Expect(err).ToNot(HaveOccurred())
			Expect(state).To(Equal(`{"persistent_disk":200}`))
		})
	})

	Describe("JobNextState", func() {
		It("returns empty string because next state is never available", func() {
			state, err := NewUpdateParams(oldSpec, newSpec).ToStatusParams().JobNextState()
			Expect(err).ToNot(HaveOccurred())
			Expect(state).To(Equal(""))
		})
	})
})

var _ = Describe("NewUpdateParams", func() {
	Describe("UpdatedPackages", func() {
		It("returns list of packages that changed or got added in lexical order", func() {
			oldPkgs := map[string]boshas.PackageSpec{
				"foo": boshas.PackageSpec{
					Name: "foo",
					Sha1: "foo-sha1-old",
				},
				"bar": boshas.PackageSpec{
					Name: "bar",
					Sha1: "bar-sha1",
				},
			}

			newPkgs := map[string]boshas.PackageSpec{
				"foo": boshas.PackageSpec{
					Name: "foo",
					Sha1: "foo-sha1-new",
				},
				"bar": boshas.PackageSpec{
					Name: "bar",
					Sha1: "bar-sha1",
				},
				"baz": boshas.PackageSpec{
					Name: "baz",
					Sha1: "baz-sha1",
				},
			}

			oldSpec := boshas.V1ApplySpec{
				PackageSpecs: oldPkgs,
			}

			newSpec := boshas.V1ApplySpec{
				PackageSpecs: newPkgs,
			}

			params := NewUpdateParams(oldSpec, newSpec)

			Expect(params.UpdatedPackages()).To(Equal([]string{"baz", "foo"}))
		})
	})

	Describe("JobState", func() {
		It("returns JSON serialized current spec that only includes persistent disk", func() {
			oldSpec := boshas.V1ApplySpec{PersistentDisk: 200}
			newSpec := boshas.V1ApplySpec{PersistentDisk: 301}
			params := NewUpdateParams(oldSpec, newSpec)

			state, err := params.JobState()
			Expect(err).ToNot(HaveOccurred())
			Expect(state).To(Equal(`{"persistent_disk":200}`))
		})
	})

	Describe("JobNextState", func() {
		It("returns JSON serialized future spec that only includes persistent disk", func() {
			oldSpec := boshas.V1ApplySpec{PersistentDisk: 200}
			newSpec := boshas.V1ApplySpec{PersistentDisk: 301}
			params := NewUpdateParams(oldSpec, newSpec)

			state, err := params.JobNextState()
			Expect(err).ToNot(HaveOccurred())
			Expect(state).To(Equal(`{"persistent_disk":301}`))
		})
	})
})
