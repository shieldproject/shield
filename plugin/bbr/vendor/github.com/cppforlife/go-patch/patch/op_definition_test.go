package patch_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/go-patch/patch"
)

var _ = Describe("NewOpsFromDefinitions", func() {
	var (
		path                    = "/abc"
		invalidPath             = "abc"
		errorMsg                = "error"
		val         interface{} = 123
		complexVal  interface{} = map[interface{}]interface{}{123: 123}
	)

	It("supports 'replace' and 'remove' operations", func() {
		opDefs := []OpDefinition{
			{Type: "replace", Path: &path, Value: &val},
			{Type: "remove", Path: &path},
		}

		ops, err := NewOpsFromDefinitions(opDefs)
		Expect(err).ToNot(HaveOccurred())

		Expect(ops).To(Equal(Ops([]Op{
			ReplaceOp{Path: MustNewPointerFromString("/abc"), Value: 123},
			RemoveOp{Path: MustNewPointerFromString("/abc")},
		})))
	})

	It("returns error if operation type is unknown", func() {
		_, err := NewOpsFromDefinitions([]OpDefinition{{Type: "test"}})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(`Unknown operation [0] with type 'test' within
{
  "Type": "test"
}`))
	})

	It("returns error if operation type is find since it's not useful in list of operations", func() {
		_, err := NewOpsFromDefinitions([]OpDefinition{{Type: "find"}})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Unknown operation [0] with type 'find'"))
	})

	It("allows values to be complex in error messages", func() {
		_, err := NewOpsFromDefinitions([]OpDefinition{{Type: "test", Path: &invalidPath, Value: &complexVal}})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(`Unknown operation [0] with type 'test' within
{
  "Type": "test",
  "Path": "abc",
  "Value": "<redacted>"
}`))
	})

	Describe("replace", func() {
		It("allows error description", func() {
			opDefs := []OpDefinition{{Type: "replace", Path: &path, Value: &val, Error: &errorMsg}}

			ops, err := NewOpsFromDefinitions(opDefs)
			Expect(err).ToNot(HaveOccurred())

			Expect(ops).To(Equal(Ops([]Op{
				DescriptiveOp{
					Op:       ReplaceOp{Path: MustNewPointerFromString("/abc"), Value: 123},
					ErrorMsg: errorMsg,
				},
			})))
		})

		It("requires path", func() {
			_, err := NewOpsFromDefinitions([]OpDefinition{{Type: "replace"}})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(`Replace operation [0]: Missing path within
{
  "Type": "replace"
}`))
		})

		It("requires value", func() {
			_, err := NewOpsFromDefinitions([]OpDefinition{{Type: "replace", Path: &path}})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(`Replace operation [0]: Missing value within
{
  "Type": "replace",
  "Path": "/abc"
}`))
		})

		It("requires valid path", func() {
			_, err := NewOpsFromDefinitions([]OpDefinition{{Type: "replace", Path: &invalidPath, Value: &val}})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(`Replace operation [0]: Invalid path: Expected to start with '/' within
{
  "Type": "replace",
  "Path": "abc",
  "Value": "<redacted>"
}`))
		})
	})

	Describe("remove", func() {
		It("allows error description", func() {
			opDefs := []OpDefinition{{Type: "remove", Path: &path, Error: &errorMsg}}

			ops, err := NewOpsFromDefinitions(opDefs)
			Expect(err).ToNot(HaveOccurred())

			Expect(ops).To(Equal(Ops([]Op{
				DescriptiveOp{
					Op:       RemoveOp{Path: MustNewPointerFromString("/abc")},
					ErrorMsg: errorMsg,
				},
			})))
		})

		It("requires path", func() {
			_, err := NewOpsFromDefinitions([]OpDefinition{{Type: "remove"}})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(`Remove operation [0]: Missing path within
{
  "Type": "remove"
}`))
		})

		It("does not allow value", func() {
			_, err := NewOpsFromDefinitions([]OpDefinition{{Type: "remove", Path: &path, Value: &val}})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(`Remove operation [0]: Cannot specify value within
{
  "Type": "remove",
  "Path": "/abc",
  "Value": "<redacted>"
}`))
		})

		It("requires valid path", func() {
			_, err := NewOpsFromDefinitions([]OpDefinition{{Type: "remove", Path: &invalidPath}})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(`Remove operation [0]: Invalid path: Expected to start with '/' within
{
  "Type": "remove",
  "Path": "abc"
}`))
		})
	})
})
