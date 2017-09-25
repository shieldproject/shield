package patch_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"

	. "github.com/cppforlife/go-patch/patch"
)

var _ = Describe("Integration", func() {
	It("works in a basic way", func() {
		inStr := `
releases:
- name: capi
  version: 0.1

instance_groups:
- name: cloud_controller
  instances: 0
  jobs:
  - name: cloud_controller
    release: capi

- name: uaa
  instances: 0
`

		var in interface{}

		err := yaml.Unmarshal([]byte(inStr), &in)
		Expect(err).ToNot(HaveOccurred())

		ops1Str := `
- type: replace
  path: /instance_groups/name=cloud_controller/instances
  value: 1

- type: replace
  path: /instance_groups/name=cloud_controller/jobs/name=cloud_controller/consumes?/db
  value:
    instances:
    - address: some-db.local
    properties:
      username: user
      password: pass

- type: replace
  path: /instance_groups/name=uaa/instances
  value: 1

- type: replace
  path: /instance_groups/-
  value:
    name: uaadb
    instances: 2
`

		var opDefs1 []OpDefinition

		err = yaml.Unmarshal([]byte(ops1Str), &opDefs1)
		Expect(err).ToNot(HaveOccurred())

		ops1, err := NewOpsFromDefinitions(opDefs1)
		Expect(err).ToNot(HaveOccurred())

		ops2Str := `
- type: replace
  path: /releases/name=capi/version
  value: latest
`

		var opDefs2 []OpDefinition

		err = yaml.Unmarshal([]byte(ops2Str), &opDefs2)
		Expect(err).ToNot(HaveOccurred())

		ops2, err := NewOpsFromDefinitions(opDefs2)
		Expect(err).ToNot(HaveOccurred())

		ops := append(ops1, ops2...)

		res, err := ops.Apply(in)
		Expect(err).ToNot(HaveOccurred())

		outStr := `
releases:
- name: capi
  version: latest

instance_groups:
- name: cloud_controller
  instances: 1
  jobs:
  - name: cloud_controller
    release: capi
    consumes:
      db:
        instances:
        - address: some-db.local
        properties:
          username: user
          password: pass

- name: uaa
  instances: 1

- name: uaadb
  instances: 2
`

		var out interface{}

		err = yaml.Unmarshal([]byte(outStr), &out)
		Expect(err).ToNot(HaveOccurred())

		Expect(res).To(Equal(out))
	})

	It("works with find op", func() {
		inStr := `
releases:
- name: capi
  version: 0.1

instance_groups:
- name: cloud_controller
  instances: 0
  jobs:
  - name: cloud_controller
    release: capi

- name: uaa
  instances: 0
`

		var in interface{}

		err := yaml.Unmarshal([]byte(inStr), &in)
		Expect(err).ToNot(HaveOccurred())

		path := MustNewPointerFromString("/instance_groups/name=cloud_controller")

		res, err := FindOp{Path: path}.Apply(in)
		Expect(err).ToNot(HaveOccurred())

		outStr := `
name: cloud_controller
instances: 0
jobs:
- name: cloud_controller
  release: capi
`

		var out interface{}

		err = yaml.Unmarshal([]byte(outStr), &out)
		Expect(err).ToNot(HaveOccurred())

		Expect(res).To(Equal(out))
	})

	It("shows custom error messages", func() {
		inStr := `
releases:
- name: capi
  version: 0.1
`

		var in interface{}

		err := yaml.Unmarshal([]byte(inStr), &in)
		Expect(err).ToNot(HaveOccurred())

		opsStr := `
- type: remove
  path: /releases/0/not-there
  error: "Custom error message"
`

		var opDefs []OpDefinition

		err = yaml.Unmarshal([]byte(opsStr), &opDefs)
		Expect(err).ToNot(HaveOccurred())

		ops, err := NewOpsFromDefinitions(opDefs)
		Expect(err).ToNot(HaveOccurred())

		_, err = ops.Apply(in)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(
			"Error 'Custom error message': Expected to find a map key 'not-there' for path '/releases/0/not-there' (found map keys: 'name', 'version')"))
	})
})
