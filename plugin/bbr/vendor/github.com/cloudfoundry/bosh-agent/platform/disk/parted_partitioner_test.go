package disk_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"fmt"
	fakeboshaction "github.com/cloudfoundry/bosh-agent/agent/action/fakes"
	. "github.com/cloudfoundry/bosh-agent/platform/disk"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	"regexp"
)

const partitionNamePrefix = "bosh-partition"

func scrubPartitionNames(commands [][]string) [][]string {
	scrubbedCommands := make([][]string, len(commands))
	partitionNameSyntax := regexp.MustCompile("^" + partitionNamePrefix + "-[0-9]+$")

	// deep copy
	for i := range commands {
		scrubbedCommands[i] = make([]string, len(commands[i]))
		copy(scrubbedCommands[i], commands[i])
	}

	for _, command := range scrubbedCommands {
		for index, part := range command {
			if partitionNameSyntax.MatchString(part) {
				command[index] = partitionNamePrefix + "-x"
			}
		}
	}

	return scrubbedCommands
}

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
							Stdout:     "Error: /dev/sda: unrecognised disk label",
							ExitStatus: 1,
							Error:      errors.New("Error: /dev/sda: unrecognised disk label"),
						},
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

					Expect(len(fakeCmdRunner.RunCommands)).To(Equal(9))

					scrubbedCommands := scrubPartitionNames(fakeCmdRunner.RunCommands)
					Expect(scrubbedCommands).To(ContainElement([]string{"parted", "-m", "/dev/sda", "unit", "B", "print"}))
					Expect(scrubbedCommands).To(ContainElement([]string{"parted", "-s", "/dev/sda", "mklabel", "gpt"}))
					Expect(scrubbedCommands).To(ContainElement([]string{"parted", "-s", "/dev/sda", "unit", "B", "mkpart", "bosh-partition-x", "1048576", "8590983167"}))
					Expect(scrubbedCommands).To(ContainElement([]string{"parted", "-s", "/dev/sda", "unit", "B", "mkpart", "bosh-partition-x", "8590983168", "17180917759"}))
					Expect(scrubbedCommands).To(ContainElement([]string{"partprobe", "/dev/sda"}))
					Expect(scrubbedCommands).To(ContainElement([]string{"udevadm", "settle"}))
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

					Expect(len(fakeCmdRunner.RunCommands)).To(Equal(7))

					scrubbedCommands := scrubPartitionNames(fakeCmdRunner.RunCommands)
					Expect(scrubbedCommands).To(ContainElement([]string{"parted", "-m", "/dev/sda", "unit", "B", "print"}))
					Expect(scrubbedCommands).To(ContainElement([]string{"parted", "-s", "/dev/sda", "unit", "B", "mkpart", "bosh-partition-x", "1048576", "8590983167"}))
					Expect(scrubbedCommands).To(ContainElement([]string{"parted", "-s", "/dev/sda", "unit", "B", "mkpart", "bosh-partition-x", "8590983168", "17180917759"}))
					Expect(scrubbedCommands).To(ContainElement([]string{"partprobe", "/dev/sda"}))
					Expect(scrubbedCommands).To(ContainElement([]string{"udevadm", "settle"}))
				})
			})

			Context("when there are existing partitions", func() {

				Context("and none of the partitions were created by BOSH", func() {
					BeforeEach(func() {
						fakeCmdRunner.AddCmdResult(
							"parted -m /dev/sda unit B print",
							fakesys.FakeCmdResult{
								Stdout: `BYT;
/dev/xvdf:221190815744B:xvd:512:512:gpt:Xen Virtual Block Device;
1:512B:2048576B:199680B:ext4:primary:;
`},
						)
					})

					It("creates partitions using parted overwriting the existing partitions", func() {
						partitions := []Partition{
							{SizeInBytes: 8589934592}, // (8GiB)
							{SizeInBytes: 8589934592}, // (8GiB)
						}

						// Calculating "aligned" partition start/end/size
						// (513) % 1048576 = 513
						// (513) + 1048576 - 513 = 1048576 (aligned start)
						// 1048576 + 8589934592 = 8590983168
						// 8590983168 % 1048576 = 0
						// 8590983168 - 0 - 1 = 8590983167 (desired end)
						// first start=1048576, end=8590983167, size=8589934592

						// (8590983167 + 1) % 1048576 = 0
						// (8590983167 + 1) = 8590983168 (aligned start)
						// 8590983168 + 8589934592 = 17180917760 (desired end)
						// 17180917760 % 1048576 = 0
						// 17180917760 - 0 - 1 = 17180917759
						// second start=8590983168, end=17180917759, size=8589934592

						err := partitioner.Partition("/dev/sda", partitions)
						Expect(err).ToNot(HaveOccurred())

						Expect(len(fakeCmdRunner.RunCommands)).To(Equal(7))

						scrubbedCommands := scrubPartitionNames(fakeCmdRunner.RunCommands)
						Expect(scrubbedCommands).To(ContainElement([]string{"parted", "-m", "/dev/sda", "unit", "B", "print"}))
						Expect(scrubbedCommands).To(ContainElement([]string{"parted", "-s", "/dev/sda", "unit", "B", "mkpart", "bosh-partition-x", "1048576", "8590983167"}))
						Expect(scrubbedCommands).To(ContainElement([]string{"parted", "-s", "/dev/sda", "unit", "B", "mkpart", "bosh-partition-x", "8590983168", "17180917759"}))
						Expect(scrubbedCommands).To(ContainElement([]string{"partprobe", "/dev/sda"}))
						Expect(scrubbedCommands).To(ContainElement([]string{"udevadm", "settle"}))
					})
				})

				Context("and a partition was created by BOSH", func() {
					BeforeEach(func() {
						fakeCmdRunner.AddCmdResult(
							"parted -m /dev/sda unit B print",
							fakesys.FakeCmdResult{
								Stdout: `BYT;
/dev/xvdf:221190815744B:xvd:512:512:gpt:Xen Virtual Block Device;
1:512B:2048576B:199680B:ext4:bosh-partition-0:;
`},
						)
					})

					It("does NOT partition the disk, and returns an error", func() {
						partitions := []Partition{
							{SizeInBytes: 8589934592}, // (8GiB)
							{SizeInBytes: 8589934592}, // (8GiB)
						}

						// Calculating "aligned" partition start/end/size
						// (513) % 1048576 = 513
						// (513) + 1048576 - 513 = 1048576 (aligned start)
						// 1048576 + 8589934592 = 8590983168
						// 8590983168 % 1048576 = 0
						// 8590983168 - 0 - 1 = 8590983167 (desired end)
						// first start=1048576, end=8590983167, size=8589934592

						// (8590983167 + 1) % 1048576 = 0
						// (8590983167 + 1) = 8590983168 (aligned start)
						// 8590983168 + 8589934592 = 17180917760 (desired end)
						// 17180917760 % 1048576 = 0
						// 17180917760 - 0 - 1 = 17180917759
						// second start=8590983168, end=17180917759, size=8589934592

						err := partitioner.Partition("/dev/sda", partitions)
						Expect(err.Error()).To(Equal("'/dev/sda' contains a partition created by bosh. No partitioning is allowed."))

						Expect(len(fakeCmdRunner.RunCommands)).To(Equal(1))
						scrubbedCommands := scrubPartitionNames(fakeCmdRunner.RunCommands)
						Expect(scrubbedCommands).To(ContainElement([]string{"parted", "-m", "/dev/sda", "unit", "B", "print"}))
					})
				})

			})

			Context("when the type does not match", func() {
				BeforeEach(func() {
					fakeCmdRunner.AddCmdResult(
						"parted -m /dev/sdf unit B print",
						fakesys.FakeCmdResult{
							Stdout: `BYT;
/dev/xvdf:3146063544320B:xvd:512:512:gpt:Xen Virtual Block Device;
1:1048576B:3146062496255B:3146062495744B:Golden Bow:primary:;
`},
					)
				})

				It("replaces the partition", func() {
					partitions := []Partition{
						{Type: PartitionTypeLinux},
					}

					// Calculating "aligned" partition start/end/size
					// (513) % 1048576 = 513
					// (513) + 1048576 - 513 = 1048576 (aligned start)
					// 1048576 + 3146063544320 = 3146064592896
					// min(3146064592896, 3146063544320 - 1) = 3146063544319
					// 3146063544319 % 1048576 = 1048575
					// 3146063544319 - 1048575 - 1 = 3146062495743 (desired end)
					// first start=1048576, end=3146062495743, size=3146062495743

					err := partitioner.Partition("/dev/sdf", partitions)
					Expect(err).ToNot(HaveOccurred())

					Expect(len(fakeCmdRunner.RunCommands)).To(Equal(4))

					scrubbedCommands := scrubPartitionNames(fakeCmdRunner.RunCommands)
					Expect(scrubbedCommands).To(ContainElement([]string{"parted", "-m", "/dev/sdf", "unit", "B", "print"}))
					Expect(scrubbedCommands).To(ContainElement([]string{"parted", "-s", "/dev/sdf", "unit", "B", "mkpart", "bosh-partition-x", "1048576", "3146062495743"}))
					Expect(scrubbedCommands).To(ContainElement([]string{"partprobe", "/dev/sdf"}))
					Expect(scrubbedCommands).To(ContainElement([]string{"udevadm", "settle"}))
				})
			})

			Context("when the partition is not yet formatted", func() {
				BeforeEach(func() {
					fakeCmdRunner.AddCmdResult(
						"parted -m /dev/sdf unit B print",
						fakesys.FakeCmdResult{
							Stdout: `BYT;
/dev/xvdf:3146063544320B:xvd:512:512:gpt:Xen Virtual Block Device;
1:1048576B:3146062496255B:3146062495744B::primary:;
`},
					)
				})

				It("repartitions", func() {
					partitions := []Partition{
						{Type: PartitionTypeLinux},
					}

					// Calculating "aligned" partition start/end/size
					// (513) % 1048576 = 513
					// (513) + 1048576 - 513 = 1048576 (aligned start)
					// 1048576 + 3146063544320 = 3146064592896
					// min(3146064592896, 3146063544320 - 1) = 3146063544319
					// 3146063544319 % 1048576 = 1048575
					// 3146063544319 - 1048575 - 1 = 3146062495743 (desired end)
					// first start=1048576, end=3146062495743, size=3146062495743

					err := partitioner.Partition("/dev/sdf", partitions)
					Expect(err).ToNot(HaveOccurred())
					Expect(len(fakeCmdRunner.RunCommands)).To(Equal(4))

					scrubbedCommands := scrubPartitionNames(fakeCmdRunner.RunCommands)
					Expect(scrubbedCommands).To(ContainElement([]string{"parted", "-m", "/dev/sdf", "unit", "B", "print"}))
					Expect(scrubbedCommands).To(ContainElement([]string{"parted", "-s", "/dev/sdf", "unit", "B", "mkpart", "bosh-partition-x", "1048576", "3146062495743"}))
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
					// (513) % 1048576 = 513
					// (513) + 1048576 - 513 = 1048576 (aligned start)
					// 1048576 + 8589934592 = 8590983168
					// 8590983168 % 1048576 = 0
					// 8590983168 - 0 - 1 = 8590983167 (desired end)
					// first start=1048576, end=8590983167, size=8589934592

					// (8590983167 + 1) % 1048576 = 0
					// (8590983167 + 1) = 8590983168 (aligned start)
					// 8590983168 + 221190815744 = 229781798912 (desired end)
					// min(229781798912, 221190815744 - 1) = 221190815743
					// 221190815743 % 1048576 = 1048575
					// 221190815743 - 1048575 - 1 = 221189767167
					// second start=8590983168, end=221189767167, size=212599832575

					err := partitioner.Partition("/dev/sda", partitions)
					Expect(err).ToNot(HaveOccurred())

					Expect(len(fakeCmdRunner.RunCommands)).To(Equal(7))

					scrubbedCommands := scrubPartitionNames(fakeCmdRunner.RunCommands)
					Expect(scrubbedCommands).To(ContainElement([]string{"parted", "-m", "/dev/sda", "unit", "B", "print"}))
					Expect(scrubbedCommands).To(ContainElement([]string{"parted", "-s", "/dev/sda", "unit", "B", "mkpart", "bosh-partition-x", "1048576", "8590983167"}))
					Expect(scrubbedCommands).To(ContainElement([]string{"parted", "-s", "/dev/sda", "unit", "B", "mkpart", "bosh-partition-x", "8590983168", "221189767167"}))
					Expect(scrubbedCommands).To(ContainElement([]string{"partprobe", "/dev/sda"}))
					Expect(scrubbedCommands).To(ContainElement([]string{"udevadm", "settle"}))
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
						{SizeInBytes: 8589934592, Type: PartitionTypeLinux}, // (8GiB)
						{SizeInBytes: 8589934592, Type: PartitionTypeLinux}, // (8GiB)
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
						{SizeInBytes: 8589934592, Type: PartitionTypeLinux}, // (8GiB)
						{SizeInBytes: 8589934592, Type: PartitionTypeLinux}, // (8GiB)
					}
					err := partitioner.Partition("/dev/sda", partitions)
					Expect(err).ToNot(HaveOccurred())

					Expect(len(fakeCmdRunner.RunCommands)).To(Equal(1))
					Expect(fakeCmdRunner.RunCommands).To(ContainElement([]string{"parted", "-m", "/dev/sda", "unit", "B", "print"}))
				})
			})

			Context("when we have extra partitions", func() {
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
						{SizeInBytes: 8589934592, Type: PartitionTypeLinux}, // (8GiB)
					}
					err := partitioner.Partition("/dev/sda", partitions)
					Expect(err).ToNot(HaveOccurred())

					Expect(len(fakeCmdRunner.RunCommands)).To(Equal(1))
					Expect(fakeCmdRunner.RunCommands).To(ContainElement([]string{"parted", "-m", "/dev/sda", "unit", "B", "print"}))
				})
			})

			Context("when there is an existing partition within the expected size and type", func() {
				for _, fsFormat := range []string{"ext4", "xfs"} {
					Context(fmt.Sprintf("with %s filesystem", fsFormat), func() {
						BeforeEach(func() {
							fakeCmdRunner.AddCmdResult(
								"parted -m /dev/sdf unit B print",
								fakesys.FakeCmdResult{
									Stdout: fmt.Sprintf(`BYT;
/dev/xvdf:3146063544320B:xvd:512:512:gpt:Xen Virtual Block Device;
1:1048576B:3146062496255B:3146062495744B:%s:primary:;
`, fsFormat)},
							)
						})

						It("reuses the existing partition", func() {
							partitions := []Partition{
								{Type: PartitionTypeLinux},
							}

							err := partitioner.Partition("/dev/sdf", partitions)
							Expect(err).ToNot(HaveOccurred())

							Expect(len(fakeCmdRunner.RunCommands)).To(Equal(1))
							Expect(fakeCmdRunner.RunCommands).To(ContainElement([]string{"parted", "-m", "/dev/sdf", "unit", "B", "print"}))
						})
					})
				}
			})

			Context("when a swap partition is used", func() {
				BeforeEach(func() {
					fakeCmdRunner.AddCmdResult(
						"parted -m /dev/sdf unit B print",
						fakesys.FakeCmdResult{
							Stdout: `BYT;
/dev/xvdf:3146063544320B:xvd:512:512:gpt:Xen Virtual Block Device;
1:1048576B:3146062496255B:3146062495744B:linux-swap(v1):primary:;
`},
					)
				})

				It("reuses the existing partition", func() {
					partitions := []Partition{
						{Type: PartitionTypeSwap},
					}

					err := partitioner.Partition("/dev/sdf", partitions)
					Expect(err).ToNot(HaveOccurred())

					Expect(len(fakeCmdRunner.RunCommands)).To(Equal(1))
					Expect(fakeCmdRunner.RunCommands).To(ContainElement([]string{"parted", "-m", "/dev/sdf", "unit", "B", "print"}))
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
							Stderr: "Error: /dev/sda: unrecognised disk label", ExitStatus: 0},
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
							Stderr: "Error: /dev/sda: unrecognised disk label", ExitStatus: 0},
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

	Describe("GetDeviceSizeInBytes", func() {
		It("returns error if lsblk fails", func() {
			fakeCmdRunner.AddCmdResult(
				"lsblk --nodeps -nb -o SIZE /dev/path",
				fakesys.FakeCmdResult{Error: errors.New("fake-err")},
			)

			_, err := partitioner.GetDeviceSizeInBytes("/dev/path")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("returns error if lsblk doesnt return number", func() {
			fakeCmdRunner.AddCmdResult(
				"lsblk --nodeps -nb -o SIZE /dev/path",
				fakesys.FakeCmdResult{Stdout: "not-number"},
			)

			_, err := partitioner.GetDeviceSizeInBytes("/dev/path")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Converting block device size"))
			Expect(err.Error()).To(ContainSubstring(`parsing "not-number"`))
		})

		It("returns number in bytes (stripping newline) from lsblk", func() {
			fakeCmdRunner.AddCmdResult(
				"lsblk --nodeps -nb -o SIZE /dev/path",
				fakesys.FakeCmdResult{Stdout: "123\n"},
			)

			num, err := partitioner.GetDeviceSizeInBytes("/dev/path")
			Expect(err).ToNot(HaveOccurred())
			Expect(num).To(Equal(uint64(123)))
		})
	})
})
