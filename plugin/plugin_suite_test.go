package plugin_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestPluginFramework(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin Framework Test Suite")
}
