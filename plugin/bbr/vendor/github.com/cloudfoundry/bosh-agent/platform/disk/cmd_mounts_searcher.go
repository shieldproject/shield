package disk

import (
	"strings"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type cmdMountsSearcher struct {
	runner boshsys.CmdRunner
}

func NewCmdMountsSearcher(runner boshsys.CmdRunner) MountsSearcher {
	return cmdMountsSearcher{runner}
}

func (s cmdMountsSearcher) SearchMounts() ([]Mount, error) {
	var mounts []Mount

	stdout, _, _, err := s.runner.RunCommand("mount")
	if err != nil {
		return mounts, bosherr.WrapError(err, "Running mount")
	}

	// e.g. '/dev/sda on /boot type ext2 (rw)'
	for _, mountEntry := range strings.Split(stdout, "\n") {
		if mountEntry == "" {
			continue
		}

		mountFields := strings.Fields(mountEntry)

		mounts = append(mounts, Mount{
			PartitionPath: mountFields[0],
			MountPoint:    mountFields[2],
		})
	}

	return mounts, nil
}
