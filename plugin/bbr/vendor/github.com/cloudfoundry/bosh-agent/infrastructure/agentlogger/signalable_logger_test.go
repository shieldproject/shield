package agentlogger_test

import (
	"bytes"
	"os"
	"syscall"

	"github.com/cloudfoundry/bosh-agent/infrastructure/agentlogger"
	"github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Signalable logger debug", func() {
	Describe("when SIGSEGV is recieved", func() {
		It("it dumps all goroutines to the given buffer", func() {
			outBuf := new(bytes.Buffer)
			signalChannel := make(chan os.Signal, 1)
			writerLogger := logger.NewWriterLogger(logger.LevelError, outBuf)
			_, doneChannel := agentlogger.NewSignalableLogger(writerLogger, signalChannel)

			signalChannel <- syscall.SIGSEGV
			<-doneChannel

			Expect(outBuf).To(ContainSubstring("Dumping goroutines"))
			Expect(outBuf).To(MatchRegexp(`goroutine (\d+) \[(syscall|running)\]`))
		})
	})
})
