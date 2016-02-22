package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/starkandwayne/shield/cmd/shield"
)

var _ = Describe("CLI Duration", func() {
	It("Should convert duration correctly ignoring whitespace characters", func() {
		DAYS := " 30d"
		WKS := "4w "
		YRS := "1 y"
		day_val, err := ParseDuration(DAYS)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(day_val.String()).Should(Equal("30d"))

		wk_val, err := ParseDuration(WKS)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(wk_val.String()).Should(Equal("4w"))

		yr_val, err := ParseDuration(YRS)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(yr_val.String()).Should(Equal("1y"))
	})

	It("Should default to days ignoring whitespace characters", func() {
		DEF := "          \t        32    \t     \t\t\t\t\r\n\r\n"
		def_val, err := ParseDuration(DEF)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(def_val.String()).Should(Equal("32d"))
	})

	It("Should fail with an invalid duration or units", func() {
		INVV := "ty"
		INVU := "4u"
		INVAL := "#&*"

		invv_val, err := ParseDuration(INVV)
		Ω(invv_val).Should(BeNil())
		Ω(err).Should(HaveOccurred())

		invu_val, err := ParseDuration(INVU)
		Ω(invu_val).Should(BeNil())
		Ω(err).Should(HaveOccurred())

		inval_val, err := ParseDuration(INVAL)
		Ω(inval_val).Should(BeNil())
		Ω(err).Should(HaveOccurred())
	})
})
