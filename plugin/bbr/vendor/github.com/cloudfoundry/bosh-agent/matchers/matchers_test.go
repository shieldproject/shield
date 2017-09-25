package matchers_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/matchers"
)

var _ = Describe("MatchesOneOf", func() {
	It("requires at least two matchers", func() {
		Expect(func() {
			MatchOneOf()
		}).To(Panic())

		matcher1 := BeTrue()
		Expect(func() {
			MatchOneOf(matcher1)
		}).To(Panic())

		matcher2 := BeFalse()
		Expect(func() {
			MatchOneOf(matcher1, matcher2)
		}).ToNot(Panic())
	})
})
