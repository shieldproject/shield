package template_test

import (
	"errors"

	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/director/template"
)

var _ = Describe("VarsFileArg", func() {
	Describe("UnmarshalFlag", func() {
		var (
			fs  *fakesys.FakeFileSystem
			arg VarsFileArg
		)

		BeforeEach(func() {
			fs = fakesys.NewFakeFileSystem()
			arg = VarsFileArg{FS: fs}
		})

		It("sets read vars", func() {
			fs.WriteFileString("/some/path", "name1: var1\nname2: var2")

			err := (&arg).UnmarshalFlag("/some/path")
			Expect(err).ToNot(HaveOccurred())
			Expect(arg.Vars).To(Equal(StaticVariables{
				"name1": "var1",
				"name2": "var2",
			}))
		})

		It("returns objects", func() {
			fs.WriteFileString("/some/path", "name1: \n  key: value")

			err := (&arg).UnmarshalFlag("/some/path")
			Expect(err).ToNot(HaveOccurred())
			Expect(arg.Vars["name1"]).To(Equal(map[interface{}]interface{}{"key": "value"}))
		})

		It("returns yaml parsed objects of expected type", func() {
			fs.WriteFileString("/some/path", "name1: 1\nname2: nil\nname3: true\nname4: \"\"\nname5: \nname6: ~\n")

			err := (&arg).UnmarshalFlag("/some/path")
			Expect(err).ToNot(HaveOccurred())
			Expect(arg.Vars).To(Equal(StaticVariables{
				"name1": 1,
				"name2": "nil",
				"name3": true,
				"name4": "",
				"name5": nil,
				"name6": nil,
			}))
		})

		It("returns an error if reading file fails", func() {
			fs.WriteFileString("/some/path", "content")
			fs.ReadFileError = errors.New("fake-err")

			err := (&arg).UnmarshalFlag("/some/path")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("returns an error if parsing file fails", func() {
			fs.WriteFileString("/some/path", "content")

			err := (&arg).UnmarshalFlag("/some/path")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Deserializing variables file '/some/path'"))
		})

		It("returns an error when it's empty", func() {
			err := (&arg).UnmarshalFlag("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected file path to be non-empty"))
		})
	})
})
