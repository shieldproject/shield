package disk

import (
	"strings"
	"time"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type linuxMounter struct {
	runner            boshsys.CmdRunner
	mountsSearcher    MountsSearcher
	maxUnmountRetries int
	unmountRetrySleep time.Duration
}

func NewLinuxMounter(
	runner boshsys.CmdRunner,
	mountsSearcher MountsSearcher,
	unmountRetrySleep time.Duration,
) Mounter {
	return linuxMounter{
		runner:            runner,
		mountsSearcher:    mountsSearcher,
		maxUnmountRetries: 600,
		unmountRetrySleep: unmountRetrySleep,
	}
}

func (m linuxMounter) Mount(partitionPath, mountPoint string, mountOptions ...string) error {
	shouldMount, err := m.shouldMount(partitionPath, mountPoint)
	if !shouldMount {
		return err
	}

	if err != nil {
		return bosherr.WrapError(err, "Checking whether partition should be mounted")
	}

	mountArgs := []string{partitionPath, mountPoint}
	mountArgs = append(mountArgs, mountOptions...)

	_, _, _, err = m.runner.RunCommand("mount", mountArgs...)
	if err != nil {
		return bosherr.WrapError(err, "Shelling out to mount")
	}

	return nil
}

func (m linuxMounter) RemountAsReadonly(mountPoint string) error {
	return m.Remount(mountPoint, mountPoint, "-o", "ro")
}

func (m linuxMounter) Remount(fromMountPoint, toMountPoint string, mountOptions ...string) error {
	partitionPath, found, err := m.IsMountPoint(fromMountPoint)
	if err != nil || !found {
		return bosherr.WrapErrorf(err, "Error finding device for mount point %s", fromMountPoint)
	}

	_, err = m.Unmount(fromMountPoint)
	if err != nil {
		return bosherr.WrapErrorf(err, "Unmounting %s", fromMountPoint)
	}

	return m.Mount(partitionPath, toMountPoint, mountOptions...)
}

func (m linuxMounter) SwapOn(partitionPath string) (err error) {
	out, _, _, _ := m.runner.RunCommand("swapon", "-s")

	for i, swapOnLines := range strings.Split(out, "\n") {
		swapOnFields := strings.Fields(swapOnLines)

		switch {
		case i == 0:
			continue
		case len(swapOnFields) == 0:
			continue
		case swapOnFields[0] == partitionPath:
			return nil
		}
	}

	_, _, _, err = m.runner.RunCommand("swapon", partitionPath)
	if err != nil {
		return bosherr.WrapError(err, "Shelling out to swapon")
	}

	return nil
}

func (m linuxMounter) Unmount(partitionOrMountPoint string) (bool, error) {
	isMounted, err := m.IsMounted(partitionOrMountPoint)
	if err != nil || !isMounted {
		return false, err
	}

	_, _, _, err = m.runner.RunCommand("umount", partitionOrMountPoint)

	for i := 1; i < m.maxUnmountRetries && err != nil; i++ {
		time.Sleep(m.unmountRetrySleep)
		_, _, _, err = m.runner.RunCommand("umount", partitionOrMountPoint)
	}

	return err == nil, err
}

func (m linuxMounter) IsMountPoint(path string) (string, bool, error) {
	mounts, err := m.mountsSearcher.SearchMounts()
	if err != nil {
		return "", false, bosherr.WrapError(err, "Searching mounts")
	}

	for _, mount := range mounts {
		if mount.MountPoint == path {
			return mount.PartitionPath, true, nil
		}
	}

	return "", false, nil
}

func (m linuxMounter) IsMounted(partitionOrMountPoint string) (bool, error) {
	mounts, err := m.mountsSearcher.SearchMounts()
	if err != nil {
		return false, bosherr.WrapError(err, "Searching mounts")
	}

	for _, mount := range mounts {
		if mount.PartitionPath == partitionOrMountPoint || mount.MountPoint == partitionOrMountPoint {
			return true, nil
		}
	}

	return false, nil
}

func (m linuxMounter) shouldMount(partitionPath, mountPoint string) (bool, error) {
	mounts, err := m.mountsSearcher.SearchMounts()
	if err != nil {
		return false, bosherr.WrapError(err, "Searching mounts")
	}

	for _, mount := range mounts {
		switch {
		case mount.PartitionPath == partitionPath && mount.MountPoint == mountPoint:
			return false, nil
		case mount.PartitionPath == partitionPath && mount.MountPoint != mountPoint && partitionPath != "tmpfs":
			return false, bosherr.Errorf("Device %s is already mounted to %s, can't mount to %s",
				mount.PartitionPath, mount.MountPoint, mountPoint)
		case mount.MountPoint == mountPoint:
			return false, bosherr.Errorf("Device %s is already mounted to %s, can't mount %s",
				mount.PartitionPath, mount.MountPoint, partitionPath)
		}
	}

	return true, nil
}
