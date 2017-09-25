package agentclient_test

import (
	"errors"

	. "github.com/cloudfoundry/bosh-agent/agentclient"
	fakeagentclient "github.com/cloudfoundry/bosh-agent/agentclient/fakes"
	boshretry "github.com/cloudfoundry/bosh-utils/retrystrategy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PingRetryable", func() {
	Describe("Attempt", func() {
		var (
			fakeAgentClient *fakeagentclient.FakeAgentClient
			pingRetryable   boshretry.Retryable
		)

		BeforeEach(func() {
			fakeAgentClient = &fakeagentclient.FakeAgentClient{}
			pingRetryable = NewPingRetryable(fakeAgentClient)
		})

		It("tells the agent client to ping", func() {
			isRetryable, err := pingRetryable.Attempt()
			Expect(err).ToNot(HaveOccurred())
			Expect(isRetryable).To(BeTrue())
			Expect(fakeAgentClient.PingCallCount()).To(Equal(1))
		})

		Context("when pinging fails", func() {
			BeforeEach(func() {
				fakeAgentClient.PingReturns("", errors.New("fake-agent-client-ping-error"))
			})

			It("returns an error", func() {
				isRetryable, err := pingRetryable.Attempt()
				Expect(err).To(HaveOccurred())
				Expect(isRetryable).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring("fake-agent-client-ping-error"))
			})
		})
	})
})
