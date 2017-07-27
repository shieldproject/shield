package index_test

import (
	. "github.com/cloudfoundry/bosh-cli/index"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("InMemoryIndex", func() {
	var (
		index Index
	)

	BeforeEach(func() {
		index = NewInMemoryIndex()
	})

	Describe("Save/Find", func() {
		It("returns true if item is found by key", func() {
			k1 := Key{Key: "key-1"}
			v1 := Value{Name: "value-1", Count: 1}
			err := index.Save(k1, v1)
			Expect(err).ToNot(HaveOccurred())

			var value Value

			err = index.Find(k1, &value)
			Expect(err).ToNot(HaveOccurred())
			Expect(err).ToNot(Equal(ErrNotFound))

			Expect(value).To(Equal(v1))
		})

		It("returns false if item is not found by key", func() {
			k1 := Key{Key: "key-1"}
			v1 := Value{Name: "value-1", Count: 1}
			err := index.Save(k1, v1)
			Expect(err).ToNot(HaveOccurred())

			var value Value

			err = index.Find(Key{Key: "key-2"}, &value)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(ErrNotFound))

			Expect(value).To(Equal(Value{}))
		})

		Context("when the values include arrays", func() {
			It("returns true and correctly deserializes item with nil", func() {
				k1 := Key{Key: "key-1"}
				v1 := ArrayValue{} // nil
				err := index.Save(k1, v1)
				Expect(err).ToNot(HaveOccurred())

				var value ArrayValue

				err = index.Find(k1, &value)
				Expect(err).ToNot(HaveOccurred())
				Expect(err).ToNot(Equal(ErrNotFound))

				Expect(value).To(Equal(v1))
			})

			It("returns true and correctly deserializes item with empty slice", func() {
				k1 := Key{Key: "key-1"}
				v1 := ArrayValue{Names: []string{}} // empty slice
				err := index.Save(k1, v1)
				Expect(err).ToNot(HaveOccurred())

				var value ArrayValue

				err = index.Find(k1, &value)
				Expect(err).ToNot(HaveOccurred())
				Expect(err).ToNot(Equal(ErrNotFound))

				Expect(value).To(Equal(v1))
			})

			It("returns true and correctly deserializes item with multiple items", func() {
				k1 := Key{Key: "key-1"}
				v1 := ArrayValue{Names: []string{"name-1-1", "name-1-2"}} // multiple
				err := index.Save(k1, v1)
				Expect(err).ToNot(HaveOccurred())

				var value ArrayValue

				err = index.Find(k1, &value)
				Expect(err).ToNot(HaveOccurred())
				Expect(err).ToNot(Equal(ErrNotFound))

				Expect(value).To(Equal(v1))
			})
		})

		Context("when the values include structs", func() {
			It("returns true and correctly deserializes item with zero value", func() {
				k1 := Key{Key: "key-1"}
				v1 := StructValue{} // zero value
				err := index.Save(k1, v1)
				Expect(err).ToNot(HaveOccurred())

				var value StructValue

				err = index.Find(k1, &value)
				Expect(err).ToNot(HaveOccurred())
				Expect(err).ToNot(Equal(ErrNotFound))

				Expect(value).To(Equal(v1))
			})

			It("returns true and correctly deserializes item with filled struct", func() {
				k1 := Key{Key: "key-1"}
				v1 := StructValue{
					Name: Name{
						First: "first-name-1", Last: "last-name-1",
					},
				} // struct
				err := index.Save(k1, v1)
				Expect(err).ToNot(HaveOccurred())

				var value StructValue

				err = index.Find(k1, &value)
				Expect(err).ToNot(HaveOccurred())
				Expect(err).ToNot(Equal(ErrNotFound))

				Expect(value).To(Equal(v1))
			})

			It("returns saved value of a modified pointer", func() {
				middleName := "middle-name-1"
				k1 := Key{Key: "key-1"}
				v1 := StructValue{
					Name: Name{
						First:  "first-name-1",
						Middle: &middleName,
						Last:   "last-name-1",
					},
				}
				err := index.Save(k1, v1)
				Expect(err).ToNot(HaveOccurred())

				middleName = "middle-name-2"

				var value StructValue

				err = index.Find(k1, &value)
				Expect(err).ToNot(HaveOccurred())

				Expect(*value.Name.Middle).To(Equal("middle-name-1"))
			})
		})
	})
})
