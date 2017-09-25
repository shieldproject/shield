// +build windows

package monitor

import (
	"fmt"
	"math"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var cpuTestCases = []struct {
	prevKernel, prevUser, prevIdle uint64
	currKernel, currUser, currIdle uint64
	loadKernel, loadUser, loadIdle float64
}{
	{
		0, 0, 0,
		300, 500, 200,
		0.125, 0.625, 0.25,
	},
	{
		300, 500, 200,
		600, 1000, 400,
		0.125, 0.625, 0.25,
	},
	{
		300, 500, 200,
		300, 500, 100, // Regression
		0, 0, 0,
	},
}

func matchFloat(val, exp float64) error {
	if math.Abs(val-exp) < 0.000001 {
		return nil
	}
	return fmt.Errorf("Expected %.6f to Equal %.6f", val, exp)
}

var _ = Describe("CPU", func() {
	Context("when calculating CPU usage", func() {
		It("should correctly report SystemLevel User, Kernel and Idle", func() {
			for _, x := range cpuTestCases {
				m := Monitor{}
				m.kernel.previous = x.prevKernel
				m.user.previous = x.prevUser
				m.idle.previous = x.prevIdle
				m.calculateSystemCPU(x.currKernel, x.currUser, x.currIdle)
				Expect(matchFloat(m.kernel.load, x.loadKernel)).To(Succeed())
				Expect(matchFloat(m.user.load, x.loadUser)).To(Succeed())
				Expect(matchFloat(m.idle.load, x.loadIdle)).To(Succeed())
			}
		})
	})
})
