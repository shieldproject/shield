package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gexec"
	"testing"
)

var pathToBoshUtils string

func TestMain(t *testing.T) {
	RegisterFailHandler(Fail)
	BeforeSuite(func() {
		var err error
		pathToBoshUtils, err = gexec.Build("github.com/cloudfoundry/bosh-utils/main")
		Ω(err).ShouldNot(HaveOccurred())
	})

	AfterSuite(func() {
		gexec.CleanupBuildArtifacts()
	})
	RunSpecs(t, "Main Suite")
}
