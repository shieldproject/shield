package template_test

import (
	"errors"

	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/director/template"
)

var _ = Describe("VarFileArg", func() {
	Describe("UnmarshalFlag", func() {
		var (
			fs  *fakesys.FakeFileSystem
			arg VarFileArg
		)

		BeforeEach(func() {
			fs = fakesys.NewFakeFileSystem()
			arg = VarFileArg{FS: fs}
		})

		It("sets name and value from a file", func() {
			fs.WriteFileString("/some/path", "val\nval")

			err := (&arg).UnmarshalFlag("name=/some/path")
			Expect(err).ToNot(HaveOccurred())
			Expect(arg.Vars).To(Equal(StaticVariables{"name": "val\nval"}))
		})

		It("sets name and value when value contains a `=`", func() {
			fs.WriteFileString("/some/path=", "val")

			err := (&arg).UnmarshalFlag("name=/some/path=")
			Expect(err).ToNot(HaveOccurred())
			Expect(arg.Vars).To(Equal(StaticVariables{"name": "val"}))
		})

		It("returns error if string does not have 2 pieces", func() {
			err := (&arg).UnmarshalFlag("val")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected var 'val' to be in format 'name=path'"))
		})

		It("returns error if name is empty", func() {
			err := (&arg).UnmarshalFlag("=val")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected var '=val' to specify non-empty name"))
		})

		It("returns error if value is empty", func() {
			err := (&arg).UnmarshalFlag("name=")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected var 'name=' to specify non-empty path"))
		})

		It("returns an error if reading file fails", func() {
			fs.WriteFileString("/some/path", "content")
			fs.ReadFileError = errors.New("fake-err")

			err := (&arg).UnmarshalFlag("var=/some/path")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("returns an error if expanding path fails", func() {
			fs.ExpandPathErr = errors.New("fake-err")

			err := (&arg).UnmarshalFlag("var=/some/path")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("returns an error when it's empty", func() {
			err := (&arg).UnmarshalFlag("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected var '' to be in format 'name=path'"))
		})
	})
})
