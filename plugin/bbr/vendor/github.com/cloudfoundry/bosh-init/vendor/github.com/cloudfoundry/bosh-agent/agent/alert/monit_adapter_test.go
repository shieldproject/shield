package alert_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agent/alert"
	fakesettings "github.com/cloudfoundry/bosh-agent/settings/fakes"
	"github.com/pivotal-golang/clock/fakeclock"

	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
)

func buildMonitAlert() MonitAlert {
	return MonitAlert{
		ID:          "some-random-id",
		Service:     "nats",
		Event:       "does not exist",
		Action:      "restart",
		Date:        "Sun, 22 May 2011 20:07:41 +0500",
		Description: "process is not running",
	}
}

var _ = Describe("monitAdapter", func() {
	var (
		settingsService *fakesettings.FakeSettingsService
		timeService     *fakeclock.FakeClock
	)

	BeforeEach(func() {
		settingsService = &fakesettings.FakeSettingsService{}
		timeService = fakeclock.NewFakeClock(time.Now())
	})

	Describe("IsIgnorable", func() {
		itIgnores := func(event string) {
			monitAlert := buildMonitAlert()
			monitAlert.Event = event

			monitAdapter := NewMonitAdapter(monitAlert, settingsService, timeService)
			Expect(monitAdapter.IsIgnorable()).To(BeTrue())
		}

		itDoesNotIgnore := func(event string) {
			monitAlert := buildMonitAlert()
			monitAlert.Event = event

			monitAdapter := NewMonitAdapter(monitAlert, settingsService, timeService)
			Expect(monitAdapter.IsIgnorable()).To(BeFalse())
		}

		It("ignores some monit events", func() {
			itIgnores("action done")
			itIgnores("checksum succeeded")
			itIgnores("checksum not changed")
			itIgnores("connection succeeded")
			itIgnores("connection not changed")
		})

		It("does not ignores all monit events", func() {
			itDoesNotIgnore("checksum failed")
			itDoesNotIgnore("checksum changed")
			itDoesNotIgnore("connection failed")
			itDoesNotIgnore("connection changed")
			itDoesNotIgnore("content failed")
		})

		It("does not ignore unknown monit events", func() {
			itDoesNotIgnore("fake event")
		})
	})

	Describe("Alert", func() {
		It("defaults to severty critical, when the event is unknown", func() {
			monitAlert := buildMonitAlert()
			monitAdapter := NewMonitAdapter(monitAlert, settingsService, timeService)

			builtAlert, err := monitAdapter.Alert()
			Expect(err).ToNot(HaveOccurred())
			Expect(builtAlert.ID).To(Equal("some-random-id"))
			Expect(builtAlert.Severity).To(Equal(SeverityAlert))
			Expect(builtAlert.Title).To(Equal("nats - does not exist - restart"))
			Expect(builtAlert.Summary).To(Equal("process is not running"))
			Expect(builtAlert.CreatedAt).To(Equal(int64(1306076861)))
		})

		It("defaults to severty critical, when the event is unknown", func() {
			monitAlert := buildMonitAlert()
			monitAlert.Event = "fake-event"
			monitAdapter := NewMonitAdapter(monitAlert, settingsService, timeService)

			builtAlert, err := monitAdapter.Alert()
			Expect(err).ToNot(HaveOccurred())
			Expect(builtAlert.Severity).To(Equal(SeverityCritical))
		})

		It("recognizes events with case insensitivity", func() {
			alerts := map[string]SeverityLevel{
				"action done": SeverityIgnored,
				"Action done": SeverityIgnored,
				"action Done": SeverityIgnored,
			}

			for event, expectedSeverity := range alerts {
				monitAlert := buildMonitAlert()
				monitAlert.Event = event
				monitAdapter := NewMonitAdapter(monitAlert, settingsService, timeService)
				builtAlert, err := monitAdapter.Alert()
				Expect(err).ToNot(HaveOccurred())
				Expect(builtAlert.Severity).To(Equal(expectedSeverity))
			}
		})

		It("defaults CreatedAt to time.Now(), when parsing the supplied time fails", func() {
			monitAlert := buildMonitAlert()
			monitAlert.Date = "Thu, 02 May 2013 20:07:0"

			monitAdapter := NewMonitAdapter(monitAlert, settingsService, timeService)
			builtAlert, err := monitAdapter.Alert()
			Expect(err).ToNot(HaveOccurred())
			Expect(builtAlert.CreatedAt).To(Equal(timeService.Now().Unix()))
		})

		It("sets the title with ips", func() {
			monitAlert := buildMonitAlert()
			settingsService.Settings.Networks = boshsettings.Networks{
				"fake-net1": boshsettings.Network{IP: "192.168.0.1"},
				"fake-net2": boshsettings.Network{IP: "10.0.0.1"},
			}

			monitAdapter := NewMonitAdapter(monitAlert, settingsService, timeService)
			builtAlert, err := monitAdapter.Alert()
			Expect(err).ToNot(HaveOccurred())
			Expect(builtAlert.Title).To(Equal("nats (10.0.0.1, 192.168.0.1) - does not exist - restart"))
		})
	})
})
