package plugin_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPluginFramework(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin Framework Test Suite")
}
