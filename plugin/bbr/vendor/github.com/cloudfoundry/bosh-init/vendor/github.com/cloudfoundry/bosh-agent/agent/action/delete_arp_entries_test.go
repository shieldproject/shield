package action_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agent/action"
	fakeplatform "github.com/cloudfoundry/bosh-agent/platform/fakes"
)

func init() {
	Describe("Delete ARP Entries", func() {
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

		It("is synchronous so that is it is not queued", func() {
			Expect(action.IsAsynchronous()).To(BeFalse())
		})

		It("is not persistent", func() {
			Expect(action.IsPersistent()).To(BeFalse())
		})

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
}
