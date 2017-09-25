//+build windows

package platform

import (
	"log"
)

type windowsAuditLoggerProvider struct{}

func NewAuditLoggerProvider() AuditLoggerProvider {
	return &windowsAuditLoggerProvider{}
}

func (p *windowsAuditLoggerProvider) ProvideDebugLogger() (*log.Logger, error) {
	return nil, nil
}

func (p *windowsAuditLoggerProvider) ProvideErrorLogger() (*log.Logger, error) {
	return nil, nil
}
