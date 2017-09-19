package version_test

import (
	"sort"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/go-semi-semantic/version"
)

var _ = Describe("AscSorting", func() {
	It("sorts in ascending order", func() {
		vers := []Version{
			MustNewVersionFromString("1.2+b"),
			MustNewVersionFromString("1.2"),
			MustNewVersionFromString("1.0.a"),
		}

		sort.Sort(AscSorting(vers))

		Expect(vers).To(Equal([]Version{
			MustNewVersionFromString("1.0.a"),
			MustNewVersionFromString("1.2"),
			MustNewVersionFromString("1.2+b"),
		}))
	})
})
