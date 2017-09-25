package disk

import (
	boshdevutil "github.com/cloudfoundry/bosh-agent/platform/deviceutil"
)

type Manager interface {
	GetPartitioner() Partitioner
	GetRootDevicePartitioner() Partitioner
	GetPartedPartitioner() Partitioner
	GetFormatter() Formatter
	GetMounter() Mounter
	GetMountsSearcher() MountsSearcher
	GetDiskUtil(diskPath string) boshdevutil.DeviceUtil
}
