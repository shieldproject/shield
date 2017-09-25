package bosh_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestBosh(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bosh Suite")
}
