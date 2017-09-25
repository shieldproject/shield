// +build !windows

package vitals

import (
	"fmt"

	boshstats "github.com/cloudfoundry/bosh-agent/platform/stats"
)

func createLoadVitals(loadStats boshstats.CPULoad) []string {
	return []string{
		fmt.Sprintf("%.2f", loadStats.One),
		fmt.Sprintf("%.2f", loadStats.Five),
		fmt.Sprintf("%.2f", loadStats.Fifteen),
	}
}
