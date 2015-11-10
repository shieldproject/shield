package timespec_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestTimespec(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Timespec Test Suite")
}
