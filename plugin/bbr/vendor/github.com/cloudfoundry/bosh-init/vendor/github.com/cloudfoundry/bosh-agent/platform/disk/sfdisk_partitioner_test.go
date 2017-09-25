package disk_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	fakeboshaction "github.com/cloudfoundry/bosh-agent/agent/action/fakes"
	. "github.com/cloudfoundry/bosh-agent/platform/disk"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

const devSdaSfdiskEmptyDump = `# partition table of /dev/sda
unit: sectors

/dev/sda1 : start=        0, size=    0, Id= 0
/dev/sda2 : start=        0, size=    0, Id= 0
/dev/sda3 : start=        0, size=    0, Id= 0
/dev/sda4 : start=        0, size=    0, Id= 0
`

const devSdaSfdiskNotableDumpStderr = `
sfdisk: ERROR: sector 0 does not have an msdos signature
 /dev/sda: unrecognized partition table type
No partitions found`

const devSdaSfdiskDump = `# partition table of /dev/sda
unit: sectors

/dev/sda1 : start=        1, size= xxxx, Id=82
/dev/sda2 : start=     xxxx, size= xxxx, Id=83
/dev/sda3 : start=     xxxx, size= xxxx, Id=83
/dev/sda4 : start=        0, size=    0, Id= 0
`

const devSdaSfdiskDumpOnePartition = `# partition table of /dev/sda
unit: sectors

/dev/sda1 : start=        1, size= xxxx, Id=83
/dev/sda2 : start=     xxxx, size= xxxx, Id=83
/dev/sda3 : start=        0, size=    0, Id= 0
/dev/sda4 : start=        0, size=    0, Id= 0
`

const expectedDmSetupLs = `
xxxxxx-part1	(252:1)
xxxxxx	(252:0)
`

