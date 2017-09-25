package http_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAgentclient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HTTP Agent Client Suite")
}
