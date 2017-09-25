package manifest_test

import (
	. "github.com/cloudfoundry/bosh-cli/deployment/manifest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("WatchTime", func() {
	Describe("NewWatchTime", func() {
		It("returns the correct WatchTime", func() {
			watchTime, err := NewWatchTime("2000-5000")
			Expect(err).ToNot(HaveOccurred())
			Expect(watchTime.Start).To(Equal(2000))
			Expect(watchTime.End).To(Equal(5000))
		})

		Context("when the start is later than the end", func() {
			It("returns an error", func() {
				watchTime, err := NewWatchTime("5000-2000")
				Expect(err).To(HaveOccurred())
				Expect(watchTime).To(Equal(WatchTime{}))
			})
		})

		Context("when range fails to parse", func() {
			It("returns an error", func() {
				watchTime, err := NewWatchTime("not-an-integer")
				Expect(err).To(HaveOccurred())
				Expect(watchTime).To(Equal(WatchTime{}))
			})
		})
	})
})
