package cmd_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	boshdir "github.com/cloudfoundry/bosh-cli/director"
)

var _ = Describe("NewValueStringPercent", func() {
	Describe("String", func() {
		It("returns empty string when is empty", func() {
			Expect(NewValueStringPercent("").String()).To(Equal(""))
		})

		It("returns suffixed percent", func() {
			Expect(NewValueStringPercent("10").String()).To(Equal("10%"))
		})
	})
})

var _ = Describe("ValueCPUTotal", func() {
	Describe("String", func() {
		It("returns empty string when it's nil", func() {
			Expect(ValueCPUTotal{nil}.String()).To(Equal(""))
		})

		It("returns formatted percent", func() {
			val := float64(110)
			Expect(ValueCPUTotal{&val}.String()).To(Equal("110.0%"))

			val = float64(40.11)
			Expect(ValueCPUTotal{&val}.String()).To(Equal("40.1%"))

			val = float64(0)
			Expect(ValueCPUTotal{&val}.String()).To(Equal("0.0%"))
		})
	})
})

var _ = Describe("ValueMemSize", func() {
	Describe("String", func() {
		It("returns empty string when percent or kb is 0", func() {
			size := boshdir.VMInfoVitalsMemSize{KB: "1", Percent: ""}
			Expect(ValueMemSize{size}.String()).To(Equal(""))

			size = boshdir.VMInfoVitalsMemSize{KB: "", Percent: "1"}
			Expect(ValueMemSize{size}.String()).To(Equal(""))
		})

		It("returns empty string when cannot parse kb", func() {
			size := boshdir.VMInfoVitalsMemSize{KB: "non-num", Percent: "100"}
			Expect(ValueMemSize{size}.String()).To(Equal(""))
		})

		It("returns percent and value", func() {
			size := boshdir.VMInfoVitalsMemSize{KB: "77", Percent: "10"}
			Expect(ValueMemSize{size}.String()).To(Equal("10% (77 kB)"))

			size = boshdir.VMInfoVitalsMemSize{KB: "123456", Percent: "10"}
			Expect(ValueMemSize{size}.String()).To(Equal("10% (124 MB)"))
		})
	})
})

var _ = Describe("ValueMemIntSize", func() {
	Describe("String", func() {
		It("returns empty string when percent or kb is 0", func() {
			size := boshdir.VMInfoVitalsMemIntSize{}
			Expect(ValueMemIntSize{size}.String()).To(Equal(""))

			kb := uint64(0)
			size = boshdir.VMInfoVitalsMemIntSize{KB: &kb, Percent: nil}
			Expect(ValueMemIntSize{size}.String()).To(Equal(""))

			per := float64(0)
			size = boshdir.VMInfoVitalsMemIntSize{KB: nil, Percent: &per}
			Expect(ValueMemIntSize{size}.String()).To(Equal(""))
		})

		It("returns percent and value", func() {
			kb := uint64(77)
			per := float64(100)
			size := boshdir.VMInfoVitalsMemIntSize{KB: &kb, Percent: &per}
			Expect(ValueMemIntSize{size}.String()).To(Equal("100.0% (77 kB)"))
		})
	})
})

var _ = Describe("ValueDiskSize", func() {
	Describe("String", func() {
		It("returns empty string when percent or kb is 0", func() {
			size := boshdir.VMInfoVitalsDiskSize{InodePercent: "1", Percent: ""}
			Expect(ValueDiskSize{size}.String()).To(Equal(""))

			size = boshdir.VMInfoVitalsDiskSize{InodePercent: "", Percent: "1"}
			Expect(ValueDiskSize{size}.String()).To(Equal(""))
		})

		It("returns percents", func() {
			size := boshdir.VMInfoVitalsDiskSize{InodePercent: "77", Percent: "11"}
			Expect(ValueDiskSize{size}.String()).To(Equal("11% (77i%)"))
		})
	})
})

var _ = Describe("ValueUptime", func() {
	Describe("String", func() {
		It("returns empty string when it's nil", func() {
			Expect(ValueUptime{nil}.String()).To(Equal(""))
		})

		It("returns days, hours, mins, sec", func() {
			secs := uint64(40)
			Expect(ValueUptime{&secs}.String()).To(Equal("0d 0h 0m 40s"))

			secs = uint64(60)
			Expect(ValueUptime{&secs}.String()).To(Equal("0d 0h 1m 0s"))

			secs = uint64(90)
			Expect(ValueUptime{&secs}.String()).To(Equal("0d 0h 1m 30s"))

			secs = uint64(3600)
			Expect(ValueUptime{&secs}.String()).To(Equal("0d 1h 0m 0s"))

			secs = uint64(3600 + 90)
			Expect(ValueUptime{&secs}.String()).To(Equal("0d 1h 1m 30s"))

			secs = uint64(3600*24 + 3690)
			Expect(ValueUptime{&secs}.String()).To(Equal("1d 1h 1m 30s"))
		})
	})
})
