package platform

import (
	"time"

	"code.cloudfoundry.org/clock"
	"github.com/cloudfoundry/bosh-agent/infrastructure/devicepathresolver"

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
	platforms map[string]func() Platform
}

type Options struct {
	Linux LinuxOptions
}

func NewProvider(logger boshlog.Logger, dirProvider boshdirs.Provider, statsCollector boshstats.Collector, fs boshsys.FileSystem, options Options, bootstrapState *BootstrapState, clock clock.Clock, auditLogger AuditLogger) Provider {
	runner := boshsys.NewExecCmdRunner(logger)

	diskManagerOpts := boshdisk.LinuxDiskManagerOpts{
		BindMount:       options.Linux.BindMountPersistentDisk,
		PartitionerType: options.Linux.PartitionerType,
	}

	auditLogger.StartLogging()

	linuxDiskManager := boshdisk.NewLinuxDiskManager(logger, runner, fs, diskManagerOpts)
	udev := boshudev.NewConcreteUdevDevice(runner, logger)
	linuxCdrom := boshcdrom.NewLinuxCdrom("/dev/sr0", udev, runner)
	linuxCdutil := boshcdrom.NewCdUtil(dirProvider.SettingsDir(), fs, linuxCdrom, logger)

	compressor := boshcmd.NewTarballCompressor(runner, fs)
	copier := boshcmd.NewGenericCpCopier(fs, logger)

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
	opensuseNetManager := boshnet.NewOpensuseNetManager(fs, runner, ipResolver, interfaceConfigurationCreator, interfaceAddressesValidator, dnsValidator, arping, logger)

	windowsNetManager := boshnet.NewWindowsNetManager(
		runner,
		interfaceConfigurationCreator,
		boshnet.NewMACAddressDetector(),
		logger,
		clock,
		fs,
		dirProvider,
	)

	centosCertManager := boshcert.NewCentOSCertManager(fs, runner, 0, logger)
	ubuntuCertManager := boshcert.NewUbuntuCertManager(fs, runner, 60, logger)
	windowsCertManager := boshcert.NewWindowsCertManager(fs, runner, dirProvider, logger)
	opensuseCertManager := boshcert.NewOpensuseOSCertManager(fs, runner, 0, logger)

	routesSearcher := boshnet.NewRoutesSearcher(runner)
	defaultNetworkResolver := boshnet.NewDefaultNetworkResolver(routesSearcher, ipResolver)

	monitRetryable := NewMonitRetryable(runner)
	monitRetryStrategy := boshretry.NewAttemptRetryStrategy(10, 1*time.Second, monitRetryable, logger)

	var devicePathResolver devicepathresolver.DevicePathResolver
	switch options.Linux.DevicePathResolutionType {
	case "virtio":
		udev := boshudev.NewConcreteUdevDevice(runner, logger)
		idDevicePathResolver := devicepathresolver.NewIDDevicePathResolver(500*time.Millisecond, udev, fs)
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

	var centos = func() Platform {
		return NewLinuxPlatform(
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
			auditLogger,
		)
	}

	var ubuntu = func() Platform {
		return NewLinuxPlatform(
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
			auditLogger,
		)
	}

	var windows = func() Platform {
		return NewWindowsPlatform(
			statsCollector,
			fs,
			runner,
			dirProvider,
			windowsNetManager,
			windowsCertManager,
			devicePathResolver,
			logger,
			defaultNetworkResolver,
			auditLogger,
			uuidGenerator,
		)
	}

	var dummy = func() Platform {
		return NewDummyPlatform(
			statsCollector,
			fs,
			runner,
			dirProvider,
			devicePathResolver,
			logger,
			auditLogger,
		)
	}

	var opensuse = func() Platform {
		return NewLinuxPlatform(
			fs,
			runner,
			statsCollector,
			compressor,
			copier,
			dirProvider,
			vitalsService,
			linuxCdutil,
			linuxDiskManager,
			opensuseNetManager,
			opensuseCertManager,
			monitRetryStrategy,
			devicePathResolver,
			bootstrapState,
			options.Linux,
			logger,
			defaultNetworkResolver,
			uuidGenerator,
			auditLogger,
		)
	}

	return provider{
		platforms: map[string]func() Platform{
			"ubuntu":   ubuntu,
			"centos":   centos,
			"dummy":    dummy,
			"windows":  windows,
			"opensuse": opensuse,
		},
	}
}

func (p provider) Get(name string) (Platform, error) {
	plat, found := p.platforms[name]
	if !found {
		return nil, bosherror.Errorf("Platform %s could not be found", name)
	}
	return plat(), nil
}
