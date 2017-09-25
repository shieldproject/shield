package disk

import (
	"strings"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type procMountsSearcher struct {
	fs boshsys.FileSystem
}

func NewProcMountsSearcher(fs boshsys.FileSystem) MountsSearcher {
	return procMountsSearcher{fs}
}

func (s procMountsSearcher) SearchMounts() ([]Mount, error) {
	var mounts []Mount

	mountInfo, err := s.fs.ReadFileString("/proc/mounts")
	if err != nil {
		return mounts, bosherr.WrapError(err, "Reading /proc/mounts")
	}

	for _, mountEntry := range strings.Split(mountInfo, "\n") {
		if mountEntry == "" {
			continue
		}

		mountFields := strings.Fields(mountEntry)

		mounts = append(mounts, Mount{
			PartitionPath: mountFields[0],
			MountPoint:    mountFields[1],
		})
	}

	return mounts, nil
}
