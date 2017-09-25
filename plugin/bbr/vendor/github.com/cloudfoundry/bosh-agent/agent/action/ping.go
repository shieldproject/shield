package action

import (
	"errors"
)

type PingAction struct{}

func NewPing() PingAction {
	return PingAction{}
}

func (a PingAction) IsAsynchronous(_ ProtocolVersion) bool {
	return false
}

func (a PingAction) IsPersistent() bool {
	return false
}

func (a PingAction) IsLoggable() bool {
	return true
}

func (a PingAction) Run() (string, error) {
	return "pong", nil
}

func (a PingAction) Resume() (interface{}, error) {
	return nil, errors.New("not supported")
}

func (a PingAction) Cancel() error {
	return errors.New("not supported")
}
