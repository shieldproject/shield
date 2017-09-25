package action_test

import (
	. "github.com/cloudfoundry/bosh-agent/agent/action"
	fakeactions "github.com/cloudfoundry/bosh-agent/agent/action/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func init() {
	Describe("configureNetworks", func() {
		var (
			action ConfigureNetworksAction
		)

		BeforeEach(func() {
			action = NewConfigureNetworks(fakeactions.NewFakeAgentKiller())
		})

		It("is asynchronous", func() {
			Expect(action.IsAsynchronous()).To(BeTrue())
		})

		It("is persistent because director expects configure_networks task to become done after agent is restarted", func() {
			Expect(action.IsPersistent()).To(BeTrue())
		})

		Describe("Run", func() {
			// restarts agent process
		})

		Describe("Resume", func() {
			It("returns ok because agent was restarted and connections were re-initialized", func() {
				value, err := action.Resume()
				Expect(value).To(Equal("ok"))
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
}
