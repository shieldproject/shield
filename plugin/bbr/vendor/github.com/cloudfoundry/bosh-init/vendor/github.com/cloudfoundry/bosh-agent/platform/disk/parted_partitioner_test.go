package disk_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	fakeboshaction "github.com/cloudfoundry/bosh-agent/agent/action/fakes"
	. "github.com/cloudfoundry/bosh-agent/platform/disk"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("PartedPartitioner", func() {
	var (
		fakeCmdRunner *fakesys.FakeCmdRunner
		partitioner   Partitioner
		fakeclock     *fakeboshaction.FakeClock
		logger        boshlog.Logger
	)

	BeforeEach(func() {
		logger = boshlog.NewLogger(boshlog.LevelNone)
		fakeCmdRunner = fakesys.NewFakeCmdRunner()
		fakeclock = &fakeboshaction.FakeClock{}
		partitioner = NewPartedPartitioner(logger, fakeCmdRunner, fakeclock)
	})

	Describe("Partition", func() {
		Context("when the desired partitions do not exist", func() {
			Context("when there is no partition table", func() {
				BeforeEach(func() {
					fakeCmdRunner.AddCmdResult(
						"parted -m /dev/sda unit B print",
						fakesys.FakeCmdResult{
							Stdout: "Error: /dev/sda: unrecognised disk label", ExitStatus: 1, Error: errors.New("Error: /dev/sda: unrecognised disk label")},
					)
					fakeCmdRunner.AddCmdResult(
						"parted -m /dev/sda unit B print",
						fakesys.FakeCmdResult{
							Stdout: `BYT;
/dev/xvdf:221190815744B:xvd:512:512:gpt:Xen Virtual Block Device;
`})
					fakeCmdRunner.AddCmdResult(
						"parted -s /dev/sda mklabel gpt",
						fakesys.FakeCmdResult{Stdout: "", ExitStatus: 0})
				})

				It("makes a gpt label and then creates partitions using parted", func() {
					partitions := []Partition{
						{SizeInBytes: 8589934592}, // (8GiB)
						{SizeInBytes: 8589934592}, // (8GiB)
					}

					// Calculating "aligned" partition start/end/size
					// (512 + 1) % 1048576 = 513
					// (512 + 1) + 1048576 - 513 = 1048576 (aligned start)
					// 1048576 + 8589934592 = 8590983168
					// 8590983168 % 1048576 = 0
					// 8590983168 - 0 - 1 = 8590983167 (desired end)
					// first start=1048576, end=8590983167, size=8589934592

					// (8590983167 + 1) % 1048576 = 0
					// (8590983167 + 1) = 8590983168 (aligned start)
					// 8590983168 + 8589934592 = 17180917760 (desired end)
					// 17180917760 % 1048576 = 0
					// 17180917760 - 0 - 1 = 17180917759
					// second start=11661213696, end=17180917759, size=8589934592

					err := partitioner.Partition("/dev/sda", partitions)
					Expect(err).ToNot(HaveOccurred())

					Expect(len(fakeCmdRunner.RunCommands)).To(Equal(5))
					Expect(fakeCmdRunner.RunCommands).To(ContainElement([]string{"parted", "-m", "/dev/sda", "unit", "B", "print"}))
					Expect(fakeCmdRunner.RunCommands).To(ContainElement([]string{"parted", "-s", "/dev/sda", "mklabel", "gpt"}))
					Expect(fakeCmdRunner.RunCommands).To(ContainElement([]string{"parted", "-s", "/dev/sda", "unit", "B", "mkpart", "primary", "1048576", "8590983167"}))
					Expect(fakeCmdRunner.RunCommands).To(ContainElement([]string{"parted", "-s", "/dev/sda", "unit", "B", "mkpart", "primary", "8590983168", "17180917759"}))
				})
			})

			Context("when there are no partitions", func() {
				BeforeEach(func() {
					fakeCmdRunner.AddCmdResult(
						"parted -m /dev/sda unit B print",
						fakesys.FakeCmdResult{
							Stdout: `BYT;
/dev/xvdf:221190815744B:xvd:512:512:gpt:Xen Virtual Block Device;
`},
					)
				})

				It("creates partitions using parted starting at the 1048576 byte", func() {
					partitions := []Partition{
						{SizeInBytes: 8589934592}, // (8GiB)
						{SizeInBytes: 8589934592}, // (8GiB)
					}

					// Calculating "aligned" partition start/end/size
					// (512 + 1) % 1048576 = 513
					// (512 + 1) + 1048576 - 513 = 1048576 (aligned start)
					// 1048576 + 8589934592 = 8590983168
					// 8590983168 % 1048576 = 0
					// 8590983168 - 0 - 1 = 8590983167 (desired end)
					// first start=1048576, end=8590983167, size=8589934592

					// (8590983167 + 1) % 1048576 = 0
					// (8590983167 + 1) = 8590983168 (aligned start)
					// 8590983168 + 8589934592 = 17180917760 (desired end)
					// 17180917760 % 1048576 = 0
					// 17180917760 - 0 - 1 = 17180917759
					// second start=11661213696, end=17180917759, size=8589934592

					err := partitioner.Partition("/dev/sda", partitions)
					Expect(err).ToNot(HaveOccurred())

					Expect(len(fakeCmdRunner.RunCommands)).To(Equal(3))
					Expect(fakeCmdRunner.RunCommands).To(ContainElement([]string{"parted", "-m", "/dev/sda", "unit", "B", "print"}))
					Expect(fakeCmdRunner.RunCommands).To(ContainElement([]string{"parted", "-s", "/dev/sda", "unit", "B", "mkpart", "primary", "1048576", "8590983167"}))
					Expect(fakeCmdRunner.RunCommands).To(ContainElement([]string{"parted", "-s", "/dev/sda", "unit", "B", "mkpart", "primary", "8590983168", "17180917759"}))
				})
			})

			Context("when there are existing partitions", func() {
				BeforeEach(func() {
					fakeCmdRunner.AddCmdResult(
						"parted -m /dev/sda unit B print",
						fakesys.FakeCmdResult{
							Stdout: `BYT;
/dev/xvdf:221190815744B:xvd:512:512:gpt:Xen Virtual Block Device;
1:512B:2048576B:199680B:ext4::;
`},
					)
				})

				It("creates partitions using parted respecting the existing partitions", func() {
					partitions := []Partition{
						{SizeInBytes: 8589934592}, // (8GiB)
						{SizeInBytes: 8589934592}, // (8GiB)
					}

					// Calculating "aligned" partition start/end/size
					// (2048576 + 1) % 1048576 = 1000001
					// (2048576 + 1) + 1048576 - 1000001 = 2097152 (aligned start)
					// 2097152 + 8589934592 = 8592031744
					// 8592031744 % 1048576 = 0
					// 8592031744 - 0 - 1 = 8592031743 (desired end)
					// first start=2097152, end=8592031743, size=8589934592

					// (8592031743 + 1) % 1048576 = 0
					// (8592031743 + 1) = 8592031744 (aligned start)
					// 8592031744 + 8589934592 = 17181966336 (desired end)
					// 17181966336 % 1048576 = 0
					// 17181966336 - 0 - 1 = 17181966335
					// second start=8592031744, end=17181966335, size=8589934592

					err := partitioner.Partition("/dev/sda", partitions)
					Expect(err).ToNot(HaveOccurred())

					Expect(len(fakeCmdRunner.RunCommands)).To(Equal(3))
					Expect(fakeCmdRunner.RunCommands).To(ContainElement([]string{"parted", "-m", "/dev/sda", "unit", "B", "print"}))
					Expect(fakeCmdRunner.RunCommands).To(ContainElement([]string{"parted", "-s", "/dev/sda", "unit", "B", "mkpart", "primary", "2097152", "8592031743"}))
					Expect(fakeCmdRunner.RunCommands).To(ContainElement([]string{"parted", "-s", "/dev/sda", "unit", "B", "mkpart", "primary", "8592031744", "17181966335"}))
				})
			})

			Context("when the required partition over-flows the device", func() {
				BeforeEach(func() {
					fakeCmdRunner.AddCmdResult(
						"parted -m /dev/sda unit B print",
						fakesys.FakeCmdResult{
							Stdout: `BYT;
/dev/xvdf:221190815744B:xvd:512:512:gpt:Xen Virtual Block Device;
1:512B:2048576B:199680B:ext4::;
`},
					)
				})

				It("creates partitions using parted but truncates the partition", func() {
					partitions := []Partition{
						{SizeInBytes: 8589934592},   // (8GiB)
						{SizeInBytes: 221190815744}, // (197GiB)
					}

					// Calculating "aligned" partition start/end/size
					// (2048576 + 1) % 1048576 = 1000001
					// (2048576 + 1) + 1048576 - 1000001 = 2097152 (aligned start)
					// 2097152 + 8589934592 = 8592031744
					// 8592031744 % 1048576 = 0
					// 8592031744 - 0 - 1 = 8592031743 (desired end)
					// first start=2097152, end=8592031743, size=8589934592

					// (8592031743 + 1) % 1048576 = 0
					// (8592031743 + 1) = 8592031744 (aligned start)
					// 8592031744 + 8589934592 = 221190815743 (desired end)
					// 221190815743 % 1048576 = 1048575
					// 221190815743 - 1048575 - 1 = 221189767167
					// second start=8592031744, end=221189767167, size=212597735423

					err := partitioner.Partition("/dev/sda", partitions)
					Expect(err).ToNot(HaveOccurred())

					Expect(len(fakeCmdRunner.RunCommands)).To(Equal(3))
					Expect(fakeCmdRunner.RunCommands).To(ContainElement([]string{"parted", "-m", "/dev/sda", "unit", "B", "print"}))
					Expect(fakeCmdRunner.RunCommands).To(ContainElement([]string{"parted", "-s", "/dev/sda", "unit", "B", "mkpart", "primary", "2097152", "8592031743"}))
					Expect(fakeCmdRunner.RunCommands).To(ContainElement([]string{"parted", "-s", "/dev/sda", "unit", "B", "mkpart", "primary", "8592031744", "221189767167"}))
				})
			})
		})

		Context("when the existing partitions match desired partitions", func() {
			Context("when the partitions match exactly", func() {
				BeforeEach(func() {
					fakeCmdRunner.AddCmdResult(
						"parted -m /dev/sda unit B print",
						fakesys.FakeCmdResult{
							Stdout: `BYT;
/dev/xvdf:221190815744B:xvd:512:512:gpt:Xen Virtual Block Device;
1:512B:8589935104B:8589934592B:ext4::;
2:8589935105B:17179869697B:8589934592B:ext4::;
`},
					)
				})

				It("checks the existing partitions and does nothing", func() {
					partitions := []Partition{
						{SizeInBytes: 8589934592}, // (8GiB)
						{SizeInBytes: 8589934592}, // (8GiB)
					}
					err := partitioner.Partition("/dev/sda", partitions)
					Expect(err).ToNot(HaveOccurred())

					Expect(len(fakeCmdRunner.RunCommands)).To(Equal(1))
					Expect(fakeCmdRunner.RunCommands).To(ContainElement([]string{"parted", "-m", "/dev/sda", "unit", "B", "print"}))
				})
			})

			Context("when the partitions are within delta", func() {
				BeforeEach(func() {
					fakeCmdRunner.AddCmdResult(
						"parted -m /dev/sda unit B print",
						fakesys.FakeCmdResult{
							Stdout: `BYT;
/dev/xvdf:221190815744B:xvd:512:512:gpt:Xen Virtual Block Device;
1:512B:8589935104B:8568963072B:ext4::;
2:8589935105B:17179869697B:8568963072B:ext4::;
`},
					)
				})

				It("checks the existing partitions and does nothing", func() {
					partitions := []Partition{
						{SizeInBytes: 8589934592}, // (8GiB)
						{SizeInBytes: 8589934592}, // (8GiB)
					}
					err := partitioner.Partition("/dev/sda", partitions)
					Expect(err).ToNot(HaveOccurred())

					Expect(len(fakeCmdRunner.RunCommands)).To(Equal(1))
					Expect(fakeCmdRunner.RunCommands).To(ContainElement([]string{"parted", "-m", "/dev/sda", "unit", "B", "print"}))
				})
			})
		})

		Context("when getting existing partitions returns an error", func() {
			Context("when the first call to parted print fails", func() {
				BeforeEach(func() {
					fakeCmdRunner.AddCmdResult(
						"parted -m /dev/sda unit B print",
						fakesys.FakeCmdResult{
							Stdout: "Some weird error", ExitStatus: 1, Error: errors.New("Some weird error")},
					)
				})

				It("throw an error", func() {
					partitions := []Partition{
						{SizeInBytes: 8589934592}, // (8GiB)
						{SizeInBytes: 8589934592}, // (8GiB)
					}

					err := partitioner.Partition("/dev/sda", partitions)
					Expect(err).To(HaveOccurred())

					Expect(len(fakeCmdRunner.RunCommands)).To(Equal(1))
					Expect(err.Error()).To(ContainSubstring("Getting existing partitions of `/dev/sda': Running parted print: Some weird error"))
				})
			})

			Context("when parted fails to make device label", func() {
				BeforeEach(func() {
					fakeCmdRunner.AddCmdResult(
						"parted -m /dev/sda unit B print",
						fakesys.FakeCmdResult{
							Stdout: "Error: /dev/sda: unrecognised disk label", ExitStatus: 1, Error: errors.New("Error: /dev/sda: unrecognised disk label")},
					)
					fakeCmdRunner.AddCmdResult(
						"parted -s /dev/sda mklabel gpt",
						fakesys.FakeCmdResult{Stdout: "Some weird error", ExitStatus: 1, Error: errors.New("Some weird error")})
				})

				It("throw an error", func() {
					partitions := []Partition{
						{SizeInBytes: 8589934592}, // (8GiB)
						{SizeInBytes: 8589934592}, // (8GiB)
					}

					err := partitioner.Partition("/dev/sda", partitions)
					Expect(err).To(HaveOccurred())

					Expect(len(fakeCmdRunner.RunCommands)).To(Equal(2))
					Expect(err.Error()).To(ContainSubstring("Getting existing partitions of `/dev/sda': Running parted print: Parted making label: Some weird error"))
				})
			})

			Context("when parted makes a label but fails print the second time", func() {
				BeforeEach(func() {
					fakeCmdRunner.AddCmdResult(
						"parted -m /dev/sda unit B print",
						fakesys.FakeCmdResult{
							Stdout: "Error: /dev/sda: unrecognised disk label", ExitStatus: 1, Error: errors.New("Error: /dev/sda: unrecognised disk label")},
					)
					fakeCmdRunner.AddCmdResult(
						"parted -m /dev/sda unit B print",
						fakesys.FakeCmdResult{
							Stdout: `Some weird error`, Error: errors.New("Some weird error")})
					fakeCmdRunner.AddCmdResult(
						"parted -s /dev/sda mklabel gpt",
						fakesys.FakeCmdResult{Stdout: "", ExitStatus: 0})
				})

				It("throw an error", func() {
					partitions := []Partition{
						{SizeInBytes: 8589934592}, // (8GiB)
						{SizeInBytes: 8589934592}, // (8GiB)
					}

					err := partitioner.Partition("/dev/sda", partitions)
					Expect(err).To(HaveOccurred())

					Expect(len(fakeCmdRunner.RunCommands)).To(Equal(3))
					Expect(err.Error()).To(ContainSubstring("Getting existing partitions of `/dev/sda': Running parted print: Some weird error"))
				})
			})
		})
	})
})
