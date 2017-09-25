package platform

import (
	"github.com/cloudfoundry/bosh-agent/platform/cert"

	"log"

	boshdpresolv "github.com/cloudfoundry/bosh-agent/infrastructure/devicepathresolver"
	boshvitals "github.com/cloudfoundry/bosh-agent/platform/vitals"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshdir "github.com/cloudfoundry/bosh-agent/settings/directories"
	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type AuditLogger interface {
	Debug(string)
	Err(string)
	StartLogging()
}

type AuditLoggerProvider interface {
	ProvideDebugLogger() (*log.Logger, error)
	ProvideErrorLogger() (*log.Logger, error)
}

type Platform interface {
	GetFs() boshsys.FileSystem
	GetRunner() boshsys.CmdRunner
	GetCompressor() boshcmd.Compressor
	GetCopier() boshcmd.Copier
	GetDirProvider() boshdir.Provider
	GetVitalsService() boshvitals.Service
	GetAuditLogger() AuditLogger
	GetDevicePathResolver() (devicePathResolver boshdpresolv.DevicePathResolver)

	// User management
	CreateUser(username, basePath string) (err error)
	AddUserToGroups(username string, groups []string) (err error)
	DeleteEphemeralUsersMatching(regex string) (err error)

	// Bootstrap functionality
	SetupRootDisk(ephemeralDiskPath string) (err error)
	SetupSSH(publicKey []string, username string) (err error)
	SetUserPassword(user, encryptedPwd string) (err error)
	SetupIPv6(boshsettings.IPv6) error
	SetupHostname(hostname string) (err error)
	SetupNetworking(networks boshsettings.Networks) (err error)
	SetupLogrotate(groupName, basePath, size string) (err error)
	SetTimeWithNtpServers(servers []string) (err error)
	SetupEphemeralDiskWithPath(devicePath string, desiredSwapSizeInBytes *uint64) (err error)
	SetupRawEphemeralDisks(devices []boshsettings.DiskSettings) (err error)
	SetupDataDir() (err error)
	SetupTmpDir() (err error)
	SetupHomeDir() (err error)
	SetupBlobsDir() (err error)
	SetupMonitUser() (err error)
	StartMonit() (err error)
	SetupRuntimeConfiguration() (err error)
	SetupLogDir() (err error)
	SetupLoggingAndAuditing() (err error)
	SetupRecordsJSONPermission(path string) error

	// Disk management
	MountPersistentDisk(diskSettings boshsettings.DiskSettings, mountPoint string) error
	UnmountPersistentDisk(diskSettings boshsettings.DiskSettings) (didUnmount bool, err error)
	MigratePersistentDisk(fromMountPoint, toMountPoint string) (err error)
	GetEphemeralDiskPath(diskSettings boshsettings.DiskSettings) string
	IsMountPoint(path string) (partitionPath string, result bool, err error)
	IsPersistentDiskMounted(diskSettings boshsettings.DiskSettings) (result bool, err error)
	IsPersistentDiskMountable(diskSettings boshsettings.DiskSettings) (bool, error)
	AssociateDisk(name string, settings boshsettings.DiskSettings) error

	GetFileContentsFromCDROM(filePath string) (contents []byte, err error)
	GetFilesContentsFromDisk(diskPath string, fileNames []string) (contents [][]byte, err error)

	// Network misc
	GetDefaultNetwork() (boshsettings.Network, error)
	GetConfiguredNetworkInterfaces() ([]string, error)
	PrepareForNetworkingChange() error
	DeleteARPEntryWithIP(ip string) error
	SaveDNSRecords(dnsRecords boshsettings.DNSRecords, hostname string) error

	// Additional monit management
	GetMonitCredentials() (username, password string, err error)

	GetCertManager() cert.Manager

	GetHostPublicKey() (string, error)

	RemoveDevTools(packageFileListPath string) error
	RemoveStaticLibraries(packageFileListPath string) error
}
