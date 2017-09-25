package platform_test

import (
	"errors"
	"os"
	"path"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/platform"

	fakedpresolv "github.com/cloudfoundry/bosh-agent/infrastructure/devicepathresolver/fakes"
	fakecert "github.com/cloudfoundry/bosh-agent/platform/cert/fakes"
	fakedevutil "github.com/cloudfoundry/bosh-agent/platform/deviceutil/fakes"
	fakedisk "github.com/cloudfoundry/bosh-agent/platform/disk/fakes"
	fakenet "github.com/cloudfoundry/bosh-agent/platform/net/fakes"
	fakestats "github.com/cloudfoundry/bosh-agent/platform/stats/fakes"
	fakeretry "github.com/cloudfoundry/bosh-utils/retrystrategy/fakes"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	fakeuuidgen "github.com/cloudfoundry/bosh-utils/uuid/fakes"

	boshdisk "github.com/cloudfoundry/bosh-agent/platform/disk"
	boshvitals "github.com/cloudfoundry/bosh-agent/platform/vitals"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshdirs "github.com/cloudfoundry/bosh-agent/settings/directories"
	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var _ = Describe("LinuxPlatform", describeLinuxPlatform)

func describeLinuxPlatform() {
	var (
		collector                  *fakestats.FakeCollector
		fs                         *fakesys.FakeFileSystem
		cmdRunner                  *fakesys.FakeCmdRunner
		diskManager                *fakedisk.FakeDiskManager
		dirProvider                boshdirs.Provider
		devicePathResolver         *fakedpresolv.FakeDevicePathResolver
		platform                   Platform
		cdutil                     *fakedevutil.FakeDeviceUtil
		compressor                 boshcmd.Compressor
		copier                     boshcmd.Copier
		vitalsService              boshvitals.Service
		netManager                 *fakenet.FakeManager
		certManager                *fakecert.FakeManager
		monitRetryStrategy         *fakeretry.FakeRetryStrategy
		fakeDefaultNetworkResolver *fakenet.FakeDefaultNetworkResolver

		fakeUUIDGenerator *fakeuuidgen.FakeGenerator

		state    *BootstrapState
		stateErr error
		options  LinuxOptions

		logger boshlog.Logger
	)

	BeforeEach(func() {
		logger = boshlog.NewLogger(boshlog.LevelNone)

		collector = &fakestats.FakeCollector{}
		fs = fakesys.NewFakeFileSystem()
		cmdRunner = fakesys.NewFakeCmdRunner()
		diskManager = fakedisk.NewFakeDiskManager()
		dirProvider = boshdirs.NewProvider("/fake-dir")
		cdutil = fakedevutil.NewFakeDeviceUtil()
		compressor = boshcmd.NewTarballCompressor(cmdRunner, fs)
		copier = boshcmd.NewCpCopier(cmdRunner, fs, logger)
		vitalsService = boshvitals.NewService(collector, dirProvider)
		netManager = &fakenet.FakeManager{}
		certManager = new(fakecert.FakeManager)
		monitRetryStrategy = fakeretry.NewFakeRetryStrategy()
		devicePathResolver = fakedpresolv.NewFakeDevicePathResolver()
		fakeDefaultNetworkResolver = &fakenet.FakeDefaultNetworkResolver{}

		fakeUUIDGenerator = fakeuuidgen.NewFakeGenerator()

		state, stateErr = NewBootstrapState(fs, "/agent-state.json")
		Expect(stateErr).NotTo(HaveOccurred())

		options = LinuxOptions{}

		fs.SetGlob("/sys/bus/scsi/devices/*:0:0:0/block/*", []string{
			"/sys/bus/scsi/devices/0:0:0:0/block/sr0",
			"/sys/bus/scsi/devices/6:0:0:0/block/sdd",
			"/sys/bus/scsi/devices/fake-host-id:0:0:0/block/sda",
		})

		fs.SetGlob("/sys/bus/scsi/devices/fake-host-id:0:fake-disk-id:0/block/*", []string{
			"/sys/bus/scsi/devices/fake-host-id:0:fake-disk-id:0/block/sdf",
		})
	})

	JustBeforeEach(func() {
		platform = NewLinuxPlatform(
			fs,
			cmdRunner,
			collector,
			compressor,
			copier,
			dirProvider,
			vitalsService,
			cdutil,
			diskManager,
			netManager,
			certManager,
			monitRetryStrategy,
			devicePathResolver,
			state,
			options,
			logger,
			fakeDefaultNetworkResolver,
			fakeUUIDGenerator,
		)
	})

	Describe("SetupRuntimeConfiguration", func() {
		It("setups runtime configuration", func() {
			err := platform.SetupRuntimeConfiguration()
			Expect(err).NotTo(HaveOccurred())

			Expect(len(cmdRunner.RunCommands)).To(Equal(1))
			Expect(cmdRunner.RunCommands[0]).To(Equal([]string{"bosh-agent-rc"}))
		})
	})

	Describe("CreateUser", func() {
		It("creates user", func() {
			expectedUseradd := []string{
				"useradd",
				"-m",
				"-b", "/some/path/to/home",
				"-s", "/bin/bash",
				"-p", "bar-pwd",
				"foo-user",
			}

			err := platform.CreateUser("foo-user", "bar-pwd", "/some/path/to/home")
			Expect(err).NotTo(HaveOccurred())

			basePathStat := fs.GetFileTestStat("/some/path/to/home")
			Expect(basePathStat.FileType).To(Equal(fakesys.FakeFileTypeDir))
			Expect(basePathStat.FileMode).To(Equal(os.FileMode(0755)))

			Expect(cmdRunner.RunCommands).To(Equal([][]string{expectedUseradd}))
		})

		It("creates user with an empty password", func() {
			expectedUseradd := []string{
				"useradd",
				"-m",
				"-b", "/some/path/to/home",
				"-s", "/bin/bash",
				"foo-user",
			}

			err := platform.CreateUser("foo-user", "", "/some/path/to/home")
			Expect(err).NotTo(HaveOccurred())

			basePathStat := fs.GetFileTestStat("/some/path/to/home")
			Expect(basePathStat.FileType).To(Equal(fakesys.FakeFileTypeDir))
			Expect(basePathStat.FileMode).To(Equal(os.FileMode(0755)))

			Expect(cmdRunner.RunCommands).To(Equal([][]string{expectedUseradd}))
		})
	})

	Describe("AddUserToGroups", func() {
		It("adds user to groups", func() {
			err := platform.AddUserToGroups("foo-user", []string{"group1", "group2", "group3"})
			Expect(err).NotTo(HaveOccurred())

			Expect(len(cmdRunner.RunCommands)).To(Equal(1))

			usermod := []string{"usermod", "-G", "group1,group2,group3", "foo-user"}
			Expect(cmdRunner.RunCommands[0]).To(Equal(usermod))
		})
	})

	Describe("DeleteEphemeralUsersMatching", func() {
		It("deletes users with prefix and regex", func() {
			passwdFile := `bosh_foo:...
bosh_bar:...
foo:...
bar:...
foobar:...
bosh_foobar:...`

			fs.WriteFileString("/etc/passwd", passwdFile)

			err := platform.DeleteEphemeralUsersMatching("bar$")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(cmdRunner.RunCommands)).To(Equal(2))
			Expect(cmdRunner.RunCommands[0]).To(Equal([]string{"userdel", "-r", "bosh_bar"}))
			Expect(cmdRunner.RunCommands[1]).To(Equal([]string{"userdel", "-r", "bosh_foobar"}))
		})
	})

	Describe("SetupRootDisk", func() {
		BeforeEach(func() {
			mountsSearcher := diskManager.FakeMountsSearcher

			mountsSearcher.SearchMountsMounts = []boshdisk.Mount{{
				PartitionPath: "/dev/sda1",
				MountPoint:    "/",
			}}

			devicePathResolver.GetRealDevicePathStub = func(diskSettings boshsettings.DiskSettings) (string, bool, error) {
				return diskSettings.Path, false, nil
			}
		})

		Context("when growpart is installed", func() {
			BeforeEach(func() {
				cmdRunner.CommandExistsValue = true
				options.CreatePartitionIfNoEphemeralDisk = false
			})

			It("runs growpart and resize2fs", func() {
				cmdRunner.AddCmdResult(
					"readlink -f /dev/sda1",
					fakesys.FakeCmdResult{Error: nil, Stdout: "/dev/sda1"},
				)

				err := platform.SetupRootDisk("/dev/sdb")

				Expect(err).NotTo(HaveOccurred())
				Expect(len(cmdRunner.RunCommands)).To(Equal(3))
				Expect(cmdRunner.RunCommands[1]).To(Equal([]string{"growpart", "/dev/sda", "1"}))
				Expect(cmdRunner.RunCommands[2]).To(Equal([]string{"resize2fs", "-f", "/dev/sda1"}))
			})

			It("runs growpart and resize2fs for the right root device number", func() {
				err := platform.SetupEphemeralDiskWithPath("/dev/sda")
				Expect(err).NotTo(HaveOccurred())

				mountsSearcher := diskManager.FakeMountsSearcher
				mountsSearcher.SearchMountsMounts = []boshdisk.Mount{{
					PartitionPath: "/dev/sda2",
					MountPoint:    "/",
				}}

				cmdRunner.AddCmdResult(
					"readlink -f /dev/sda2",
					fakesys.FakeCmdResult{Error: nil, Stdout: "/dev/sda2"},
				)

				err = platform.SetupRootDisk("/dev/sdb")

				Expect(err).NotTo(HaveOccurred())
				Expect(len(cmdRunner.RunCommands)).To(Equal(3))
				Expect(cmdRunner.RunCommands[1]).To(Equal([]string{"growpart", "/dev/sda", "2"}))
				Expect(cmdRunner.RunCommands[2]).To(Equal([]string{"resize2fs", "-f", "/dev/sda2"}))
			})

			It("returns error if it can't find the root device", func() {
				cmdRunner.AddCmdResult(
					"readlink -f /dev/sda1",
					fakesys.FakeCmdResult{Error: errors.New("fake-readlink-error")},
				)

				err := platform.SetupRootDisk("/dev/sdb")

				Expect(err).To(HaveOccurred())
				Expect(len(cmdRunner.RunCommands)).To(Equal(1))
			})

			It("returns an error if growing the partiton fails", func() {
				cmdRunner.AddCmdResult(
					"readlink -f /dev/sda1",
					fakesys.FakeCmdResult{Error: nil, Stdout: "/dev/sda1"},
				)

				cmdRunner.AddCmdResult(
					"growpart /dev/sda 1",
					fakesys.FakeCmdResult{Error: errors.New("fake-growpart-error")},
				)

				err := platform.SetupRootDisk("/dev/sdb")

				Expect(err).To(HaveOccurred())
				Expect(len(cmdRunner.RunCommands)).To(Equal(2))
			})

			It("returns error if resizing the filesystem fails", func() {
				cmdRunner.AddCmdResult(
					"readlink -f /dev/sda1",
					fakesys.FakeCmdResult{Error: nil, Stdout: "/dev/sda1"},
				)

				cmdRunner.AddCmdResult(
					"resize2fs -f /dev/sda1",
					fakesys.FakeCmdResult{Error: errors.New("fake-resize2fs-error")},
				)

				err := platform.SetupRootDisk("/dev/sdb")

				Expect(err).To(HaveOccurred())
				Expect(len(cmdRunner.RunCommands)).To(Equal(3))
			})

			It("skips growing root fs if no ephemerial disk is provided", func() {
				var platformWithNoEphemeralDisk Platform

				options.CreatePartitionIfNoEphemeralDisk = true
				platformWithNoEphemeralDisk = NewLinuxPlatform(
					fs,
					cmdRunner,
					collector,
					compressor,
					copier,
					dirProvider,
					vitalsService,
					cdutil,
					diskManager,
					netManager,
					certManager,
					monitRetryStrategy,
					devicePathResolver,
					state,
					options,
					logger,
					fakeDefaultNetworkResolver,
					fakeUUIDGenerator,
				)
				err := platformWithNoEphemeralDisk.SetupRootDisk("")

				Expect(err).ToNot(HaveOccurred())
				Expect(len(cmdRunner.RunCommands)).To(Equal(0))
			})
		})

		Context("when growpart is not installed", func() {
			BeforeEach(func() {
				cmdRunner.CommandExistsValue = false
				options.CreatePartitionIfNoEphemeralDisk = false
			})

			It("does not return error if growpart is not installed and skips growing fs", func() {
				err := platform.SetupRootDisk("/dev/sdb")

				Expect(err).ToNot(HaveOccurred())
				Expect(len(cmdRunner.RunCommands)).To(Equal(0))
			})
		})

		Context("when SkipDiskSetup is true", func() {
			BeforeEach(func() {
				options.SkipDiskSetup = true
				cmdRunner.CommandExistsValue = true
			})

			It("does nothing", func() {
				err := platform.SetupRootDisk("/dev/sdb")

				Expect(err).ToNot(HaveOccurred())
				Expect(len(cmdRunner.RunCommands)).To(Equal(0))
			})
		})
	})

	Describe("SetupSSH", func() {
		It("setup ssh", func() {
			fs.HomeDirHomePath = "/some/home/dir"

			platform.SetupSSH("some public key", "vcap")

			sshDirPath := "/some/home/dir/.ssh"
			sshDirStat := fs.GetFileTestStat(sshDirPath)

			Expect("vcap").To(Equal(fs.HomeDirUsername))

			Expect(sshDirStat).NotTo(BeNil())
			Expect(sshDirStat.FileType).To(Equal(fakesys.FakeFileTypeDir))
			Expect(os.FileMode(0700)).To(Equal(sshDirStat.FileMode))
			Expect("vcap").To(Equal(sshDirStat.Username))

			authKeysStat := fs.GetFileTestStat(path.Join(sshDirPath, "authorized_keys"))

			Expect(authKeysStat).NotTo(BeNil())
			Expect(fakesys.FakeFileTypeFile).To(Equal(authKeysStat.FileType))
			Expect(os.FileMode(0600)).To(Equal(authKeysStat.FileMode))
			Expect("vcap").To(Equal(authKeysStat.Username))
			Expect("some public key").To(Equal(authKeysStat.StringContents()))
		})

	})

	Describe("SetUserPassword", func() {
		It("set user password", func() {
			platform.SetUserPassword("my-user", "my-encrypted-password")
			Expect(len(cmdRunner.RunCommands)).To(Equal(1))
			Expect(cmdRunner.RunCommands[0]).To(Equal([]string{"usermod", "-p", "my-encrypted-password", "my-user"}))
		})
	})

	Describe("SetupHostname", func() {
		const expectedEtcHosts = `127.0.0.1 localhost foobar.local

# The following lines are desirable for IPv6 capable hosts
::1 localhost ip6-localhost ip6-loopback foobar.local
fe00::0 ip6-localnet
ff00::0 ip6-mcastprefix
ff02::1 ip6-allnodes
ff02::2 ip6-allrouters
ff02::3 ip6-allhosts
`
		Context("When running command to get hostname fails", func() {
			It("returns an error", func() {
				result := fakesys.FakeCmdResult{Error: errors.New("Oops!")}
				cmdRunner.AddCmdResult("hostname foobar.local", result)

				err := platform.SetupHostname("foobar.local")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Setting hostname: Oops!"))
			})
		})

		Context("When writing to the /etc/hostname file fails", func() {
			It("returns an error", func() {
				fs.WriteFileErrors["/etc/hostname"] = errors.New("ENXIO: disk failed")

				err := platform.SetupHostname("foobar.local")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Writing to /etc/hostname: ENXIO: disk failed"))
			})
		})

		Context("When writing to /etc/hosts file fails", func() {
			It("returns an error", func() {
				fs.WriteFileErrors["/etc/hosts"] = errors.New("ENXIO: disk failed")

				err := platform.SetupHostname("foobar.local")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Writing to /etc/hosts: ENXIO: disk failed"))
			})
		})

		Context("When saving bootstrap state fails", func() {
			It("returns an error", func() {
				fs.WriteFileErrors["/agent-state.json"] = errors.New("ENXIO: disk failed")

				err := platform.SetupHostname("foobar.local")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Setting up hostname: Writing bootstrap state to file: ENXIO: disk failed"))
			})
		})

		Context("When host files have not yet been configured", func() {
			It("sets up hostname", func() {
				platform.SetupHostname("foobar.local")
				Expect(len(cmdRunner.RunCommands)).To(Equal(1))
				Expect(cmdRunner.RunCommands[0]).To(Equal([]string{"hostname", "foobar.local"}))

				hostnameFileContent, err := fs.ReadFileString("/etc/hostname")
				Expect(err).NotTo(HaveOccurred())
				Expect(hostnameFileContent).To(Equal("foobar.local"))

				hostsFileContent, err := fs.ReadFileString("/etc/hosts")
				Expect(err).NotTo(HaveOccurred())
				Expect(hostsFileContent).To(Equal(expectedEtcHosts))
			})
		})

		Context("When host files have already been configured", func() {
			It("skips setting up hostname to prevent overriding changes made by the release author", func() {
				platform.SetupHostname("foobar.local")
				platform.SetupHostname("newfoo.local")

				Expect(len(cmdRunner.RunCommands)).To(Equal(1))
				Expect(cmdRunner.RunCommands[0]).To(Equal([]string{"hostname", "foobar.local"}))

				hostnameFileContent, err := fs.ReadFileString("/etc/hostname")
				Expect(err).NotTo(HaveOccurred())
				Expect(hostnameFileContent).To(Equal("foobar.local"))

				hostsFileContent, err := fs.ReadFileString("/etc/hosts")
				Expect(err).NotTo(HaveOccurred())
				Expect(hostsFileContent).To(Equal(expectedEtcHosts))
			})
		})
	})

	Describe("SetupLogrotate", func() {
		const expectedEtcLogrotate = `# Generated by bosh-agent

fake-base-path/data/sys/log/*.log fake-base-path/data/sys/log/.*.log fake-base-path/data/sys/log/*/*.log fake-base-path/data/sys/log/*/.*.log fake-base-path/data/sys/log/*/*/*.log fake-base-path/data/sys/log/*/*/.*.log {
  missingok
  rotate 7
  compress
  delaycompress
  copytruncate
  size=fake-size
}
`

		It("sets up logrotate", func() {
			platform.SetupLogrotate("fake-group-name", "fake-base-path", "fake-size")

			logrotateFileContent, err := fs.ReadFileString("/etc/logrotate.d/fake-group-name")
			Expect(err).NotTo(HaveOccurred())
			Expect(logrotateFileContent).To(Equal(expectedEtcLogrotate))
		})
	})

	Describe("SetTimeWithNtpServers", func() {
		It("sets time with ntp servers", func() {
			platform.SetTimeWithNtpServers([]string{"0.north-america.pool.ntp.org", "1.north-america.pool.ntp.org"})

			ntpConfig := fs.GetFileTestStat("/fake-dir/bosh/etc/ntpserver")
			Expect(ntpConfig.StringContents()).To(Equal("0.north-america.pool.ntp.org 1.north-america.pool.ntp.org"))
			Expect(ntpConfig.FileType).To(Equal(fakesys.FakeFileTypeFile))

			Expect(len(cmdRunner.RunCommands)).To(Equal(1))
			Expect(cmdRunner.RunCommands[0]).To(Equal([]string{"ntpdate"}))
		})

		It("sets time with ntp servers is noop when no ntp server provided", func() {
			platform.SetTimeWithNtpServers([]string{})
			Expect(len(cmdRunner.RunCommands)).To(Equal(0))

			ntpConfig := fs.GetFileTestStat("/fake-dir/bosh/etc/ntpserver")
			Expect(ntpConfig).To(BeNil())
		})
	})

	Describe("SetupEphemeralDiskWithPath", func() {
		var (
			partitioner *fakedisk.FakePartitioner
			formatter   *fakedisk.FakeFormatter
			mounter     *fakedisk.FakeMounter
		)

		BeforeEach(func() {
			partitioner = diskManager.FakePartitioner
			formatter = diskManager.FakeFormatter
			mounter = diskManager.FakeMounter
		})

		itSetsUpEphemeralDisk := func(act func() error) {
			It("sets up ephemeral disk with path", func() {
				err := act()
				Expect(err).NotTo(HaveOccurred())

				dataDir := fs.GetFileTestStat("/fake-dir/data")
				Expect(dataDir.FileType).To(Equal(fakesys.FakeFileTypeDir))
				Expect(dataDir.FileMode).To(Equal(os.FileMode(0750)))
			})

			It("creates new partition even if the data directory is not empty", func() {
				fs.SetGlob(path.Join("/fake-dir", "data", "*"), []string{"something"})

				err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(partitioner.PartitionCalled).To(BeTrue())
				Expect(formatter.FormatCalled).To(BeTrue())
				Expect(mounter.MountCalled).To(BeTrue())
			})
		}

		Context("when ephemeral disk path is provided", func() {
			act := func() error { return platform.SetupEphemeralDiskWithPath("/dev/xvda") }

			itSetsUpEphemeralDisk(act)

			It("returns error if creating data dir fails", func() {
				fs.MkdirAllError = errors.New("fake-mkdir-all-err")

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-mkdir-all-err"))
				Expect(partitioner.PartitionCalled).To(BeFalse())
				Expect(formatter.FormatCalled).To(BeFalse())
				Expect(mounter.MountCalled).To(BeFalse())
			})

			It("returns err when the data directory cannot be globbed", func() {
				fs.GlobErr = errors.New("fake-glob-err")

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Globbing ephemeral disk mount point `/fake-dir/data/*'"))
				Expect(err.Error()).To(ContainSubstring("fake-glob-err"))
				Expect(partitioner.PartitionCalled).To(BeFalse())
				Expect(formatter.FormatCalled).To(BeFalse())
				Expect(mounter.MountCalled).To(BeFalse())
			})

			It("returns err when mem stats are unavailable", func() {
				collector.MemStatsErr = errors.New("fake-memstats-error")
				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Calculating partition sizes"))
				Expect(err.Error()).To(ContainSubstring("fake-memstats-error"))
				Expect(partitioner.PartitionCalled).To(BeFalse())
				Expect(formatter.FormatCalled).To(BeFalse())
				Expect(mounter.MountCalled).To(BeFalse())
			})

			It("returns an error when partitioning fails", func() {
				partitioner.PartitionErr = errors.New("fake-partition-error")
				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Partitioning ephemeral disk `/dev/xvda'"))
				Expect(err.Error()).To(ContainSubstring("fake-partition-error"))
				Expect(formatter.FormatCalled).To(BeFalse())
				Expect(mounter.MountCalled).To(BeFalse())
			})

			It("formats swap and data partitions", func() {
				err := act()
				Expect(err).NotTo(HaveOccurred())

				Expect(len(formatter.FormatPartitionPaths)).To(Equal(2))
				Expect(formatter.FormatPartitionPaths[0]).To(Equal("/dev/xvda1"))
				Expect(formatter.FormatPartitionPaths[1]).To(Equal("/dev/xvda2"))

				Expect(len(formatter.FormatFsTypes)).To(Equal(2))
				Expect(formatter.FormatFsTypes[0]).To(Equal(boshdisk.FileSystemSwap))
				Expect(formatter.FormatFsTypes[1]).To(Equal(boshdisk.FileSystemExt4))
			})

			It("mounts swap and data partitions", func() {
				err := act()
				Expect(err).NotTo(HaveOccurred())

				Expect(len(mounter.MountMountPoints)).To(Equal(1))
				Expect(mounter.MountMountPoints[0]).To(Equal("/fake-dir/data"))
				Expect(len(mounter.MountPartitionPaths)).To(Equal(1))
				Expect(mounter.MountPartitionPaths[0]).To(Equal("/dev/xvda2"))

				Expect(len(mounter.SwapOnPartitionPaths)).To(Equal(1))
				Expect(mounter.SwapOnPartitionPaths[0]).To(Equal("/dev/xvda1"))
			})

			It("creates swap the size of the memory and the rest for data when disk is bigger than twice the memory", func() {
				memSizeInBytes := uint64(1024 * 1024 * 1024)
				diskSizeInBytes := 2*memSizeInBytes + 64
				fakePartitioner := partitioner
				fakePartitioner.GetDeviceSizeInBytesSizes["/dev/xvda"] = diskSizeInBytes
				collector.MemStats.Total = memSizeInBytes

				err := act()
				Expect(err).NotTo(HaveOccurred())
				Expect(fakePartitioner.PartitionPartitions).To(Equal([]boshdisk.Partition{
					{SizeInBytes: memSizeInBytes, Type: boshdisk.PartitionTypeSwap},
					{SizeInBytes: diskSizeInBytes - memSizeInBytes, Type: boshdisk.PartitionTypeLinux},
				}))
			})

			It("creates equal swap and data partitions when disk is twice the memory or smaller", func() {
				memSizeInBytes := uint64(1024 * 1024 * 1024)
				diskSizeInBytes := 2*memSizeInBytes - 64
				fakePartitioner := partitioner
				fakePartitioner.GetDeviceSizeInBytesSizes["/dev/xvda"] = diskSizeInBytes
				collector.MemStats.Total = memSizeInBytes

				err := act()
				Expect(err).NotTo(HaveOccurred())
				Expect(fakePartitioner.PartitionPartitions).To(Equal([]boshdisk.Partition{
					{SizeInBytes: diskSizeInBytes / 2, Type: boshdisk.PartitionTypeSwap},
					{SizeInBytes: diskSizeInBytes / 2, Type: boshdisk.PartitionTypeLinux},
				}))
			})
		})

		Context("when ephemeral disk path is not provided", func() {
			act := func() error { return platform.SetupEphemeralDiskWithPath("") }

			Context("when agent should partition ephemeral disk on root disk", func() {
				BeforeEach(func() {
					partitioner = diskManager.FakeRootDevicePartitioner
					options.CreatePartitionIfNoEphemeralDisk = true
				})

				Context("when root device fails to be determined", func() {
					BeforeEach(func() {
						diskManager.FakeMountsSearcher.SearchMountsErr = errors.New("fake-mounts-searcher-error")
					})

					It("returns an error", func() {
						err := act()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("Finding root partition device"))
						Expect(partitioner.PartitionCalled).To(BeFalse())
						Expect(formatter.FormatCalled).To(BeFalse())
						Expect(mounter.MountCalled).To(BeFalse())
					})
				})

				Context("when root device is determined", func() {
					BeforeEach(func() {
						diskManager.FakeMountsSearcher.SearchMountsMounts = []boshdisk.Mount{
							{MountPoint: "/", PartitionPath: "rootfs"},
							{MountPoint: "/", PartitionPath: "/dev/vda1"},
						}
					})

					Context("when getting absolute path fails", func() {
						BeforeEach(func() {
							cmdRunner.AddCmdResult(
								"readlink -f /dev/vda1",
								fakesys.FakeCmdResult{Error: errors.New("fake-readlink-error")},
							)
						})

						It("returns an error", func() {
							err := act()
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(ContainSubstring("fake-readlink-error"))
							Expect(partitioner.PartitionCalled).To(BeFalse())
							Expect(formatter.FormatCalled).To(BeFalse())
							Expect(mounter.MountCalled).To(BeFalse())
						})
					})

					Context("when getting absolute path suceeds", func() {
						BeforeEach(func() {
							cmdRunner.AddCmdResult(
								"readlink -f /dev/vda1",
								fakesys.FakeCmdResult{Stdout: "/dev/vda1"},
							)
						})

						Context("when root device has insufficient space for ephemeral partitions", func() {
							BeforeEach(func() {
								partitioner.GetDeviceSizeInBytesSizes["/dev/vda"] = 1024*1024*1024 - 1
								collector.MemStats.Total = 8
							})

							It("returns an error", func() {
								err := act()
								Expect(err).To(HaveOccurred())
								Expect(err.Error()).To(ContainSubstring("Insufficient remaining disk"))
								Expect(partitioner.PartitionCalled).To(BeFalse())
								Expect(formatter.FormatCalled).To(BeFalse())
								Expect(mounter.MountCalled).To(BeFalse())
							})
						})

						Context("when root device has sufficient space for ephemeral partitions", func() {
							BeforeEach(func() {
								partitioner.GetDeviceSizeInBytesSizes["/dev/vda"] = 1024 * 1024 * 1024
								collector.MemStats.Total = 256 * 1024 * 1024
							})

							itSetsUpEphemeralDisk(act)

							It("returns err when mem stats are unavailable", func() {
								collector.MemStatsErr = errors.New("fake-memstats-error")
								err := act()
								Expect(err).To(HaveOccurred())
								Expect(err.Error()).To(ContainSubstring("Calculating partition sizes"))
								Expect(err.Error()).To(ContainSubstring("fake-memstats-error"))
								Expect(partitioner.PartitionCalled).To(BeFalse())
								Expect(formatter.FormatCalled).To(BeFalse())
								Expect(mounter.MountCalled).To(BeFalse())
							})

							It("returns an error when partitioning fails", func() {
								partitioner.PartitionErr = errors.New("fake-partition-error")
								err := act()
								Expect(err).To(HaveOccurred())
								Expect(err.Error()).To(ContainSubstring("Partitioning root device `/dev/vda'"))
								Expect(err.Error()).To(ContainSubstring("fake-partition-error"))
								Expect(formatter.FormatCalled).To(BeFalse())
								Expect(mounter.MountCalled).To(BeFalse())
							})

							It("formats swap and data partitions", func() {
								err := act()
								Expect(err).NotTo(HaveOccurred())

								Expect(len(formatter.FormatPartitionPaths)).To(Equal(2))
								Expect(formatter.FormatPartitionPaths[0]).To(Equal("/dev/vda2"))
								Expect(formatter.FormatPartitionPaths[1]).To(Equal("/dev/vda3"))

								Expect(len(formatter.FormatFsTypes)).To(Equal(2))
								Expect(formatter.FormatFsTypes[0]).To(Equal(boshdisk.FileSystemSwap))
								Expect(formatter.FormatFsTypes[1]).To(Equal(boshdisk.FileSystemExt4))
							})

							It("mounts swap and data partitions", func() {
								err := act()
								Expect(err).NotTo(HaveOccurred())

								Expect(len(mounter.MountMountPoints)).To(Equal(1))
								Expect(mounter.MountMountPoints[0]).To(Equal("/fake-dir/data"))
								Expect(len(mounter.MountPartitionPaths)).To(Equal(1))
								Expect(mounter.MountPartitionPaths[0]).To(Equal("/dev/vda3"))

								Expect(len(mounter.SwapOnPartitionPaths)).To(Equal(1))
								Expect(mounter.SwapOnPartitionPaths[0]).To(Equal("/dev/vda2"))
							})

							It("creates swap the size of the memory and the rest for data when disk is bigger than twice the memory", func() {
								memSizeInBytes := uint64(1024 * 1024 * 1024)
								diskSizeInBytes := 2*memSizeInBytes + 64
								partitioner.GetDeviceSizeInBytesSizes["/dev/vda"] = diskSizeInBytes
								collector.MemStats.Total = memSizeInBytes

								err := act()
								Expect(err).ToNot(HaveOccurred())
								Expect(partitioner.PartitionDevicePath).To(Equal("/dev/vda"))
								Expect(partitioner.PartitionPartitions).To(ContainElement(
									boshdisk.Partition{
										SizeInBytes: memSizeInBytes,
										Type:        boshdisk.PartitionTypeSwap,
									}),
								)
								Expect(partitioner.PartitionPartitions).To(ContainElement(
									boshdisk.Partition{
										SizeInBytes: diskSizeInBytes - memSizeInBytes,
										Type:        boshdisk.PartitionTypeLinux,
									}),
								)
							})

							It("creates equal swap and data partitions when disk is twice the memory or smaller", func() {
								memSizeInBytes := uint64(1024 * 1024 * 1024)
								diskSizeInBytes := 2*memSizeInBytes - 64
								partitioner.GetDeviceSizeInBytesSizes["/dev/vda"] = diskSizeInBytes
								collector.MemStats.Total = memSizeInBytes

								err := act()
								Expect(err).ToNot(HaveOccurred())
								Expect(partitioner.PartitionDevicePath).To(Equal("/dev/vda"))
								Expect(partitioner.PartitionPartitions).To(ContainElement(
									boshdisk.Partition{
										SizeInBytes: diskSizeInBytes / 2,
										Type:        boshdisk.PartitionTypeSwap,
									}),
								)
								Expect(partitioner.PartitionPartitions).To(ContainElement(
									boshdisk.Partition{
										SizeInBytes: diskSizeInBytes / 2,
										Type:        boshdisk.PartitionTypeLinux,
									}),
								)
							})
						})

						Context("when getting root device remaining size fails", func() {
							BeforeEach(func() {
								partitioner.GetDeviceSizeInBytesErr = errors.New("fake-get-remaining-size-error")
							})

							It("returns an error", func() {
								err := act()
								Expect(err).To(HaveOccurred())
								Expect(err.Error()).To(ContainSubstring("Getting root device remaining size"))
								Expect(err.Error()).To(ContainSubstring("fake-get-remaining-size-error"))
								Expect(partitioner.PartitionCalled).To(BeFalse())
								Expect(formatter.FormatCalled).To(BeFalse())
								Expect(mounter.MountCalled).To(BeFalse())
							})
						})
					})
				})

				Context("when root device is determined and root partition is not the first one", func() {
					BeforeEach(func() {
						diskManager.FakeMountsSearcher.SearchMountsMounts = []boshdisk.Mount{
							{MountPoint: "/boot", PartitionPath: "/dev/vda1"},
							{MountPoint: "/", PartitionPath: "rootfs"},
							{MountPoint: "/", PartitionPath: "/dev/vda2"},
						}
					})

					Context("when getting absolute path suceeds", func() {
						BeforeEach(func() {
							cmdRunner.AddCmdResult(
								"readlink -f /dev/vda2",
								fakesys.FakeCmdResult{Stdout: "/dev/vda2"},
							)
						})

						Context("when root device has sufficient space for ephemeral partitions", func() {
							BeforeEach(func() {
								partitioner.GetDeviceSizeInBytesSizes["/dev/vda"] = 1024 * 1024 * 1024
								collector.MemStats.Total = 256 * 1024 * 1024
							})

							itSetsUpEphemeralDisk(act)

							It("formats swap and data partitions", func() {
								err := act()
								Expect(err).NotTo(HaveOccurred())

								Expect(len(formatter.FormatPartitionPaths)).To(Equal(2))
								Expect(formatter.FormatPartitionPaths[0]).To(Equal("/dev/vda3"))
								Expect(formatter.FormatPartitionPaths[1]).To(Equal("/dev/vda4"))

								Expect(len(formatter.FormatFsTypes)).To(Equal(2))
								Expect(formatter.FormatFsTypes[0]).To(Equal(boshdisk.FileSystemSwap))
								Expect(formatter.FormatFsTypes[1]).To(Equal(boshdisk.FileSystemExt4))
							})

							It("mounts swap and data partitions", func() {
								err := act()
								Expect(err).NotTo(HaveOccurred())

								Expect(len(mounter.MountMountPoints)).To(Equal(1))
								Expect(mounter.MountMountPoints[0]).To(Equal("/fake-dir/data"))
								Expect(len(mounter.MountPartitionPaths)).To(Equal(1))
								Expect(mounter.MountPartitionPaths[0]).To(Equal("/dev/vda4"))

								Expect(len(mounter.SwapOnPartitionPaths)).To(Equal(1))
								Expect(mounter.SwapOnPartitionPaths[0]).To(Equal("/dev/vda3"))
							})

							It("creates swap the size of the memory and the rest for data when disk is bigger than twice the memory", func() {
								memSizeInBytes := uint64(1024 * 1024 * 1024)
								diskSizeInBytes := 2*memSizeInBytes + 64
								partitioner.GetDeviceSizeInBytesSizes["/dev/vda"] = diskSizeInBytes
								collector.MemStats.Total = memSizeInBytes

								err := act()
								Expect(err).ToNot(HaveOccurred())
								Expect(partitioner.PartitionDevicePath).To(Equal("/dev/vda"))
								Expect(partitioner.PartitionPartitions).To(ContainElement(
									boshdisk.Partition{
										SizeInBytes: memSizeInBytes,
										Type:        boshdisk.PartitionTypeSwap,
									}),
								)
								Expect(partitioner.PartitionPartitions).To(ContainElement(
									boshdisk.Partition{
										SizeInBytes: diskSizeInBytes - memSizeInBytes,
										Type:        boshdisk.PartitionTypeLinux,
									}),
								)
							})

							It("creates equal swap and data partitions when disk is twice the memory or smaller", func() {
								memSizeInBytes := uint64(1024 * 1024 * 1024)
								diskSizeInBytes := 2*memSizeInBytes - 64
								partitioner.GetDeviceSizeInBytesSizes["/dev/vda"] = diskSizeInBytes
								collector.MemStats.Total = memSizeInBytes

								err := act()
								Expect(err).ToNot(HaveOccurred())
								Expect(partitioner.PartitionDevicePath).To(Equal("/dev/vda"))
								Expect(partitioner.PartitionPartitions).To(ContainElement(
									boshdisk.Partition{
										SizeInBytes: diskSizeInBytes / 2,
										Type:        boshdisk.PartitionTypeSwap,
									}),
								)
								Expect(partitioner.PartitionPartitions).To(ContainElement(
									boshdisk.Partition{
										SizeInBytes: diskSizeInBytes / 2,
										Type:        boshdisk.PartitionTypeLinux,
									}),
								)
							})
						})
					})
				})

				It("returns error if creating data dir fails", func() {
					fs.MkdirAllError = errors.New("fake-mkdir-all-err")

					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-mkdir-all-err"))
					Expect(partitioner.PartitionCalled).To(BeFalse())
					Expect(formatter.FormatCalled).To(BeFalse())
					Expect(mounter.MountCalled).To(BeFalse())
				})

				It("returns err when the data directory cannot be globbed", func() {
					fs.GlobErr = errors.New("fake-glob-err")

					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Globbing ephemeral disk mount point `/fake-dir/data/*'"))
					Expect(err.Error()).To(ContainSubstring("fake-glob-err"))
					Expect(partitioner.PartitionCalled).To(BeFalse())
					Expect(formatter.FormatCalled).To(BeFalse())
					Expect(mounter.MountCalled).To(BeFalse())
				})
			})

			Context("when agent should not partition ephemeral disk on root disk", func() {
				BeforeEach(func() {
					options.CreatePartitionIfNoEphemeralDisk = false
				})

				It("returns an error", func() {
					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("cannot use root partition as ephemeral disk"))
				})

				It("does not try to partition anything", func() {
					err := act()
					Expect(err).To(HaveOccurred())
					Expect(partitioner.PartitionCalled).To(BeFalse())
				})

				It("does not try to format anything", func() {
					err := act()
					Expect(err).To(HaveOccurred())
					Expect(formatter.FormatCalled).To(BeFalse())
				})

				It("does not try to mount anything", func() {
					err := act()
					Expect(err).To(HaveOccurred())
					Expect(mounter.MountCalled).To(BeFalse())
				})
			})
		})

		Context("when SkipDiskSetup is true", func() {
			BeforeEach(func() {
				options.SkipDiskSetup = true
			})

			It("does nothing", func() {
				err := platform.SetupEphemeralDiskWithPath("/dev/xvda")

				Expect(err).ToNot(HaveOccurred())
				Expect(partitioner.PartitionCalled).To(BeFalse())
				Expect(formatter.FormatCalled).To(BeFalse())
				Expect(mounter.MountCalled).To(BeFalse())
			})
		})
	})

	Describe("SetupRawEphemeralDisks", func() {
		It("labels the raw ephemeral paths for unpartitioned disks", func() {
			result := fakesys.FakeCmdResult{
				Error:      nil,
				ExitStatus: 0,
				Stderr:     "",
				Stdout: `Model: Xen Virtual Block Device (xvd)
Disk /dev/xvdb: 40.3GB
Sector size (logical/physical): 512B/512B
Partition Table: loop

Number  Start  End     Size    File system  Flags
1      0.00B  40.3GB  40.3GB  ext3
`,
			}

			cmdRunner.AddCmdResult("parted -s /dev/xvdb p", result)

			result = fakesys.FakeCmdResult{
				Error:      nil,
				ExitStatus: 0,
				Stderr:     "",
				Stdout: `Model: Xen Virtual Block Device (xvd)
Disk /dev/xvdc: 40.3GB
Sector size (logical/physical): 512B/512B
Partition Table: loop

Number  Start  End     Size    File system  Flags
1      0.00B  40.3GB  40.3GB  ext3
`,
			}

			cmdRunner.AddCmdResult("parted -s /dev/xvdc p", result)

			devicePathResolver.GetRealDevicePathStub = func(diskSettings boshsettings.DiskSettings) (string, bool, error) {
				return diskSettings.Path, false, nil
			}

			err := platform.SetupRawEphemeralDisks([]boshsettings.DiskSettings{{Path: "/dev/xvdb"}, {Path: "/dev/xvdc"}})

			Expect(err).ToNot(HaveOccurred())
			Expect(len(cmdRunner.RunCommands)).To(Equal(4))
			Expect(cmdRunner.RunCommands[0]).To(Equal([]string{"parted", "-s", "/dev/xvdb", "p"}))
			Expect(cmdRunner.RunCommands[1]).To(Equal([]string{"parted", "-s", "/dev/xvdb", "mklabel", "gpt", "unit", "%", "mkpart", "raw-ephemeral-0", "0", "100"}))
			Expect(cmdRunner.RunCommands[2]).To(Equal([]string{"parted", "-s", "/dev/xvdc", "p"}))
			Expect(cmdRunner.RunCommands[3]).To(Equal([]string{"parted", "-s", "/dev/xvdc", "mklabel", "gpt", "unit", "%", "mkpart", "raw-ephemeral-1", "0", "100"}))
		})

		It("does not label the raw ephemeral paths for already partitioned disks", func() {
			result := fakesys.FakeCmdResult{
				Error:      nil,
				ExitStatus: 0,
				Stderr:     "",
				Stdout: `Model: Xen Virtual Block Device (xvd)
Disk /dev/xvdb: 40.3GB
Sector size (logical/physical): 512B/512B
Partition Table: gpt

Number  Start   End     Size    File system  Name             Flags
 1      1049kB  40.3GB  40.3GB               raw-ephemeral-0
`,
			}

			cmdRunner.AddCmdResult("parted -s /dev/xvdb p", result)

			result = fakesys.FakeCmdResult{
				Error:      nil,
				ExitStatus: 0,
				Stderr:     "",
				Stdout: `Model: Xen Virtual Block Device (xvd)
Disk /dev/xvdc: 40.3GB
Sector size (logical/physical): 512B/512B
Partition Table: gpt

Number  Start   End     Size    File system  Name             Flags
 1      1049kB  40.3GB  40.3GB               raw-ephemeral-1
`,
			}

			cmdRunner.AddCmdResult("parted -s /dev/xvdc p", result)

			devicePathResolver.GetRealDevicePathStub = func(diskSettings boshsettings.DiskSettings) (string, bool, error) {
				return diskSettings.Path, false, nil
			}

			err := platform.SetupRawEphemeralDisks([]boshsettings.DiskSettings{{Path: "/dev/xvdb"}, {Path: "/dev/xvdc"}})

			Expect(err).ToNot(HaveOccurred())
			Expect(len(cmdRunner.RunCommands)).To(Equal(2))
			Expect(cmdRunner.RunCommands[0]).To(Equal([]string{"parted", "-s", "/dev/xvdb", "p"}))
			Expect(cmdRunner.RunCommands[1]).To(Equal([]string{"parted", "-s", "/dev/xvdc", "p"}))
		})

		It("does not give an error if parted prints 'unrecognised disk label' to stdout and returns an error", func() {
			result := fakesys.FakeCmdResult{
				Error:      errors.New("fake-parted-error"),
				ExitStatus: 0,
				Stderr:     "",
				Stdout: `Model: Xen Virtual Block Device (xvd)
Error: /dev/xvda: unrecognised disk label
Disk /dev/xvda: 40.3GB
Sector size (logical/physical): 512B/512B
Partition Table: gpt

Number  Start   End     Size    File system  Name             Flags
 1      1049kB  40.3GB  40.3GB               raw-ephemeral-0
`,
			}

			cmdRunner.AddCmdResult("parted -s /dev/xvda p", result)

			devicePathResolver.GetRealDevicePathStub = func(diskSettings boshsettings.DiskSettings) (string, bool, error) {
				return diskSettings.Path, false, nil
			}

			err := platform.SetupRawEphemeralDisks([]boshsettings.DiskSettings{{Path: "/dev/xvda"}})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(cmdRunner.RunCommands)).To(Equal(1))
			Expect(cmdRunner.RunCommands[0]).To(Equal([]string{"parted", "-s", "/dev/xvda", "p"}))
		})

		It("does not give an error if parted prints 'unrecognised disk label' to stderr and returns an error", func() {
			result := fakesys.FakeCmdResult{
				Error:      errors.New("fake-parted-error"),
				ExitStatus: 0,
				Stderr:     "Error: /dev/xvda: unrecognised disk label",
				Stdout: `Model: Xen Virtual Block Device (xvd)
Disk /dev/xvda: 40.3GB
Sector size (logical/physical): 512B/512B
Partition Table: gpt

Number  Start   End     Size    File system  Name             Flags
 1      1049kB  40.3GB  40.3GB               raw-ephemeral-0
`,
			}

			cmdRunner.AddCmdResult("parted -s /dev/xvda p", result)

			devicePathResolver.GetRealDevicePathStub = func(diskSettings boshsettings.DiskSettings) (string, bool, error) {
				return diskSettings.Path, false, nil
			}

			err := platform.SetupRawEphemeralDisks([]boshsettings.DiskSettings{{Path: "/dev/xvda"}})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(cmdRunner.RunCommands)).To(Equal(1))
			Expect(cmdRunner.RunCommands[0]).To(Equal([]string{"parted", "-s", "/dev/xvda", "p"}))
		})

		Context("when SkipDiskSetup is true", func() {
			BeforeEach(func() {
				options.SkipDiskSetup = true
			})

			It("does nothing", func() {
				err := platform.SetupRawEphemeralDisks([]boshsettings.DiskSettings{{Path: "/dev/xvdb"}, {Path: "/dev/xvdc"}})

				Expect(err).ToNot(HaveOccurred())
				Expect(len(cmdRunner.RunCommands)).To(Equal(0))
			})
		})
	})

	Describe("SetupDataDir", func() {
		var mounter *fakedisk.FakeMounter
		BeforeEach(func() {
			mounter = diskManager.FakeMounter
		})

		Context("when sys/run is already mounted", func() {
			BeforeEach(func() {
				mounter.IsMountPointResult = true
			})

			It("creates sys/log directory in data directory", func() {
				err := platform.SetupDataDir()
				Expect(err).NotTo(HaveOccurred())

				sysLogStats := fs.GetFileTestStat("/fake-dir/data/sys/log")
				Expect(sysLogStats).ToNot(BeNil())
				Expect(sysLogStats.FileType).To(Equal(fakesys.FakeFileTypeDir))
				Expect(sysLogStats.FileMode).To(Equal(os.FileMode(0750)))
				Expect(cmdRunner.RunCommands[0]).To(Equal([]string{"chown", "root:vcap", "/fake-dir/data/sys"}))
				Expect(cmdRunner.RunCommands[1]).To(Equal([]string{"chown", "root:vcap", "/fake-dir/data/sys/log"}))
			})

			It("creates symlink from sys to data/sys", func() {
				err := platform.SetupDataDir()
				Expect(err).NotTo(HaveOccurred())

				sysStats := fs.GetFileTestStat("/fake-dir/sys")
				Expect(sysStats).ToNot(BeNil())
				Expect(sysStats.FileType).To(Equal(fakesys.FakeFileTypeSymlink))
				Expect(sysStats.SymlinkTarget).To(Equal("/fake-dir/data/sys"))
			})

			It("does not create new sys/run dir", func() {
				err := platform.SetupDataDir()
				Expect(err).NotTo(HaveOccurred())

				sysRunStats := fs.GetFileTestStat("/fake-dir/data/sys/run")
				Expect(sysRunStats).To(BeNil())
			})

			It("does not mount tmpfs again", func() {
				err := platform.SetupDataDir()
				Expect(err).NotTo(HaveOccurred())
				Expect(len(mounter.MountPartitionPaths)).To(Equal(0))
			})
		})

		Context("when sys/run is not yet mounted", func() {
			BeforeEach(func() {
				mounter.IsMountPointResult = false
			})

			It("creates sys/log directory in data directory", func() {
				err := platform.SetupDataDir()
				Expect(err).NotTo(HaveOccurred())

				sysLogStats := fs.GetFileTestStat("/fake-dir/data/sys/log")
				Expect(sysLogStats).ToNot(BeNil())
				Expect(sysLogStats.FileType).To(Equal(fakesys.FakeFileTypeDir))
				Expect(sysLogStats.FileMode).To(Equal(os.FileMode(0750)))
				Expect(cmdRunner.RunCommands[0]).To(Equal([]string{"chown", "root:vcap", "/fake-dir/data/sys"}))
				Expect(cmdRunner.RunCommands[1]).To(Equal([]string{"chown", "root:vcap", "/fake-dir/data/sys/log"}))
			})

			It("creates symlink from sys to data/sys", func() {
				err := platform.SetupDataDir()
				Expect(err).NotTo(HaveOccurred())

				sysStats := fs.GetFileTestStat("/fake-dir/sys")
				Expect(sysStats).ToNot(BeNil())
				Expect(sysStats.FileType).To(Equal(fakesys.FakeFileTypeSymlink))
				Expect(sysStats.SymlinkTarget).To(Equal("/fake-dir/data/sys"))
			})

			It("creates new sys/run dir", func() {
				err := platform.SetupDataDir()
				Expect(err).NotTo(HaveOccurred())

				sysRunStats := fs.GetFileTestStat("/fake-dir/data/sys/run")
				Expect(sysRunStats).ToNot(BeNil())
				Expect(sysRunStats.FileType).To(Equal(fakesys.FakeFileTypeDir))
				Expect(sysRunStats.FileMode).To(Equal(os.FileMode(0750)))
				Expect(cmdRunner.RunCommands[2]).To(Equal([]string{"chown", "root:vcap", "/fake-dir/data/sys/run"}))
			})

			It("mounts tmpfs to sys/run", func() {
				err := platform.SetupDataDir()
				Expect(err).NotTo(HaveOccurred())

				Expect(len(mounter.MountPartitionPaths)).To(Equal(1))
				Expect(mounter.MountPartitionPaths[0]).To(Equal("tmpfs"))
				Expect(mounter.MountMountPoints[0]).To(Equal("/fake-dir/data/sys/run"))
				Expect(mounter.MountMountOptions[0]).To(Equal([]string{"-t", "tmpfs", "-o", "size=1m"}))
			})

			It("returns an error if creation of mount point fails", func() {
				fs.MkdirAllError = errors.New("fake-mkdir-error")

				err := platform.SetupDataDir()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-mkdir-error"))
			})

			It("returns an error if mounting tmpfs fails", func() {
				mounter.MountErr = errors.New("fake-mount-error")

				err := platform.SetupDataDir()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-mount-error"))
			})
		})
	})

	Describe("SetupTmpDir", func() {
		act := func() error { return platform.SetupTmpDir() }

		var mounter *fakedisk.FakeMounter
		BeforeEach(func() {
			mounter = diskManager.FakeMounter
		})

		It("changes permissions on /tmp", func() {
			err := act()
			Expect(err).NotTo(HaveOccurred())

			Expect(cmdRunner.RunCommands[0]).To(Equal([]string{"chown", "root:vcap", "/tmp"}))
			Expect(cmdRunner.RunCommands[1]).To(Equal([]string{"chmod", "0770", "/tmp"}))
			Expect(cmdRunner.RunCommands[2]).To(Equal([]string{"chmod", "0700", "/var/tmp"}))
		})

		It("creates new temp dir", func() {
			err := act()
			Expect(err).NotTo(HaveOccurred())

			fileStats := fs.GetFileTestStat("/fake-dir/data/tmp")
			Expect(fileStats).NotTo(BeNil())
			Expect(fileStats.FileType).To(Equal(fakesys.FakeFileType(fakesys.FakeFileTypeDir)))
			Expect(fileStats.FileMode).To(Equal(os.FileMode(0755)))
		})

		It("returns error if creating new temp dir errs", func() {
			fs.MkdirAllError = errors.New("fake-mkdir-error")

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-mkdir-error"))
		})

		It("sets TMPDIR environment variable so that children of this process will use new temp dir", func() {
			err := act()
			Expect(err).NotTo(HaveOccurred())
			Expect(os.Getenv("TMPDIR")).To(Equal("/fake-dir/data/tmp"))
		})

		It("returns error if setting TMPDIR errs", func() {
			// uses os package; no way to trigger err
		})

		ItDoesNotTryToUseLoopDevice := func() {
			It("does not create new tmp filesystem", func() {
				act()
				for _, cmd := range cmdRunner.RunCommands {
					Expect(cmd[0]).ToNot(Equal("truncate"))
					Expect(cmd[0]).ToNot(Equal("mke2fs"))
				}
			})

			It("does not try to mount anything /tmp", func() {
				act()
				Expect(len(mounter.MountPartitionPaths)).To(Equal(0))
			})
		}

		Context("when UseDefaultTmpDir option is set to false", func() {
			BeforeEach(func() {
				options.UseDefaultTmpDir = false
			})

			Context("when /tmp is not a mount point", func() {
				BeforeEach(func() {
					mounter.IsMountPointResult = false
				})

				It("creates a root_tmp folder", func() {
					err := act()
					Expect(err).NotTo(HaveOccurred())
					Expect(cmdRunner.RunCommands[3]).To(Equal([]string{"mkdir", "-p", "/fake-dir/data/root_tmp"}))
				})

				It("changes permissions on the new bind mount folder", func() {
					err := act()
					Expect(err).NotTo(HaveOccurred())

					Expect(cmdRunner.RunCommands[4]).To(Equal([]string{"chmod", "0700", "/fake-dir/data/root_tmp"}))
				})

				It("bind mounts it in /tmp", func() {
					err := act()
					Expect(err).NotTo(HaveOccurred())

					Expect(len(mounter.MountPartitionPaths)).To(Equal(1))
					Expect(mounter.MountPartitionPaths[0]).To(Equal("/fake-dir/data/root_tmp"))
					Expect(mounter.MountMountOptions[0]).To(ConsistOf("-o", "nodev", "-o", "noexec", "-o", "nosuid", "--bind"))
				})

				It("changes permissions for the system /tmp folder", func() {
					err := act()
					Expect(err).NotTo(HaveOccurred())

					Expect(cmdRunner.RunCommands[5]).To(Equal([]string{"chown", "root:vcap", "/tmp"}))
				})
			})

			Context("when /tmp is a mount point", func() {
				BeforeEach(func() {
					mounter.IsMountedResult = true
				})

				It("returns without an error", func() {
					err := act()
					Expect(mounter.IsMountedDevicePathOrMountPoint).To(Equal("/tmp"))
					Expect(err).ToNot(HaveOccurred())
				})

				ItDoesNotTryToUseLoopDevice()
			})

			Context("when /tmp cannot be determined if it is a mount point", func() {
				BeforeEach(func() {
					mounter.IsMountedErr = errors.New("fake-is-mounted-error")
				})

				It("returns error", func() {
					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-is-mounted-error"))
				})

				ItDoesNotTryToUseLoopDevice()
			})
		})

		Context("when UseDefaultTmpDir option is set to true", func() {
			BeforeEach(func() {
				options.UseDefaultTmpDir = true
			})

			It("returns without an error", func() {
				err := act()
				Expect(err).ToNot(HaveOccurred())
			})

			ItDoesNotTryToUseLoopDevice()
		})
	})

	Describe("MountPersistentDisk", func() {
		act := func() error {
			return platform.MountPersistentDisk(
				boshsettings.DiskSettings{Path: "fake-volume-id"},
				"/mnt/point",
			)
		}

		var (
			partitioner *fakedisk.FakePartitioner
			formatter   *fakedisk.FakeFormatter
			mounter     *fakedisk.FakeMounter
		)
		BeforeEach(func() {
			partitioner = diskManager.FakePartitioner
			formatter = diskManager.FakeFormatter
			mounter = diskManager.FakeMounter
		})

		Context("when the size of the disk is larger than or equals 2 Terrabytes", func() {

			BeforeEach(func() {
				diskManager.FakeDiskUtil.GetBlockDeviceSizeSize = uint64(2199023255552)
			})

			It("uses parted partitioner", func() {
				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(diskManager.PartedPartitionerCalled).To(BeTrue())
				Expect(diskManager.PartitionerCalled).To(BeFalse())
			})
		})

		Context("when the size of the disk is less than 2 Terabytes", func() {

			BeforeEach(func() {
				diskManager.FakeDiskUtil.GetBlockDeviceSizeSize = uint64(2199023255551)
			})

			It("uses fdisk partitioner", func() {
				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(diskManager.PartitionerCalled).To(BeTrue())
				Expect(diskManager.PartedPartitionerCalled).To(BeFalse())
			})
		})

		Context("when the lsblk command returns an error", func() {

			BeforeEach(func() {
				diskManager.FakeDiskUtil.GetBlockDeviceSizeSize = uint64(3199023255556)
				diskManager.FakeDiskUtil.GetBlockDeviceSizeError = errors.New("Some error")
			})

			It("uses fdisk partitioner", func() {
				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(diskManager.PartitionerCalled).To(BeTrue())
				Expect(diskManager.PartedPartitionerCalled).To(BeFalse())
			})
		})

		Context("when device real path contains /dev/mapper/ and is successfully resolved", func() {
			BeforeEach(func() {
				devicePathResolver.RealDevicePath = "/dev/mapper/fake-real-device-path"
			})

			Context("when store directory is already mounted", func() {
				BeforeEach(func() {
					mounter.IsMountPointResult = true
				})

				Context("when mounting the same device", func() {
					BeforeEach(func() {
						mounter.IsMountPointPartitionPath = "/dev/mapper/fake-real-device-path-part1"
					})

					It("skips mounting", func() {
						err := act()
						Expect(err).ToNot(HaveOccurred())
						Expect(mounter.MountCalled).To(BeFalse())
					})
				})

				Context("when mounting a different device", func() {
					BeforeEach(func() {
						mounter.IsMountPointPartitionPath = "/dev/mapper/another-device"
					})

					It("mounts the store migration directory", func() {
						err := act()
						Expect(err).ToNot(HaveOccurred())
						Expect(fs.GetFileTestStat("/fake-dir/store_migration_target").FileType).To(Equal(fakesys.FakeFileTypeDir))
						Expect(mounter.MountPartitionPaths).To(Equal([]string{"/dev/mapper/fake-real-device-path-part1"}))
						Expect(mounter.MountMountPoints).To(Equal([]string{"/fake-dir/store_migration_target"}))
						Expect(mounter.MountMountOptions).To(Equal([][]string{nil}))
					})
				})
			})

			Context("when failing to determine if store directory is mounted", func() {
				BeforeEach(func() {
					mounter.IsMountPointErr = errors.New("fake-is-mount-point-err")
				})

				It("returns an error", func() {
					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-is-mount-point-err"))
					Expect(mounter.MountCalled).To(BeFalse())
				})
			})

			Context("when UsePreformattedPersistentDisk set to false", func() {
				It("creates the mount directory with the correct permissions", func() {
					err := act()
					Expect(err).ToNot(HaveOccurred())

					mountPoint := fs.GetFileTestStat("/mnt/point")
					Expect(mountPoint.FileType).To(Equal(fakesys.FakeFileTypeDir))
					Expect(mountPoint.FileMode).To(Equal(os.FileMode(0700)))
				})

				It("returns error when creating mount directory fails", func() {
					fs.MkdirAllError = errors.New("fake-mkdir-all-err")

					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-mkdir-all-err"))
				})

				It("partitions the disk", func() {
					err := act()
					Expect(err).ToNot(HaveOccurred())

					partitions := []boshdisk.Partition{{Type: boshdisk.PartitionTypeLinux}}
					Expect(partitioner.PartitionDevicePath).To(Equal("/dev/mapper/fake-real-device-path"))
					Expect(partitioner.PartitionPartitions).To(Equal(partitions))
				})

				It("formats the disk", func() {
					err := act()
					Expect(err).ToNot(HaveOccurred())
					Expect(formatter.FormatPartitionPaths).To(Equal([]string{"/dev/mapper/fake-real-device-path-part1"}))
					Expect(formatter.FormatFsTypes).To(Equal([]boshdisk.FileSystemType{boshdisk.FileSystemExt4}))
				})

				It("mounts the disk", func() {
					err := act()
					Expect(err).ToNot(HaveOccurred())
					Expect(mounter.MountPartitionPaths).To(Equal([]string{"/dev/mapper/fake-real-device-path-part1"}))
					Expect(mounter.MountMountPoints).To(Equal([]string{"/mnt/point"}))
					Expect(mounter.MountMountOptions).To(Equal([][]string{nil}))
				})
			})
		})

		Context("when device path is successfully resolved", func() {
			BeforeEach(func() {
				devicePathResolver.RealDevicePath = "fake-real-device-path"
			})

			Context("when UsePreformattedPersistentDisk set to false", func() {
				It("creates the mount directory with the correct permissions", func() {
					err := act()
					Expect(err).ToNot(HaveOccurred())

					mountPoint := fs.GetFileTestStat("/mnt/point")
					Expect(mountPoint.FileType).To(Equal(fakesys.FakeFileTypeDir))
					Expect(mountPoint.FileMode).To(Equal(os.FileMode(0700)))
				})

				It("returns error when creating mount directory fails", func() {
					fs.MkdirAllError = errors.New("fake-mkdir-all-err")

					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-mkdir-all-err"))
				})

				It("partitions the disk", func() {
					err := act()
					Expect(err).ToNot(HaveOccurred())

					partitions := []boshdisk.Partition{{Type: boshdisk.PartitionTypeLinux}}
					Expect(partitioner.PartitionDevicePath).To(Equal("fake-real-device-path"))
					Expect(partitioner.PartitionPartitions).To(Equal(partitions))
				})

				Context("when settings do NOT specify persistentDiskFS", func() {
					It("formats in ext4 format", func() {
						err := act()
						Expect(err).ToNot(HaveOccurred())
						Expect(formatter.FormatPartitionPaths).To(Equal([]string{"fake-real-device-path1"}))
						Expect(formatter.FormatFsTypes).To(Equal([]boshdisk.FileSystemType{boshdisk.FileSystemExt4}))
					})
				})

				Context("when settings specify persistentDiskFS", func() {
					Context("with ext4", func() {
						It("formats in using the given format", func() {
							err := platform.MountPersistentDisk(
								boshsettings.DiskSettings{Path: "fake-volume-id", FileSystemType: boshdisk.FileSystemExt4},
								"/mnt/point",
							)

							Expect(err).ToNot(HaveOccurred())
							Expect(formatter.FormatFsTypes).To(Equal([]boshdisk.FileSystemType{boshdisk.FileSystemExt4}))
						})
					})

					Context("with xfs", func() {
						It("formats in using the given format", func() {
							err := platform.MountPersistentDisk(
								boshsettings.DiskSettings{Path: "fake-volume-id", FileSystemType: boshdisk.FileSystemXFS},
								"/mnt/point",
							)

							Expect(err).ToNot(HaveOccurred())
							Expect(formatter.FormatFsTypes).To(Equal([]boshdisk.FileSystemType{boshdisk.FileSystemXFS}))
						})
					})

					Context("with an unsupported type", func() {
						It("it errors", func() {
							err := platform.MountPersistentDisk(
								boshsettings.DiskSettings{Path: "fake-volume-id", FileSystemType: boshdisk.FileSystemType("blahblah")},
								"/mnt/point",
							)

							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(Equal(`The filesystem type "blahblah" is not supported`))
						})
					})
				})

				It("returns an error when disk could not be formatted", func() {
					formatter.FormatError = errors.New("Oh noes!")
					err := platform.MountPersistentDisk(
						boshsettings.DiskSettings{Path: "fake-volume-id", FileSystemType: boshdisk.FileSystemXFS},
						"/mnt/point",
					)

					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("Formatting partition with xfs: Oh noes!"))
				})

				It("mounts the disk", func() {
					err := act()
					Expect(err).ToNot(HaveOccurred())
					Expect(mounter.MountPartitionPaths).To(Equal([]string{"fake-real-device-path1"}))
					Expect(mounter.MountMountPoints).To(Equal([]string{"/mnt/point"}))
					Expect(mounter.MountMountOptions).To(Equal([][]string{nil}))
				})
			})

			Context("when UsePreformattedPersistentDisk set to true", func() {
				BeforeEach(func() {
					options.UsePreformattedPersistentDisk = true
				})

				It("creates the mount directory with the correct permissions", func() {
					err := act()
					Expect(err).ToNot(HaveOccurred())

					mountPoint := fs.GetFileTestStat("/mnt/point")
					Expect(mountPoint.FileType).To(Equal(fakesys.FakeFileTypeDir))
					Expect(mountPoint.FileMode).To(Equal(os.FileMode(0700)))
				})

				It("returns error when creating mount directory fails", func() {
					fs.MkdirAllError = errors.New("fake-mkdir-all-err")

					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-mkdir-all-err"))
				})

				It("mounts volume at mount point", func() {
					err := act()
					Expect(err).ToNot(HaveOccurred())

					Expect(len(mounter.MountPartitionPaths)).To(Equal(1))
					Expect(mounter.MountPartitionPaths).To(Equal([]string{"fake-real-device-path"})) // no '1' because no partition
					Expect(mounter.MountMountPoints).To(Equal([]string{"/mnt/point"}))
					Expect(mounter.MountMountOptions).To(Equal([][]string{nil}))
				})

				It("returns error when mounting fails", func() {
					mounter.MountErr = errors.New("fake-mount-err")

					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-mount-err"))
				})

				It("does not partition the disk", func() {
					err := act()
					Expect(err).ToNot(HaveOccurred())
					Expect(partitioner.PartitionCalled).To(BeFalse())
				})

				It("does not format the disk", func() {
					err := act()
					Expect(err).ToNot(HaveOccurred())
					Expect(formatter.FormatCalled).To(BeFalse())
				})
			})
		})

		Context("when device path is not successfully resolved", func() {
			It("return an error", func() {
				devicePathResolver.GetRealDevicePathErr = errors.New("fake-get-real-device-path-err")

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-get-real-device-path-err"))
			})
		})
	})

	Describe("UnmountPersistentDisk", func() {
		act := func() (bool, error) {
			return platform.UnmountPersistentDisk(boshsettings.DiskSettings{Path: "fake-device-path"})
		}

		var mounter *fakedisk.FakeMounter
		BeforeEach(func() {
			mounter = diskManager.FakeMounter
		})

		Context("when device real path contains /dev/mapper/ and can be resolved", func() {
			BeforeEach(func() {
				devicePathResolver.RealDevicePath = "/dev/mapper/fake-real-device-path"
			})

			ItUnmountsPersistentDisk := func(expectedUnmountMountPoint string) {
				It("returs true without an error if unmounting succeeded", func() {
					mounter.UnmountDidUnmount = true

					didUnmount, err := act()
					Expect(err).NotTo(HaveOccurred())
					Expect(didUnmount).To(BeTrue())
					Expect(mounter.UnmountPartitionPathOrMountPoint).To(Equal(expectedUnmountMountPoint))
				})

				It("returs false without an error if was already unmounted", func() {
					mounter.UnmountDidUnmount = false

					didUnmount, err := act()
					Expect(err).NotTo(HaveOccurred())
					Expect(didUnmount).To(BeFalse())
					Expect(mounter.UnmountPartitionPathOrMountPoint).To(Equal(expectedUnmountMountPoint))
				})

				It("returns error if unmounting fails", func() {
					mounter.UnmountDidUnmount = false
					mounter.UnmountErr = errors.New("fake-unmount-err")

					didUnmount, err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-unmount-err"))
					Expect(didUnmount).To(BeFalse())
					Expect(mounter.UnmountPartitionPathOrMountPoint).To(Equal(expectedUnmountMountPoint))
				})
			}

			Context("UsePreformattedPersistentDisk is set to false", func() {
				ItUnmountsPersistentDisk("/dev/mapper/fake-real-device-path-part1") // note partition '-part1'
			})

		})

		Context("when device path can be resolved", func() {
			BeforeEach(func() {
				devicePathResolver.RealDevicePath = "fake-real-device-path"
			})

			ItUnmountsPersistentDisk := func(expectedUnmountMountPoint string) {
				It("returs true without an error if unmounting succeeded", func() {
					mounter.UnmountDidUnmount = true

					didUnmount, err := act()
					Expect(err).NotTo(HaveOccurred())
					Expect(didUnmount).To(BeTrue())
					Expect(mounter.UnmountPartitionPathOrMountPoint).To(Equal(expectedUnmountMountPoint))
				})

				It("returs false without an error if was already unmounted", func() {
					mounter.UnmountDidUnmount = false

					didUnmount, err := act()
					Expect(err).NotTo(HaveOccurred())
					Expect(didUnmount).To(BeFalse())
					Expect(mounter.UnmountPartitionPathOrMountPoint).To(Equal(expectedUnmountMountPoint))
				})

				It("returns error if unmounting fails", func() {
					mounter.UnmountDidUnmount = false
					mounter.UnmountErr = errors.New("fake-unmount-err")

					didUnmount, err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-unmount-err"))
					Expect(didUnmount).To(BeFalse())
					Expect(mounter.UnmountPartitionPathOrMountPoint).To(Equal(expectedUnmountMountPoint))
				})
			}

			Context("UsePreformattedPersistentDisk is set to false", func() {
				ItUnmountsPersistentDisk("fake-real-device-path1") // note partition '1'
			})

			Context("UsePreformattedPersistentDisk is set to true", func() {
				BeforeEach(func() {
					options.UsePreformattedPersistentDisk = true
				})

				ItUnmountsPersistentDisk("fake-real-device-path") // note no '1'; no partitions
			})
		})

		Context("when device path cannot be resolved", func() {
			BeforeEach(func() {
				devicePathResolver.GetRealDevicePathErr = errors.New("fake-get-real-device-path-err")
				devicePathResolver.GetRealDevicePathTimedOut = false
			})

			It("returns error", func() {
				isMounted, err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-get-real-device-path-err"))
				Expect(isMounted).To(BeFalse())
			})
		})

		Context("when device path cannot be resolved due to timeout", func() {
			BeforeEach(func() {
				devicePathResolver.GetRealDevicePathErr = errors.New("fake-get-real-device-path-err")
				devicePathResolver.GetRealDevicePathTimedOut = true
			})

			It("does not return error", func() {
				isMounted, err := act()
				Expect(err).NotTo(HaveOccurred())
				Expect(isMounted).To(BeFalse())
			})
		})
	})

	Describe("GetFileContentsFromCDROM", func() {
		It("delegates to cdutil", func() {
			cdutil.GetFilesContentsContents = [][]byte{[]byte("fake-contents")}
			filename := "fake-env"
			contents, err := platform.GetFileContentsFromCDROM(filename)
			Expect(err).NotTo(HaveOccurred())
			Expect(cdutil.GetFilesContentsFileNames[0]).To(Equal(filename))
			Expect(contents).To(Equal([]byte("fake-contents")))
		})
	})

	Describe("GetFilesContentsFromDisk", func() {
		It("delegates to diskutil", func() {
			diskManager.FakeDiskUtil.GetFilesContentsContents = [][]byte{
				[]byte("fake-contents-1"),
				[]byte("fake-contents-2"),
			}
			contents, err := platform.GetFilesContentsFromDisk(
				"fake-disk-path",
				[]string{"fake-file-path-1", "fake-file-path-2"},
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(diskManager.DiskUtilDiskPath).To(Equal("fake-disk-path"))
			Expect(diskManager.FakeDiskUtil.GetFilesContentsFileNames).To(Equal(
				[]string{"fake-file-path-1", "fake-file-path-2"},
			))
			Expect(contents).To(Equal([][]byte{
				[]byte("fake-contents-1"),
				[]byte("fake-contents-2"),
			}))
		})
	})

	Describe("GetEphemeralDiskPath", func() {
		Context("when real device path was resolved without an error", func() {
			It("returns real device path and true", func() {
				devicePathResolver.RealDevicePath = "fake-real-device-path"
				realPath := platform.GetEphemeralDiskPath(boshsettings.DiskSettings{Path: "fake-device-path"})
				Expect(realPath).To(Equal("fake-real-device-path"))
			})
		})

		Context("when real device path was not resolved without an error", func() {
			It("returns real device path and true", func() {
				devicePathResolver.GetRealDevicePathErr = errors.New("fake-get-real-device-path-err")
				realPath := platform.GetEphemeralDiskPath(boshsettings.DiskSettings{Path: "fake-device-path"})
				Expect(realPath).To(Equal(""))
			})
		})
	})

	Describe("MigratePersistentDisk", func() {
		var mounter *fakedisk.FakeMounter
		BeforeEach(func() {
			mounter = diskManager.FakeMounter
		})

		It("migrate persistent disk", func() {
			err := platform.MigratePersistentDisk("/from/path", "/to/path")
			Expect(err).ToNot(HaveOccurred())

			Expect(mounter.RemountAsReadonlyPath).To(Equal("/from/path"))

			Expect(len(cmdRunner.RunCommands)).To(Equal(1))
			Expect(cmdRunner.RunCommands[0]).To(Equal([]string{"sh", "-c", "(tar -C /from/path -cf - .) | (tar -C /to/path -xpf -)"}))

			Expect(mounter.UnmountPartitionPathOrMountPoint).To(Equal("/from/path"))
			Expect(mounter.RemountFromMountPoint).To(Equal("/to/path"))
			Expect(mounter.RemountToMountPoint).To(Equal("/from/path"))
		})
	})

	Describe("IsPersistentDiskMounted", func() {
		act := func() (bool, error) {
			return platform.IsPersistentDiskMounted(boshsettings.DiskSettings{Path: "fake-device-path"})
		}

		var mounter *fakedisk.FakeMounter
		BeforeEach(func() {
			mounter = diskManager.FakeMounter
		})

		Context("when device real path contains /dev/mapper/ and can be resolved", func() {
			BeforeEach(func() {
				devicePathResolver.RealDevicePath = "/dev/mapper/fake-real-device-path"
			})

			ItChecksPersistentDiskMountPoint := func(expectedCheckedMountPoint string) {
				Context("when checking persistent disk mount point succeeds", func() {
					It("returns true if mount point exists", func() {
						mounter.IsMountedResult = true

						isMounted, err := act()
						Expect(err).NotTo(HaveOccurred())
						Expect(isMounted).To(BeTrue())
						Expect(mounter.IsMountedDevicePathOrMountPoint).To(Equal(expectedCheckedMountPoint))
					})

					It("returns false if mount point does not exist", func() {
						mounter.IsMountedResult = false

						isMounted, err := act()
						Expect(err).NotTo(HaveOccurred())
						Expect(isMounted).To(BeFalse())
						Expect(mounter.IsMountedDevicePathOrMountPoint).To(Equal(expectedCheckedMountPoint))
					})
				})

				Context("checking persistent disk mount points fails", func() {
					It("returns error", func() {
						mounter.IsMountedResult = false
						mounter.IsMountedErr = errors.New("fake-is-mounted-err")

						isMounted, err := act()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("fake-is-mounted-err"))
						Expect(isMounted).To(BeFalse())
						Expect(mounter.IsMountedDevicePathOrMountPoint).To(Equal(expectedCheckedMountPoint))
					})
				})
			}

			Context("UsePreformattedPersistentDisk is set to false", func() {
				ItChecksPersistentDiskMountPoint("/dev/mapper/fake-real-device-path-part1") // note partition '-part1'
			})
		})

		Context("when device path can be resolved", func() {
			BeforeEach(func() {
				devicePathResolver.RealDevicePath = "fake-real-device-path"
			})

			ItChecksPersistentDiskMountPoint := func(expectedCheckedMountPoint string) {
				Context("when checking persistent disk mount point succeeds", func() {
					It("returns true if mount point exists", func() {
						mounter.IsMountedResult = true

						isMounted, err := act()
						Expect(err).NotTo(HaveOccurred())
						Expect(isMounted).To(BeTrue())
						Expect(mounter.IsMountedDevicePathOrMountPoint).To(Equal(expectedCheckedMountPoint))
					})

					It("returns false if mount point does not exist", func() {
						mounter.IsMountedResult = false

						isMounted, err := act()
						Expect(err).NotTo(HaveOccurred())
						Expect(isMounted).To(BeFalse())
						Expect(mounter.IsMountedDevicePathOrMountPoint).To(Equal(expectedCheckedMountPoint))
					})
				})

				Context("checking persistent disk mount points fails", func() {
					It("returns error", func() {
						mounter.IsMountedResult = false
						mounter.IsMountedErr = errors.New("fake-is-mounted-err")

						isMounted, err := act()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("fake-is-mounted-err"))
						Expect(isMounted).To(BeFalse())
						Expect(mounter.IsMountedDevicePathOrMountPoint).To(Equal(expectedCheckedMountPoint))
					})
				})
			}

			Context("UsePreformattedPersistentDisk is set to false", func() {
				ItChecksPersistentDiskMountPoint("fake-real-device-path1") // note partition '1'
			})

			Context("UsePreformattedPersistentDisk is set to true", func() {
				BeforeEach(func() {
					options.UsePreformattedPersistentDisk = true
				})

				ItChecksPersistentDiskMountPoint("fake-real-device-path") // note no '1'; no partitions
			})
		})

		Context("when device path cannot be resolved", func() {
			BeforeEach(func() {
				devicePathResolver.GetRealDevicePathErr = errors.New("fake-get-real-device-path-err")
				devicePathResolver.GetRealDevicePathTimedOut = false
			})

			It("returns error", func() {
				isMounted, err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-get-real-device-path-err"))
				Expect(isMounted).To(BeFalse())
			})
		})

		Context("when device path cannot be resolved due to timeout", func() {
			BeforeEach(func() {
				devicePathResolver.GetRealDevicePathErr = errors.New("fake-get-real-device-path-err")
				devicePathResolver.GetRealDevicePathTimedOut = true
			})

			It("does not return error", func() {
				isMounted, err := act()
				Expect(err).NotTo(HaveOccurred())
				Expect(isMounted).To(BeFalse())
			})
		})
	})

	Describe("IsPersistentDiskMountable", func() {
		BeforeEach(func() {
			devicePathResolver.RealDevicePath = "/fake/device"
		})

		Context("when the specified drive does not exist", func() {
			It("returns error", func() {
				devicePathResolver.GetRealDevicePathTimedOut = true
				devicePathResolver.GetRealDevicePathErr = errors.New("fake-timeout-error")
				diskSettings := boshsettings.DiskSettings{
					Path: "/fake/device",
				}

				isMounted, err := platform.IsPersistentDiskMountable(diskSettings)
				Expect(err).To(HaveOccurred())
				Expect(isMounted).To(Equal(false))
			})
		})

		Context("when there is no partition on drive", func() {
			It("returns false", func() {
				result := fakesys.FakeCmdResult{
					Error:      nil,
					ExitStatus: 0,
					Stderr: `
dfdisk: ERROR: sector 0 does not have an msdos signature
/fake/device: unrecognized partition table type
No partitions found
`,
					Stdout: "",
				}

				cmdRunner.AddCmdResult("sfdisk -d /fake/device", result)

				diskSettings := boshsettings.DiskSettings{
					Path: "/fake/device",
				}

				isMounted, err := platform.IsPersistentDiskMountable(diskSettings)
				Expect(err).ToNot(HaveOccurred())
				Expect(isMounted).To(Equal(false))
			})
		})

		Context("when drive is partitioned", func() {
			It("returns true", func() {
				result := fakesys.FakeCmdResult{
					Error:      nil,
					ExitStatus: 0,
					Stderr:     "",
					Stdout: `# partition table of /fake/device
unit: sectors

/fake/device1 : start=       63, size=  5997984, Id=83
/fake/device2 : start=  5998592, size= 32691088, Id=83
/fake/device3 : start= 38690816, size=195750832, Id=83
/fake/device4 : start=        0, size=        0, Id= 0
`,
				}

				cmdRunner.AddCmdResult("sfdisk -d /fake/device", result)

				diskSettings := boshsettings.DiskSettings{
					Path: "/fake/device",
				}

				isMounted, err := platform.IsPersistentDiskMountable(diskSettings)
				Expect(err).ToNot(HaveOccurred())
				Expect(isMounted).To(Equal(true))
			})
		})
	})

	Describe("StartMonit", func() {
		It("creates a symlink between /etc/service/monit and /etc/sv/monit", func() {
			err := platform.StartMonit()
			Expect(err).NotTo(HaveOccurred())
			target, _ := fs.ReadLink(path.Join("/etc", "service", "monit"))
			Expect(target).To(Equal(path.Join("/etc", "sv", "monit")))
		})

		It("retries to start monit", func() {
			err := platform.StartMonit()
			Expect(err).NotTo(HaveOccurred())
			Expect(monitRetryStrategy.TryCalled).To(BeTrue())
		})

		It("returns error if retrying to start monit fails", func() {
			monitRetryStrategy.TryErr = errors.New("fake-retry-monit-error")

			err := platform.StartMonit()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-retry-monit-error"))
		})
	})

	Describe("SetupMonitUser", func() {
		It("setup monit user", func() {
			err := platform.SetupMonitUser()
			Expect(err).NotTo(HaveOccurred())

			monitUserFileStats := fs.GetFileTestStat("/fake-dir/monit/monit.user")
			Expect(monitUserFileStats).ToNot(BeNil())
			Expect(monitUserFileStats.StringContents()).To(Equal("vcap:random-password"))
		})
	})

	Describe("GetMonitCredentials", func() {
		It("get monit credentials reads monit file from disk", func() {
			fs.WriteFileString("/fake-dir/monit/monit.user", "fake-user:fake-random-password")

			username, password, err := platform.GetMonitCredentials()
			Expect(err).NotTo(HaveOccurred())

			Expect(username).To(Equal("fake-user"))
			Expect(password).To(Equal("fake-random-password"))
		})

		It("get monit credentials errs when invalid file format", func() {
			fs.WriteFileString("/fake-dir/monit/monit.user", "fake-user")

			_, _, err := platform.GetMonitCredentials()
			Expect(err).To(HaveOccurred())
		})

		It("get monit credentials leaves colons in password intact", func() {
			fs.WriteFileString("/fake-dir/monit/monit.user", "fake-user:fake:random:password")

			username, password, err := platform.GetMonitCredentials()
			Expect(err).NotTo(HaveOccurred())

			Expect(username).To(Equal("fake-user"))
			Expect(password).To(Equal("fake:random:password"))
		})
	})

	Describe("PrepareForNetworkingChange", func() {
		It("removes the network persistent rules file", func() {
			fs.WriteFile("/etc/udev/rules.d/70-persistent-net.rules", []byte{})

			err := platform.PrepareForNetworkingChange()
			Expect(err).NotTo(HaveOccurred())

			Expect(fs.FileExists("/etc/udev/rules.d/70-persistent-net.rules")).To(BeFalse())
		})

		It("returns error if removing persistent rules file fails", func() {
			fs.RemoveAllError = errors.New("fake-remove-all-error")

			err := platform.PrepareForNetworkingChange()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-remove-all-error"))
		})
	})

	Describe("SetupNetworking", func() {
		It("delegates to the NetManager", func() {
			networks := boshsettings.Networks{}

			err := platform.SetupNetworking(networks)
			Expect(err).ToNot(HaveOccurred())

			Expect(netManager.SetupNetworkingNetworks).To(Equal(networks))
		})
	})

	Describe("GetConfiguredNetworkInterfaces", func() {
		It("delegates to the NetManager", func() {
			netmanagerInterfaces := []string{"fake-eth0", "fake-eth1"}
			netManager.GetConfiguredNetworkInterfacesInterfaces = netmanagerInterfaces

			interfaces, err := platform.GetConfiguredNetworkInterfaces()
			Expect(err).ToNot(HaveOccurred())
			Expect(interfaces).To(Equal(netmanagerInterfaces))
		})
	})

	Describe("GetDefaultNetwork", func() {
		It("delegates to the defaultNetworkResolver", func() {
			defaultNetwork := boshsettings.Network{IP: "1.2.3.4"}
			fakeDefaultNetworkResolver.GetDefaultNetworkNetwork = defaultNetwork

			network, err := platform.GetDefaultNetwork()
			Expect(err).ToNot(HaveOccurred())

			Expect(network).To(Equal(defaultNetwork))
		})
	})

	Describe("GetHostPublicKey", func() {
		It("gets host public key if file exists", func() {
			fs.WriteFileString("/etc/ssh/ssh_host_rsa_key.pub", "public-key")
			hostPublicKey, err := platform.GetHostPublicKey()
			Expect(err).ToNot(HaveOccurred())
			Expect(hostPublicKey).To(Equal("public-key"))
		})

		It("throws error if file does not exist", func() {
			hostPublicKey, err := platform.GetHostPublicKey()
			Expect(err).To(HaveOccurred())
			Expect(hostPublicKey).To(Equal(""))
		})
	})

	Describe("DeleteARPEntryWithIP", func() {
		It("cleans the arp entry for the given ip", func() {
			err := platform.DeleteARPEntryWithIP("1.2.3.4")
			deleteArpEntry := []string{"arp", "-d", "1.2.3.4"}
			Expect(cmdRunner.RunCommands[0]).To(Equal(deleteArpEntry))
			Expect(err).ToNot(HaveOccurred())
		})

		It("fails if arp command fails", func() {
			result := fakesys.FakeCmdResult{
				Error:      errors.New("failure"),
				ExitStatus: 1,
				Stderr:     "",
				Stdout:     "",
			}
			cmdRunner.AddCmdResult("arp -d 1.2.3.4", result)

			err := platform.DeleteARPEntryWithIP("1.2.3.4")

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("RemoveDevTools", func() {
		It("removes listed packages", func() {
			devToolsListPath := path.Join(dirProvider.EtcDir(), "dev_tools_file_list")
			fs.WriteFileString(devToolsListPath, "dummy-compiler")
			err := platform.RemoveDevTools(devToolsListPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(cmdRunner.RunCommands)).To(Equal(1))
			Expect(cmdRunner.RunCommands[0]).To(Equal([]string{"rm", "-rf", "dummy-compiler"}))
		})
	})

	Describe("SaveDNSRecords", func() {
		var (
			dnsRecords boshsettings.DNSRecords

			defaultEtcHosts string
		)

		BeforeEach(func() {
			dnsRecords = boshsettings.DNSRecords{
				Records: [][2]string{
					{"fake-ip0", "fake-name0"},
					{"fake-ip1", "fake-name1"},
				},
			}

			defaultEtcHosts = strings.Replace(EtcHostsTemplate, "{{ . }}", "fake-hostname", -1)
		})

		It("fails generating a UUID", func() {
			fakeUUIDGenerator.GenerateError = errors.New("fake-error")

			err := platform.SaveDNSRecords(dnsRecords, "fake-hostname")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Generating UUID"))
		})

		It("fails to create intermediary /etc/hosts-<uuid> file", func() {
			fs.WriteFileErrors["/etc/hosts-fake-uuid-0"] = errors.New("fake-error")

			err := platform.SaveDNSRecords(dnsRecords, "fake-hostname")
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(ContainSubstring("Writing to /etc/hosts-fake-uuid-0"))
		})

		It("fails to renames intermediary /etc/hosts-<uuid> file to /etc/hosts", func() {
			fs.RenameError = errors.New("fake-error")

			err := platform.SaveDNSRecords(dnsRecords, "fake-hostname")
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(ContainSubstring("Renaming /etc/hosts-fake-uuid-0 to /etc/hosts"))
		})

		It("renames intermediary /etc/hosts-<uuid> atomically to /etc/hosts", func() {
			err := platform.SaveDNSRecords(dnsRecords, "fake-hostname")
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.RenameError).ToNot(HaveOccurred())

			Expect(len(fs.RenameOldPaths)).To(Equal(1))
			Expect(fs.RenameOldPaths).To(ContainElement("/etc/hosts-fake-uuid-0"))

			Expect(len(fs.RenameNewPaths)).To(Equal(1))
			Expect(fs.RenameNewPaths).To(ContainElement("/etc/hosts"))
		})

		It("preserves the default DNS records in '/etc/hosts'", func() {
			err := platform.SaveDNSRecords(dnsRecords, "fake-hostname")
			Expect(err).ToNot(HaveOccurred())

			hostsFileContents, err := fs.ReadFile("/etc/hosts")
			Expect(err).ToNot(HaveOccurred())
			Expect(string(hostsFileContents)).To(ContainSubstring(defaultEtcHosts))
		})

		It("writes the new DNS records in '/etc/hosts'", func() {
			err := platform.SaveDNSRecords(dnsRecords, "fake-hostname")
			Expect(err).ToNot(HaveOccurred())

			hostsFileContents, err := fs.ReadFile("/etc/hosts")
			Expect(err).ToNot(HaveOccurred())

			Expect(hostsFileContents).Should(MatchRegexp("fake-ip0\\s+fake-name0\\n"))
			Expect(hostsFileContents).Should(MatchRegexp("fake-ip1\\s+fake-name1\\n"))
		})
	})
}
