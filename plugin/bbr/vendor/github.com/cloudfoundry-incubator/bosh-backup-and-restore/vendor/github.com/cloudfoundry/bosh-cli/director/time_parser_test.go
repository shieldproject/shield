package director_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/director"
)

var _ = Describe("TimeParser", func() {
	Describe("Parse", func() {
		It("returns parsed time", func() {
			in := "2016-01-09 06:23:25 +0000"

			parsed, err := TimeParser{}.Parse(in)
			Expect(err).ToNot(HaveOccurred())
			Expect(parsed).To(Equal(time.Date(2016, time.January, 9, 6, 23, 25, 0, time.UTC)))

			in2 := "2016-08-25 00:17:16 UTC"

			parsed, err = TimeParser{}.Parse(in2)
			Expect(err).ToNot(HaveOccurred())
			Expect(parsed).To(Equal(time.Date(2016, time.August, 25, 0, 17, 16, 0, time.UTC)))
		})

		It("returns error if none of the formats match", func() {
			_, err := TimeParser{}.Parse("2016")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`parsing time "2016"`))
		})
	})
})
