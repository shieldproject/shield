package sshtunnel

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestSshtunnel(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sshtunnel Suite")
}
