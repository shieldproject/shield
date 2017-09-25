package fakes

import (
	"path"

	boshdpresolv "github.com/cloudfoundry/bosh-agent/infrastructure/devicepathresolver"
	fakedpresolv "github.com/cloudfoundry/bosh-agent/infrastructure/devicepathresolver/fakes"
	boshcert "github.com/cloudfoundry/bosh-agent/platform/cert"
	fakecert "github.com/cloudfoundry/bosh-agent/platform/cert/fakes"
	boshvitals "github.com/cloudfoundry/bosh-agent/platform/vitals"
	fakevitals "github.com/cloudfoundry/bosh-agent/platform/vitals/fakes"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshdir "github.com/cloudfoundry/bosh-agent/settings/directories"
	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	fakecmd "github.com/cloudfoundry/bosh-utils/fileutil/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

type FakePlatform struct {
	Fs                *fakesys.FakeFileSystem
	Runner            *fakesys.FakeCmdRunner
	FakeCompressor    *fakecmd.FakeCompressor
	FakeCopier        *fakecmd.FakeCopier
	FakeVitalsService *fakevitals.FakeService
	fsType            string
	logger            boshlog.Logger

	DevicePathResolver boshdpresolv.DevicePathResolver

	SetupRuntimeConfigurationWasInvoked bool

	CreateUserUsername string
	CreateUserPassword string
	CreateUserBasePath string

	AddUserToGroupsGroups             map[string][]string
	DeleteEphemeralUsersMatchingRegex string
	SetupSSHPublicKeys                map[string]string

	SetupSSHCalled    bool
	SetupSSHPublicKey string
	SetupSSHUsername  string
	SetupSSHErr       error

	UserPasswords         map[string]string
	SetupHostnameHostname string

	SaveDNSRecordsError      error
	SaveDNSRecordsHostname   string
	SaveDNSRecordsDNSRecords boshsettings.DNSRecords

	SetTimeWithNtpServersServers []string

	SetupEphemeralDiskWithPathDevicePath string
	SetupEphemeralDiskWithPathErr        error

	SetupRawEphemeralDisksDevices   []boshsettings.DiskSettings
	SetupRawEphemeralDisksErr       error
	SetupRawEphemeralDisksCallCount int

	SetupDataDirCalled bool
	SetupDataDirErr    error

	SetupTmpDirCalled bool
	SetupTmpDirErr    error

	SetupNetworkingCalled   bool
	SetupNetworkingNetworks boshsettings.Networks
	SetupNetworkingErr      error

	MountPersistentDiskCalled     bool
	MountPersistentDiskSettings   boshsettings.DiskSettings
	MountPersistentDiskMountPoint string
	MountPersistentDiskErr        error

	UnmountPersistentDiskDidUnmount bool
	UnmountPersistentDiskSettings   boshsettings.DiskSettings

	GetFileContentsFromCDROMPath        string
	GetFileContentsFromCDROMContents    []byte
	GetFileContentsFromCDROMErr         error
	GetFileContentsFromCDROMCalledTimes int

	GetFileContentsFromDiskDiskPaths   []string
	GetFileContentsFromDiskFileNames   [][]string
	GetFileContentsFromDiskContents    map[string][]byte
	GetFileContentsFromDiskErrs        map[string]error
	GetFileContentsFromDiskCalledTimes int

	GetEphemeralDiskPathCalled   bool
	GetEphemeralDiskPathSettings boshsettings.DiskSettings
	GetEphemeralDiskPathRealPath string

	ScsiDiskMap map[string]string

	MigratePersistentDiskFromMountPoint string
	MigratePersistentDiskToMountPoint   string

	IsPersistentDiskMountableResult bool
	IsPersistentDiskMountableErr    error

	IsMountPointPath          string
	IsMountPointPartitionPath string
	IsMountPointResult        bool
	IsMountPointErr           error

	PackageFileListPath    string
	IsRemoveDevToolsCalled bool
	IsRemoveDevToolsError  error

	MountedDevicePaths []string

	StartMonitStarted           bool
	SetupMonitUserSetup         bool
	GetMonitCredentialsUsername string
	GetMonitCredentialsPassword string

	PrepareForNetworkingChangeCalled bool
	PrepareForNetworkingChangeErr    error

	GetDefaultNetworkNetwork boshsettings.Network
	GetDefaultNetworkErr     error

	GetConfiguredNetworkInterfacesInterfaces []string
	GetConfiguredNetworkInterfacesErr        error

	LastIPDeletedFromARP    string
	DeleteARPEntryWithIPErr error

	certManager boshcert.Manager

	GetHostPublicKeyValue string
	GetHostPublicKeyError error

	SetupRootDiskCalledTimes int
	SetupRootDiskError       error
}

func NewFakePlatform() (platform *FakePlatform) {
	platform = new(FakePlatform)
	platform.Fs = fakesys.NewFakeFileSystem()
	platform.Runner = fakesys.NewFakeCmdRunner()
	platform.FakeCompressor = fakecmd.NewFakeCompressor()
	platform.FakeCopier = fakecmd.NewFakeCopier()
	platform.FakeVitalsService = fakevitals.NewFakeService()
	platform.DevicePathResolver = fakedpresolv.NewFakeDevicePathResolver()
	platform.AddUserToGroupsGroups = make(map[string][]string)
	platform.SetupSSHPublicKeys = make(map[string]string)
	platform.UserPasswords = make(map[string]string)
	platform.ScsiDiskMap = make(map[string]string)
	platform.GetFileContentsFromDiskDiskPaths = []string{}
	platform.GetFileContentsFromDiskFileNames = [][]string{}
	platform.GetFileContentsFromDiskContents = map[string][]byte{}
	platform.GetFileContentsFromDiskErrs = map[string]error{}
	platform.certManager = new(fakecert.FakeManager)
	platform.SetupRawEphemeralDisksCallCount = 0
	platform.SetupRawEphemeralDisksDevices = nil
	platform.SetupRawEphemeralDisksErr = nil
	platform.GetHostPublicKeyError = nil
	platform.SetupRootDiskCalledTimes = 0
	platform.SetupRootDiskError = nil
	platform.IsPersistentDiskMountableErr = nil
	return
}

func (p *FakePlatform) GetFs() (fs boshsys.FileSystem) {
	return p.Fs
}

func (p *FakePlatform) GetPersistentDiskFS() (fsType string) {
	return p.fsType
}

func (p *FakePlatform) GetRunner() (runner boshsys.CmdRunner) {
	return p.Runner
}

func (p *FakePlatform) GetCompressor() (compressor boshcmd.Compressor) {
	return p.FakeCompressor
}

func (p *FakePlatform) GetCopier() (copier boshcmd.Copier) {
	return p.FakeCopier
}

func (p *FakePlatform) GetDirProvider() (dirProvider boshdir.Provider) {
	return boshdir.NewProvider("/var/vcap")
}

func (p *FakePlatform) GetVitalsService() (service boshvitals.Service) {
	return p.FakeVitalsService
}

func (p *FakePlatform) GetDevicePathResolver() (devicePathResolver boshdpresolv.DevicePathResolver) {
	return p.DevicePathResolver
}

func (p *FakePlatform) SetupRuntimeConfiguration() (err error) {
	p.SetupRuntimeConfigurationWasInvoked = true
	return
}

func (p *FakePlatform) CreateUser(username, password, basePath string) (err error) {
	p.CreateUserUsername = username
	p.CreateUserPassword = password
	p.CreateUserBasePath = basePath
	return
}

func (p *FakePlatform) AddUserToGroups(username string, groups []string) (err error) {
	p.AddUserToGroupsGroups[username] = groups
	return
}

func (p *FakePlatform) DeleteEphemeralUsersMatching(regex string) (err error) {
	p.DeleteEphemeralUsersMatchingRegex = regex
	return
}

func (p *FakePlatform) SetupRootDisk(ephemeralDiskPath string) (err error) {
	p.SetupRootDiskCalledTimes++
	if p.SetupRootDiskError != nil {
		err = p.SetupRootDiskError
	}
	return
}

func (p *FakePlatform) SetupSSH(publicKey, username string) error {
	p.SetupSSHCalled = true
	p.SetupSSHPublicKeys[username] = publicKey
	p.SetupSSHPublicKey = publicKey
	p.SetupSSHUsername = username
	return p.SetupSSHErr
}

func (p *FakePlatform) SetUserPassword(user, encryptedPwd string) (err error) {
	p.UserPasswords[user] = encryptedPwd
	return
}

func (p *FakePlatform) SaveDNSRecords(dnsRecords boshsettings.DNSRecords, hostname string) error {
	p.SaveDNSRecordsDNSRecords = dnsRecords
	p.SaveDNSRecordsHostname = hostname
	return p.SaveDNSRecordsError
}

func (p *FakePlatform) SetupHostname(hostname string) (err error) {
	p.SetupHostnameHostname = hostname
	return
}

func (p *FakePlatform) SetupNetworking(networks boshsettings.Networks) error {
	p.SetupNetworkingCalled = true
	p.SetupNetworkingNetworks = networks
	return p.SetupNetworkingErr
}

func (p *FakePlatform) GetConfiguredNetworkInterfaces() ([]string, error) {
	return p.GetConfiguredNetworkInterfacesInterfaces, p.GetConfiguredNetworkInterfacesErr
}

func (p *FakePlatform) GetCertManager() (certManager boshcert.Manager) {
	return p.certManager
}

func (p *FakePlatform) SetupLogrotate(groupName, basePath, size string) (err error) {
	return
}

func (p *FakePlatform) SetTimeWithNtpServers(servers []string) (err error) {
	p.SetTimeWithNtpServersServers = servers
	return
}

func (p *FakePlatform) SetupEphemeralDiskWithPath(devicePath string) (err error) {
	p.SetupEphemeralDiskWithPathDevicePath = devicePath
	return p.SetupEphemeralDiskWithPathErr
}

func (p *FakePlatform) SetupRawEphemeralDisks(devices []boshsettings.DiskSettings) (err error) {
	p.SetupRawEphemeralDisksDevices = devices
	p.SetupRawEphemeralDisksCallCount++
	return p.SetupRawEphemeralDisksErr
}

func (p *FakePlatform) SetupDataDir() error {
	p.SetupDataDirCalled = true
	return p.SetupDataDirErr
}

func (p *FakePlatform) SetupTmpDir() error {
	p.SetupTmpDirCalled = true
	return p.SetupTmpDirErr
}

func (p *FakePlatform) MountPersistentDisk(diskSettings boshsettings.DiskSettings, mountPoint string) (err error) {
	p.MountPersistentDiskCalled = true
	p.MountPersistentDiskSettings = diskSettings
	p.MountPersistentDiskMountPoint = mountPoint
	return p.MountPersistentDiskErr
}

func (p *FakePlatform) UnmountPersistentDisk(diskSettings boshsettings.DiskSettings) (didUnmount bool, err error) {
	p.UnmountPersistentDiskSettings = diskSettings
	didUnmount = p.UnmountPersistentDiskDidUnmount
	return
}

func (p *FakePlatform) GetEphemeralDiskPath(diskSettings boshsettings.DiskSettings) string {
	p.GetEphemeralDiskPathCalled = true
	p.GetEphemeralDiskPathSettings = diskSettings
	return p.GetEphemeralDiskPathRealPath
}

func (p *FakePlatform) GetFileContentsFromCDROM(path string) ([]byte, error) {
	p.GetFileContentsFromCDROMCalledTimes++
	p.GetFileContentsFromCDROMPath = path
	return p.GetFileContentsFromCDROMContents, p.GetFileContentsFromCDROMErr
}

func (p *FakePlatform) GetFilesContentsFromDisk(diskPath string, fileNames []string) ([][]byte, error) {
	p.GetFileContentsFromDiskCalledTimes++

	p.GetFileContentsFromDiskDiskPaths = append(p.GetFileContentsFromDiskDiskPaths, diskPath)
	p.GetFileContentsFromDiskFileNames = append(p.GetFileContentsFromDiskFileNames, fileNames)

	result := [][]byte{}
	for _, fileName := range fileNames {
		fileDiskPath := path.Join(diskPath, fileName)
		err := p.GetFileContentsFromDiskErrs[fileDiskPath]
		if err != nil {
			return [][]byte{}, err
		}

		result = append(result, p.GetFileContentsFromDiskContents[fileDiskPath])
	}

	return result, nil
}

func (p *FakePlatform) SetGetFilesContentsFromDisk(fileName string, contents []byte, err error) {
	p.GetFileContentsFromDiskContents[fileName] = contents
	p.GetFileContentsFromDiskErrs[fileName] = err
}

func (p *FakePlatform) MigratePersistentDisk(fromMountPoint, toMountPoint string) (err error) {
	p.MigratePersistentDiskFromMountPoint = fromMountPoint
	p.MigratePersistentDiskToMountPoint = toMountPoint
	return
}

func (p *FakePlatform) IsMountPoint(path string) (string, bool, error) {
	p.IsMountPointPath = path
	return p.IsMountPointPartitionPath, p.IsMountPointResult, p.IsMountPointErr
}

func (p *FakePlatform) IsPersistentDiskMounted(diskSettings boshsettings.DiskSettings) (result bool, err error) {
	for _, mountedPath := range p.MountedDevicePaths {
		if mountedPath == diskSettings.Path {
			return true, nil
		}
	}
	return
}

func (p *FakePlatform) SetIsPersistentDiskMountable(isPartitioned bool, err error) {
	p.IsPersistentDiskMountableResult = isPartitioned
	p.IsPersistentDiskMountableErr = err
}

func (p *FakePlatform) IsPersistentDiskMountable(diskSettings boshsettings.DiskSettings) (bool, error) {
	return p.IsPersistentDiskMountableResult, p.IsPersistentDiskMountableErr
}

func (p *FakePlatform) StartMonit() (err error) {
	p.StartMonitStarted = true
	return
}

func (p *FakePlatform) SetupMonitUser() (err error) {
	p.SetupMonitUserSetup = true
	return
}

func (p *FakePlatform) GetMonitCredentials() (username, password string, err error) {
	username = p.GetMonitCredentialsUsername
	password = p.GetMonitCredentialsPassword
	return
}

func (p *FakePlatform) DeleteARPEntryWithIP(ip string) error {
	p.LastIPDeletedFromARP = ip
	return p.DeleteARPEntryWithIPErr
}

func (p *FakePlatform) PrepareForNetworkingChange() error {
	p.PrepareForNetworkingChangeCalled = true
	return p.PrepareForNetworkingChangeErr
}

func (p *FakePlatform) GetDefaultNetwork() (boshsettings.Network, error) {
	return p.GetDefaultNetworkNetwork, p.GetDefaultNetworkErr
}

func (p *FakePlatform) GetHostPublicKey() (string, error) {
	return p.GetHostPublicKeyValue, p.GetHostPublicKeyError
}

func (p *FakePlatform) RemoveDevTools(packageFileListPath string) error {
	p.IsRemoveDevToolsCalled = true
	p.PackageFileListPath = packageFileListPath
	return p.IsRemoveDevToolsError
}
