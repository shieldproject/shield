package mbus_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestMbus(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Message Bus Suite")
}
