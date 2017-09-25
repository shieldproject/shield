package pkg_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"

	boshrelpkg "github.com/cloudfoundry/bosh-cli/release/pkg"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
)

func TestReg(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "state/pkg")
}

func newPkg(name, fp string, deps []string) *boshrelpkg.Package {
	resource := NewResourceWithBuiltArchive(name, fp, "", "")
	return boshrelpkg.NewPackage(resource, deps)
}
