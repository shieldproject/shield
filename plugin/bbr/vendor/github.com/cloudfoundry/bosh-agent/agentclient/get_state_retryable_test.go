package agentclient_test

import (
	"errors"

	. "github.com/cloudfoundry/bosh-agent/agentclient"
	fakeagentclient "github.com/cloudfoundry/bosh-agent/agentclient/fakes"
	boshretry "github.com/cloudfoundry/bosh-utils/retrystrategy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetStateRetryable", func() {
	Describe("Attempt", func() {
		var (
			fakeAgentClient   *fakeagentclient.FakeAgentClient
			getStateRetryable boshretry.Retryable
		)

		BeforeEach(func() {
			fakeAgentClient = &fakeagentclient.FakeAgentClient{}
			getStateRetryable = NewGetStateRetryable(fakeAgentClient)
		})

		Context("when get_state fails", func() {
			BeforeEach(func() {
				fakeAgentClient.GetStateReturns(AgentState{}, errors.New("fake-get-state-error"))
			})

			It("returns an error", func() {
				isRetryable, err := getStateRetryable.Attempt()
				Expect(err).To(HaveOccurred())
				Expect(isRetryable).To(BeFalse())
				Expect(err.Error()).To(ContainSubstring("fake-get-state-error"))
				Expect(fakeAgentClient.GetStateCallCount()).To(Equal(1))
			})
		})

		Context("when get_state returns state as pending", func() {
			BeforeEach(func() {
				fakeAgentClient.GetStateReturns(AgentState{JobState: "pending"}, nil)
			})

			It("returns retryable error", func() {
				isRetryable, err := getStateRetryable.Attempt()
				Expect(err).To(HaveOccurred())
				Expect(isRetryable).To(BeTrue())
				Expect(fakeAgentClient.GetStateCallCount()).To(Equal(1))
			})
		})

		Context("when get_state returns state as running", func() {
			BeforeEach(func() {
				fakeAgentClient.GetStateReturns(AgentState{JobState: "running"}, nil)
			})

			It("returns no error", func() {
				isRetryable, err := getStateRetryable.Attempt()
				Expect(err).ToNot(HaveOccurred())
				Expect(isRetryable).To(BeTrue())
				Expect(fakeAgentClient.GetStateCallCount()).To(Equal(1))
			})
		})
	})
})
