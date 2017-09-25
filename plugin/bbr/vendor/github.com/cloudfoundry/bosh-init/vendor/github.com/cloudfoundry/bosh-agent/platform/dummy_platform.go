package platform

import (
	"encoding/json"
	"path"

	boshdpresolv "github.com/cloudfoundry/bosh-agent/infrastructure/devicepathresolver"
	boshcert "github.com/cloudfoundry/bosh-agent/platform/cert"
	boshstats "github.com/cloudfoundry/bosh-agent/platform/stats"
	boshvitals "github.com/cloudfoundry/bosh-agent/platform/vitals"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshdir "github.com/cloudfoundry/bosh-agent/settings/directories"
	boshdirs "github.com/cloudfoundry/bosh-agent/settings/directories"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type mount struct {
	MountDir string
	DiskCid  string
}

type diskMigration struct {
	FromDiskCid string
	ToDiskCid   string
}

const CredentialFileName = "password"

type dummyPlatform struct {
	collector          boshstats.Collector
	fs                 boshsys.FileSystem
	cmdRunner          boshsys.CmdRunner
	compressor         boshcmd.Compressor
	copier             boshcmd.Copier
	dirProvider        boshdirs.Provider
	vitalsService      boshvitals.Service
	devicePathResolver boshdpresolv.DevicePathResolver
	logger             boshlog.Logger
	certManager        boshcert.Manager
}

func NewDummyPlatform(
	collector boshstats.Collector,
	fs boshsys.FileSystem,
	cmdRunner boshsys.CmdRunner,
	dirProvider boshdirs.Provider,
	devicePathResolver boshdpresolv.DevicePathResolver,
	logger boshlog.Logger,
) Platform {
	return &dummyPlatform{
		fs:                 fs,
		cmdRunner:          cmdRunner,
		collector:          collector,
		compressor:         boshcmd.NewTarballCompressor(cmdRunner, fs),
		copier:             boshcmd.NewCpCopier(cmdRunner, fs, logger),
		dirProvider:        dirProvider,
		devicePathResolver: devicePathResolver,
		vitalsService:      boshvitals.NewService(collector, dirProvider),
		certManager:        boshcert.NewDummyCertManager(fs, cmdRunner, 0, logger),
	}
}

func (p dummyPlatform) GetFs() (fs boshsys.FileSystem) {
	return p.fs
}

func (p dummyPlatform) GetRunner() (runner boshsys.CmdRunner) {
	return p.cmdRunner
}

func (p dummyPlatform) GetCompressor() (compressor boshcmd.Compressor) {
	return p.compressor
}

func (p dummyPlatform) GetCopier() (copier boshcmd.Copier) {
	return p.copier
}

func (p dummyPlatform) GetDirProvider() (dirProvider boshdir.Provider) {
	return p.dirProvider
}

func (p dummyPlatform) GetVitalsService() (service boshvitals.Service) {
	return p.vitalsService
}

func (p dummyPlatform) GetDevicePathResolver() (devicePathResolver boshdpresolv.DevicePathResolver) {
	return p.devicePathResolver
}

func (p dummyPlatform) SetupRuntimeConfiguration() (err error) {
	return
}

func (p dummyPlatform) CreateUser(username, password, basePath string) (err error) {
	return
}

func (p dummyPlatform) AddUserToGroups(username string, groups []string) (err error) {
	return
}

func (p dummyPlatform) DeleteEphemeralUsersMatching(regex string) (err error) {
	return
}

func (p dummyPlatform) SetupRootDisk(ephemeralDiskPath string) (err error) {
	return
}

func (p dummyPlatform) SetupSSH(publicKey, username string) (err error) {
	return
}

func (p dummyPlatform) SetUserPassword(user, encryptedPwd string) (err error) {
	credentialsPath := path.Join(p.dirProvider.BoshDir(), user, CredentialFileName)
	return p.fs.WriteFileString(credentialsPath, encryptedPwd)
}

func (p dummyPlatform) SaveDNSRecords(dnsRecords boshsettings.DNSRecords, hostname string) (err error) {
	return
}

func (p dummyPlatform) SetupHostname(hostname string) (err error) {
	return
}

func (p dummyPlatform) SetupNetworking(networks boshsettings.Networks) (err error) {
	return
}

func (p dummyPlatform) GetConfiguredNetworkInterfaces() (interfaces []string, err error) {
	return
}

func (p dummyPlatform) GetCertManager() (certManager boshcert.Manager) {
	return p.certManager
}

func (p dummyPlatform) SetupLogrotate(groupName, basePath, size string) (err error) {
	return
}

func (p dummyPlatform) SetTimeWithNtpServers(servers []string) (err error) {
	return
}

func (p dummyPlatform) SetupEphemeralDiskWithPath(devicePath string) (err error) {
	return
}

func (p dummyPlatform) SetupRawEphemeralDisks(devices []boshsettings.DiskSettings) (err error) {
	return
}

func (p dummyPlatform) SetupDataDir() error {
	dataDir := p.dirProvider.DataDir()

	sysDataDir := path.Join(dataDir, "sys")

	logDir := path.Join(sysDataDir, "log")
	err := p.fs.MkdirAll(logDir, logDirPermissions)
	if err != nil {
		return bosherr.WrapErrorf(err, "Making %s dir", logDir)
	}

	sysDir := path.Join(path.Dir(dataDir), "sys")
	err = p.fs.Symlink(sysDataDir, sysDir)
	if err != nil {
		return bosherr.WrapErrorf(err, "Symlinking '%s' to '%s'", sysDir, sysDataDir)
	}

	return nil
}

func (p dummyPlatform) SetupTmpDir() error {
	return nil
}

func (p dummyPlatform) MountPersistentDisk(diskSettings boshsettings.DiskSettings, mountPoint string) error {
	mounts, err := p.existingMounts()
	if err != nil {
		return err
	}

	_, isMountPoint, err := p.IsMountPoint(mountPoint)
	if err != nil {
		return err
	}

	if isMountPoint {
		mountPoint = p.dirProvider.StoreMigrationDir()
	}

	mounts = append(mounts, mount{MountDir: mountPoint, DiskCid: diskSettings.ID})
	mountsJSON, err := json.Marshal(mounts)
	if err != nil {
		return err
	}

	return p.fs.WriteFile(p.mountsPath(), mountsJSON)
}

func (p dummyPlatform) UnmountPersistentDisk(diskSettings boshsettings.DiskSettings) (didUnmount bool, err error) {
	mounts, err := p.existingMounts()
	if err != nil {
		return false, err
	}

	var updatedMounts []mount
	for _, mount := range mounts {
		if mount.DiskCid != diskSettings.ID {
			updatedMounts = append(updatedMounts, mount)
		}
	}

	updatedMountsJSON, err := json.Marshal(updatedMounts)
	if err != nil {
		return false, err
	}

	err = p.fs.WriteFile(p.mountsPath(), updatedMountsJSON)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (p dummyPlatform) GetEphemeralDiskPath(diskSettings boshsettings.DiskSettings) string {
	return "/dev/sdb"
}

func (p dummyPlatform) GetFileContentsFromCDROM(filePath string) (contents []byte, err error) {
	return
}

func (p dummyPlatform) GetFilesContentsFromDisk(diskPath string, fileNames []string) (contents [][]byte, err error) {
	return
}

func (p dummyPlatform) MigratePersistentDisk(fromMountPoint, toMountPoint string) (err error) {
	diskMigrationsPath := path.Join(p.dirProvider.BoshDir(), "disk_migrations.json")
	var diskMigrations []diskMigration
	if p.fs.FileExists(diskMigrationsPath) {
		bytes, err := p.fs.ReadFile(diskMigrationsPath)
		if err != nil {
			return err
		}
		err = json.Unmarshal(bytes, &diskMigrations)
		if err != nil {
			return err
		}
	}

	mounts, err := p.existingMounts()
	if err != nil {
		return err
	}
	fromDiskCid := p.getDiskCidByMountPoint(fromMountPoint, mounts)
	toDiskCid := p.getDiskCidByMountPoint(toMountPoint, mounts)

	diskMigrations = append(diskMigrations, diskMigration{FromDiskCid: fromDiskCid, ToDiskCid: toDiskCid})

	diskMigrationsJSON, err := json.Marshal(diskMigrations)
	if err != nil {
		return err
	}

	return p.fs.WriteFile(diskMigrationsPath, diskMigrationsJSON)
}

func (p dummyPlatform) IsMountPoint(mountPointPath string) (partitionPath string, result bool, err error) {
	mounts, err := p.existingMounts()
	if err != nil {
		return "", false, err
	}

	for _, mount := range mounts {
		if mount.MountDir == mountPointPath {
			return "", true, nil
		}
	}

	return "", false, nil
}

func (p dummyPlatform) IsPersistentDiskMounted(diskSettings boshsettings.DiskSettings) (bool, error) {
	return true, nil
}

func (p dummyPlatform) IsPersistentDiskMountable(diskSettings boshsettings.DiskSettings) (bool, error) {
	return false, nil
}

func (p dummyPlatform) StartMonit() (err error) {
	return
}

func (p dummyPlatform) SetupMonitUser() (err error) {
	return
}

func (p dummyPlatform) GetMonitCredentials() (username, password string, err error) {
	return
}

func (p dummyPlatform) PrepareForNetworkingChange() error {
	return nil
}

func (p dummyPlatform) DeleteARPEntryWithIP(ip string) error {
	return nil
}

func (p dummyPlatform) GetDefaultNetwork() (boshsettings.Network, error) {
	var network boshsettings.Network

	networkPath := path.Join(p.dirProvider.BoshDir(), "dummy-default-network-settings.json")
	contents, err := p.fs.ReadFile(networkPath)
	if err != nil {
		return network, nil
	}

	err = json.Unmarshal([]byte(contents), &network)
	if err != nil {
		return network, bosherr.WrapError(err, "Unmarshal json settings")
	}

	return network, nil
}

func (p dummyPlatform) GetHostPublicKey() (string, error) {
	return "dummy-public-key", nil
}

func (p dummyPlatform) RemoveDevTools(packageFileListPath string) error {
	return nil
}

func (p dummyPlatform) getDiskCidByMountPoint(mountPoint string, mounts []mount) string {
	var diskCid string
	for _, mount := range mounts {
		if mount.MountDir == mountPoint {
			diskCid = mount.DiskCid
		}
	}
	return diskCid
}

func (p dummyPlatform) mountsPath() string {
	return path.Join(p.dirProvider.BoshDir(), "mounts.json")
}

func (p dummyPlatform) existingMounts() ([]mount, error) {
	mountsPath := p.mountsPath()
	var mounts []mount

	if !p.fs.FileExists(mountsPath) {
		return mounts, nil
	}

	bytes, err := p.fs.ReadFile(mountsPath)
	if err != nil {
		return mounts, err
	}
	err = json.Unmarshal(bytes, &mounts)
	return mounts, err
}
