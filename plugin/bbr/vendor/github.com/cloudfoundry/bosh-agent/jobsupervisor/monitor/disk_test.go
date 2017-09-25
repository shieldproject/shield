// +build windows

package monitor

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Disk", func() {
	It("should report the percent used", func() {
		du := diskUsage{
			SectorsPerCluster:     8,
			BytesPerSector:        512,
			NumberOfFreeClusters:  6948117,
			TotalNumberOfClusters: 15638527,
		}
		expPer := 0.55570515049148
		exp := DiskUsage{
			Total: 64055406592,
			Used:  35595919360,
		}
		u := newDiskUsage(du)
		Expect(u).To(Equal(exp))
		Expect(matchFloat(u.Percent(), expPer)).To(Succeed())
	})
})
