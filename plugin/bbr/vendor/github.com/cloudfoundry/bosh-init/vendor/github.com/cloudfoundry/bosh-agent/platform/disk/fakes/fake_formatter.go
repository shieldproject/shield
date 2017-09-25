package fakes

import (
	boshdisk "github.com/cloudfoundry/bosh-agent/platform/disk"
)

type FakeFormatter struct {
	FormatCalled         bool
	FormatPartitionPaths []string
	FormatFsTypes        []boshdisk.FileSystemType
	FormatError          error
}

func (p *FakeFormatter) Format(partitionPath string, fsType boshdisk.FileSystemType) (err error) {
	if p.FormatError != nil {
		return p.FormatError
	}
	p.FormatCalled = true
	p.FormatPartitionPaths = append(p.FormatPartitionPaths, partitionPath)
	p.FormatFsTypes = append(p.FormatFsTypes, fsType)
	return
}
