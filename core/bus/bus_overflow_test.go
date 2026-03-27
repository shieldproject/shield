package bus_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/shieldproject/shield/core/bus"
)

var _ = Describe("Bus backlog overflow behavior", func() {
	var b *bus.Bus

	ev := func(name string) bus.Event {
		return bus.Event{
			Event: name,
			Queue: "*",
			Data:  nil,
		}
	}

	BeforeEach(func() {
		// 1 slot, backlog channel capacity of 1
		b = bus.New(1, 1)
	})

	Context("when a client's backlog channel is full", func() {
		It("drops the client connection when a second event overflows the backlog", func() {
			ch, _, err := b.Register([]string{"*"})
			Ω(err).ShouldNot(HaveOccurred())
			Ω(ch).ShouldNot(BeNil())

			// First event fills the buffered channel (capacity=1); non-blocking send succeeds.
			b.SendEvent([]string{"*"}, ev("create-object"))

			// Second event finds the channel full; the default branch fires,
			// unregister() is called, and dropped.connections is incremented.
			b.SendEvent([]string{"*"}, ev("update-object"))

			m := b.DumpState()

			// Connection was dropped — current count returns to zero.
			Ω(m.Connections.Current).Should(Equal(int64(0)))

			// Exactly one connection was dropped.
			Ω(m.Connections.Dropped).Should(Equal(int64(1)))

			// The slot list is empty after the drop.
			Ω(m.Slots).Should(BeEmpty())

			// The channel is closed after unregister; a receive returns
			// the buffered event then signals closure.
			firstEvent, ok := <-ch
			Ω(ok).Should(BeTrue())
			Ω(firstEvent.Event).Should(Equal("create-object"))

			_, ok = <-ch
			Ω(ok).Should(BeFalse(), "channel should be closed after overflow drop")
		})

		It("frees the slot so a new client can register after drop", func() {
			ch, _, _ := b.Register([]string{"*"})
			Ω(ch).ShouldNot(BeNil())

			// Fill then overflow to trigger the drop.
			b.SendEvent([]string{"*"}, ev("create-object"))
			b.SendEvent([]string{"*"}, ev("update-object"))

			// The single slot must now be free for a new registration.
			ch2, _, err := b.Register([]string{"*"})
			Ω(err).ShouldNot(HaveOccurred())
			Ω(ch2).ShouldNot(BeNil())

			m := b.DumpState()
			Ω(m.Connections.Current).Should(Equal(int64(1)))
			Ω(len(m.Slots)).Should(Equal(1))
		})

		It("tracks dropped connection count in metrics", func() {
			_, _, _ = b.Register([]string{"*"})

			b.SendEvent([]string{"*"}, ev("create-object"))
			b.SendEvent([]string{"*"}, ev("update-object"))

			m := b.DumpState()
			Ω(m.Connections.Dropped).Should(Equal(int64(1)))
			Ω(m.Connections.Lifetime).Should(Equal(int64(1)))
			Ω(m.Connections.Current).Should(Equal(int64(0)))
		})

		It("records the first event type in the events metric before drop", func() {
			_, _, _ = b.Register([]string{"*"})

			b.SendEvent([]string{"*"}, ev("create-object"))
			b.SendEvent([]string{"*"}, ev("update-object"))

			m := b.DumpState()
			// Both SendEvent calls reach the events counter before the channel
			// send is attempted; the drop happens inside the select default.
			Ω(m.Events["create-object"]).Should(Equal(int64(1)))
			Ω(m.Events["update-object"]).Should(Equal(int64(1)))

			// Only the first message was delivered; the second triggered the drop.
			Ω(m.Messages["create-object"]).Should(Equal(int64(1)))
			Ω(m.Messages["update-object"]).Should(Equal(int64(0)))
		})
	})
})
