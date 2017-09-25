package fakes

import (
	"bytes"
	"log"
	"sync"
)

type FakeAuditLoggerProvider struct {
	debugBuffer          *bytes.Buffer
	debugBufferLock      sync.RWMutex
	errorBuffer          *bytes.Buffer
	errorBufferLock      sync.RWMutex
	debugLoggerError     error
	debugLoggerErrorLock sync.RWMutex
	errorLoggerError     error
	errorLoggerErrorLock sync.RWMutex
}

func NewFakeAuditLoggerProvider() *FakeAuditLoggerProvider {
	return &FakeAuditLoggerProvider{
		debugBuffer: bytes.NewBuffer([]byte{}),
		errorBuffer: bytes.NewBuffer([]byte{}),
	}
}

type synchronousWriter struct {
	buffer *bytes.Buffer
	mutex  *sync.RWMutex
}

func (w synchronousWriter) Write(bytes []byte) (int, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.buffer.Write(bytes)
}

func (p *FakeAuditLoggerProvider) ProvideDebugLogger() (*log.Logger, error) {
	p.debugLoggerErrorLock.Lock()
	if p.debugLoggerError != nil {
		debugErr := p.debugLoggerError
		p.debugLoggerError = nil
		return nil, debugErr
	}
	p.debugLoggerErrorLock.Unlock()

	writer := synchronousWriter{
		buffer: p.debugBuffer,
		mutex:  &p.debugBufferLock,
	}
	return log.New(writer, "", log.LstdFlags), nil
}

func (p *FakeAuditLoggerProvider) SetDebugLoggerError(err error) {
	p.debugLoggerErrorLock.Lock()
	p.debugLoggerError = err
	p.debugLoggerErrorLock.Unlock()
}

func (p *FakeAuditLoggerProvider) SetErrorLoggerError(err error) {
	p.errorLoggerErrorLock.Lock()
	p.errorLoggerError = err
	p.errorLoggerErrorLock.Unlock()
}

func (p *FakeAuditLoggerProvider) GetDebugLogsAt(index int) string {
	p.debugBufferLock.RLock()
	debugString := string(p.debugBuffer.Bytes())
	p.debugBufferLock.RUnlock()
	return debugString
}

func (p *FakeAuditLoggerProvider) ProvideErrorLogger() (*log.Logger, error) {
	p.errorLoggerErrorLock.Lock()
	if p.errorLoggerError != nil {
		errorErr := p.errorLoggerError
		p.errorLoggerError = nil
		return nil, errorErr
	}
	p.errorLoggerErrorLock.Unlock()

	writer := synchronousWriter{
		buffer: p.errorBuffer,
		mutex:  &p.errorBufferLock,
	}
	return log.New(writer, "", log.LstdFlags), nil
}

func (p *FakeAuditLoggerProvider) GetErrorLogsAt(index int) string {
	p.errorBufferLock.RLock()
	errorString := string(p.errorBuffer.Bytes())
	p.errorBufferLock.RUnlock()
	return errorString
}
