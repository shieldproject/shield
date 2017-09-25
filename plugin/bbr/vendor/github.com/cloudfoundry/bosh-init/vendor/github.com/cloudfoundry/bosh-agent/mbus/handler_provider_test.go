package mbus_test

import (
	gourl "net/url"
	"reflect"

	"github.com/cloudfoundry/yagnats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/mbus"
	"github.com/cloudfoundry/bosh-agent/micro"
	fakeplatform "github.com/cloudfoundry/bosh-agent/platform/fakes"
	boshdir "github.com/cloudfoundry/bosh-agent/settings/directories"
	fakesettings "github.com/cloudfoundry/bosh-agent/settings/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var _ = Describe("HandlerProvider", func() {
	var (
		settingsService *fakesettings.FakeSettingsService
		platform        *fakeplatform.FakePlatform
		dirProvider     boshdir.Provider
		logger          boshlog.Logger
		provider        HandlerProvider
	)

	BeforeEach(func() {
		settingsService = &fakesettings.FakeSettingsService{}
		logger = boshlog.NewLogger(boshlog.LevelNone)
		platform = fakeplatform.NewFakePlatform()
		dirProvider = boshdir.NewProvider("/var/vcap")
		provider = NewHandlerProvider(settingsService, logger)
	})

	Describe("Get", func() {
		It("returns nats handler", func() {
			settingsService.Settings.Mbus = "nats://lol"
			handler, err := provider.Get(platform, dirProvider)
			Expect(err).ToNot(HaveOccurred())

			// yagnats.NewClient returns new object every time
			expectedHandler := NewNatsHandler(settingsService, yagnats.NewClient(), logger, platform)
			Expect(reflect.TypeOf(handler)).To(Equal(reflect.TypeOf(expectedHandler)))
		})

		It("returns https handler", func() {
			url, err := gourl.Parse("https://foo:bar@lol")
			Expect(err).ToNot(HaveOccurred())

			settingsService.Settings.Mbus = "https://foo:bar@lol"
			handler, err := provider.Get(platform, dirProvider)
			Expect(err).ToNot(HaveOccurred())
			Expect(handler).To(Equal(micro.NewHTTPSHandler(url, logger, platform.GetFs(), dirProvider)))
		})

		It("returns an error if not supported", func() {
			settingsService.Settings.Mbus = "unknown-scheme://lol"
			_, err := provider.Get(platform, dirProvider)
			Expect(err).To(HaveOccurred())
		})
	})
})
