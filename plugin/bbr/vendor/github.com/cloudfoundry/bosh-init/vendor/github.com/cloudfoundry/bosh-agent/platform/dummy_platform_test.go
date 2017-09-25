package platform_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	boshdpresolv "github.com/cloudfoundry/bosh-agent/infrastructure/devicepathresolver"
	fakedpresolv "github.com/cloudfoundry/bosh-agent/infrastructure/devicepathresolver/fakes"
	. "github.com/cloudfoundry/bosh-agent/platform"
	boshstats "github.com/cloudfoundry/bosh-agent/platform/stats"
	fakestats "github.com/cloudfoundry/bosh-agent/platform/stats/fakes"
	"github.com/cloudfoundry/bosh-agent/settings"
	boshdirs "github.com/cloudfoundry/bosh-agent/settings/directories"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	"path"
)

type mount struct {
	MountDir string
	DiskCid  string
}

var _ = Describe("DummyPlatform", describeDummyPlatform)

func describeDummyPlatform() {
	var (
		platform           Platform
		collector          boshstats.Collector
		fs                 *fakesys.FakeFileSystem
		cmdRunner          boshsys.CmdRunner
		dirProvider        boshdirs.Provider
		devicePathResolver boshdpresolv.DevicePathResolver
		logger             boshlog.Logger
	)

	BeforeEach(func() {
		collector = &fakestats.FakeCollector{}
		fs = fakesys.NewFakeFileSystem()
		cmdRunner = fakesys.NewFakeCmdRunner()
		dirProvider = boshdirs.NewProvider("/fake-dir")
		devicePathResolver = fakedpresolv.NewFakeDevicePathResolver()
		logger = boshlog.NewLogger(boshlog.LevelNone)
	})

	JustBeforeEach(func() {
		platform = NewDummyPlatform(
			collector,
			fs,
			cmdRunner,
			dirProvider,
			devicePathResolver,
			logger,
		)
	})

	Describe("GetDefaultNetwork", func() {
		It("returns the contents of dummy-defaults-network-settings.json since that's what the dummy cpi writes", func() {
			settingsFilePath := "/fake-dir/bosh/dummy-default-network-settings.json"
			fs.WriteFileString(settingsFilePath, `{"IP": "1.2.3.4"}`)

			network, err := platform.GetDefaultNetwork()
			Expect(err).NotTo(HaveOccurred())

			Expect(network.IP).To(Equal("1.2.3.4"))
		})
	})

	Describe("GetCertManager", func() {
		It("returs a dummy cert manager", func() {
			certManager := platform.GetCertManager()

			Expect(certManager.UpdateCertificates("")).Should(BeNil())
		})
	})

	Describe("UnmountPersistentDisk", func() {
		Context("when there are two mounted persistent disks in the mounts json", func() {
			BeforeEach(func() {

				var mounts []mount
				mounts = append(mounts, mount{MountDir: "dir1", DiskCid: "cid1"})
				mounts = append(mounts, mount{MountDir: "dir2", DiskCid: "cid2"})
				mountsJSON, _ := json.Marshal(mounts)

				mountsPath := path.Join(dirProvider.BoshDir(), "mounts.json")
				fs.WriteFile(mountsPath, mountsJSON)
			})

			It("removes one of the disks from the mounts json", func() {
				unmounted, err := platform.UnmountPersistentDisk(settings.DiskSettings{ID: "cid1"})
				Expect(err).NotTo(HaveOccurred())
				Expect(unmounted).To(Equal(true))

				_, isMountPoint, err := platform.IsMountPoint("dir1")
				Expect(isMountPoint).To(Equal(false))

				_, isMountPoint, err = platform.IsMountPoint("dir2")
				Expect(isMountPoint).To(Equal(true))
			})
		})
	})

	Describe("SetUserPassword", func() {
		It("writes the password to a file", func() {
			err := platform.SetUserPassword("user-name", "fake-password")
			Expect(err).NotTo(HaveOccurred())

			userPasswordsPath := path.Join(dirProvider.BoshDir(), "user-name", CredentialFileName)
			password, err := fs.ReadFileString(userPasswordsPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(password).To(Equal("fake-password"))
		})

		It("writes the passwords to different files for each user", func() {
			err := platform.SetUserPassword("user-name1", "fake-password1")
			Expect(err).NotTo(HaveOccurred())
			err = platform.SetUserPassword("user-name2", "fake-password2")
			Expect(err).NotTo(HaveOccurred())

			userPasswordsPath := path.Join(dirProvider.BoshDir(), "user-name1", CredentialFileName)
			password, err := fs.ReadFileString(userPasswordsPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(password).To(Equal("fake-password1"))

			userPasswordsPath = path.Join(dirProvider.BoshDir(), "user-name2", CredentialFileName)
			password, err = fs.ReadFileString(userPasswordsPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(password).To(Equal("fake-password2"))
		})
	})

	Describe("SetupDataDir", func() {
		It("creates a link from BASEDIR/sys to BASEDIR/data/sys", func() {
			err := platform.SetupDataDir()
			Expect(err).NotTo(HaveOccurred())

			stat := fs.GetFileTestStat("/fake-dir/sys")

			Expect(stat).ToNot(BeNil())
			Expect(stat.SymlinkTarget).To(Equal("/fake-dir/data/sys"))
		})
	})
}
