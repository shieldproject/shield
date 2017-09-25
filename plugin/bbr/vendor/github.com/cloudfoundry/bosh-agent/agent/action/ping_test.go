package action_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agent/action"
)

var _ = Describe("Ping", func() {

	var (
		action PingAction
	)

	BeforeEach(func() {
		action = NewPing()
	})

	AssertActionIsNotAsynchronous(action)
	AssertActionIsNotPersistent(action)
	AssertActionIsLoggable(action)

	AssertActionIsNotResumable(action)
	AssertActionIsNotCancelable(action)

	It("ping run returns pong", func() {
		pong, err := action.Run()
		Expect(err).ToNot(HaveOccurred())
		Expect(pong).To(Equal("pong"))
	})
})
