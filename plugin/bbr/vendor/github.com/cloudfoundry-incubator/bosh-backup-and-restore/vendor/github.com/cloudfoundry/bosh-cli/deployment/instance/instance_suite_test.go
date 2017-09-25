package instance_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestInstance(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Instance Suite")
}
