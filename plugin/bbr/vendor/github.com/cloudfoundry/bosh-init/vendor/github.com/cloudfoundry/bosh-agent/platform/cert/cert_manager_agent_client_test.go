package cert_test

import (
	. "github.com/cloudfoundry/bosh-agent/agentclient/http"

	"github.com/cloudfoundry/bosh-agent/agentclient"
	"github.com/cloudfoundry/bosh-agent/settings"
	fakehttpclient "github.com/cloudfoundry/bosh-utils/httpclient/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CertManager", func() {
	var (
		fakeHTTPClient *fakehttpclient.FakeHTTPClient
		agentClient    agentclient.AgentClient
	)

	BeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		fakeHTTPClient = fakehttpclient.NewFakeHTTPClient()
		agentClient = NewAgentClient("http://localhost:6305", "fake-uuid", 0, 10, fakeHTTPClient, logger)
		fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)
		fakeHTTPClient.SetPostBehavior(`{"value":"updated"}`, 200, nil)
	})

	It("adds and registers new certs on a fresh machine", func() {
		var cert string = "This certificate is the first one. It's more awesome than the other one.\n-----BEGIN CERTIFICATE-----\nMIIEJDCCAwygAwIBAgIJAO+CqgiJnCgpMA0GCSqGSIb3DQEBBQUAMGkxCzAJBgNV\nBAYTAkNBMRMwEQYDVQQIEwpTb21lLVN0YXRlMSIBAgIJAO+CqgiJnCgpMA0GCSqGSIb3DQEBBQUAMGkxCzAJBgNV\nBAYTAkNBMRMwEQYDVQQIEwpTb21lLVN0YXRlMSEwHwYDVQQKExhJbnRlcm5ldCBX\naWRnaXRzIFB0eSBMdGQxIjAgBgNVBAMTGWR4MTkwLnRvci5waXZvdGFsbGFicy5EwHwYDVQQKExhJbnRlcm5ldCBX\naWRnaXRzIFB0eSBMdGQxIjAgBgNVBAMTGWR4MTkwLnRvci5waXZvdGFsbGFicy5j\nb20wHhcNMTUwNTEzMTM1NjA2WhcNMjUwNTEwMTM1NjA2WjBpMQswCQYDVQQGEwJD\nQTETMBEGA1UECBMKU29tZGackAF\nqokoSBXzJCJTt2P681gyqBDr/hUYzqpoXUsOTRisScbEbaSv8hTiTeFJUMyNQAqn\nDtmvI8bXKxU=\n-----END CERTIFICATE-----\n"
		settings := settings.Settings{TrustedCerts: cert}

		err := agentClient.UpdateSettings(settings)
		Expect(err).NotTo(HaveOccurred())
	})
})
