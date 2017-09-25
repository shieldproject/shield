package action_test

import (
	. "github.com/cloudfoundry/bosh-agent/agent/action"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Info", func() {
	var (
		action InfoAction
	)

	BeforeEach(func() {
		action = NewInfo()
	})

	AssertActionIsNotAsynchronous(action)
	AssertActionIsNotPersistent(action)
	AssertActionIsLoggable(action)

	AssertActionIsNotResumable(action)
	AssertActionIsNotCancelable(action)

	It("returns the api version", func() {
		infoResponse, err := action.Run()
		Expect(err).ToNot(HaveOccurred())
		Expect(infoResponse.APIVersion).To(Equal(1))
	})
})