var _ = Describe("sfdiskPartitioner", func() {
	var (
		runner      *fakesys.FakeCmdRunner
		partitioner Partitioner
		fakeclock   *fakeboshaction.FakeClock
	)

	BeforeEach(func() {
		runner = fakesys.NewFakeCmdRunner()
		logger := boshlog.NewLogger(boshlog.LevelNone)
		fakeclock = &fakeboshaction.FakeClock{}

		partitioner = NewSfdiskPartitioner(logger, runner, fakeclock)
	})

	It("sfdisk partition", func() {
		runner.AddCmdResult("sfdisk -d /dev/sda", fakesys.FakeCmdResult{Stdout: devSdaSfdiskEmptyDump})

		partitions := []Partition{
			{Type: PartitionTypeSwap, SizeInBytes: 512 * 1024 * 1024},
			{Type: PartitionTypeLinux, SizeInBytes: 1024 * 1024 * 1024},
			{Type: PartitionTypeLinux, SizeInBytes: 512 * 1024 * 1024},
		}

		partitioner.Partition("/dev/sda", partitions)

		Expect(1).To(Equal(len(runner.RunCommandsWithInput)))
		Expect(runner.RunCommandsWithInput[0]).To(Equal([]string{",512,S\n,1024,L\n,,L\n", "sfdisk", "-uM", "/dev/sda"}))
	})

	It("sfdisk partition with no partition table", func() {
		runner.AddCmdResult("sfdisk -d /dev/sda", fakesys.FakeCmdResult{Stderr: devSdaSfdiskNotableDumpStderr})

		partitions := []Partition{
			{Type: PartitionTypeSwap, SizeInBytes: 512 * 1024 * 1024},
			{Type: PartitionTypeLinux, SizeInBytes: 1024 * 1024 * 1024},
			{Type: PartitionTypeLinux, SizeInBytes: 512 * 1024 * 1024},
		}

		partitioner.Partition("/dev/sda", partitions)

		Expect(1).To(Equal(len(runner.RunCommandsWithInput)))
		Expect(runner.RunCommandsWithInput[0]).To(Equal([]string{",512,S\n,1024,L\n,,L\n", "sfdisk", "-uM", "/dev/sda"}))
	})

	It("sfdisk partition for multipath", func() {
		partitions := []Partition{
			{Type: PartitionTypeSwap, SizeInBytes: 512 * 1024 * 1024},
			{Type: PartitionTypeLinux, SizeInBytes: 1024 * 1024 * 1024},
			{Type: PartitionTypeLinux, SizeInBytes: 512 * 1024 * 1024},
		}

		partitioner.Partition("/dev/mapper/xxxxxx", partitions)

		Expect(1).To(Equal(len(runner.RunCommandsWithInput)))
		Expect(runner.RunCommandsWithInput[0]).To(Equal([]string{",512,S\n,1024,L\n,,L\n", "sfdisk", "-uM", "/dev/mapper/xxxxxx"}))
		Expect(22).To(Equal(len(runner.RunCommands)))
		Expect(runner.RunCommands[1]).To(Equal([]string{"/etc/init.d/open-iscsi", "restart"}))
	})

	It("sfdisk get device size in mb", func() {
		runner.AddCmdResult("sfdisk -s /dev/sda", fakesys.FakeCmdResult{Stdout: fmt.Sprintf("%d\n", 40000*1024)})

		size, err := partitioner.GetDeviceSizeInBytes("/dev/sda")
		Expect(err).ToNot(HaveOccurred())

		Expect(size).To(Equal(uint64(40000 * 1024 * 1024)))
	})

	It("sfdisk partition when partitions already match", func() {
		runner.AddCmdResult("sfdisk -d /dev/sda", fakesys.FakeCmdResult{Stdout: devSdaSfdiskDump})
		runner.AddCmdResult("sfdisk -s /dev/sda", fakesys.FakeCmdResult{Stdout: fmt.Sprintf("%d\n", 2048*1024)})
		runner.AddCmdResult("sfdisk -s /dev/sda1", fakesys.FakeCmdResult{Stdout: fmt.Sprintf("%d\n", 525*1024)})
		runner.AddCmdResult("sfdisk -s /dev/sda2", fakesys.FakeCmdResult{Stdout: fmt.Sprintf("%d\n", 1020*1024)})
		runner.AddCmdResult("sfdisk -s /dev/sda3", fakesys.FakeCmdResult{Stdout: fmt.Sprintf("%d\n", 500*1024)})

		partitions := []Partition{
			{Type: PartitionTypeSwap, SizeInBytes: 512 * 1024 * 1024},
			{Type: PartitionTypeLinux, SizeInBytes: 1024 * 1024 * 1024},
			{Type: PartitionTypeLinux, SizeInBytes: 512 * 1024 * 1024},
		}

		partitioner.Partition("/dev/sda", partitions)

		Expect(len(runner.RunCommandsWithInput)).To(Equal(0))
	})

	It("sfdisk partition with last partition not matching size", func() {
		runner.AddCmdResult("sfdisk -d /dev/sda", fakesys.FakeCmdResult{Stdout: devSdaSfdiskDumpOnePartition})
		runner.AddCmdResult("sfdisk -s /dev/sda", fakesys.FakeCmdResult{Stdout: fmt.Sprintf("%d\n", 2048*1024)})
		runner.AddCmdResult("sfdisk -s /dev/sda1", fakesys.FakeCmdResult{Stdout: fmt.Sprintf("%d\n", 1024*1024)})
		runner.AddCmdResult("sfdisk -s /dev/sda2", fakesys.FakeCmdResult{Stdout: fmt.Sprintf("%d\n", 512*1024)})

		partitions := []Partition{
			{Type: PartitionTypeLinux, SizeInBytes: 1024 * 1024 * 1024},
			{Type: PartitionTypeLinux},
		}

		partitioner.Partition("/dev/sda", partitions)

		Expect(len(runner.RunCommandsWithInput)).To(Equal(1))
		Expect(runner.RunCommandsWithInput[0]).To(Equal([]string{",1024,L\n,,L\n", "sfdisk", "-uM", "/dev/sda"}))
	})

	It("sfdisk partition with last partition filling disk", func() {
		runner.AddCmdResult("sfdisk -d /dev/sda", fakesys.FakeCmdResult{Stdout: devSdaSfdiskDumpOnePartition})
		runner.AddCmdResult("sfdisk -s /dev/sda", fakesys.FakeCmdResult{Stdout: fmt.Sprintf("%d\n", 2048*1024)})
		runner.AddCmdResult("sfdisk -s /dev/sda1", fakesys.FakeCmdResult{Stdout: fmt.Sprintf("%d\n", 1024*1024)})
		runner.AddCmdResult("sfdisk -s /dev/sda2", fakesys.FakeCmdResult{Stdout: fmt.Sprintf("%d\n", 1024*1024)})

		partitions := []Partition{
			{Type: PartitionTypeLinux, SizeInBytes: 1024 * 1024 * 1024},
			{Type: PartitionTypeLinux},
		}

		partitioner.Partition("/dev/sda", partitions)

		Expect(0).To(Equal(len(runner.RunCommandsWithInput)))
	})

	It("sfdisk command is retried 20 times", func() {
		for i := 0; i < 19; i++ {
			testError := fmt.Errorf("test error")
			runner.AddCmdResult(" sfdisk -uM /dev/sda", fakesys.FakeCmdResult{ExitStatus: 1, Error: testError})
		}
		runner.AddCmdResult(" sfdisk -uM /dev/sda", fakesys.FakeCmdResult{Stdout: devSdaSfdiskDumpOnePartition})

		partitions := []Partition{}

		err := partitioner.Partition("/dev/sda", partitions)
		Expect(err).To(BeNil())
		Expect(fakeclock.SleepCallCount()).To(Equal(19))
		Expect(len(runner.RunCommandsWithInput)).To(Equal(20))
	})

	It("dmsetup command is retried 20 times", func() {
		for i := 0; i < 19; i++ {
			testError := fmt.Errorf("test error")
			runner.AddCmdResult("dmsetup ls", fakesys.FakeCmdResult{ExitStatus: 1, Error: testError})
		}
		runner.AddCmdResult("dmsetup ls", fakesys.FakeCmdResult{Stdout: expectedDmSetupLs})

		partitions := []Partition{}

		err := partitioner.Partition("/dev/mapper/xxxxxx", partitions)
		Expect(err).To(BeNil())
		Expect(fakeclock.SleepCallCount()).To(Equal(19))
		Expect(len(runner.RunCommands)).To(Equal(23))
	})
})
