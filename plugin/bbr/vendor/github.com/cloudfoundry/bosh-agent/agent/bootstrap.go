package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"path/filepath"

	boshplatform "github.com/cloudfoundry/bosh-agent/platform"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshdir "github.com/cloudfoundry/bosh-agent/settings/directories"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type Bootstrap interface {
	Run() error
}

type bootstrap struct {
	fs              boshsys.FileSystem
	platform        boshplatform.Platform
	dirProvider     boshdir.Provider
	settingsService boshsettings.Service
	logger          boshlog.Logger
}

func NewBootstrap(
	platform boshplatform.Platform,
	dirProvider boshdir.Provider,
	settingsService boshsettings.Service,
	logger boshlog.Logger,
) Bootstrap {
	return bootstrap{
		fs:              platform.GetFs(),
		platform:        platform,
		dirProvider:     dirProvider,
		settingsService: settingsService,
		logger:          logger,
	}
}

func (boot bootstrap) Run() (err error) {
	if err = boot.platform.SetupRuntimeConfiguration(); err != nil {
		return bosherr.WrapError(err, "Setting up runtime configuration")
	}

	iaasPublicKey, err := boot.settingsService.PublicSSHKeyForUsername(boshsettings.VCAPUsername)
	if err != nil {
		return bosherr.WrapError(err, "Setting up ssh: Getting iaas public key")
	}

	if len(iaasPublicKey) > 0 {
		if err = boot.platform.SetupSSH([]string{iaasPublicKey}, boshsettings.VCAPUsername); err != nil {
			return bosherr.WrapError(err, "Setting up iaas ssh")
		}
	}

	if err = boot.settingsService.LoadSettings(); err != nil {
		return bosherr.WrapError(err, "Fetching settings")
	}

	settings := boot.settingsService.GetSettings()
	envPublicKeys := settings.Env.GetAuthorizedKeys()

	if len(envPublicKeys) > 0 {
		publicKeys := envPublicKeys

		if len(iaasPublicKey) > 0 {
			publicKeys = append(publicKeys, iaasPublicKey)
		}

		if err = boot.platform.SetupSSH(publicKeys, boshsettings.VCAPUsername); err != nil {
			return bosherr.WrapError(err, "Adding env-configured ssh keys")
		}
	}

	if err = boot.setUserPasswords(settings.Env); err != nil {
		return bosherr.WrapError(err, "Settings user password")
	}

	if err = boot.platform.SetupIPv6(settings.Env.Bosh.IPv6); err != nil {
		return bosherr.WrapError(err, "Setting up IPv6")
	}

	if err = boot.platform.SetupHostname(settings.AgentID); err != nil {
		return bosherr.WrapError(err, "Setting up hostname")
	}

	if err = boot.platform.SetupNetworking(settings.Networks); err != nil {
		return bosherr.WrapError(err, "Setting up networking")
	}

	if err = boot.platform.SetTimeWithNtpServers(settings.Ntp); err != nil {
		return bosherr.WrapError(err, "Setting up NTP servers")
	}

	if err = boot.platform.SetupRawEphemeralDisks(settings.RawEphemeralDiskSettings()); err != nil {
		return bosherr.WrapError(err, "Setting up raw ephemeral disk")
	}

	ephemeralDiskPath := boot.platform.GetEphemeralDiskPath(settings.EphemeralDiskSettings())
	desiredSwapSizeInBytes := settings.Env.GetSwapSizeInBytes()
	if err = boot.platform.SetupEphemeralDiskWithPath(ephemeralDiskPath, desiredSwapSizeInBytes); err != nil {
		return bosherr.WrapError(err, "Setting up ephemeral disk")
	}

	if err = boot.platform.SetupRootDisk(ephemeralDiskPath); err != nil {
		return bosherr.WrapError(err, "Setting up root disk")
	}

	if err = boot.platform.SetupLogDir(); err != nil {
		return bosherr.WrapError(err, "Setting up log dir")
	}

	if err = boot.platform.SetupLoggingAndAuditing(); err != nil {
		return bosherr.WrapError(err, "Starting up logging and auditing utilities")
	}

	if err = boot.platform.SetupDataDir(); err != nil {
		return bosherr.WrapError(err, "Setting up data dir")
	}

	if err = boot.platform.SetupTmpDir(); err != nil {
		return bosherr.WrapError(err, "Setting up tmp dir")
	}

	if err = boot.platform.SetupHomeDir(); err != nil {
		return bosherr.WrapError(err, "Setting up home dir")
	}

	if err = boot.platform.SetupBlobsDir(); err != nil {
		return bosherr.WrapError(err, "Setting up blobs dir")
	}

	if err = boot.comparePersistentDisk(); err != nil {
		return bosherr.WrapError(err, "Comparing persistent disks")
	}

	for diskID := range settings.Disks.Persistent {
		var lastDiskID string
		diskSettings, _ := settings.PersistentDiskSettings(diskID)

		isPartitioned, err := boot.platform.IsPersistentDiskMountable(diskSettings)
		if err != nil {
			return bosherr.WrapError(err, "Checking if persistent disk is partitioned")
		}

		lastDiskID, err = boot.lastMountedCid()
		if err != nil {
			return bosherr.WrapError(err, "Fetching last mounted disk CID")
		}
		if isPartitioned && diskID == lastDiskID {
			if err = boot.platform.MountPersistentDisk(diskSettings, boot.dirProvider.StoreDir()); err != nil {
				return bosherr.WrapError(err, "Mounting persistent disk")
			}
		}
	}

	if err = boot.platform.SetupMonitUser(); err != nil {
		return bosherr.WrapError(err, "Setting up monit user")
	}

	if err = boot.platform.StartMonit(); err != nil {
		return bosherr.WrapError(err, "Starting monit")
	}

	if settings.Env.GetRemoveDevTools() {
		packageFileListPath := path.Join(boot.dirProvider.EtcDir(), "dev_tools_file_list")

		if !boot.fs.FileExists(packageFileListPath) {
			return nil
		}

		if err = boot.platform.RemoveDevTools(packageFileListPath); err != nil {
			return bosherr.WrapError(err, "Removing Development Tools Packages")
		}
	}

	if settings.Env.GetRemoveStaticLibraries() {
		staticLibrariesListPath := path.Join(boot.dirProvider.EtcDir(), "static_libraries_list")

		if !boot.fs.FileExists(staticLibrariesListPath) {
			return nil
		}

		if err = boot.platform.RemoveStaticLibraries(staticLibrariesListPath); err != nil {
			return bosherr.WrapError(err, "Removing static libraries")
		}
	}

	return nil
}

