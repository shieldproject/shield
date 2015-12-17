package supervisor_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/starkandwayne/shield/supervisor"
)

var _ = Describe("Supervisor Error Handling", func() {
	Context("When handling sentences", func() {
		strings := func(l ...string) []string {
			return l
		}

		It("handles the empty array", func() {
			Ω(Sentencify(strings())).Should(Equal(""))
		})

		It("handles an array of one word", func() {
			Ω(Sentencify(strings("thing"))).Should(Equal("thing"))
		})

		It("handles an array of two words", func() {
			Ω(Sentencify(strings("this", "that"))).Should(Equal("this and that"))
		})

		It("handles an array of three words", func() {
			Ω(Sentencify(strings("this", "that", "the other"))).Should(Equal("this, that and the other"))
		})

		It("handles an array of four words", func() {
			Ω(Sentencify(strings("a", "b", "c", "d"))).Should(Equal("a, b, c and d"))
		})
	})

	Context("Missing Parameter errors", func() {
		It("stringifies errors properly", func() {
			err := MissingParameters("name", "summary")
			Ω(err).ShouldNot(BeNil())
			Ω(err.Error()).Should(Equal("Missing name and summary parameters"))
		})

		It("jsonifies errors properly", func() {
			err := MissingParameters("name", "summary")
			Ω(err).ShouldNot(BeNil())
			Ω(err.JSON()).Should(Equal(`{"missing":["name","summary"]}`))
		})
	})
})
