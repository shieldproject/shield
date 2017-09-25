// +build windows

package platform

import (
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type WindowsAuditLogger struct{}

func NewDelayedAuditLogger(auditLoggerProvider AuditLoggerProvider, logger boshlog.Logger) AuditLogger {
	return &WindowsAuditLogger{}
}

func (w *WindowsAuditLogger) StartLogging() {
}

func (w *WindowsAuditLogger) Debug(msg string) {
}

func (w *WindowsAuditLogger) Err(msg string) {
}
