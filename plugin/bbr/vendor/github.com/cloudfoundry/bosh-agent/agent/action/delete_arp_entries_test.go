package action_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agent/action"
	fakeplatform "github.com/cloudfoundry/bosh-agent/platform/fakes"
)

var _ = Describe("Delete ARP Entries", func() {
	var (
		platform  *fakeplatform.FakePlatform
		action    DeleteARPEntriesAction
		addresses []string
		args      DeleteARPEntriesActionArgs
	)

	BeforeEach(func() {
		platform = new(fakeplatform.FakePlatform)
		action = NewDeleteARPEntries(platform)
		addresses = []string{"10.0.0.1", "10.0.0.2"}
		args = DeleteARPEntriesActionArgs{
			Ips: addresses,
		}
	})

	AssertActionIsNotAsynchronous(action)
	AssertActionIsNotPersistent(action)
	AssertActionIsLoggable(action)

	AssertActionIsNotCancelable(action)
	AssertActionIsNotResumable(action)

	It("requests deletion of all provided IPs from the ARP cache", func() {
		Expect(platform.LastIPDeletedFromARP).To(Equal(""))
		_, err := action.Run(args)
		Expect(err).ToNot(HaveOccurred())
		Expect(platform.LastIPDeletedFromARP).To(Equal("10.0.0.2"))
	})

	It("returns an empty map", func() {
		response, err := action.Run(args)

		Expect(err).ToNot(HaveOccurred())
		Expect(response).To(Equal(map[string]interface{}{}))
	})
})