func (boot bootstrap) comparePersistentDisk() error {
	settings := boot.settingsService.GetSettings()
	updateSettingsPath := filepath.Join(boot.platform.GetDirProvider().BoshDir(), "update_settings.json")

	if err := boot.checkLastMountedCid(settings); err != nil {
		return err
	}

	var updateSettings boshsettings.UpdateSettings

	if boot.platform.GetFs().FileExists(updateSettingsPath) {
		contents, err := boot.platform.GetFs().ReadFile(updateSettingsPath)
		if err != nil {
			return bosherr.WrapError(err, "Reading update_settings.json")
		}

		if err = json.Unmarshal(contents, &updateSettings); err != nil {
			return bosherr.WrapError(err, "Unmarshalling update_settings.json")
		}
	}

	for _, diskAssociation := range updateSettings.DiskAssociations {
		if _, ok := settings.PersistentDiskSettings(diskAssociation.DiskCID); !ok {
			return fmt.Errorf("Disk %s is not attached", diskAssociation.DiskCID)
		}
	}

	if len(settings.Disks.Persistent) > 1 {
		if len(settings.Disks.Persistent) > len(updateSettings.DiskAssociations) {
			return errors.New("Unexpected disk attached")
		}
	}

	return nil
}

func (boot bootstrap) setUserPasswords(env boshsettings.Env) error {
	password := env.GetPassword()

	if !env.GetKeepRootPassword() {
		err := boot.platform.SetUserPassword(boshsettings.RootUsername, password)
		if err != nil {
			return bosherr.WrapError(err, "Setting root password")
		}
	}

	err := boot.platform.SetUserPassword(boshsettings.VCAPUsername, password)
	if err != nil {
		return bosherr.WrapError(err, "Setting vcap password")
	}

	return nil
}

func (boot bootstrap) checkLastMountedCid(settings boshsettings.Settings) error {
	lastMountedCid, err := boot.lastMountedCid()
	if err != nil {
		return bosherr.WrapError(err, "Fetching last mounted disk CID")
	}

	if len(settings.Disks.Persistent) == 0 || lastMountedCid == "" {
		return nil
	}

	if _, ok := settings.PersistentDiskSettings(lastMountedCid); !ok {
		return fmt.Errorf("Attached disk disagrees with previous mount")
	}

	return nil
}

func (boot bootstrap) lastMountedCid() (string, error) {
	managedDiskSettingsPath := filepath.Join(boot.platform.GetDirProvider().BoshDir(), "managed_disk_settings.json")
	var lastMountedCid string

	if boot.platform.GetFs().FileExists(managedDiskSettingsPath) {
		contents, err := boot.platform.GetFs().ReadFile(managedDiskSettingsPath)
		if err != nil {
			return "", bosherr.WrapError(err, "Reading managed_disk_settings.json")
		}
		lastMountedCid = string(contents)

		return lastMountedCid, nil
	}

	return "", nil
}
