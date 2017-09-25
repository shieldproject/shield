package action

import "errors"

type InfoAction struct{}

type InfoResponse struct {
	APIVersion int `json:"api_version"`
}

func NewInfo() InfoAction {
	return InfoAction{}
}

func (a InfoAction) IsAsynchronous(_ ProtocolVersion) bool {
	return false
}

func (a InfoAction) IsPersistent() bool {
	return false
}

func (a InfoAction) IsLoggable() bool {
	return true
}

func (a InfoAction) Run() (InfoResponse, error) {
	return InfoResponse{APIVersion: 1}, nil
}

func (a InfoAction) Resume() (interface{}, error) {
	return nil, errors.New("not supported")
}

func (a InfoAction) Cancel() error {
	return errors.New("not supported")
}
