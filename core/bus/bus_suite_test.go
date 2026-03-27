package bus_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

func TestBus(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bus Suite")
}
