package index_test

import (
	. "github.com/cloudfoundry/bosh-cli/index"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("FileIndex", func() {
	var (
		fs            boshsys.FileSystem
		indexFilePath string
		index         FileIndex
	)

	BeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		fs = boshsys.NewOsFileSystem(logger)

		file, err := fs.TempFile("file-index")
		Expect(err).ToNot(HaveOccurred())

		indexFilePath = file.Name()

		err = file.Close()
		Expect(err).ToNot(HaveOccurred())

		err = fs.RemoveAll(indexFilePath)
		Expect(err).ToNot(HaveOccurred())

		index = NewFileIndex(indexFilePath, fs)
	})

	AfterEach(func() {
		err := fs.RemoveAll(indexFilePath)
		Expect(err).ToNot(HaveOccurred())
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
				v1 := StructValue{Name: Name{First: "first-name-1", Last: "last-name-1"}} // struct
				err := index.Save(k1, v1)
				Expect(err).ToNot(HaveOccurred())

				var value StructValue

				err = index.Find(k1, &value)
				Expect(err).ToNot(HaveOccurred())
				Expect(err).ToNot(Equal(ErrNotFound))

				Expect(value).To(Equal(v1))
			})
		})

		Context("when a new FileIndex is constructed backed by the same file", func() {
			var (
				index2 FileIndex
			)

			BeforeEach(func() {
				index2 = NewFileIndex(indexFilePath, fs)
			})

			It("returns the value saved by the original FileIndex", func() {
				err := index.Save(Key{Key: "key-1"}, Value{Name: "value-1", Count: 1})
				Expect(err).ToNot(HaveOccurred())

				var value Value

				err = index2.Find(Key{Key: "key-1"}, &value)
				Expect(err).ToNot(HaveOccurred())
				Expect(err).ToNot(Equal(ErrNotFound))

				Expect(value).To(Equal(Value{Name: "value-1", Count: 1}))
			})
		})
	})
})
