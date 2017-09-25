package fakes

import (
	boshdevutil "github.com/cloudfoundry/bosh-agent/platform/deviceutil"
	fakedevutil "github.com/cloudfoundry/bosh-agent/platform/deviceutil/fakes"
	boshdisk "github.com/cloudfoundry/bosh-agent/platform/disk"
)

type FakeDiskManager struct {
	FakePartitioner           *FakePartitioner
	FakeFormatter             *FakeFormatter
	FakeMounter               *FakeMounter
	FakeMountsSearcher        *FakeMountsSearcher
	FakeRootDevicePartitioner *FakePartitioner
	FakeDiskUtil              *fakedevutil.FakeDeviceUtil
	DiskUtilDiskPath          string
	PartedPartitionerCalled   bool
	PartitionerCalled         bool
}

func NewFakeDiskManager() *FakeDiskManager {
	return &FakeDiskManager{
		FakePartitioner:           NewFakePartitioner(),
		FakeFormatter:             &FakeFormatter{},
		FakeMounter:               &FakeMounter{},
		FakeMountsSearcher:        &FakeMountsSearcher{},
		FakeRootDevicePartitioner: NewFakePartitioner(),
		FakeDiskUtil:              fakedevutil.NewFakeDeviceUtil(),
		PartedPartitionerCalled:   false,
		PartitionerCalled:         false,
	}
}

func (m *FakeDiskManager) GetPartitioner() boshdisk.Partitioner {
	m.PartitionerCalled = true
	return m.FakePartitioner
}

func (m *FakeDiskManager) GetPartedPartitioner() boshdisk.Partitioner {
	m.PartedPartitionerCalled = true
	return m.FakePartitioner
}

func (m *FakeDiskManager) GetRootDevicePartitioner() boshdisk.Partitioner {
	return m.FakeRootDevicePartitioner
}

func (m *FakeDiskManager) GetFormatter() boshdisk.Formatter {
	return m.FakeFormatter
}

func (m *FakeDiskManager) GetMounter() boshdisk.Mounter {
	return m.FakeMounter
}

func (m *FakeDiskManager) GetMountsSearcher() boshdisk.MountsSearcher {
	return m.FakeMountsSearcher
}

func (m *FakeDiskManager) GetDiskUtil(diskPath string) boshdevutil.DeviceUtil {
	m.DiskUtilDiskPath = diskPath
	return m.FakeDiskUtil
}
