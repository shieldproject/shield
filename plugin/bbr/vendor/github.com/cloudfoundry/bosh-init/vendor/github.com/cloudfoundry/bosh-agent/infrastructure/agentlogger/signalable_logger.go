package agentlogger

import (
	"os"
	"runtime"

	"github.com/cloudfoundry/bosh-utils/logger"
)

func NewSignalableLogger(writerLogger logger.Logger, signalChannel chan os.Signal) (logger.Logger, chan bool) {
	doneChannel := make(chan bool, 1)
	go func() {
		for {
			<-signalChannel
			writerLogger.Error("Received SIGSEGV", "Dumping goroutines...")
			stackTrace := make([]byte, 10000)
			n := runtime.Stack(stackTrace, true)
			writerLogger.Error("Received SIGSEGV", string(stackTrace[:n]))
			doneChannel <- true
		}
	}()
	return writerLogger, doneChannel
}
