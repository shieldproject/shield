package system_test

import (
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/bosh-utils/logger/loggerfakes"
	. "github.com/cloudfoundry/bosh-utils/system"
)

var _ = Describe("execProcess", func() {
	Describe("Wait", func() {
		var err error
		var logger *loggerfakes.FakeLogger

		BeforeEach(func() {
			logger = &loggerfakes.FakeLogger{}
		})

		Context("When quiet logging is enabled", func() {
			It("only excludes logging stdout and stderr contents", func() {
				quietLogging := true

				command := exec.Command(CatExePath, "--stdout", "someStdout", "--stderr", "someStderr")
				process := NewExecProcess(command, false, quietLogging, logger)
				err = process.Start()
				Expect(err).ToNot(HaveOccurred())
				time.Sleep(1 * time.Second)

				waitCh := process.Wait()
				Expect(err).ToNot(HaveOccurred())

				result := <-waitCh
				Expect(result.Stdout).To(ContainSubstring("someStdout"))
				Expect(result.Stderr).To(ContainSubstring("someStderr"))
				Expect(logger.DebugCallCount()).To(Equal(2))

				_, runningLogMessage, _ := logger.DebugArgsForCall(0)
				_, successfulLogMessage, _ := logger.DebugArgsForCall(1)
				Expect(runningLogMessage).To(ContainSubstring("Running command"))
				Expect(successfulLogMessage).To(ContainSubstring("Successful:"))
			})
		})

		Context("when quiet logging is not enabled", func() {
			It("logs the contents of stderr and stdout", func() {
				quietLogging := false

				command := exec.Command(CatExePath, "--stdout", "someStdout", "--stderr", "someStderr")
				process := NewExecProcess(command, false, quietLogging, logger)
				err = process.Start()
				Expect(err).ToNot(HaveOccurred())
				time.Sleep(1 * time.Second)

				waitCh := process.Wait()
				Expect(err).ToNot(HaveOccurred())

				result := <-waitCh
				Expect(result.Stdout).To(ContainSubstring("someStdout"))
				Expect(result.Stderr).To(ContainSubstring("someStderr"))
				Expect(logger.DebugCallCount()).To(Equal(4))

				_, stdoutLogMessage, stdoutLogArgs := logger.DebugArgsForCall(1)
				_, stderrLogMessage, stderrLogArgs := logger.DebugArgsForCall(2)
				Expect(stdoutLogMessage).To(ContainSubstring("Stdout: %s"))
				Expect(stdoutLogArgs[0]).To(ContainSubstring("someStdout"))
				Expect(stderrLogMessage).To(ContainSubstring("Stderr: %s"))
				Expect(stderrLogArgs[0]).To(ContainSubstring("someStderr"))
			})
		})
	})
})
