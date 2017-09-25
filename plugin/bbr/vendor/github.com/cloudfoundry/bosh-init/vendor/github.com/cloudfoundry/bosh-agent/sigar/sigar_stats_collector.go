package sigar

import (
	"sync"
	"time"

	sigar "github.com/cloudfoundry/gosigar"

	boshstats "github.com/cloudfoundry/bosh-agent/platform/stats"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type sigarStatsCollector struct {
	statsSigar         sigar.Sigar
	latestCPUStats     boshstats.CPUStats
	latestCPUStatsLock sync.RWMutex
}

func NewSigarStatsCollector(sigar sigar.Sigar) boshstats.Collector {
	return &sigarStatsCollector{
		statsSigar: sigar,
	}
}

func (s *sigarStatsCollector) StartCollecting(collectionInterval time.Duration, latestGotUpdated chan struct{}) {
	cpuSamplesCh, _ := s.statsSigar.CollectCpuStats(collectionInterval)

	go func() {
		for cpuSample := range cpuSamplesCh {
			s.latestCPUStatsLock.Lock()
			s.latestCPUStats.User = cpuSample.User
			s.latestCPUStats.Nice = cpuSample.Nice
			s.latestCPUStats.Sys = cpuSample.Sys
			s.latestCPUStats.Wait = cpuSample.Wait
			s.latestCPUStats.Total = cpuSample.Total()
			s.latestCPUStatsLock.Unlock()

			if latestGotUpdated != nil {
				latestGotUpdated <- struct{}{}
			}
		}
	}()
}

func (s *sigarStatsCollector) GetCPULoad() (load boshstats.CPULoad, err error) {
	l, err := s.statsSigar.GetLoadAverage()
	if err != nil {
		if err != sigar.ErrNotImplemented {
			err = bosherr.WrapError(err, "Getting Sigar Load Average")
		}
		return
	}

	load.One = l.One
	load.Five = l.Five
	load.Fifteen = l.Fifteen

	return
}

func (s *sigarStatsCollector) GetCPUStats() (boshstats.CPUStats, error) {
	s.latestCPUStatsLock.RLock()
	defer s.latestCPUStatsLock.RUnlock()

	return s.latestCPUStats, nil
}

func (s *sigarStatsCollector) GetMemStats() (usage boshstats.Usage, err error) {
	mem, err := s.statsSigar.GetMem()
	if err != nil {
		err = bosherr.WrapError(err, "Getting Sigar Mem")
		return
	}

	usage.Total = mem.Total

	// actual_used = mem->used - (kern_buffers + kern_cached)
	// (https://github.com/hyperic/sigar/blob/1898438/src/os/linux/linux_sigar.c#L344)
	usage.Used = mem.ActualUsed

	return
}

func (s *sigarStatsCollector) GetSwapStats() (usage boshstats.Usage, err error) {
	swap, err := s.statsSigar.GetSwap()
	if err != nil {
		err = bosherr.WrapError(err, "Getting Sigar Swap")
		return
	}

	usage.Total = swap.Total
	usage.Used = swap.Used

	return
}

func (s *sigarStatsCollector) GetDiskStats(mountedPath string) (stats boshstats.DiskStats, err error) {
	fsUsage, err := s.statsSigar.GetFileSystemUsage(mountedPath)
	if err != nil {
		err = bosherr.WrapError(err, "Getting Sigar File System Usage")
		return
	}

	stats.DiskUsage.Total = fsUsage.Total
	stats.DiskUsage.Used = fsUsage.Used
	stats.InodeUsage.Total = fsUsage.Files
	stats.InodeUsage.Used = fsUsage.Files - fsUsage.FreeFiles

	return
}
