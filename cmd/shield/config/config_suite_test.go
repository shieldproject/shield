package config_test

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/starkandwayne/goutils/ansi"

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
	ansi.Fprintf(os.Stderr, "@Y{WARNING: %s}\n", fmt.Sprintf(format, args...))
}
