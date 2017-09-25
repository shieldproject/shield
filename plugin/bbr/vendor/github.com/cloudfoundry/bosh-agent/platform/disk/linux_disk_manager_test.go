package disk_test

import (
	"time"

	"code.cloudfoundry.org/clock"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/platform/disk"
)

var _ = Describe("NewLinuxDiskManager", func() {
	var (
		runner *fakesys.FakeCmdRunner
		fs     *fakesys.FakeFileSystem
		logger boshlog.Logger
	)

	BeforeEach(func() {
		runner = fakesys.NewFakeCmdRunner()
		fs = fakesys.NewFakeFileSystem()
		logger = boshlog.NewLogger(boshlog.LevelNone)
	})

	Context("when bindMount is set to false", func() {
		It("returns disk manager configured not to do bind mounting", func() {
			expectedMountsSearcher := NewProcMountsSearcher(fs)
			expectedMounter := NewLinuxMounter(runner, expectedMountsSearcher, 1*time.Second)

			diskManager := NewLinuxDiskManager(logger, runner, fs, LinuxDiskManagerOpts{})
			Expect(diskManager.GetMounter()).To(Equal(expectedMounter))
		})
	})

	Context("when bindMount is set to true", func() {
		It("returns disk manager configured to do bind mounting", func() {
			expectedMountsSearcher := NewCmdMountsSearcher(runner)
			expectedMounter := NewLinuxBindMounter(NewLinuxMounter(runner, expectedMountsSearcher, 1*time.Second))

			opts := LinuxDiskManagerOpts{BindMount: true}
			diskManager := NewLinuxDiskManager(logger, runner, fs, opts)
			Expect(diskManager.GetMounter()).To(Equal(expectedMounter))
		})
	})

	Context("when partitioner type is not set", func() {
		It("returns disk manager configured to use sfdisk", func() {
			opts := LinuxDiskManagerOpts{}
			diskManager := NewLinuxDiskManager(logger, runner, fs, opts)
			Expect(diskManager.GetPartitioner()).To(Equal(NewSfdiskPartitioner(logger, runner, clock.NewClock())))
		})
	})

	Context("when partitioner type is 'parted'", func() {
		It("returns disk manager configured to use parted", func() {
			opts := LinuxDiskManagerOpts{PartitionerType: "parted"}
			diskManager := NewLinuxDiskManager(logger, runner, fs, opts)
			Expect(diskManager.GetPartitioner()).To(Equal(NewPartedPartitioner(logger, runner, clock.NewClock())))
		})
	})

	Context("when partitioner type is unknown", func() {
		It("panics", func() {
			opts := LinuxDiskManagerOpts{PartitionerType: "unknown"}
			Expect(func() { NewLinuxDiskManager(logger, runner, fs, opts) }).To(Panic())
		})
	})
})
