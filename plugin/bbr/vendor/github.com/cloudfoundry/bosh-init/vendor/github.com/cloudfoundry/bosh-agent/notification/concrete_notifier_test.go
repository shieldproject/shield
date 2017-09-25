package notification_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshhandler "github.com/cloudfoundry/bosh-agent/handler"
	fakembus "github.com/cloudfoundry/bosh-agent/mbus/fakes"
	. "github.com/cloudfoundry/bosh-agent/notification"
)

var _ = Describe("concreteNotifier", func() {
	Describe("NotifyShutdown", func() {
		var (
			handler  *fakembus.FakeHandler
			notifier Notifier
		)

		BeforeEach(func() {
			handler = fakembus.NewFakeHandler()
			notifier = NewNotifier(handler)
		})

		It("sends shutdown message to health manager", func() {
			err := notifier.NotifyShutdown()
			Expect(err).ToNot(HaveOccurred())

			Expect(handler.SendInputs()).To(Equal([]fakembus.SendInput{
				{
					Target:  boshhandler.HealthMonitor,
					Topic:   boshhandler.Shutdown,
					Message: nil,
				},
			}))
		})

		It("returns error if sending shutdown message fails", func() {
			handler.SendErr = errors.New("fake-send-error")

			err := notifier.NotifyShutdown()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-send-error"))
		})
	})
})
