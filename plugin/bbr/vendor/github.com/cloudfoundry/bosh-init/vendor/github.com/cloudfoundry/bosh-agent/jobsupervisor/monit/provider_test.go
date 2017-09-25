package monit_test

import (
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	fakeplatform "github.com/cloudfoundry/bosh-agent/platform/fakes"
	boshhttp "github.com/cloudfoundry/bosh-utils/http"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	. "github.com/cloudfoundry/bosh-agent/jobsupervisor/monit"
)

var _ = Describe("clientProvider", func() {
	It("Get", func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		platform := fakeplatform.NewFakePlatform()

		platform.GetMonitCredentialsUsername = "fake-user"
		platform.GetMonitCredentialsPassword = "fake-pass"

		client, err := NewProvider(platform, logger).Get()
		Expect(err).ToNot(HaveOccurred())

		httpClient := http.DefaultClient

		shortHTTPClient := boshhttp.NewRetryClient(httpClient, 20, 1*time.Second, logger)
		longHTTPClient := NewMonitRetryClient(httpClient, 300, 20, 1*time.Second, logger)

		expectedClient := NewHTTPClient(
			"127.0.0.1:2822",
			"fake-user",
			"fake-pass",
			shortHTTPClient,
			longHTTPClient,
			logger,
		)
		Expect(client).To(Equal(expectedClient))
	})
})
