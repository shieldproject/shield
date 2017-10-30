package config_test

import (
	"os"

	fmt "github.com/jhunt/go-ansi"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var err error

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}

var _ = AfterEach(func() {
	err = nil
})

func Warn(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "@Y{WARNING: %s}\n", fmt.Sprintf(format, args...))
}
