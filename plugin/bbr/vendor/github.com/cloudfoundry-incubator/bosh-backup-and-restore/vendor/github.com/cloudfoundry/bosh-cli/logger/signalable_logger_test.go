package logger_test

import (
	"io/ioutil"
	"os"
	"syscall"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	bilog "github.com/cloudfoundry/bosh-cli/logger"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

func captureOutputs(f func()) (stdout, stderr []byte) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, err := os.Pipe()
	Expect(err).ToNot(HaveOccurred())

	rErr, wErr, err := os.Pipe()
	Expect(err).ToNot(HaveOccurred())

	os.Stdout = wOut
	os.Stderr = wErr

	f()

	outC := make(chan []byte)
	errC := make(chan []byte)

	go func() {
		bytes, _ := ioutil.ReadAll(rOut)
		outC <- bytes

		bytes, _ = ioutil.ReadAll(rErr)
		errC <- bytes
	}()

	err = wOut.Close()
	Expect(err).ToNot(HaveOccurred())

	err = wErr.Close()
	Expect(err).ToNot(HaveOccurred())

	stdout = <-outC
	stderr = <-errC

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return
}

var _ = Describe("SignalableLogger", func() {
	var (
		signalChannel chan os.Signal
	)
	BeforeEach(func() {
		signalChannel = make(chan os.Signal, 1)
	})

	Describe("Toggling forced debug", func() {
		Describe("when the log level is error", func() {
			It("outputs at debug level", func() {
				stdout, stderr := captureOutputs(func() {
					logger, doneChannel := bilog.NewSignalableLogger(boshlog.NewLogger(boshlog.LevelError), signalChannel)

					signalChannel <- syscall.SIGHUP
					<-doneChannel

					logger.Debug("TOGGLED_DEBUG", "some debug log")
					logger.Info("TOGGLED_INFO", "some info log")
					logger.Warn("TOGGLED_WARN", "some warn log")
					logger.Error("TOGGLED_ERROR", "some error log")
				})

				Expect(stdout).To(ContainSubstring("TOGGLED_DEBUG"))
				Expect(stdout).To(ContainSubstring("TOGGLED_INFO"))
				Expect(stderr).To(ContainSubstring("TOGGLED_WARN"))
				Expect(stderr).To(ContainSubstring("TOGGLED_ERROR"))
			})

			It("outputs at error level when toggled back", func() {
				stdout, stderr := captureOutputs(func() {
					logger, doneChannel := bilog.NewSignalableLogger(boshlog.NewLogger(boshlog.LevelError), signalChannel)

					signalChannel <- syscall.SIGHUP
					<-doneChannel
					signalChannel <- syscall.SIGHUP
					<-doneChannel

					logger.Debug("STANDARD_DEBUG", "some debug log")
					logger.Info("STANDARD_INFO", "some info log")
					logger.Warn("STANDARD_WARN", "some warn log")
					logger.Error("STANDARD_ERROR", "some error log")
				})

				Expect(stdout).ToNot(ContainSubstring("STANDARD_DEBUG"))
				Expect(stdout).ToNot(ContainSubstring("STANDARD_INFO"))
				Expect(stderr).ToNot(ContainSubstring("STANDARD_WARN"))
				Expect(stderr).To(ContainSubstring("STANDARD_ERROR"))
			})
		})
	})
})
