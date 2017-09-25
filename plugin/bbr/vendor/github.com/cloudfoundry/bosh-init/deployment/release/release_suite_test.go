package release_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestAction(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deployment Release Suite")
}
