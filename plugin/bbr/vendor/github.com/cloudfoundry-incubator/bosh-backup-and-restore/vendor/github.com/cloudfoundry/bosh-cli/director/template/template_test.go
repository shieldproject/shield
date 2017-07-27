package template_test

import (
	"errors"
	"fmt"

	"github.com/cppforlife/go-patch/patch"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/director/template"
)

var _ = Describe("Template", func() {
	It("can interpolate values into a struct with byte slice", func() {
		template := NewTemplate([]byte("((key))"))
		vars := StaticVariables{"key": "foo"}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte("foo\n")))
	})

	It("can interpolate multiple values into a byte slice", func() {
		template := NewTemplate([]byte("((key)): ((value))"))
		vars := StaticVariables{
			"key":   "foo",
			"value": "bar",
		}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte("foo: bar\n")))
	})

	It("can interpolate boolean values into a byte slice", func() {
		template := NewTemplate([]byte("otherstuff: ((boule))"))
		vars := StaticVariables{"boule": true}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte("otherstuff: true\n")))
	})

	It("can interpolate a different data types into a byte slice", func() {
		hashValue := map[string]interface{}{"key2": []string{"value1", "value2"}}
		template := NewTemplate([]byte("name1: ((name1))\nname2: ((name2))\nname3: ((name3))\nname4: ((name4))\nname5: ((name5))\nname6: ((name6))\n1234: value\n"))
		vars := StaticVariables{
			"name1": 1,
			"name2": "nil",
			"name3": true,
			"name4": "",
			"name5": nil,
			"name6": map[string]interface{}{"key": hashValue},
		}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte(`1234: value
name1: 1
name2: nil
name3: true
name4: ""
name5: null
name6:
  key:
    key2:
    - value1
    - value2
`)))
	})

	It("can interpolate different data types into a byte slice with !key", func() {
		template := NewTemplate([]byte("otherstuff: ((!boule))"))
		vars := StaticVariables{"boule": true}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte("otherstuff: true\n")))
	})

	It("return errors if there are missing variable keys and ExpectAllKeys is true", func() {
		template := NewTemplate([]byte(`
((key)): ((key2))
((key3)): 2
dup-key: ((key3))
((key4))_array:
- ((key_in_array))
`))
		vars := StaticVariables{"key3": "foo"}

		_, err := template.Evaluate(vars, nil, EvaluateOpts{ExpectAllKeys: true})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Expected to find variables: key\nkey2\nkey4\nkey_in_array"))
	})

	It("does not return error if there are missing variable keys and ExpectAllKeys is false", func() {
		template := NewTemplate([]byte("((key)): ((key2))\n((key3)): 2"))
		vars := StaticVariables{"key3": "foo"}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(Equal([]byte("((key)): ((key2))\nfoo: 2\n")))
	})

	It("return errors if there are unused variable keys and ExpectAllVarsUsed is true", func() {
		template := NewTemplate([]byte("((key2))"))
		vars := StaticVariables{"key1": "1", "key2": "2", "key3": "3"}

		_, err := template.Evaluate(vars, nil, EvaluateOpts{ExpectAllVarsUsed: true})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Expected to use variables: key1\nkey3"))
	})

	It("does not return error if there are unused variable keys and ExpectAllVarsUsed is false", func() {
		template := NewTemplate([]byte("((key)): ((key2))"))
		vars := StaticVariables{"key3": "foo"}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(Equal([]byte("((key)): ((key2))\n")))
	})

	It("return errors if there are not found and unused variables and ExpectAllKeys and ExpectAllVarsUsed are true", func() {
		template := NewTemplate([]byte("((key2))"))
		vars := StaticVariables{"key1": "1", "key3": "3"}

		_, err := template.Evaluate(vars, nil, EvaluateOpts{ExpectAllKeys: true, ExpectAllVarsUsed: true})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Expected to find variables: key2\nExpected to use variables: key1\nkey3"))
	})

	Context("When template is a number", func() {
		It("returns it", func() {
			template := NewTemplate([]byte(`1234`))
			vars := StaticVariables{"key": "not key"}

			result, err := template.Evaluate(vars, nil, EvaluateOpts{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal([]byte("1234\n")))
		})
	})

	Context("When variable has nil as value for key", func() {
		It("uses null", func() {
			template := NewTemplate([]byte("((key)): value"))
			vars := StaticVariables{"key": nil}

			result, err := template.Evaluate(vars, nil, EvaluateOpts{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal([]byte("null: value\n")))
		})
	})

	It("can interpolate unicode values into a byte slice", func() {
		template := NewTemplate([]byte("((Ω))"))
		vars := StaticVariables{"Ω": "☃"}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte("☃\n")))
	})

	It("can interpolate keys with dashes and underscores into a byte slice", func() {
		template := NewTemplate([]byte("((with-a-dash)): ((with_an_underscore))"))
		vars := StaticVariables{
			"with-a-dash":        "dash",
			"with_an_underscore": "underscore",
		}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte("dash: underscore\n")))
	})

	It("can interpolate a secret key in the middle of a string", func() {
		template := NewTemplate([]byte("url: https://((ip))"))
		vars := StaticVariables{
			"ip": "10.0.0.0",
		}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte("url: https://10.0.0.0\n")))
	})

	It("can interpolate multiple secret keys in the middle of a string", func() {
		template := NewTemplate([]byte("uri: nats://nats:((password))@((ip)):4222"))
		vars := StaticVariables{
			"password": "secret",
			"ip":       "10.0.0.0",
		}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte("uri: nats://nats:secret@10.0.0.0:4222\n")))
	})

	It("can interpolate multiple secret keys in the middle of a string even if keys have ! marks", func() {
		template := NewTemplate([]byte("uri: nats://nats:((!password))@((ip)):4222"))
		vars := StaticVariables{
			"password": "secret",
			"ip":       "10.0.0.0",
		}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte("uri: nats://nats:secret@10.0.0.0:4222\n")))
	})

	It("can interpolate multiple keys of type string and int in the middle of a string", func() {
		template := NewTemplate([]byte("address: ((ip)):((port))"))
		vars := StaticVariables{
			"port": 4222,
			"ip":   "10.0.0.0",
		}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte("address: 10.0.0.0:4222\n")))
	})

	It("raises error when interpolating an unsupported type in the middle of a string", func() {
		template := NewTemplate([]byte("address: ((definition)):((eulers_number))"))
		vars := StaticVariables{
			"eulers_number": 2.717,
			"definition":    "natural_log",
		}

		_, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("2.717"))
		Expect(err.Error()).To(ContainSubstring("eulers_number"))
	})

	It("can interpolate a single key multiple times in the middle of a string", func() {
		template := NewTemplate([]byte("acct_and_password: ((user)):((user))"))
		vars := StaticVariables{
			"user": "nats",
		}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte("acct_and_password: nats:nats\n")))
	})

	It("can interpolate values into the middle of a key", func() {
		template := NewTemplate([]byte("((iaas))_cpi: props"))
		vars := StaticVariables{
			"iaas": "aws",
		}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte("aws_cpi: props\n")))
	})

	It("can interpolate the same value multiple times into a byte slice", func() {
		template := NewTemplate([]byte("((key)): ((key))"))
		vars := StaticVariables{"key": "foo"}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte("foo: foo\n")))
	})

	It("can interpolate values with strange newlines", func() {
		template := NewTemplate([]byte("((key))"))
		vars := StaticVariables{"key": "this\nhas\nmany\nlines"}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte("|-\n  this\n  has\n  many\n  lines\n")))
	})

	It("ignores if operation is not specified", func() {
		template := NewTemplate([]byte("((key))"))
		vars := StaticVariables{"key": "val"}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte("val\n")))
	})

	It("ignores an invalid input", func() {
		template := NewTemplate([]byte("(()"))
		vars := StaticVariables{}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte("(()\n")))
	})

	It("strips away ! from variable keys", func() {
		template := NewTemplate([]byte("abc: ((!key))\nxyz: [((!key))]"))
		vars := StaticVariables{"key": "val"}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte("abc: val\nxyz:\n- val\n")))
	})

	It("can run operations to modify document", func() {
		template := NewTemplate([]byte("a: b"))
		vars := StaticVariables{}
		ops := patch.ReplaceOp{Path: patch.MustNewPointerFromString("/a"), Value: "c"}

		result, err := template.Evaluate(vars, ops, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte("a: c\n")))
	})

	It("interpolates after running operations", func() {
		template := NewTemplate([]byte("a: b"))
		vars := StaticVariables{"c": "x"}
		ops := patch.ReplaceOp{Path: patch.MustNewPointerFromString("/a"), Value: "((c))"}

		result, err := template.Evaluate(vars, ops, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte("a: x\n")))
	})

	It("returns an error if variables added by operations are not found", func() {
		template := NewTemplate([]byte("a: b"))
		vars := StaticVariables{}
		ops := patch.ReplaceOp{Path: patch.MustNewPointerFromString("/a"), Value: "((c))"}

		_, err := template.Evaluate(vars, ops, EvaluateOpts{ExpectAllKeys: true})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Expected to find variables: c"))
	})

	It("returns an error if operation fails", func() {
		template := NewTemplate([]byte("a: b"))
		vars := StaticVariables{}
		ops := patch.ReplaceOp{Path: patch.MustNewPointerFromString("/x/y"), Value: "c"}

		_, err := template.Evaluate(vars, ops, EvaluateOpts{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Expected to find a map key 'x' for path '/x' (found map keys: 'a')"))
	})

	It("runs PostVarSubstitutionOp after running regular operations and interpolation", func() {
		template := NewTemplate([]byte("a: b"))

		vars := StaticVariables{
			"c": map[interface{}]interface{}{"d": "e"},
		}
		ops := patch.ReplaceOp{Path: patch.MustNewPointerFromString("/a"), Value: "((c))"}
		opts := EvaluateOpts{
			PostVarSubstitutionOp: patch.FindOp{Path: patch.MustNewPointerFromString("/a/d")},
		}

		result, err := template.Evaluate(vars, ops, opts)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte("e\n")))
	})

	It("returns an error if PostVarSubstitutionOp fails", func() {
		template := NewTemplate([]byte("a: b"))
		vars := StaticVariables{}
		opts := EvaluateOpts{
			PostVarSubstitutionOp: patch.FindOp{Path: patch.MustNewPointerFromString("/x")},
		}

		_, err := template.Evaluate(vars, nil, opts)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Expected to find a map key 'x' for path '/x' (found map keys: 'a')"))
	})

	It("returns raw bytes of a string if UnescapedMultiline is true", func() {
		template := NewTemplate([]byte("value"))

		result, err := template.Evaluate(StaticVariables{}, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte("value\n")))

		result, err = template.Evaluate(StaticVariables{}, nil, EvaluateOpts{UnescapedMultiline: true})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte("value\n")))
	})

	It("provides associated variable definition if found so that variables can be generated", func() {
		template := NewTemplate([]byte(`abc: ((!key1))
variables:
- name: key2
  type: key2-type
  options: {key2-opt: key2-opt-val}
- name: key1
  type: key1-type
xyz: [((!key2))]
`))

		vars := &FakeVariables{
			GetFunc: func(varDef VariableDefinition) (interface{}, bool, error) {
				switch varDef.Name {
				case "key1":
					Expect(varDef).To(Equal(VariableDefinition{Name: "key1", Type: "key1-type"}))
					return "key1-val", true, nil

				case "key2":
					Expect(varDef).To(Equal(VariableDefinition{
						Name:    "key2",
						Type:    "key2-type",
						Options: map[interface{}]interface{}{"key2-opt": "key2-opt-val"},
					}))
					return "key2-val", true, nil

				default:
					panic(fmt.Sprintf("Unexpected variable definiton: %#v", varDef))
				}
			},
		}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte(`abc: key1-val
variables: []
xyz:
- key2-val
`)))
	})

	It("allows operations to modify variable definitions for interpolation", func() {
		template := NewTemplate([]byte("abc: ((key))"))

		vars := &FakeVariables{
			GetFunc: func(varDef VariableDefinition) (interface{}, bool, error) {
				Expect(varDef).To(Equal(VariableDefinition{Name: "key", Type: "key-type"}))
				return "key-val", true, nil
			},
		}

		op := patch.ReplaceOp{
			Path:  patch.MustNewPointerFromString("/variables?/-"),
			Value: map[interface{}]interface{}{"name": "key", "type": "key-type"},
		}

		result, err := template.Evaluate(vars, op, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte(`abc: key-val
variables: []
`)))
	})

	It("goes through variables in variable definitions in order (skipping typeless variables) before interpolating other variables", func() {
		template := NewTemplate([]byte(`abc: ((key))
variables:
- name: key1
  type: key1-type
- name: missing-type
- name: key2
  type: key2-type
`))

		var interpolationOrder []string

		vars := &FakeVariables{
			GetFunc: func(varDef VariableDefinition) (interface{}, bool, error) {
				interpolationOrder = append(interpolationOrder, varDef.Name)
				return "val", true, nil
			},
		}

		_, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(interpolationOrder).To(Equal([]string{"key1", "key2", "key", "key1", "missing-type", "key2"}))
	})

	It("returns error if any variable interpolation failed from variable definitions section", func() {
		template := NewTemplate([]byte(`abc: ((key1))
variables:
- name: key2
  type: key2-type
`))

		var interpolationOrder []string

		vars := &FakeVariables{
			GetFunc: func(varDef VariableDefinition) (interface{}, bool, error) {
				interpolationOrder = append(interpolationOrder, varDef.Name)
				return nil, true, errors.New("fake-err")
			},
		}

		_, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Getting all variables from variable definitions sections: fake-err"))
		Expect(interpolationOrder).To(Equal([]string{"key2"}))
	})

	It("returns error if any variable interpolation failed inside of variable definition option section", func() {
		template := NewTemplate([]byte(`
variables:
- name: key2
  type: key2-type
  options:
    var: ((key1))
`))

		var interpolationOrder []string

		vars := &FakeVariables{
			GetFunc: func(varDef VariableDefinition) (interface{}, bool, error) {
				interpolationOrder = append(interpolationOrder, varDef.Name)
				return nil, true, errors.New("fake-err")
			},
		}

		_, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Interpolating variable 'key2' definition options"))
		Expect(err.Error()).To(ContainSubstring("fake-err"))
	})

	It("returns error if any variable interpolation failed inside of variable definition option section", func() {
		template := NewTemplate([]byte(`
variables:
- name: key2
  type: key2-type
  options:
    var: ((key1))
`))

		vars := &FakeVariables{
			GetFunc: func(varDef VariableDefinition) (interface{}, bool, error) {
				return nil, true, errors.New("fake-err")
			},
		}

		_, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Interpolating variable 'key2' definition options"))
		Expect(err.Error()).To(ContainSubstring("fake-err"))
	})

	It("returns error if any variable interpolation failed inside of variable definition option section", func() {
		template := NewTemplate([]byte(`
other_key: ((other_key))
key2: ((key2))
variables:
- name: key2
  type: key2-type
  options:
    var: ((key1))
`))

		var queriedNames []string

		vars := &FakeVariables{
			GetFunc: func(varDef VariableDefinition) (interface{}, bool, error) {
				queriedNames = append(queriedNames, varDef.Name)
				return nil, false, nil
			},
		}

		_, err := template.Evaluate(vars, nil, EvaluateOpts{ExpectAllKeys: true})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Expected to find variables: key1\nother_key"))

		Expect(queriedNames).To(ContainElement("key1"))
		Expect(queriedNames).To(ContainElement("other_key"))
		Expect(queriedNames).ToNot(ContainElement("key2"), "because it depends on presence of key1 which is not found")
	})

	It("returns error if variables are recursively defined", func() {
		template := NewTemplate([]byte(`
variables:
- name: key1
  type: key1-type
  options:
    var: ((key2))
- name: key2
  type: key2-type
  options:
    var: ((key1))
`))

		_, err := template.Evaluate(StaticVariables{}, nil, EvaluateOpts{ExpectAllKeys: true})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Detected recursion"))
		Expect(err.Error()).To(ContainSubstring("Interpolating variable 'key1' definition options"))
		Expect(err.Error()).To(ContainSubstring("Interpolating variable 'key2' definition options"))
	})

	It("returns error if variables are referenced multiple times (when checking for recursion)", func() {
		template := NewTemplate([]byte(`
top_level: ((top_level))
top_level2: ((top_level))
variables:
- name: key1
  type: key1-type
  options:
    options_level: ((options_level))
    options_level2: ((options_level))
- name: key2
  type: key2-type
  options:
    var: ((key1))
`))

		_, err := template.Evaluate(StaticVariables{}, nil, EvaluateOpts{})
		Expect(err).ToNot(HaveOccurred())
	})

	It("allows to access sub key of an interpolated value via dot syntax", func() {
		template := NewTemplate([]byte("((key.subkey))"))
		vars := StaticVariables{
			"key": map[interface{}]interface{}{"subkey": "e"},
		}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte("e\n")))
	})

	It("allows to generate variables that use sub key via dot syntax", func() {
		template := NewTemplate([]byte(`ca: ((cert.private_key))
variables:
- name: cert
  type: cert-type
  options:
    cert-opt: cert-opt-val
    key1: ((key1.subkey))
`))

		vars := &FakeVariables{
			GetFunc: func(varDef VariableDefinition) (interface{}, bool, error) {
				switch varDef.Name {
				case "cert":
					Expect(varDef).To(Equal(VariableDefinition{
						Name: "cert",
						Type: "cert-type",
						Options: map[interface{}]interface{}{
							"cert-opt": "cert-opt-val",
							"key1":     "key1-subkey",
						},
					}))
					return map[interface{}]interface{}{"private_key": "private-key-val"}, true, nil

				case "key1":
					Expect(varDef).To(Equal(VariableDefinition{Name: "key1"}))
					return map[interface{}]interface{}{"subkey": "key1-subkey"}, true, nil

				default:
					panic(fmt.Sprintf("Unexpected variable definiton: %#v", varDef))
				}
			},
		}

		opts := EvaluateOpts{
			PostVarSubstitutionOp: patch.FindOp{Path: patch.MustNewPointerFromString("/ca")},
		}

		result, err := template.Evaluate(vars, nil, opts)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte("private-key-val\n")))
	})

	It("keeps variable definitons around if that variable was not found using vars-store flag", func() {
		template := NewTemplate([]byte(`
variables:
- name: not-found
  type: password
`))

		vars := &FakeVariables{
			GetFunc: func(varDef VariableDefinition) (interface{}, bool, error) {
				return nil, false, nil
			},
		}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte(`variables:
- name: not-found
  type: password
`)))
	})

	It("keeps variable definitons around if that variable was not found using var flag", func() {
		template := NewTemplate([]byte(`
variables:
- name: not-found
`))

		vars := &FakeVariables{
			GetFunc: func(varDef VariableDefinition) (interface{}, bool, error) {
				return nil, false, nil
			},
		}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte(`variables:
- name: not-found
`)))
	})

	It("does not keep variable definition around if variable was found using vars-store flag", func() {
		template := NewTemplate([]byte(`
variables:
- name: found
  type: password
- name: not-found
  type: password
`))

		vars := &FakeVariables{
			GetFunc: func(varDef VariableDefinition) (interface{}, bool, error) {
				switch varDef.Name {
				case "found":
					return nil, true, nil

				case "not-found":
					return nil, false, nil

				default:
					panic(fmt.Sprintf("Unexpected variable definiton: %#v", varDef))
				}
			},
		}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte(`variables:
- name: not-found
  type: password
`)))
	})

	It("does not keep variable definition around if variable was found using var flag", func() {
		template := NewTemplate([]byte(`
variables:
- name: found
- name: not-found
  type: password
`))

		vars := &FakeVariables{
			GetFunc: func(varDef VariableDefinition) (interface{}, bool, error) {
				switch varDef.Name {
				case "found":
					return nil, true, nil

				case "not-found":
					return nil, false, nil

				default:
					panic(fmt.Sprintf("Unexpected variable definiton: %#v", varDef))
				}
			},
		}

		result, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal([]byte(`variables:
- name: not-found
  type: password
`)))
	})

	It("returns an error if variable is not found and is being used with a sub key", func() {
		template := NewTemplate([]byte("((key.subkey_not_found))"))
		vars := StaticVariables{}

		_, err := template.Evaluate(vars, nil, EvaluateOpts{ExpectAllKeys: true})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Expected to find variables: key"))
	})

	It("returns an error if accessing sub key of an interpolated value fails", func() {
		template := NewTemplate([]byte("((key.subkey_not_found))"))
		vars := StaticVariables{
			"key": map[interface{}]interface{}{"subkey": "e"},
		}

		_, err := template.Evaluate(vars, nil, EvaluateOpts{ExpectAllKeys: true})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Expected to find a map key 'subkey_not_found'"))
	})

	It("returns error if finding variable fails", func() {
		template := NewTemplate([]byte("((key))"))
		vars := &FakeVariables{GetErr: errors.New("fake-err")}

		_, err := template.Evaluate(vars, nil, EvaluateOpts{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("fake-err"))
	})
})
