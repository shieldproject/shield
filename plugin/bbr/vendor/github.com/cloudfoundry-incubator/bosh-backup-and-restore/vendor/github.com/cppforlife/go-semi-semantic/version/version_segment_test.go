package version_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/go-semi-semantic/version"
)

var _ = Describe("MustNewVersionSegmentFromString", func() {
	It("parses valid version segment successfully", func() {
		verSeg := MustNewVersionSegmentFromString("dev.0")
		Expect(verSeg.AsString()).To(Equal("dev.0"))
	})

	It("panics on invalid version segment", func() {
		Expect(func() { MustNewVersionSegmentFromString("") }).To(Panic())
	})
})

var _ = Describe("NewVersionSegmentFromString", func() {
	It("handles one or more non-negative numerical components", func() {
		components := []VerSegComp{VerSegCompInt{1}}
		Expect(MustNewVersionSegmentFromString("1").Components).To(Equal(components))

		components = []VerSegComp{VerSegCompInt{1}, VerSegCompInt{1}}
		Expect(MustNewVersionSegmentFromString("1.1").Components).To(Equal(components))

		components = []VerSegComp{VerSegCompInt{1}, VerSegCompInt{1}, VerSegCompInt{1}}
		Expect(MustNewVersionSegmentFromString("1.1.1").Components).To(Equal(components))

		components = []VerSegComp{VerSegCompInt{1}, VerSegCompInt{1}, VerSegCompInt{1}, VerSegCompInt{1}}
		Expect(MustNewVersionSegmentFromString("1.1.1.1").Components).To(Equal(components))

		components = []VerSegComp{VerSegCompInt{123}, VerSegCompInt{0}, VerSegCompInt{1}}
		Expect(MustNewVersionSegmentFromString("123.0.1").Components).To(Equal(components))
	})

	It("handles negative numerical components as strings", func() {
		components := []VerSegComp{VerSegCompStr{"-1"}}
		Expect(MustNewVersionSegmentFromString("-1").Components).To(Equal(components))

		components = []VerSegComp{VerSegCompInt{0}, VerSegCompStr{"-1"}}
		Expect(MustNewVersionSegmentFromString("0.-1").Components).To(Equal(components))

		components = []VerSegComp{VerSegCompStr{"-0"}}
		Expect(MustNewVersionSegmentFromString("-0").Components).To(Equal(components))
	})

	It("handles numbers that start with '0' as strings", func() {
		components := []VerSegComp{VerSegCompStr{"0000"}}
		Expect(MustNewVersionSegmentFromString("0000").Components).To(Equal(components))

		components = []VerSegComp{VerSegCompStr{"01"}}
		Expect(MustNewVersionSegmentFromString("01").Components).To(Equal(components))
	})

	It("handles alphanumerics, hyphens & underscores in components as strings", func() {
		components := []VerSegComp{VerSegCompStr{"a"}}
		Expect(MustNewVersionSegmentFromString("a").Components).To(Equal(components))

		components = []VerSegComp{VerSegCompStr{"a"}, VerSegCompStr{"b"}, VerSegCompStr{"c"}}
		Expect(MustNewVersionSegmentFromString("a.b.c").Components).To(Equal(components))

		components = []VerSegComp{VerSegCompStr{"1-1"}}
		Expect(MustNewVersionSegmentFromString("1-1").Components).To(Equal(components))

		components = []VerSegComp{VerSegCompStr{"alpha-12"}, VerSegCompInt{5}, VerSegCompStr{"-"}}
		Expect(MustNewVersionSegmentFromString("alpha-12.5.-").Components).To(Equal(components))

		components = []VerSegComp{VerSegCompInt{1}, VerSegCompInt{2}, VerSegCompInt{3}, VerSegCompStr{"alpha"}}
		Expect(MustNewVersionSegmentFromString("1.2.3.alpha").Components).To(Equal(components))

		components = []VerSegComp{VerSegCompStr{"2013-03-21_01-53-17"}}
		Expect(MustNewVersionSegmentFromString("2013-03-21_01-53-17").Components).To(Equal(components))
	})

	It("returns an error for the empty string", func() {
		_, err := NewVersionSegmentFromString("")
		Expect(err).To(HaveOccurred())
	})

	It("raises an ParseError for non-alphanumeric, non-hyphen, non-underscore characters", func() {
		for _, inalidStr := range []string{"+", "&", " ", "\\u{6666}", "1.\\u{6666}"} {
			_, err := NewVersionSegmentFromString(inalidStr)
			Expect(err).To(HaveOccurred()) // ParseError
		}
	})
})

