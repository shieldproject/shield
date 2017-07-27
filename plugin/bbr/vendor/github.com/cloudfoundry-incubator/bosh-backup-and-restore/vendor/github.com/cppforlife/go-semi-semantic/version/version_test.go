package version_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/go-semi-semantic/version"
)

var _ = Describe("MustNewVersionFromString", func() {
	It("parses valid version successfully", func() {
		ver := MustNewVersionFromString("1.0.a")
		Expect(ver.AsString()).To(Equal("1.0.a"))
	})

	It("panics on invalid version", func() {
		Expect(func() { MustNewVersionFromString("-1") }).To(Panic())
	})
})

var _ = Describe("NewVersionFromString", func() {
	It("parses up to 3 segments", func() {
		segmentA := MustNewVersionSegmentFromString("1.0.a")
		segmentB := MustNewVersionSegmentFromString("1.0.b")
		segmentC := MustNewVersionSegmentFromString("1.0.c")

		Expect(MustNewVersionFromString("1.0.a-1.0.b+1.0.c").Segments).To(
			Equal([]VersionSegment{segmentA, segmentB, segmentC}))

		Expect(MustNewVersionFromString("1.0.a-1.0.b").Segments).To(
			Equal([]VersionSegment{segmentA, segmentB, VersionSegment{}}))

		Expect(MustNewVersionFromString("1.0.a+1.0.c").Segments).To(
			Equal([]VersionSegment{segmentA, VersionSegment{}, segmentC}))

		Expect(MustNewVersionFromString("1.0.a").Segments).To(
			Equal([]VersionSegment{segmentA, VersionSegment{}, VersionSegment{}}))
	})

	It("supports hyphenation in pre/post-release segments", func() {
		v := MustNewVersionFromString("1-1-1")
		Expect(v.Release).To(Equal(MustNewVersionSegmentFromString("1")))
		Expect(v.PreRelease).To(Equal(MustNewVersionSegmentFromString("1-1")))
		Expect(v.PostRelease).To(Equal(VersionSegment{}))

		v = MustNewVersionFromString("1+1-1")
		Expect(v.Release).To(Equal(MustNewVersionSegmentFromString("1")))
		Expect(v.PreRelease).To(Equal(VersionSegment{}))
		Expect(v.PostRelease).To(Equal(MustNewVersionSegmentFromString("1-1")))

		v = MustNewVersionFromString("1-1-1+1-1")
		Expect(v.Release).To(Equal(MustNewVersionSegmentFromString("1")))
		Expect(v.PreRelease).To(Equal(MustNewVersionSegmentFromString("1-1")))
		Expect(v.PostRelease).To(Equal(MustNewVersionSegmentFromString("1-1")))
	})

	It("raises a ParseError for empty segments", func() {
		for _, invalidStr := range []string{"+1", "1+", "-1", "1-", "1-+1", "1-1+"} {
			_, err := NewVersionFromString(invalidStr)
			Expect(err).To(HaveOccurred())
		}
	})

	It("raises a ParseError if multiple post-release segments", func() {
		_, err := NewVersionFromString("1+1+1")
		Expect(err).To(HaveOccurred())
	})

	It("raises an ArgumentError for the empty string", func() {
		_, err := NewVersionFromString("")
		Expect(err).To(HaveOccurred())
	})

	It("raises a ParseError for invalid characters", func() {
		_, err := NewVersionFromString(" ")
		Expect(err).To(HaveOccurred())

		_, err = NewVersionFromString("1 1")
		Expect(err).To(HaveOccurred())

		_, err = NewVersionFromString("can\"t do it cap\"n")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("Version", func() {
	Describe("release", func() {
		It("returns the first VersionCluster", func() {
			Expect(MustNewVersionFromString("1.0").Release).To(
				Equal(MustNewVersionSegmentFromString("1.0")))

			Expect(MustNewVersionFromString("1.0-alpha").Release).To(
				Equal(MustNewVersionSegmentFromString("1.0")))

			Expect(MustNewVersionFromString("1.0+dev").Release).To(
				Equal(MustNewVersionSegmentFromString("1.0")))

			Expect(MustNewVersionFromString("1.0-alpha+dev").Release).To(
				Equal(MustNewVersionSegmentFromString("1.0")))
		})
	})

	Describe("PreRelease", func() {
		It("returns the VersionCluster following the \"-\"", func() {
			Expect(MustNewVersionFromString("1.0").PreRelease).To(Equal(VersionSegment{}))

			Expect(MustNewVersionFromString("1.0-alpha").PreRelease).To(
				Equal(MustNewVersionSegmentFromString("alpha")))

			Expect(MustNewVersionFromString("1.0+dev").PreRelease).To(Equal(VersionSegment{}))

			Expect(MustNewVersionFromString("1.0-alpha+dev").PreRelease).To(
				Equal(MustNewVersionSegmentFromString("alpha")))
		})
	})

	Describe("PostRelease", func() {
		It("returns the VersionCluster following the \"+\"", func() {
			Expect(MustNewVersionFromString("1.0").PostRelease).To(Equal(VersionSegment{}))

			Expect(MustNewVersionFromString("1.0-alpha").PostRelease).To(Equal(VersionSegment{}))

			Expect(MustNewVersionFromString("1.0+dev").PostRelease).To(
				Equal(MustNewVersionSegmentFromString("dev")))

			Expect(MustNewVersionFromString("1.0-alpha+dev").PostRelease).To(
				Equal(MustNewVersionSegmentFromString("dev")))
		})
	})

	Describe("IncrementRelease", func() {
		It("increases release and does not keep pre and post releases", func() {
			ver, err := MustNewVersionFromString("1").IncrementRelease()
			Expect(err).ToNot(HaveOccurred())
			Expect(ver.AsString()).To(Equal("2"))

			ver, err = MustNewVersionFromString("1.1.1+1-1").IncrementRelease()
			Expect(err).ToNot(HaveOccurred())
			Expect(ver.AsString()).To(Equal("1.1.2"))

			ver, err = MustNewVersionFromString("1.a.0.1-1").IncrementRelease()
			Expect(err).ToNot(HaveOccurred())
			Expect(ver.AsString()).To(Equal("1.a.0.2"))
		})

		It("does not affect original version", func() {
			origVer := MustNewVersionFromString("1.1.1-1+1")
			newVer, err := origVer.IncrementRelease()
			Expect(err).ToNot(HaveOccurred())
			Expect(newVer.AsString()).To(Equal("1.1.2"))
			Expect(origVer.AsString()).To(Equal("1.1.1-1+1"))
		})

		It("raises an error if release cannot be incremented", func() {
			_, err := MustNewVersionFromString("1.a+1").IncrementRelease()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("IncrementPostRelease", func() {
		def := MustNewVersionSegmentFromString("0")

		It("increases post release keeping release and pre releases (does not use default)", func() {
			ver, err := MustNewVersionFromString("1.1+100").IncrementPostRelease(def)
			Expect(err).ToNot(HaveOccurred())
			Expect(ver.AsString()).To(Equal("1.1+101"))

			ver, err = MustNewVersionFromString("1.1.1-1+1").IncrementPostRelease(def)
			Expect(err).ToNot(HaveOccurred())
			Expect(ver.AsString()).To(Equal("1.1.1-1+2"))

			ver, err = MustNewVersionFromString("1.a.0.1-1+1.b.10").IncrementPostRelease(def)
			Expect(err).ToNot(HaveOccurred())
			Expect(ver.AsString()).To(Equal("1.a.0.1-1+1.b.11"))
		})

		It("uses default when post release is not present", func() {
			def := MustNewVersionSegmentFromString("dev.0")

			ver, err := MustNewVersionFromString("1.1").IncrementPostRelease(def)
			Expect(err).ToNot(HaveOccurred())
			Expect(ver.AsString()).To(Equal("1.1+dev.0"))

			ver, err = MustNewVersionFromString("1.1.1-1").IncrementPostRelease(def)
			Expect(err).ToNot(HaveOccurred())
			Expect(ver.AsString()).To(Equal("1.1.1-1+dev.0"))
		})

		It("does not affect original version", func() {
			origVer := MustNewVersionFromString("1.1.1-1+1")
			newVer, err := origVer.IncrementPostRelease(def)
			Expect(err).ToNot(HaveOccurred())
			Expect(newVer.AsString()).To(Equal("1.1.1-1+2"))
			Expect(origVer.AsString()).To(Equal("1.1.1-1+1"))
		})

		It("raises an error if default post release is empty", func() {
			_, err := MustNewVersionFromString("1.1+a").IncrementPostRelease(VersionSegment{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected default post relase to be non-empty"))
		})

		It("raises an error if release cannot be incremented", func() {
			_, err := MustNewVersionFromString("1.1+a").IncrementPostRelease(def)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Empty", func() {
		It("returns true when version is empty (init-ed as zero value)", func() {
			Expect(Version{}.Empty()).To(BeTrue())
		})

		It("returns false when version is not empty", func() {
			Expect(MustNewVersionFromString("1").Empty()).To(BeFalse())
			Expect(MustNewVersionFromString("1-1+1").Empty()).To(BeFalse())
		})
	})

	Describe("String", func() {
		It("returns friendly value", func() {
			Expect(MustNewVersionSegmentFromString("1.1.1.1").String()).To(Equal("1.1.1.1"))
		})
	})

	Describe("AsString", func() {
		It("joins the version clusters with separators", func() {
			release := MustNewVersionSegmentFromString("1.1.1.1")
			preRelease := MustNewVersionSegmentFromString("2.2.2.2")
			postRelease := MustNewVersionSegmentFromString("3.3.3.3")

			Expect(MustNewVersion(release, VersionSegment{}, VersionSegment{}).AsString()).To(Equal("1.1.1.1"))
			Expect(MustNewVersion(release, preRelease, VersionSegment{}).AsString()).To(Equal("1.1.1.1-2.2.2.2"))
			Expect(MustNewVersion(release, VersionSegment{}, postRelease).AsString()).To(Equal("1.1.1.1+3.3.3.3"))
			Expect(MustNewVersion(release, preRelease, postRelease).AsString()).To(Equal("1.1.1.1-2.2.2.2+3.3.3.3"))
		})
	})

	Describe("Compare", func() {
		It("handles equivalence", func() {
			Expect(MustNewVersionFromString("1.0").IsEq(MustNewVersionFromString("1.0"))).To(BeTrue())
			Expect(MustNewVersionFromString("1.0").IsEq(MustNewVersionFromString("1.0.0"))).To(BeTrue())
			Expect(MustNewVersionFromString("1-1+1").IsEq(MustNewVersionFromString("1-1+1"))).To(BeTrue())
			Expect(MustNewVersionFromString("1-1+0").IsEq(MustNewVersionFromString("1-1"))).To(BeFalse())
		})

		It("treats nil pre/post-release as distinct from zeroed pre/post-release", func() {
			Expect(MustNewVersionFromString("1-0+1").IsEq(MustNewVersionFromString("1+1"))).To(BeFalse())
			Expect(MustNewVersionFromString("1-1+0").IsEq(MustNewVersionFromString("1-1"))).To(BeFalse())
		})

		It("treats pre-release as less than release", func() {
			Expect(MustNewVersionFromString("1.0-alpha").IsLt(MustNewVersionFromString("1.0"))).To(BeTrue())
		})

		It("treats post-release as greater than release", func() {
			Expect(MustNewVersionFromString("1.0+dev").IsGt(MustNewVersionFromString("1.0"))).To(BeTrue())
		})
	})
})
