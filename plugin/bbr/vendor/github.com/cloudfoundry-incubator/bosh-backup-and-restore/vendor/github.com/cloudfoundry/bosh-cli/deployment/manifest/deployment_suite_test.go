package manifest_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestDeployment(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deployment Suite")
}
