package fmt_test

import (
	"time"

	. "github.com/cloudfoundry/bosh-init/ui/fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Duration", describeDuration)

func describeDuration() {
	var duration time.Duration

	Context("when given a duration less than one minute", func() {
		BeforeEach(func() {
			duration = 59 * time.Second
		})

		It("returns a string in 00:00:00 format", func() {
			Expect(Duration(duration)).To(Equal("00:00:59"))
		})
	})

	Context("when given a duration greater than one minute", func() {
		BeforeEach(func() {
			duration = 69 * time.Second
		})

		It("returns a string in 00:00:00 format", func() {
			Expect(Duration(duration)).To(Equal("00:01:09"))
		})
	})

	Context("when given a duration greater than one hour", func() {
		BeforeEach(func() {
			duration = 3669 * time.Second
		})

		It("returns a string in 00:00:00 format", func() {
			Expect(Duration(duration)).To(Equal("01:01:09"))
		})
	})
}
