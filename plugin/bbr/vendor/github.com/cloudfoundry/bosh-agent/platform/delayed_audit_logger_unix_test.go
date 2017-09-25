// +build !windows

package platform_test

import (
	"errors"
	"fmt"

	"github.com/cloudfoundry/bosh-agent/platform"
	"github.com/cloudfoundry/bosh-agent/platform/fakes"
	"github.com/cloudfoundry/bosh-utils/logger/loggerfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Delayed Audit Logger", func() {
	var (
		delayedAuditLogger  *platform.DelayedAuditLogger
		auditLoggerProvider *fakes.FakeAuditLoggerProvider
		logger              *loggerfakes.FakeLogger
	)

	Context("StartLogging", func() {
		BeforeEach(func() {
			logger = &loggerfakes.FakeLogger{}
			auditLoggerProvider = fakes.NewFakeAuditLoggerProvider()
			delayedAuditLogger = platform.NewDelayedAuditLogger(auditLoggerProvider, logger)
		})

		Context("when there is an audit logger available", func() {
			BeforeEach(func() {
				delayedAuditLogger.StartLogging()
			})

			It("should start logging to the debug log", func() {
				delayedAuditLogger.Debug("Debugging")

				Eventually(func() string {
					return auditLoggerProvider.GetDebugLogsAt(0)
				}).Should(ContainSubstring("Debugging"))
			})

			It("should start logging to the error log", func() {
				delayedAuditLogger.Err("Oh noes!")

				Eventually(func() string {
					return auditLoggerProvider.GetErrorLogsAt(0)
				}).Should(ContainSubstring("Oh noes!"))
			})
		})

		Context("when there are over a 1000 long messages that have piled up in the debug log", func() {
			It("should overflow and drop messages", func() {
				for i := 0; i < 1001; i++ {
					delayedAuditLogger.Debug(fmt.Sprintf("Message %d", i))
				}

				delayedAuditLogger.StartLogging()
				Eventually(func() string {
					return auditLoggerProvider.GetDebugLogsAt(0)
				}).Should(ContainSubstring("Message 999"))
				Consistently(func() string {
					return auditLoggerProvider.GetDebugLogsAt(1000)
				}).ShouldNot(ContainSubstring("Message 1000"))

				_, debugLog, _ := logger.DebugArgsForCall(1001)
				Expect(debugLog).To(ContainSubstring("Debug message 'Message 1000' not sent to syslog"))
			})
		})

		Context("when there are over a 1000 long messages that have piled up in the error log", func() {
			It("should overflow and drop messages", func() {
				for i := 0; i < 1001; i++ {
					delayedAuditLogger.Err(fmt.Sprintf("Message %d", i))
				}

				delayedAuditLogger.StartLogging()
				Eventually(func() string {
					return auditLoggerProvider.GetErrorLogsAt(0)
				}).Should(ContainSubstring("Message 999"))
				Consistently(func() string {
					return auditLoggerProvider.GetErrorLogsAt(1000)
				}).ShouldNot(ContainSubstring("Message 1000"))

				_, errorLog, _ := logger.DebugArgsForCall(1001)
				Expect(errorLog).To(ContainSubstring("Error message 'Message 1000' not sent to syslog"))
			})
		})

		Context("when there is no debug audit logger available", func() {
			It("should retry until audit logger is available", func() {
				auditLoggerProvider.SetDebugLoggerError(errors.New("Problems!"))
				delayedAuditLogger.StartLogging()

				Eventually(func() int {
					return logger.ErrorCallCount()
				}).Should(Equal(1))

				_, err, _ := logger.ErrorArgsForCall(0)
				Expect(err).To(ContainSubstring("Problems!"))
			})
		})

		Context("when there is no error audit logger available", func() {
			It("should retry until audit logger is available", func() {
				auditLoggerProvider.SetErrorLoggerError(errors.New("Problems!"))
				delayedAuditLogger.StartLogging()

				Eventually(func() int {
					return logger.ErrorCallCount()
				}).Should(Equal(1))

				_, err, _ := logger.ErrorArgsForCall(0)
				Expect(err).To(ContainSubstring("Problems!"))
			})
		})
	})
})
