package disk

import (
	"time"

	boshdevutil "github.com/cloudfoundry/bosh-agent/platform/deviceutil"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	"github.com/pivotal-golang/clock"
)

type linuxDiskManager struct {
	partitioner           Partitioner
	rootDevicePartitioner Partitioner
	partedPartitioner     Partitioner
	formatter             Formatter
	mounter               Mounter
	mountsSearcher        MountsSearcher
	fs                    boshsys.FileSystem
	logger                boshlog.Logger
	runner                boshsys.CmdRunner
}

func NewLinuxDiskManager(
	logger boshlog.Logger,
	runner boshsys.CmdRunner,
	fs boshsys.FileSystem,
	bindMount bool,
) (manager Manager) {
	var mounter Mounter
	var mountsSearcher MountsSearcher

	// By default we want to use most reliable source of
	// mount information which is /proc/mounts
	mountsSearcher = NewProcMountsSearcher(fs)

	// Bind mounting in a container (warden) will not allow
	// reliably determine which device backs a mount point,
	// so we use less reliable source of mount information:
	// the mount command which returns information from /etc/mtab.
	if bindMount {
		mountsSearcher = NewCmdMountsSearcher(runner)
	}

	mounter = NewLinuxMounter(runner, mountsSearcher, 1*time.Second)

	if bindMount {
		mounter = NewLinuxBindMounter(mounter)
	}

	return linuxDiskManager{
		partitioner:           NewSfdiskPartitioner(logger, runner, clock.NewClock()),
		rootDevicePartitioner: NewRootDevicePartitioner(logger, runner, uint64(20*1024*1024)),
		partedPartitioner:     NewPartedPartitioner(logger, runner, clock.NewClock()),
		formatter:             NewLinuxFormatter(runner, fs),
		mounter:               mounter,
		mountsSearcher:        mountsSearcher,
		fs:                    fs,
		logger:                logger,
		runner:                runner,
	}
}

func (m linuxDiskManager) GetPartitioner() Partitioner { return m.partitioner }

func (m linuxDiskManager) GetPartedPartitioner() Partitioner { return m.partedPartitioner }

func (m linuxDiskManager) GetRootDevicePartitioner() Partitioner {
	return m.rootDevicePartitioner
}

func (m linuxDiskManager) GetFormatter() Formatter           { return m.formatter }
func (m linuxDiskManager) GetMounter() Mounter               { return m.mounter }
func (m linuxDiskManager) GetMountsSearcher() MountsSearcher { return m.mountsSearcher }

func (m linuxDiskManager) GetDiskUtil(diskPath string) boshdevutil.DeviceUtil {
	return NewDiskUtil(diskPath, m.runner, m.mounter, m.fs, m.logger)
}
