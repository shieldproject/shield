package core

import (
	"github.com/goccy/go-yaml"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Duration YAML Unmarshaling", func() {
	type wrapper struct {
		D duration `yaml:"d"`
	}

	DescribeTable("parses duration strings",
		func(input string, expected int) {
			var w wrapper
			err := yaml.Unmarshal([]byte("d: "+input), &w)
			Expect(err).NotTo(HaveOccurred())
			Expect(int(w.D)).To(Equal(expected))
		},
		Entry("seconds", "30s", 30),
		Entry("minutes", "5m", 300),
		Entry("hours", "1h", 3600),
		Entry("days", "90d", 90*86400),
		Entry("weeks", "2w", 2*7*86400),
		Entry("years", "1y", 365*86400),
		Entry("bare integer as string", "\"300\"", 300),
		Entry("fractional hours", "1.5h", 5400),
		Entry("uppercase unit", "2D", 2*86400),
	)

	It("parses bare integer YAML values", func() {
		var w wrapper
		err := yaml.Unmarshal([]byte("d: 300"), &w)
		Expect(err).NotTo(HaveOccurred())
		Expect(int(w.D)).To(Equal(300))
	})

	It("round-trips a struct with multiple duration fields", func() {
		type config struct {
			Fast duration `yaml:"fast"`
			Slow duration `yaml:"slow"`
		}

		input := "fast: 1s\nslow: 5m\n"
		var c config
		err := yaml.Unmarshal([]byte(input), &c)
		Expect(err).NotTo(HaveOccurred())
		Expect(int(c.Fast)).To(Equal(1))
		Expect(int(c.Slow)).To(Equal(300))
	})

	It("handles nested struct with duration", func() {
		type nested struct {
			Scheduler struct {
				FastLoop duration `yaml:"fast-loop"`
				SlowLoop duration `yaml:"slow-loop"`
			} `yaml:"scheduler"`
		}

		input := "scheduler:\n  fast-loop: 1s\n  slow-loop: 300s\n"
		var n nested
		err := yaml.Unmarshal([]byte(input), &n)
		Expect(err).NotTo(HaveOccurred())
		Expect(int(n.Scheduler.FastLoop)).To(Equal(1))
		Expect(int(n.Scheduler.SlowLoop)).To(Equal(300))
	})
})
