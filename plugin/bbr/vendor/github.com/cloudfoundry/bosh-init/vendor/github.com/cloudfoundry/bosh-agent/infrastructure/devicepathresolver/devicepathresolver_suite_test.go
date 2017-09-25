package devicepathresolver_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestDevicepathresolver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Device Path Resolver Suite")
}
