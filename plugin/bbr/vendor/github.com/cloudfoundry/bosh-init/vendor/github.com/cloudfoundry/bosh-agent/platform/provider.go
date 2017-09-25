package platform

import (
	"time"

	"github.com/cloudfoundry/bosh-agent/infrastructure/devicepathresolver"
	"github.com/pivotal-golang/clock"

	boshcdrom "github.com/cloudfoundry/bosh-agent/platform/cdrom"
	boshcert "github.com/cloudfoundry/bosh-agent/platform/cert"
	boshdisk "github.com/cloudfoundry/bosh-agent/platform/disk"
	boshnet "github.com/cloudfoundry/bosh-agent/platform/net"
	bosharp "github.com/cloudfoundry/bosh-agent/platform/net/arp"
	boship "github.com/cloudfoundry/bosh-agent/platform/net/ip"
	boshstats "github.com/cloudfoundry/bosh-agent/platform/stats"
	boshudev "github.com/cloudfoundry/bosh-agent/platform/udevdevice"
	boshvitals "github.com/cloudfoundry/bosh-agent/platform/vitals"
	boshdirs "github.com/cloudfoundry/bosh-agent/settings/directories"
	bosherror "github.com/cloudfoundry/bosh-utils/errors"
	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshretry "github.com/cloudfoundry/bosh-utils/retrystrategy"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
)

const (
	ArpIterations          = 20
	ArpIterationDelay      = 5 * time.Second
	ArpInterfaceCheckDelay = 100 * time.Millisecond
)

const (
	SigarStatsCollectionInterval = 10 * time.Second
)

type Provider interface {
	Get(name string) (Platform, error)
}

type provider struct {
	platforms map[string]Platform
}

type Options struct {
	Linux LinuxOptions
}

func NewProvider(logger boshlog.Logger, dirProvider boshdirs.Provider, statsCollector boshstats.Collector, fs boshsys.FileSystem, options Options, bootstrapState *BootstrapState, clock clock.Clock) Provider {
	runner := boshsys.NewExecCmdRunner(logger)
	linuxDiskManager := boshdisk.NewLinuxDiskManager(logger, runner, fs, options.Linux.BindMountPersistentDisk)

	udev := boshudev.NewConcreteUdevDevice(runner, logger)
	linuxCdrom := boshcdrom.NewLinuxCdrom("/dev/sr0", udev, runner)
	linuxCdutil := boshcdrom.NewCdUtil(dirProvider.SettingsDir(), fs, linuxCdrom, logger)

	compressor := boshcmd.NewTarballCompressor(runner, fs)
	copier := boshcmd.NewCpCopier(runner, fs, logger)

	// Kick of stats collection as soon as possible
	statsCollector.StartCollecting(SigarStatsCollectionInterval, nil)

	vitalsService := boshvitals.NewService(statsCollector, dirProvider)

	ipResolver := boship.NewResolver(boship.NetworkInterfaceToAddrsFunc)

	arping := bosharp.NewArping(runner, fs, logger, ArpIterations, ArpIterationDelay, ArpInterfaceCheckDelay)
	interfaceConfigurationCreator := boshnet.NewInterfaceConfigurationCreator(logger)

	interfaceAddressesProvider := boship.NewSystemInterfaceAddressesProvider()
	interfaceAddressesValidator := boship.NewInterfaceAddressesValidator(interfaceAddressesProvider)
	dnsValidator := boshnet.NewDNSValidator(fs)

	centosNetManager := boshnet.NewCentosNetManager(fs, runner, ipResolver, interfaceConfigurationCreator, interfaceAddressesValidator, dnsValidator, arping, logger)
	ubuntuNetManager := boshnet.NewUbuntuNetManager(fs, runner, ipResolver, interfaceConfigurationCreator, interfaceAddressesValidator, dnsValidator, arping, logger)

	windowsNetManager := boshnet.NewWindowsNetManager(runner, interfaceConfigurationCreator, boshnet.NewMACAddressDetector(), logger, clock)

	centosCertManager := boshcert.NewCentOSCertManager(fs, runner, 0, logger)
	ubuntuCertManager := boshcert.NewUbuntuCertManager(fs, runner, 60, logger)

	routesSearcher := boshnet.NewRoutesSearcher(runner)
	defaultNetworkResolver := boshnet.NewDefaultNetworkResolver(routesSearcher, ipResolver)

	monitRetryable := NewMonitRetryable(runner)
	monitRetryStrategy := boshretry.NewAttemptRetryStrategy(10, 1*time.Second, monitRetryable, logger)

	var devicePathResolver devicepathresolver.DevicePathResolver
	switch options.Linux.DevicePathResolutionType {
	case "virtio":
		udev := boshudev.NewConcreteUdevDevice(runner, logger)
		idDevicePathResolver := devicepathresolver.NewIDDevicePathResolver(500*time.Millisecond, options.Linux.VirtioDevicePrefix, udev, fs)
		mappedDevicePathResolver := devicepathresolver.NewMappedDevicePathResolver(30000*time.Millisecond, fs)
		devicePathResolver = devicepathresolver.NewVirtioDevicePathResolver(idDevicePathResolver, mappedDevicePathResolver, logger)
	case "scsi":
		scsiIDPathResolver := devicepathresolver.NewSCSIIDDevicePathResolver(50000*time.Millisecond, fs, logger)
		scsiVolumeIDPathResolver := devicepathresolver.NewSCSIVolumeIDDevicePathResolver(500*time.Millisecond, fs)
		scsiLunPathResolver := devicepathresolver.NewSCSILunDevicePathResolver(50000*time.Millisecond, fs, logger)
		devicePathResolver = devicepathresolver.NewScsiDevicePathResolver(scsiVolumeIDPathResolver, scsiIDPathResolver, scsiLunPathResolver)
	default:
		devicePathResolver = devicepathresolver.NewIdentityDevicePathResolver()
	}

	uuidGenerator := boshuuid.NewGenerator()

	centos := NewLinuxPlatform(
		fs,
		runner,
		statsCollector,
		compressor,
		copier,
		dirProvider,
		vitalsService,
		linuxCdutil,
		linuxDiskManager,
		centosNetManager,
		centosCertManager,
		monitRetryStrategy,
		devicePathResolver,
		bootstrapState,
		options.Linux,
		logger,
		defaultNetworkResolver,
		uuidGenerator,
	)

	ubuntu := NewLinuxPlatform(
		fs,
		runner,
		statsCollector,
		compressor,
		copier,
		dirProvider,
		vitalsService,
		linuxCdutil,
		linuxDiskManager,
		ubuntuNetManager,
		ubuntuCertManager,
		monitRetryStrategy,
		devicePathResolver,
		bootstrapState,
		options.Linux,
		logger,
		defaultNetworkResolver,
		uuidGenerator,
	)

	windows := NewWindowsPlatform(
		statsCollector,
		fs,
		runner,
		dirProvider,
		windowsNetManager,
		devicePathResolver,
		logger,
		defaultNetworkResolver,
	)

	return provider{
		platforms: map[string]Platform{
			"ubuntu":  ubuntu,
			"centos":  centos,
			"dummy":   NewDummyPlatform(statsCollector, fs, runner, dirProvider, devicePathResolver, logger),
			"windows": windows,
		},
	}
}

func (p provider) Get(name string) (Platform, error) {
	plat, found := p.platforms[name]
	if !found {
		return nil, bosherror.Errorf("Platform %s could not be found", name)
	}
	return plat, nil
}
