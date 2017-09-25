package disk_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"errors"
	. "github.com/cloudfoundry/bosh-agent/platform/disk"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("Linux Formatter", func() {
	Describe("when using swap", func() {
		It("format as swap disk if partition has not been formatted", func() {
			fakeRunner := fakesys.NewFakeCmdRunner()
			fakeFs := fakesys.NewFakeFileSystem()
			fakeRunner.AddCmdResult("blkid -p /dev/xvda2", fakesys.FakeCmdResult{ExitStatus: 2, Error: errors.New("Exit code 2")})

			formatter := NewLinuxFormatter(fakeRunner, fakeFs)
			formatter.Format("/dev/xvda2", FileSystemSwap)

			Expect(2).To(Equal(len(fakeRunner.RunCommands)))
			Expect(fakeRunner.RunCommands[1]).To(Equal([]string{"mkswap", "/dev/xvda2"}))
		})

		It("reformats the partition if is not formatted as swap", func() {
			fakeRunner := fakesys.NewFakeCmdRunner()
			fakeFs := fakesys.NewFakeFileSystem()
			fakeRunner.AddCmdResult("blkid -p /dev/xvda1", fakesys.FakeCmdResult{Stdout: `xxxxx TYPE="ext4" yyyy zzzz`})

			formatter := NewLinuxFormatter(fakeRunner, fakeFs)
			formatter.Format("/dev/xvda1", FileSystemSwap)

			Expect(2).To(Equal(len(fakeRunner.RunCommands)))
			Expect(fakeRunner.RunCommands[1]).To(Equal([]string{"mkswap", "/dev/xvda1"}))
		})

		It("it does not reformat if it already formatted as swap", func() {
			fakeRunner := fakesys.NewFakeCmdRunner()
			fakeFs := fakesys.NewFakeFileSystem()
			fakeRunner.AddCmdResult("blkid -p /dev/xvda1", fakesys.FakeCmdResult{Stdout: `xxxxx TYPE="swap" yyyy zzzz`})

			formatter := NewLinuxFormatter(fakeRunner, fakeFs)
			formatter.Format("/dev/xvda1", FileSystemSwap)

			Expect(1).To(Equal(len(fakeRunner.RunCommands)))
			Expect(fakeRunner.RunCommands[0]).To(Equal([]string{"blkid", "-p", "/dev/xvda1"}))
		})
	})

	Describe("when using ext4", func() {
		It("allows lazy itable support", func() {
			fakeRunner := fakesys.NewFakeCmdRunner()
			fakeFs := fakesys.NewFakeFileSystem()
			fakeFs.WriteFile("/sys/fs/ext4/features/lazy_itable_init", []byte{})
			fakeRunner.AddCmdResult("blkid -p /dev/xvda2", fakesys.FakeCmdResult{Stdout: `xxxxx TYPE="ext2" yyyy zzzz`})

			formatter := NewLinuxFormatter(fakeRunner, fakeFs)
			formatter.Format("/dev/xvda2", FileSystemExt4)

			Expect(2).To(Equal(len(fakeRunner.RunCommands)))
			Expect(fakeRunner.RunCommands[1]).To(Equal([]string{"mke2fs", "-t", "ext4", "-j", "-E", "lazy_itable_init=1", "/dev/xvda2"}))

		})

		It("allows without lazy itable support", func() {
			fakeRunner := fakesys.NewFakeCmdRunner()
			fakeFs := fakesys.NewFakeFileSystem()
			fakeRunner.AddCmdResult("blkid -p /dev/xvda2", fakesys.FakeCmdResult{Stdout: `xxxxx TYPE="ext2" yyyy zzzz`})

			formatter := NewLinuxFormatter(fakeRunner, fakeFs)
			formatter.Format("/dev/xvda2", FileSystemExt4)

			Expect(2).To(Equal(len(fakeRunner.RunCommands)))
			Expect(fakeRunner.RunCommands[1]).To(Equal([]string{"mke2fs", "-t", "ext4", "-j", "/dev/xvda2"}))
		})

		It("does not re-partition if fs is already ext4", func() {
			fakeRunner := fakesys.NewFakeCmdRunner()
			fakeFs := fakesys.NewFakeFileSystem()
			fakeRunner.AddCmdResult("blkid -p /dev/xvda1", fakesys.FakeCmdResult{Stdout: `xxxxx TYPE="ext4" yyyy zzzz`})

			formatter := NewLinuxFormatter(fakeRunner, fakeFs)
			formatter.Format("/dev/xvda1", FileSystemExt4)

			Expect(1).To(Equal(len(fakeRunner.RunCommands)))
			Expect(fakeRunner.RunCommands[0]).To(Equal([]string{"blkid", "-p", "/dev/xvda1"}))
		})

		It("does not re-partition if fs is already xfs", func() {
			fakeRunner := fakesys.NewFakeCmdRunner()
			fakeFs := fakesys.NewFakeFileSystem()
			fakeRunner.AddCmdResult("blkid -p /dev/xvda2", fakesys.FakeCmdResult{Stdout: `xxxxx TYPE="xfs" yyyy zzzz`})

			formatter := NewLinuxFormatter(fakeRunner, fakeFs)
			formatter.Format("/dev/xvda2", FileSystemExt4)

			Expect(1).To(Equal(len(fakeRunner.RunCommands)))
			Expect(fakeRunner.RunCommands[0]).To(Equal([]string{"blkid", "-p", "/dev/xvda2"}))
		})

		It("reformats if fs is not a supported fs type", func() {
			fakeRunner := fakesys.NewFakeCmdRunner()
			fakeFs := fakesys.NewFakeFileSystem()
			fakeRunner.AddCmdResult("blkid -p /dev/xvda2", fakesys.FakeCmdResult{Stdout: `xxxxx TYPE="somethingelse" yyyy zzzz`})

			formatter := NewLinuxFormatter(fakeRunner, fakeFs)
			formatter.Format("/dev/xvda2", FileSystemExt4)

			Expect(2).To(Equal(len(fakeRunner.RunCommands)))
			Expect(fakeRunner.RunCommands[0]).To(Equal([]string{"blkid", "-p", "/dev/xvda2"}))
		})
	})

	Describe("when using xfs", func() {
		It("formats a blank disk with type xfs", func() {
			fakeRunner := fakesys.NewFakeCmdRunner()
			fakeFs := fakesys.NewFakeFileSystem()
			fakeRunner.AddCmdResult("blkid -p /dev/xvda2", fakesys.FakeCmdResult{ExitStatus: 2, Error: errors.New("Exit code 2")})

			formatter := NewLinuxFormatter(fakeRunner, fakeFs)
			formatter.Format("/dev/xvda2", FileSystemXFS)

			Expect(2).To(Equal(len(fakeRunner.RunCommands)))
			Expect(fakeRunner.RunCommands[1]).To(Equal([]string{"mkfs.xfs", "/dev/xvda2"}))
		})

		It("does not re-format if fs is already ext4", func() {
			fakeRunner := fakesys.NewFakeCmdRunner()
			fakeFs := fakesys.NewFakeFileSystem()
			fakeRunner.AddCmdResult("blkid -p /dev/xvda1", fakesys.FakeCmdResult{Stdout: `xxxxx TYPE="ext4" yyyy zzzz`})

			formatter := NewLinuxFormatter(fakeRunner, fakeFs)
			formatter.Format("/dev/xvda1", FileSystemXFS)

			Expect(1).To(Equal(len(fakeRunner.RunCommands)))
			Expect(fakeRunner.RunCommands[0]).To(Equal([]string{"blkid", "-p", "/dev/xvda1"}))
		})

		It("does not re-partition if fs is already xfs", func() {
			fakeRunner := fakesys.NewFakeCmdRunner()
			fakeFs := fakesys.NewFakeFileSystem()
			fakeRunner.AddCmdResult("blkid -p /dev/xvda1", fakesys.FakeCmdResult{Stdout: `xxxxx TYPE="xfs" yyyy zzzz`})

			formatter := NewLinuxFormatter(fakeRunner, fakeFs)
			formatter.Format("/dev/xvda1", FileSystemXFS)

			Expect(1).To(Equal(len(fakeRunner.RunCommands)))
			Expect(fakeRunner.RunCommands[0]).To(Equal([]string{"blkid", "-p", "/dev/xvda1"}))
		})

		It("throws an error if formatting filesystem fails", func() {
			fakeRunner := fakesys.NewFakeCmdRunner()
			fakeFs := fakesys.NewFakeFileSystem()
			fakeRunner.AddCmdResult("mkfs.xfs /dev/xvda2", fakesys.FakeCmdResult{Error: errors.New("Sadness")})
			fakeRunner.AddCmdResult("blkid -p /dev/xvda2", fakesys.FakeCmdResult{Stderr: "", ExitStatus: 2})

			formatter := NewLinuxFormatter(fakeRunner, fakeFs)
			err := formatter.Format("/dev/xvda2", FileSystemXFS)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Shelling out to mkfs.xfs: Sadness"))
		})
	})
})
