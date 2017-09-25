package main

import (
	"os"

	"os/signal"
	"syscall"

	boshapp "github.com/cloudfoundry/bosh-agent/app"
	"github.com/cloudfoundry/bosh-agent/infrastructure/agentlogger"
	"github.com/cloudfoundry/bosh-utils/logger"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

const mainLogTag = "main"

func main() {
	logger := newSignalableLogger(boshlog.NewLogger(boshlog.LevelDebug))

	defer logger.HandlePanic("Main")

	logger.Debug(mainLogTag, "Starting agent")

	fs := boshsys.NewOsFileSystem(logger)
	app := boshapp.New(logger, fs)

	err := app.Setup(os.Args)
	if err != nil {
		logger.Error(mainLogTag, "App setup %s", err.Error())
		os.Exit(1)
	}

	err = app.Run()
	if err != nil {
		logger.Error(mainLogTag, "App run %s", err.Error())
		os.Exit(1)
	}
}

func newSignalableLogger(logger logger.Logger) logger.Logger {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGSEGV)
	signalableLogger, _ := agentlogger.NewSignalableLogger(logger, c)
	return signalableLogger
}
