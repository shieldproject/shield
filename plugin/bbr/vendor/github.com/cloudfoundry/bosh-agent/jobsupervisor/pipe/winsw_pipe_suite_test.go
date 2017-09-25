package main

import (
	"runtime"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

var pathToPipeCLI string
var GoSequencePath string
var PrintPidsPath string
var ExitRunnerPath string
var ExitCodePath string
var echoCmdArgs []string

const echoOutput = "hello"

func TestWinswPipe(t *testing.T) {
	BeforeSuite(func() {
		var err error
		pathToPipeCLI, err = gexec.Build("github.com/cloudfoundry/bosh-agent/jobsupervisor/pipe")
		Expect(err).To(Succeed())

		GoSequencePath, err = gexec.Build("./testdata/gosequence/gosequence.go")
		Expect(err).To(Succeed())
		PrintPidsPath, err = gexec.Build("./testdata/printpids/printpids.go")
		Expect(err).To(Succeed())
		ExitRunnerPath, err = gexec.Build("./testdata/exitrunner/exitrunner.go")
		Expect(err).To(Succeed())
		ExitCodePath, err = gexec.Build("./testdata/exitcode/exitcode.go")
		Expect(err).To(Succeed())
	})

	BeforeEach(func() {
		if runtime.GOOS == "windows" {
			echoCmdArgs = []string{"powershell.exe", "-c", "echo", echoOutput}
			SetDefaultEventuallyTimeout(5 * time.Second)
		} else {
			echoCmdArgs = []string{"echo", echoOutput}
		}
	})

	AfterSuite(func() {
		gexec.CleanupBuildArtifacts()
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "WinswPipe Suite")
}
