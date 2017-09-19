package template_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"

	. "github.com/cloudfoundry/bosh-cli/director/template"
)

func TestReg(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "director/template")
}

type FakeVariables struct {
	GetFunc      func(VariableDefinition) (interface{}, bool, error)
	GetVarDef    VariableDefinition
	GetErr       error
	GetCallCount int
}

func (v *FakeVariables) Get(varDef VariableDefinition) (interface{}, bool, error) {
	v.GetCallCount += 1
	v.GetVarDef = varDef
	if v.GetFunc != nil {
		return v.GetFunc(varDef)
	}
	return nil, false, v.GetErr
}

func (v *FakeVariables) List() ([]VariableDefinition, error) {
	return nil, nil
}
