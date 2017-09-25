package infrastructure_test

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	fakeplat "github.com/cloudfoundry/bosh-agent/platform/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	. "github.com/cloudfoundry/bosh-agent/infrastructure"
)

var _ = Describe("InstanceMetadataSettingsSource", describeInstanceMetadataSettingsSource)

func describeInstanceMetadataSettingsSource() {
	var (
		metadataHeaders map[string]string
		settingsPath    string
		platform        *fakeplat.FakePlatform
		logger          boshlog.Logger
		metadataSource  *InstanceMetadataSettingsSource
	)

	BeforeEach(func() {
		metadataHeaders = make(map[string]string)
		metadataHeaders["key"] = "value"
		settingsPath = "/computeMetadata/v1/instance/attributes/bosh_settings"
		platform = fakeplat.NewFakePlatform()
		logger = boshlog.NewLogger(boshlog.LevelNone)
		metadataSource = NewInstanceMetadataSettingsSource("http://fake-metadata-host", metadataHeaders, settingsPath, platform, logger)
	})

	Describe("PublicSSHKeyForUsername", func() {
		It("returns an empty string", func() {
			publicKey, err := metadataSource.PublicSSHKeyForUsername("fake-username")
			Expect(err).ToNot(HaveOccurred())
			Expect(publicKey).To(Equal(""))
		})
	})

	Describe("Settings", func() {
		var (
			ts *httptest.Server
		)

		handlerFunc := func(w http.ResponseWriter, r *http.Request) {
			defer GinkgoRecover()

			Expect(r.Method).To(Equal("GET"))
			Expect(r.URL.Path).To(Equal(settingsPath))
			Expect(r.Header.Get("key")).To(Equal("value"))

			var jsonStr string

			jsonStr = `{"agent_id": "123"}`

			w.Write([]byte(jsonStr))
		}

		BeforeEach(func() {
			handler := http.HandlerFunc(handlerFunc)
			ts = httptest.NewServer(handler)
			metadataSource = NewInstanceMetadataSettingsSource(ts.URL, metadataHeaders, settingsPath, platform, logger)
		})

		AfterEach(func() {
			ts.Close()
		})

		It("returns settings read from the instance metadata endpoint", func() {
			settings, err := metadataSource.Settings()
			Expect(err).NotTo(HaveOccurred())
			Expect(settings.AgentID).To(Equal("123"))
		})

		It("returns an error if reading from the instance metadata endpoint fails", func() {
			metadataSource = NewInstanceMetadataSettingsSource("bad-registry-endpoint", metadataHeaders, settingsPath, platform, logger)
			_, err := metadataSource.Settings()
			Expect(err).To(HaveOccurred())
		})

	})
}
