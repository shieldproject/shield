package logger

import (
	"fmt"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"os"
)

func NewSignalableLogger(logger boshlog.Logger, signalChannel chan os.Signal) (boshlog.Logger, chan bool) {
	doneChannel := make(chan bool, 1)
	go func() {
		for {
			<-signalChannel
			fmt.Println("Received SIGHUP - toggling debug output")
			logger.ToggleForcedDebug()
			doneChannel <- true
		}
	}()
	return logger, doneChannel
}
