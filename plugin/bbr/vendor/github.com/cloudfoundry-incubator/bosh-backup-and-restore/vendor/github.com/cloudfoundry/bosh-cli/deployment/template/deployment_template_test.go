package template_test

import (
	"github.com/cppforlife/go-patch/patch"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/deployment/template"
	boshtpl "github.com/cloudfoundry/bosh-cli/director/template"
)

var _ = Describe("DeploymentTemplate", func() {
	It("can template values into a struct with byte slice", func() {
		deploymentTemplate := NewDeploymentTemplate([]byte(""))
		vars := boshtpl.StaticVariables{"key": "foo"}
		ops := patch.Ops{
			patch.ReplaceOp{Path: patch.MustNewPointerFromString(""), Value: "((key))"},
		}

		result, err := deploymentTemplate.Evaluate(vars, ops)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Content()).To(Equal([]byte("foo\n")))
	})

	It("returns an error if variable key is missing", func() {
		deploymentTemplate := NewDeploymentTemplate([]byte("((key)): true"))
		vars := boshtpl.StaticVariables{"key2": "foo"}
		ops := patch.Ops{}

		_, err := deploymentTemplate.Evaluate(vars, ops)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Expected to find variables: key"))
	})

	It("returns a struct that can return the SHA2 512 of the struct", func() {
		deploymentTemplate := NewDeploymentTemplate([]byte(""))
		vars := boshtpl.StaticVariables{"key": "foo"}
		ops := patch.Ops{
			patch.ReplaceOp{Path: patch.MustNewPointerFromString(""), Value: "((key))"},
		}

		result, err := deploymentTemplate.Evaluate(vars, ops)
		Expect(err).NotTo(HaveOccurred())

		asString := result.SHA()
		Expect(asString).To(Equal("0cf9180a764aba863a67b6d72f0918bc131c6772642cb2dce5a34f0a702f9470ddc2bf125c12198b1995c233c34b4afd346c54a2334c350a948a51b6e8b4e6b6"))
	})
})
