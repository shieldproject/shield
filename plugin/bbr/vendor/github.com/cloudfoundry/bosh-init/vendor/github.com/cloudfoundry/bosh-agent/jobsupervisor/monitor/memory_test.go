// +build windows

package monitor

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Memory", func() {
	Context("when create a new memstat", func() {
		It("should report the percent used", func() {
			var tests = []struct {
				Avail Byte
				Total Byte
				Exp   float64
			}{
				{Avail: 1024, Total: 2048, Exp: 0.5},
				{Avail: 0, Total: 0, Exp: 0},
				{Avail: 0, Total: 1024, Exp: 1},
				{Avail: 9216, Total: 10240, Exp: 0.10},
			}
			for _, x := range tests {
				m := MemStat{Avail: x.Avail, Total: x.Total}
				Expect(matchFloat(m.Used(), x.Exp)).To(Succeed())
			}
		})
	})
	Context("when defining Byte", func() {
		It("should print correctly", func() {
			var tests = []struct {
				Input Byte
				Exp   string
			}{
				{Input: 300, Exp: "300"},
				{Input: 1024, Exp: "1.0K"},
				{Input: 2000, Exp: "2.0K"},
				{Input: 1 * MB, Exp: "1.0M"},
				{Input: (14 * GB) / 10, Exp: "1.4G"},
			}
			for _, x := range tests {
				Expect(x.Input.String()).To(Equal(x.Exp))
			}
		})
	})
	Context("when parsing wmic output", func() {
		It("should parse the output", func() {
			res := `

AllocatedBaseSize=4791

CurrentUsage=393

Description=C:\pagefile.sys

InstallDate=20151221103329.285091-480

Name=C:\pagefile.sys

PeakUsage=2916

Status=

TempPageFile=FALSE


`
			out := []byte(res)
			num, err := parseWmicOutput(out, []byte("CurrentUsage"))
			Expect(err).To(BeNil())
			Expect(num).To(Equal(uint64(393)))

			num, err = parseWmicOutput(out, []byte("AllocatedBaseSize"))
			Expect(err).To(BeNil())
			Expect(num).To(Equal(uint64(4791)))

			num, err = parseWmicOutput(out, []byte("Status"))
			Expect(err).To(HaveOccurred())
			Expect(num).To(Equal(uint64(0)))

			num, err = parseWmicOutput(out, []byte("SOMETHINGELSE"))
			Expect(err).To(HaveOccurred())
			Expect(num).To(Equal(uint64(0)))
		})

	})

})
