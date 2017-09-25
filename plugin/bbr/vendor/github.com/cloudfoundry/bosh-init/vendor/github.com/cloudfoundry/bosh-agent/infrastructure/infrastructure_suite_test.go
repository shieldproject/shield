package infrastructure_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestInfrastructure(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Infrastructure Suite")
}
