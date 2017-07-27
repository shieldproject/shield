package template_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/director/template"
)

var _ = Describe("VarsEnvArg", func() {
	Describe("UnmarshalFlag", func() {
		var (
			arg VarsEnvArg
		)

		BeforeEach(func() {
			arg = VarsEnvArg{}
		})

		It("sets read vars that only match given prefix", func() {
			arg.EnvironFunc = func() []string {
				return []string{"something=var3", "name_key1=var1", "name_key2=var2"}
			}

			err := (&arg).UnmarshalFlag("name")
			Expect(err).ToNot(HaveOccurred())
			Expect(arg.Vars).To(Equal(StaticVariables{
				"key1": "var1",
				"key2": "var2",
			}))
		})

		It("allows values with equal signs", func() {
			arg.EnvironFunc = func() []string { return []string{"name_key1=var1=foo"} }

			err := (&arg).UnmarshalFlag("name")
			Expect(err).ToNot(HaveOccurred())
			Expect(arg.Vars).To(Equal(StaticVariables{"key1": "var1=foo"}))
		})

		It("uses native os.Environ if EnvironFunc is not given", func() {
			os.Setenv("bosh_var_env_arg_test_key", "val")

			err := (&arg).UnmarshalFlag("bosh_var_env_arg_test")
			Expect(err).ToNot(HaveOccurred())
			Expect(arg.Vars).To(Equal(StaticVariables{"key": "val"}))
		})

		It("returns objects", func() {
			arg.EnvironFunc = func() []string {
				return []string{"name_key=name1: \n  key: value"}
			}

			err := (&arg).UnmarshalFlag("name")
			Expect(err).ToNot(HaveOccurred())
			Expect(arg.Vars["key"].(map[interface{}]interface{})["name1"]).To(Equal(map[interface{}]interface{}{"key": "value"}))
		})

		It("returns yaml parsed objects of expected type", func() {
			arg.EnvironFunc = func() []string {
				return []string{"name_int=1", "name_not_nil=nil", "name_nil2=", "name_nil3=~", "name_bool=true", "name_str=\"\""}
			}

			err := (&arg).UnmarshalFlag("name")
			Expect(err).ToNot(HaveOccurred())
			Expect(arg.Vars).To(Equal(StaticVariables{
				"int":     1,
				"not_nil": "nil",
				"nil2":    nil,
				"nil3":    nil,
				"bool":    true,
				"str":     "",
			}))
		})

		It("returns an error if environment contains empty entry (invalid)", func() {
			arg.EnvironFunc = func() []string { return []string{""} }

			err := (&arg).UnmarshalFlag("name")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected environment variable to be key-value pair"))
		})

		It("returns an error if environment variable cannot be unmarshaled", func() {
			arg.EnvironFunc = func() []string { return []string{"name_key=:"} }

			err := (&arg).UnmarshalFlag("name")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Deserializing YAML from environment variable 'name_key'"))
		})

		It("returns an error when it's empty", func() {
			err := (&arg).UnmarshalFlag("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected environment variable prefix to be non-empty"))
		})
	})
})
