package tlsconfig_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestTLSConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TLS Config Suite")
}
