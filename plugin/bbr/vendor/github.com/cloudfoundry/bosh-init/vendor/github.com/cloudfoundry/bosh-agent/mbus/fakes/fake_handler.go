package fakes

import (
	"sync"

	boshhandler "github.com/cloudfoundry/bosh-agent/handler"
)

type FakeHandler struct {
	RunFunc     boshhandler.Func
	RunCallBack func()
	RunErr      error

	ReceivedRun   bool
	ReceivedStart bool
	ReceivedStop  bool

	// Keeps list of all receivd health manager requests
	sendLock   sync.Mutex
	sendInputs []SendInput

	RegisteredAdditionalFunc boshhandler.Func

	SendCallback func(SendInput)
	SendErr      error
}

type SendInput struct {
	Target  boshhandler.Target
	Topic   boshhandler.Topic
	Message interface{}
}

func NewFakeHandler() *FakeHandler {
	return &FakeHandler{sendInputs: []SendInput{}}
}

func (h *FakeHandler) Run(handlerFunc boshhandler.Func) error {
	h.ReceivedRun = true
	h.RunFunc = handlerFunc

	if h.RunCallBack != nil {
		h.RunCallBack()
	}

	return h.RunErr
}

func (h *FakeHandler) KeepOnRunning() {
	block := make(chan error)
	h.RunCallBack = func() { <-block }
}

func (h *FakeHandler) Start(handlerFunc boshhandler.Func) error {
	h.ReceivedStart = true
	h.RunFunc = handlerFunc
	return nil
}

func (h *FakeHandler) Stop() {
	h.ReceivedStop = true
}

func (h *FakeHandler) RegisterAdditionalFunc(handlerFunc boshhandler.Func) {
	h.RegisteredAdditionalFunc = handlerFunc
}

func (h *FakeHandler) Send(target boshhandler.Target, topic boshhandler.Topic, message interface{}) error {
	h.sendLock.Lock()
	defer h.sendLock.Unlock()

	sendInput := SendInput{
		Target:  target,
		Topic:   topic,
		Message: message,
	}
	h.sendInputs = append(h.sendInputs, sendInput)

	if h.SendCallback != nil {
		h.SendCallback(sendInput)
	}

	return h.SendErr
}

func (h *FakeHandler) SendInputs() []SendInput {
	h.sendLock.Lock()
	defer h.sendLock.Unlock()

	return h.sendInputs
}
