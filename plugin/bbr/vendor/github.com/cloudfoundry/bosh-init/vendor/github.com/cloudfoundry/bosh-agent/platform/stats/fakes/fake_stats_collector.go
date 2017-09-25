package fakes

import (
	"errors"
	"time"

	boshstats "github.com/cloudfoundry/bosh-agent/platform/stats"
)

type FakeCollector struct {
	StartCollectingCPUStats boshstats.CPUStats

	CPULoad  boshstats.CPULoad
	cpuStats boshstats.CPUStats

	MemStats    boshstats.Usage
	MemStatsErr error

	SwapStats boshstats.Usage
	DiskStats map[string]boshstats.DiskStats
}

func (c *FakeCollector) StartCollecting(collectionInterval time.Duration, latestGotUpdated chan struct{}) {
	c.cpuStats = c.StartCollectingCPUStats
}

func (c *FakeCollector) GetCPULoad() (load boshstats.CPULoad, err error) {
	load = c.CPULoad
	return
}

func (c *FakeCollector) GetCPUStats() (stats boshstats.CPUStats, err error) {
	stats = c.cpuStats
	return
}

func (c *FakeCollector) GetMemStats() (boshstats.Usage, error) {
	return c.MemStats, c.MemStatsErr
}

func (c *FakeCollector) GetSwapStats() (usage boshstats.Usage, err error) {
	usage = c.SwapStats
	return
}

func (c *FakeCollector) GetDiskStats(devicePath string) (stats boshstats.DiskStats, err error) {
	stats, found := c.DiskStats[devicePath]
	if !found {
		err = errors.New("Disk not found")
	}
	return
}
