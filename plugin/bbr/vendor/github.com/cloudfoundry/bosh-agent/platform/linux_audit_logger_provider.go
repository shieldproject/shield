//+build !windows

package platform

import (
	"log"
	"log/syslog"
)

type linuxAuditLoggerProvider struct{}

const topicName = "vcap.agent"

func NewAuditLoggerProvider() AuditLoggerProvider {
	return &linuxAuditLoggerProvider{}
}

func (p *linuxAuditLoggerProvider) ProvideDebugLogger() (*log.Logger, error) {
	writer, err := syslog.New(syslog.LOG_DEBUG, topicName)
	if err != nil {
		return nil, err
	}
	return log.New(writer, "", log.LstdFlags), nil
}

func (p *linuxAuditLoggerProvider) ProvideErrorLogger() (*log.Logger, error) {
	writer, err := syslog.New(syslog.LOG_ERR, topicName)
	if err != nil {
		return nil, err
	}
	return log.New(writer, "", log.LstdFlags), nil
}
