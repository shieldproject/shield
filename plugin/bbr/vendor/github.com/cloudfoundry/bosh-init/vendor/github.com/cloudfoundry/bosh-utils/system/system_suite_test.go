package system_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestSystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "System Suite")
}

var CatExePath string
var FalseExePath string
var WindowsExePath string

var _ = BeforeSuite(func() {
	var err error
	CatExePath, err = gexec.Build("exec_cmd_runner_fixtures/cat.go")
	Expect(err).ToNot(HaveOccurred())

	FalseExePath, err = gexec.Build("exec_cmd_runner_fixtures/false.go")
	Expect(err).ToNot(HaveOccurred())

	WindowsExePath, err = gexec.Build("exec_cmd_runner_fixtures/windows_exe.go")
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
