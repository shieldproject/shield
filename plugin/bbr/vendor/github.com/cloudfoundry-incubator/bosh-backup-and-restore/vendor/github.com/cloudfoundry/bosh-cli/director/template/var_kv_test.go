package template_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/director/template"
)

var _ = Describe("VarKV", func() {
	Describe("UnmarshalFlag", func() {
		var (
			arg VarKV
		)

		BeforeEach(func() {
			arg = VarKV{}
		})

		It("sets name and value", func() {
			err := (&arg).UnmarshalFlag("name=val")
			Expect(err).ToNot(HaveOccurred())
			Expect(arg).To(Equal(VarKV{Name: "name", Value: "val"}))
		})

		It("sets name and value when value contains a `=`", func() {
			err := (&arg).UnmarshalFlag("public_key=ssh-rsa G4/+VHa1aw==")
			Expect(err).ToNot(HaveOccurred())
			Expect(arg).To(Equal(VarKV{Name: "public_key", Value: "ssh-rsa G4/+VHa1aw=="}))
		})

		It("returns error if string does not have 2 pieces", func() {
			err := (&arg).UnmarshalFlag("val")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected var 'val' to be in format 'name=value'"))
		})

		It("returns error if name is empty", func() {
			err := (&arg).UnmarshalFlag("=val")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected var '=val' to specify non-empty name"))
		})

		It("returns error if value is empty", func() {
			err := (&arg).UnmarshalFlag("name=")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected var 'name=' to specify non-empty value"))
		})
		Context("When key/value is a yml", func() {
			It("works", func() {
				err := (&arg).UnmarshalFlag("name=key1: 1\nkey2: true\nkey3:\n  key31: str")
				Expect(err).ToNot(HaveOccurred())
				Expect(arg.Value.(map[interface{}]interface{})["key1"]).To(Equal(1))
				Expect(arg.Value.(map[interface{}]interface{})["key2"]).To(Equal(true))
				val3 := arg.Value.(map[interface{}]interface{})["key3"]
				Expect(val3).To(Equal(map[interface{}]interface{}{"key31": "str"}))
			})
		})
	})
})
