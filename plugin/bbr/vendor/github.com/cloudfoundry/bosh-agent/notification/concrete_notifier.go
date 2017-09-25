package notification

import (
	boshhandler "github.com/cloudfoundry/bosh-agent/handler"
)

type concreteNotifier struct {
	handler boshhandler.Handler
}

func NewNotifier(handler boshhandler.Handler) Notifier {
	return concreteNotifier{handler: handler}
}

func (n concreteNotifier) NotifyShutdown() error {
	return n.handler.Send(boshhandler.HealthMonitor, boshhandler.Shutdown, nil)
}
