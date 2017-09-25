package fakes

import (
	"sync"
)

type FakeAuditLogger struct {
	debugMutex sync.RWMutex
	errMutex   sync.RWMutex
	DebugMsgs  []string
	ErrMsgs    []string
}

func NewFakeAuditLogger() *FakeAuditLogger {
	return &FakeAuditLogger{}
}

func (f *FakeAuditLogger) Debug(msg string) {
	f.debugMutex.Lock()
	f.DebugMsgs = append(f.DebugMsgs, msg)
	f.debugMutex.Unlock()
}

func (f *FakeAuditLogger) Err(msg string) {
	f.errMutex.Lock()
	f.ErrMsgs = append(f.ErrMsgs, msg)
	f.errMutex.Unlock()
}

func (f *FakeAuditLogger) StartLogging() {}

func (f *FakeAuditLogger) GetDebugMsgs() []string {
	f.debugMutex.RLock()
	defer f.debugMutex.RUnlock()

	return f.DebugMsgs
}

func (f *FakeAuditLogger) GetErrMsgs() []string {
	f.errMutex.RLock()
	defer f.errMutex.RUnlock()

	return f.ErrMsgs
}
