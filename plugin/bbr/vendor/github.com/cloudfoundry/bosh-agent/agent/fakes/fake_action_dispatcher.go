package fakes

import (
	boshhandler "github.com/cloudfoundry/bosh-agent/handler"
)

type FakeActionDispatcher struct {
	ResumedPreviouslyDispatchedTasks bool

	DispatchReq  boshhandler.Request
	DispatchResp boshhandler.Response
}

func (dispatcher *FakeActionDispatcher) ResumePreviouslyDispatchedTasks() {
	dispatcher.ResumedPreviouslyDispatchedTasks = true
}

func (dispatcher *FakeActionDispatcher) Dispatch(req boshhandler.Request) boshhandler.Response {
	dispatcher.DispatchReq = req
	return dispatcher.DispatchResp
}
