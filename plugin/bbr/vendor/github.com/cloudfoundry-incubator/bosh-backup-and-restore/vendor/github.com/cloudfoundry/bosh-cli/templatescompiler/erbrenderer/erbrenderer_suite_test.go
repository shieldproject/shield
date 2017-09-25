package erbrenderer_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestErbrenderer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Erbrenderer Suite")
}
