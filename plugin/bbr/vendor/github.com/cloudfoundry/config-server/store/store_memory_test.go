package store_test

import (
	. "github.com/cloudfoundry/config-server/store"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StoreMemory", func() {

	Describe("Given a properly initialized MemoryStore", func() {
		var store MemoryStore

		BeforeEach(func() {
			store = NewMemoryStore()
		})

		Context("Put", func() {
			It("should not return error when adding a string type value", func() {
				_, err := store.Put("key", "value")
				Expect(err).To(BeNil())
			})

			It("returns the ID of the created configuration", func() {
				id, _ := store.Put("key", "value")
				Expect(id).To(Equal("0"))
			})

			It("generates a unique id for new record", func() {
				store.Put("key1", "value1")
				values1, _ := store.GetByName("key1")

				Expect(values1).ToNot(BeNil())
				Expect(len(values1)).To(Equal(1))
				Expect(values1[0]).To(Equal(Configuration{ID: "0", Name: "key1", Value: "value1"}))

				store.Put("key2", "value2")
				values2, _ := store.GetByName("key2")

				Expect(values2).ToNot(BeNil())
				Expect(len(values2)).To(Equal(1))
				Expect(values2[0]).To(Equal(Configuration{ID: "1", Name: "key2", Value: "value2"}))
			})

			It("generates unique ids for duplicate entries", func() {
				id1, err := store.Put("key1", "value1")
				Expect(err).To(BeNil())
				Expect(id1).ToNot(BeNil())

				id2, err := store.Put("key1", "value1")
				Expect(err).To(BeNil())
				Expect(id2).ToNot(BeNil())

				Expect(id1).ToNot(Equal(id2))
			})
		})

		Context("GetByName", func() {
			It("should return ALL associated values sorted by ID", func() {
				store.Put("some_name", "some_value")
				store.Put("some_name", "some_value")
				store.Put("some_name", "some_other_value")

				returnedValues, err := store.GetByName("some_name")
				Expect(err).To(BeNil())

				Expect(returnedValues[0]).To(Equal(Configuration{
					ID:    "2",
					Name:  "some_name",
					Value: "some_other_value",
				}))

				Expect(returnedValues[1]).To(Equal(Configuration{
					ID:    "1",
					Name:  "some_name",
					Value: "some_value",
				}))

				Expect(returnedValues[2]).To(Equal(Configuration{
					ID:    "0",
					Name:  "some_name",
					Value: "some_value",
				}))
			})
		})

		Context("GetById", func() {
			It("should return associated value", func() {
				store.Put("some_name", "some_value")

				configuration, err := store.GetByID("0")
				Expect(err).To(BeNil())
				Expect(configuration).To(Equal(Configuration{
					ID:    "0",
					Name:  "some_name",
					Value: "some_value",
				}))
			})
		})

		Context("Delete", func() {
			Context("Name exists", func() {
				BeforeEach(func() {
					store.Put("some_name", "some_value")
					store.Put("some_name", "some_value")

					values, err := store.GetByName("some_name")
					Expect(err).To(BeNil())
					Expect(values[0]).To(Equal(Configuration{
						ID:    "1",
						Name:  "some_name",
						Value: "some_value",
					}))
					Expect(values[1]).To(Equal(Configuration{
						ID:    "0",
						Name:  "some_name",
						Value: "some_value",
					}))
				})

				It("removes all values", func() {
					store.Delete("some_name")
					values, err := store.GetByName("some_name")
					Expect(err).To(BeNil())
					Expect(len(values)).To(Equal(0))
				})

				It("returns count of deleted rows", func() {
					deleted, err := store.Delete("some_name")
					Expect(err).To(BeNil())
					Expect(deleted).To(Equal(2))
				})
			})

			Context("Name does not exist", func() {
				It("returns count of deleted rows", func() {
					deleted, err := store.Delete("fake_key")
					Expect(deleted).To(Equal(0))
					Expect(err).To(BeNil())
				})
			})
		})
	})
})
