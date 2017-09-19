package cmd_test

import (
	"errors"
	"fmt"

	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	fakecfgtypes "github.com/cloudfoundry/config-server/types/typesfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	boshtpl "github.com/cloudfoundry/bosh-cli/director/template"
)

var _ = Describe("VarsFSStore", func() {
	var (
		fs    *fakesys.FakeFileSystem
		store VarsFSStore
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		store = VarsFSStore{FS: fs}
	})

	Describe("Get", func() {
		BeforeEach(func() {
			err := (&store).UnmarshalFlag("/file")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns value and found if store finds variable", func() {
			fs.WriteFileString("/file", "key: val")

			val, found, err := store.Get(boshtpl.VariableDefinition{Name: "key"})
			Expect(val).To(Equal("val"))
			Expect(found).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when store does not find variable", func() {
			BeforeEach(func() {
				fs.WriteFileString("/file", "key: val")
			})

			It("returns nil and not found if variable type is not available", func() {
				val, found, err := store.Get(boshtpl.VariableDefinition{Name: "key2"})
				Expect(val).To(BeNil())
				Expect(found).To(BeFalse())
				Expect(err).ToNot(HaveOccurred())
			})

			It("tries to generate value and save it if variable type is available", func() {
				val, found, err := store.Get(boshtpl.VariableDefinition{Name: "key2", Type: "password"})
				Expect(len(val.(string))).To(BeNumerically(">", 10))
				Expect(found).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())

				Expect(fs.ReadFileString("/file")).To(Equal(fmt.Sprintf("key: val\nkey2: %s\n", val.(string))))
			})

			It("returns error if variable type is not known", func() {
				val, found, err := store.Get(boshtpl.VariableDefinition{Name: "key2", Type: "unknown"})
				Expect(val).To(BeNil())
				Expect(found).To(BeFalse())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Generating variable 'key2': Unsupported value type: unknown"))
			})

			It("returns error if generating variable fails", func() {
				generator := &fakecfgtypes.FakeValueGenerator{}
				generator.GenerateReturns(nil, errors.New("fake-err"))

				factory := &fakecfgtypes.FakeValueGeneratorFactory{}
				factory.GetGeneratorReturns(generator, nil)

				store.ValueGeneratorFactory = factory

				val, found, err := store.Get(boshtpl.VariableDefinition{Name: "key2", Type: "type"})
				Expect(val).To(BeNil())
				Expect(found).To(BeFalse())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Generating variable 'key2': fake-err"))
			})

			It("returns error if writing file fails", func() {
				fs.WriteFileError = errors.New("fake-err")

				val, found, err := store.Get(boshtpl.VariableDefinition{Name: "key2", Type: "password"})
				Expect(val).To(BeNil())
				Expect(found).To(BeFalse())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})
		})

		Context("when store does not find backing file", func() {
			It("tries to generate value and save it if variable type is available", func() {
				val, found, err := store.Get(boshtpl.VariableDefinition{Name: "key2", Type: "password"})
				Expect(len(val.(string))).To(BeNumerically(">", 10))
				Expect(found).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())
			})
		})

		It("returns error if reading file fails", func() {
			fs.WriteFileString("/file", "contents")
			fs.ReadFileError = errors.New("fake-err")

			_, _, err := store.Get(boshtpl.VariableDefinition{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("returns an error if parsing file fails", func() {
			fs.WriteFileString("/file", "content")

			_, _, err := store.Get(boshtpl.VariableDefinition{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Deserializing variables file store '/file'"))
		})
	})

	Describe("List", func() {
		BeforeEach(func() {
			err := (&store).UnmarshalFlag("/file")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns list of names without considering nested keys", func() {
			fs.WriteFileString("/file", "key1: val\nkey2: {key3: nested}")

			defs, err := store.List()
			Expect(defs).To(ConsistOf([]boshtpl.VariableDefinition{{Name: "key1"}, {Name: "key2"}}))
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns empty list if backing file does not exist", func() {
			defs, err := store.List()
			Expect(defs).To(BeEmpty())
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if parsing file fails", func() {
			fs.WriteFileString("/file", "content")

			_, err := store.List()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Deserializing variables file store '/file'"))
		})

		It("returns error if reading file fails", func() {
			fs.WriteFileString("/file", "contents")
			fs.ReadFileError = errors.New("fake-err")

			_, err := store.List()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})

	Describe("IsSet", func() {
		It("returns true if store is configured with file path", func() {
			err := (&store).UnmarshalFlag("/file")
			Expect(err).ToNot(HaveOccurred())
			Expect(store.IsSet()).To(BeTrue())
		})

		It("returns false if store is not configured", func() {
			Expect(store.IsSet()).To(BeFalse())
		})
	})

	Describe("UnmarshalFlag", func() {
		It("returns error if file path is empty", func() {
			err := (&store).UnmarshalFlag("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected file path to be non-empty"))
		})

		It("returns error if path cannot be expanded", func() {
			fs.ExpandPathErr = errors.New("fake-err")

			err := (&store).UnmarshalFlag("/file")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
