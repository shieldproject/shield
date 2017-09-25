package action_test

import (
	. "github.com/cloudfoundry/bosh-agent/agent/action"
	fakeactions "github.com/cloudfoundry/bosh-agent/agent/action/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("configureNetworks", func() {
	var (
		action ConfigureNetworksAction
	)

	BeforeEach(func() {
		action = NewConfigureNetworks(fakeactions.NewFakeAgentKiller())
	})

	AssertActionIsAsynchronous(action)
	AssertActionIsPersistent(action)
	AssertActionIsLoggable(action)

	AssertActionIsNotCancelable(action)
	AssertActionIsResumable(action)

	It("is asynchronous", func() {
		Expect(action.IsAsynchronous(ProtocolVersion(0))).To(BeTrue())
	})

	It("is persistent because director expects configure_networks task to become done after agent is restarted", func() {
		Expect(action.IsPersistent()).To(BeTrue())
	})

	Describe("Run", func() {
		// restarts agent process
	})
})
