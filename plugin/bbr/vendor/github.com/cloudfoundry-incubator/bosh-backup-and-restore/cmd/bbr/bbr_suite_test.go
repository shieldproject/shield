package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestBbr(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BBR Cmd Suite")
}
