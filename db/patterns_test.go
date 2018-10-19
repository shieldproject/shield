package db

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DB Pattern Conversion", func() {
	It("Properly converts globs to SQL patterns", func() {
		var tests = []struct {
			Glob    string
			Pattern string
		}{
			{Glob: "test", Pattern: "%test%"},
			{Glob: "t*st", Pattern: "%t%st%"},
			{Glob: "*", Pattern: "%"},
			{Glob: "**", Pattern: "%"},
			{Glob: "**t**", Pattern: "%t%"},
			{Glob: "*test*", Pattern: "%test%"},
		}

		for _, t := range tests {
			Ω(Pattern(t.Glob)).Should(Equal(t.Pattern))
		}
	})
})
