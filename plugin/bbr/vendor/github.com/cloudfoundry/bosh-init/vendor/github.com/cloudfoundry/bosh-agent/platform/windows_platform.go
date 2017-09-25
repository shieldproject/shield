package platform

import (
	"fmt"
	"strings"

	boshdpresolv "github.com/cloudfoundry/bosh-agent/infrastructure/devicepathresolver"
	boshcert "github.com/cloudfoundry/bosh-agent/platform/cert"
	boshnet "github.com/cloudfoundry/bosh-agent/platform/net"
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

type WindowsPlatform struct {
	collector              boshstats.Collector
	fs                     boshsys.FileSystem
	cmdRunner              boshsys.CmdRunner
	compressor             boshcmd.Compressor
	copier                 boshcmd.Copier
	dirProvider            boshdirs.Provider
	vitalsService          boshvitals.Service
	netManager             boshnet.Manager
	devicePathResolver     boshdpresolv.DevicePathResolver
	certManager            boshcert.Manager
	defaultNetworkResolver boshsettings.DefaultNetworkResolver
}

func NewWindowsPlatform(
	collector boshstats.Collector,
	fs boshsys.FileSystem,
	cmdRunner boshsys.CmdRunner,
	dirProvider boshdirs.Provider,
	netManager boshnet.Manager,
	devicePathResolver boshdpresolv.DevicePathResolver,
	logger boshlog.Logger,
	defaultNetworkResolver boshsettings.DefaultNetworkResolver,
) Platform {
	return &WindowsPlatform{
		fs:                     fs,
		cmdRunner:              cmdRunner,
		collector:              collector,
		compressor:             boshcmd.NewTarballCompressor(cmdRunner, fs),
		copier:                 boshcmd.NewGenericCpCopier(fs, logger),
		dirProvider:            dirProvider,
		netManager:             netManager,
		devicePathResolver:     devicePathResolver,
		vitalsService:          boshvitals.NewService(collector, dirProvider),
		certManager:            boshcert.NewDummyCertManager(fs, cmdRunner, 0, logger),
		defaultNetworkResolver: defaultNetworkResolver,
	}
}

func (p WindowsPlatform) GetFs() (fs boshsys.FileSystem) {
	return p.fs
}

func (p WindowsPlatform) GetRunner() (runner boshsys.CmdRunner) {
	return p.cmdRunner
}

func (p WindowsPlatform) GetCompressor() (compressor boshcmd.Compressor) {
	return p.compressor
}

func (p WindowsPlatform) GetCopier() (copier boshcmd.Copier) {
	return p.copier
}

func (p WindowsPlatform) GetDirProvider() (dirProvider boshdir.Provider) {
	return p.dirProvider
}

func (p WindowsPlatform) GetVitalsService() (service boshvitals.Service) {
	return p.vitalsService
}

func (p WindowsPlatform) GetDevicePathResolver() (devicePathResolver boshdpresolv.DevicePathResolver) {
	return p.devicePathResolver
}

func (p WindowsPlatform) SetupRuntimeConfiguration() (err error) {
	return
}

func (p WindowsPlatform) CreateUser(username, password, basePath string) (err error) {
	return
}

func (p WindowsPlatform) AddUserToGroups(username string, groups []string) (err error) {
	return
}

func (p WindowsPlatform) DeleteEphemeralUsersMatching(regex string) (err error) {
	return
}

func (p WindowsPlatform) SetupRootDisk(ephemeralDiskPath string) (err error) {
	return
}

func (p WindowsPlatform) SetupSSH(publicKey, username string) (err error) {
	return
}

func (p WindowsPlatform) SetUserPassword(user, encryptedPwd string) (err error) {
	return
}

func (p WindowsPlatform) SaveDNSRecords(dnsRecords boshsettings.DNSRecords, hostname string) (err error) {
	return
}

func (p WindowsPlatform) SetupHostname(hostname string) (err error) {
	return
}

func (p WindowsPlatform) SetupNetworking(networks boshsettings.Networks) (err error) {
	return p.netManager.SetupNetworking(networks, nil)
}

func (p WindowsPlatform) GetConfiguredNetworkInterfaces() (interfaces []string, err error) {
	return
}

func (p WindowsPlatform) GetCertManager() (certManager boshcert.Manager) {
	return p.certManager
}

func (p WindowsPlatform) SetupLogrotate(groupName, basePath, size string) (err error) {
	return
}

func (p WindowsPlatform) SetTimeWithNtpServers(servers []string) (err error) {
	if len(servers) == 0 {
		return
	}
	var (
		stderr string
	)
	ntpServers := strings.Join(servers, " ")
	_, stderr, _, err = p.cmdRunner.RunCommand("powershell.exe",
		"new-netfirewallrule",
		"-displayname", "NTP",
		"-direction", "outbound",
		"-action", "allow",
		"-protocol", "udp",
		"-RemotePort", "123")
	if err != nil {
		err = bosherr.WrapErrorf(err, "SetTimeWithNtpServers  %s", stderr)
		return
	}

	_, _, _, _ = p.cmdRunner.RunCommand("net", "stop", "w32time")
	manualPeerList := fmt.Sprintf("/manualpeerlist:\"%s\"", ntpServers)
	_, stderr, _, err = p.cmdRunner.RunCommand("w32tm", "/config", "/syncfromflags:manual", manualPeerList)
	if err != nil {
		err = bosherr.WrapErrorf(err, "SetTimeWithNtpServers %s", stderr)
		return
	}
	_, _, _, _ = p.cmdRunner.RunCommand("net", "start", "w32time")
	_, stderr, _, err = p.cmdRunner.RunCommand("w32tm", "/config", "/update")
	if err != nil {
		err = bosherr.WrapErrorf(err, "SetTimeWithNtpServers %s", stderr)
		return
	}
	_, stderr, _, err = p.cmdRunner.RunCommand("w32tm", "/resync", "/rediscover")
	if err != nil {
		err = bosherr.WrapErrorf(err, "SetTimeWithNtpServers %s", stderr)
		return
	}
	return
}

func (p WindowsPlatform) SetupEphemeralDiskWithPath(devicePath string) (err error) {
	return
}

func (p WindowsPlatform) SetupRawEphemeralDisks(devices []boshsettings.DiskSettings) (err error) {
	return
}

func (p WindowsPlatform) SetupDataDir() error {
	return nil
}

func (p WindowsPlatform) SetupTmpDir() error {
	return nil
}

func (p WindowsPlatform) MountPersistentDisk(diskSettings boshsettings.DiskSettings, mountPoint string) (err error) {
	return
}

func (p WindowsPlatform) UnmountPersistentDisk(diskSettings boshsettings.DiskSettings) (didUnmount bool, err error) {
	return
}

func (p WindowsPlatform) GetEphemeralDiskPath(diskSettings boshsettings.DiskSettings) string {
	return ""
}

func (p WindowsPlatform) GetFileContentsFromCDROM(filePath string) (contents []byte, err error) {
	return p.fs.ReadFile("D:/" + filePath)
}

func (p WindowsPlatform) GetFilesContentsFromDisk(diskPath string, fileNames []string) (contents [][]byte, err error) {
	return
}

func (p WindowsPlatform) MigratePersistentDisk(fromMountPoint, toMountPoint string) (err error) {
	return
}

func (p WindowsPlatform) IsMountPoint(path string) (string, bool, error) {
	return "", true, nil
}

func (p WindowsPlatform) IsPersistentDiskMounted(diskSettings boshsettings.DiskSettings) (bool, error) {
	return true, nil
}

func (p WindowsPlatform) IsPersistentDiskMountable(diskSettings boshsettings.DiskSettings) (bool, error) {
	return true, nil
}

func (p WindowsPlatform) StartMonit() (err error) {
	return
}

func (p WindowsPlatform) SetupMonitUser() (err error) {
	return
}

func (p WindowsPlatform) GetMonitCredentials() (username, password string, err error) {
	return
}

func (p WindowsPlatform) PrepareForNetworkingChange() error {
	return nil
}

func (p WindowsPlatform) CleanIPMacAddressCache(ip string) error {
	return nil
}

func (p WindowsPlatform) RemoveDevTools(packageFileListPath string) error {
	return nil
}

func (p WindowsPlatform) GetDefaultNetwork() (boshsettings.Network, error) {
	return p.defaultNetworkResolver.GetDefaultNetwork()
}

func (p WindowsPlatform) GetHostPublicKey() (string, error) {
	return "", nil
}

func (p WindowsPlatform) DeleteARPEntryWithIP(ip string) error {
	return nil
}
