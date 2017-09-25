package installation_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestDeployer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Installation Suite")
}