var _ = Describe("NewVersionSegment", func() {
	It("saves the supplied components", func() {
		components := []VerSegComp{
			VerSegCompInt{1},
			VerSegCompInt{2},
			VerSegCompInt{3},
		}
		Expect(MustNewVersionSegment(components).Components).To(Equal(components))
	})

	It("returns an error for an empty array", func() {
		_, err := NewVersionSegment([]VerSegComp{})
		Expect(err).To(HaveOccurred())
	})

	It("returns an error for the empty string", func() {
		_, err := NewVersionSegment([]VerSegComp{VerSegCompStr{""}})
		Expect(err).To(HaveOccurred())

		_, err = NewVersionSegment([]VerSegComp{VerSegCompInt{0}, VerSegCompStr{""}})
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("VersionSegment", func() {
	Describe("Increment", func() {
		It("increases the least significant component by default", func() {
			components := []VerSegComp{VerSegCompInt{1}}
			verSeg, err := MustNewVersionSegment(components).Increment()
			Expect(err).ToNot(HaveOccurred())
			Expect(verSeg.AsString()).To(Equal("2"))

			components = []VerSegComp{VerSegCompInt{1}, VerSegCompInt{1}, VerSegCompInt{1}, VerSegCompInt{1}}
			verSeg, err = MustNewVersionSegment(components).Increment()
			Expect(err).ToNot(HaveOccurred())
			Expect(verSeg.AsString()).To(Equal("1.1.1.2"))

			components = []VerSegComp{VerSegCompInt{1}, VerSegCompStr{"a"}, VerSegCompInt{0}, VerSegCompInt{0}}
			verSeg, err = MustNewVersionSegment(components).Increment()
			Expect(err).ToNot(HaveOccurred())
			Expect(verSeg.AsString()).To(Equal("1.a.0.1"))
		})

		It("does not affect original version segment", func() {
			components := []VerSegComp{VerSegCompInt{1}}
			origVerSeg := MustNewVersionSegment(components)
			verSeg, err := origVerSeg.Increment()
			Expect(err).ToNot(HaveOccurred())
			Expect(verSeg.AsString()).To(Equal("2"))
			Expect(origVerSeg.AsString()).To(Equal("1"))

			components = []VerSegComp{VerSegCompInt{1}, VerSegCompInt{1}, VerSegCompInt{1}, VerSegCompInt{1}}
			origVerSeg = MustNewVersionSegment(components)
			verSeg, err = origVerSeg.Increment()
			Expect(err).ToNot(HaveOccurred())
			Expect(verSeg.AsString()).To(Equal("1.1.1.2"))
			Expect(origVerSeg.AsString()).To(Equal("1.1.1.1"))
		})

		It("raises an error if last index is not an integer", func() {
			components := []VerSegComp{VerSegCompStr{"a"}}
			_, err := MustNewVersionSegment(components).Increment()
			Expect(err).To(HaveOccurred())

			components = []VerSegComp{VerSegCompInt{1}, VerSegCompInt{1}, VerSegCompInt{1}, VerSegCompStr{"a"}}
			_, err = MustNewVersionSegment(components).Increment()
			Expect(err).To(HaveOccurred())

			components = []VerSegComp{VerSegCompStr{"-1"}, VerSegCompStr{"a"}}
			_, err = MustNewVersionSegment(components).Increment()
			Expect(err).To(HaveOccurred())
		})

		// does not currently support increment at specific position
		// github.com/pivotal-cf-experimental/semi_semantic/blob/master/spec/semi_semantic/version_segment_spec.rb#L88
	})

	Describe("Copy", func() {
		It("does not affect original version segment", func() {
			origVerSeg := MustNewVersionSegmentFromString("1.1")
			newVerSeg := origVerSeg.Copy()
			newVerSeg.Components = append(newVerSeg.Components, VerSegCompInt{1})
			Expect(origVerSeg.AsString()).To(Equal("1.1"))
		})
	})

	Describe("String", func() {
		It("returns friendly value", func() {
			Expect(MustNewVersionSegmentFromString("1.a").String()).To(Equal("1.a"))
		})
	})

	Describe("AsString", func() {
		It("joins the version clusters with separators", func() {
			components := []VerSegComp{VerSegCompInt{1}}
			Expect(MustNewVersionSegment(components).AsString()).To(Equal("1"))

			components = []VerSegComp{VerSegCompInt{1}, VerSegCompInt{1}, VerSegCompInt{1}, VerSegCompInt{1}}
			Expect(MustNewVersionSegment(components).AsString()).To(Equal("1.1.1.1"))

			components = []VerSegComp{VerSegCompInt{1}, VerSegCompStr{"a"}, VerSegCompInt{1}, VerSegCompInt{1}}
			Expect(MustNewVersionSegment(components).AsString()).To(Equal("1.a.1.1"))

			components = []VerSegComp{VerSegCompInt{1}, VerSegCompStr{"a"}, VerSegCompInt{1}, VerSegCompStr{"-1"}}
			Expect(MustNewVersionSegment(components).AsString()).To(Equal("1.a.1.-1"))
		})
	})

	Describe("Compare", func() {
		It("assumes appended zeros", func() {
			l := []VerSegComp{VerSegCompInt{0}}
			r := []VerSegComp{VerSegCompInt{0}, VerSegCompInt{0}}
			Expect(MustNewVersionSegment(l).IsEq(MustNewVersionSegment(r))).To(BeTrue())

			l = []VerSegComp{VerSegCompInt{1}, VerSegCompInt{0}}
			r = []VerSegComp{VerSegCompInt{1}, VerSegCompInt{0}, VerSegCompInt{0}, VerSegCompInt{0}}
			Expect(MustNewVersionSegment(l).IsEq(MustNewVersionSegment(r))).To(BeTrue())

			l = []VerSegComp{VerSegCompInt{1}, VerSegCompInt{2}, VerSegCompInt{3}}
			r = []VerSegComp{VerSegCompInt{1}, VerSegCompInt{2}, VerSegCompInt{3}, VerSegCompInt{0}}
			Expect(MustNewVersionSegment(l).IsEq(MustNewVersionSegment(r))).To(BeTrue())

			l = []VerSegComp{VerSegCompStr{"a"}}
			r = []VerSegComp{VerSegCompStr{"a"}, VerSegCompInt{0}}
			Expect(MustNewVersionSegment(l).IsEq(MustNewVersionSegment(r))).To(BeTrue())

			l = []VerSegComp{VerSegCompInt{1}, VerSegCompInt{0}}
			r = []VerSegComp{VerSegCompInt{0}}
			Expect(MustNewVersionSegment(l).IsGt(MustNewVersionSegment(r))).To(BeTrue())

			l = []VerSegComp{VerSegCompInt{1}, VerSegCompInt{1}, VerSegCompInt{1}}
			r = []VerSegComp{VerSegCompInt{1}, VerSegCompInt{1}}
			Expect(MustNewVersionSegment(l).IsGt(MustNewVersionSegment(r))).To(BeTrue())

			l = []VerSegComp{VerSegCompInt{0}}
			r = []VerSegComp{VerSegCompInt{1}, VerSegCompInt{0}}
			Expect(MustNewVersionSegment(l).IsLt(MustNewVersionSegment(r))).To(BeTrue())

			l = []VerSegComp{VerSegCompInt{1}, VerSegCompInt{1}}
			r = []VerSegComp{VerSegCompInt{1}, VerSegCompInt{1}, VerSegCompInt{1}}
			Expect(MustNewVersionSegment(l).IsLt(MustNewVersionSegment(r))).To(BeTrue())
		})

		It("compares integers numerically", func() {
			l := []VerSegComp{VerSegCompInt{1}}
			r := []VerSegComp{VerSegCompInt{1}}
			Expect(MustNewVersionSegment(l).IsEq(MustNewVersionSegment(r))).To(BeTrue())

			l = []VerSegComp{VerSegCompInt{2}}
			r = []VerSegComp{VerSegCompInt{1}}
			Expect(MustNewVersionSegment(l).IsGt(MustNewVersionSegment(r))).To(BeTrue())

			l = []VerSegComp{VerSegCompInt{1}}
			r = []VerSegComp{VerSegCompInt{0}}
			Expect(MustNewVersionSegment(l).IsGt(MustNewVersionSegment(r))).To(BeTrue())

			l = []VerSegComp{VerSegCompInt{1}, VerSegCompInt{2}, VerSegCompInt{4}}
			r = []VerSegComp{VerSegCompInt{1}, VerSegCompInt{2}, VerSegCompInt{3}}
			Expect(MustNewVersionSegment(l).IsGt(MustNewVersionSegment(r))).To(BeTrue())
		})

		It("compares strings alpha-numerically", func() {
			l := []VerSegComp{VerSegCompStr{"a"}}
			r := []VerSegComp{VerSegCompStr{"a"}}
			Expect(MustNewVersionSegment(l).IsEq(MustNewVersionSegment(r))).To(BeTrue())

			l = []VerSegComp{VerSegCompStr{"beta"}, VerSegCompInt{1}}
			r = []VerSegComp{VerSegCompStr{"alpha"}, VerSegCompInt{1}}
			Expect(MustNewVersionSegment(l).IsGt(MustNewVersionSegment(r))).To(BeTrue())

			l = []VerSegComp{VerSegCompStr{"123abc"}}
			r = []VerSegComp{VerSegCompStr{"123ab"}}
			Expect(MustNewVersionSegment(l).IsGt(MustNewVersionSegment(r))).To(BeTrue())

			l = []VerSegComp{VerSegCompStr{"a"}}
			r = []VerSegComp{VerSegCompStr{"a"}, VerSegCompInt{1}}
			Expect(MustNewVersionSegment(l).IsLt(MustNewVersionSegment(r))).To(BeTrue())

			l = []VerSegComp{VerSegCompStr{"123ab"}}
			r = []VerSegComp{VerSegCompStr{"123abc"}}
			Expect(MustNewVersionSegment(l).IsLt(MustNewVersionSegment(r))).To(BeTrue())

			l = []VerSegComp{VerSegCompStr{"2013-03-21_01-53-17"}}
			r = []VerSegComp{VerSegCompStr{"2013-03-21_12-00-00"}}
			Expect(MustNewVersionSegment(l).IsLt(MustNewVersionSegment(r))).To(BeTrue())
		})

		It("values numbers lower than non-numbers", func() {
			l := []VerSegComp{VerSegCompInt{1}}
			r := []VerSegComp{VerSegCompStr{"a"}}
			Expect(MustNewVersionSegment(l).IsLt(MustNewVersionSegment(r))).To(BeTrue())

			l = []VerSegComp{VerSegCompInt{1}, VerSegCompStr{"a"}}
			r = []VerSegComp{VerSegCompInt{1}, VerSegCompInt{0}}
			Expect(MustNewVersionSegment(l).IsGt(MustNewVersionSegment(r))).To(BeTrue())
		})
	})
})
