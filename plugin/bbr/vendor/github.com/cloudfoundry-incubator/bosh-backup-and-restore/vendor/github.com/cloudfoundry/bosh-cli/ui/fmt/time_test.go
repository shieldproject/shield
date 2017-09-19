package fmt_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/ui/fmt"
)

var _ = Describe("Duration", func() {
	Context("when given a duration less than one minute", func() {
		It("returns a string in 00:00:00 format", func() {
			duration := time.Duration(59 * time.Second)
			Expect(Duration(duration)).To(Equal("00:00:59"))
		})
	})

	Context("when given a duration greater than one minute", func() {
		It("returns a string in 00:00:00 format", func() {
			duration := time.Duration(69 * time.Second)
			Expect(Duration(duration)).To(Equal("00:01:09"))
		})
	})

	Context("when given a duration greater than one hour", func() {
		It("returns a string in 00:00:00 format", func() {
			duration := time.Duration(3669 * time.Second)
			Expect(Duration(duration)).To(Equal("01:01:09"))
		})
	})
})
