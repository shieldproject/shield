package infrastructure

import (
	"encoding/json"

	mapstruc "github.com/mitchellh/mapstructure"

	boshplat "github.com/cloudfoundry/bosh-agent/platform"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type Options struct {
	Settings SettingsOptions
}

type SettingsOptions struct {
	Sources       SourceOptionsSlice
	UseServerName bool
	UseRegistry   bool
}

// SourceOptionsSlice is used for unmarshalling different source types
type SourceOptionsSlice []SourceOptions

type SourceOptions interface {
	sourceOptionsInterface()
}

type HTTPSourceOptions struct {
	URI            string
	Headers        map[string]string
	UserDataPath   string
	InstanceIDPath string
	SSHKeysPath    string
}

func (o HTTPSourceOptions) sourceOptionsInterface() {}

type ConfigDriveSourceOptions struct {
	DiskPaths []string

	MetaDataPath string
	UserDataPath string

	SettingsPath string
}

func (o ConfigDriveSourceOptions) sourceOptionsInterface() {}

type FileSourceOptions struct {
	MetaDataPath string
	UserDataPath string

	SettingsPath string
}

func (o FileSourceOptions) sourceOptionsInterface() {}

type CDROMSourceOptions struct {
	FileName string
}

func (o CDROMSourceOptions) sourceOptionsInterface() {}

type InstanceMetadataSourceOptions struct {
	URI          string
	Headers      map[string]string
	SettingsPath string
}

func (o InstanceMetadataSourceOptions) sourceOptionsInterface() {}

type SettingsSourceFactory struct {
	options  SettingsOptions
	platform boshplat.Platform
	logger   boshlog.Logger
}

func NewSettingsSourceFactory(
	options SettingsOptions,
	platform boshplat.Platform,
	logger boshlog.Logger,
) SettingsSourceFactory {
	return SettingsSourceFactory{
		options:  options,
		platform: platform,
		logger:   logger,
	}
}

func (f SettingsSourceFactory) New() (boshsettings.Source, error) {
	if f.options.UseRegistry {
		return f.buildWithRegistry()
	}

	return f.buildWithoutRegistry()
}

func (f SettingsSourceFactory) buildWithRegistry() (boshsettings.Source, error) {
	var metadataServices []MetadataService

	digDNSResolver := NewDigDNSResolver(f.platform.GetRunner(), f.logger)
	resolver := NewRegistryEndpointResolver(digDNSResolver)

	for _, opts := range f.options.Sources {
		var metadataService MetadataService

		switch typedOpts := opts.(type) {
		case HTTPSourceOptions:
			metadataService = NewHTTPMetadataService(
				typedOpts.URI,
				typedOpts.Headers,
				typedOpts.UserDataPath,
				typedOpts.InstanceIDPath,
				typedOpts.SSHKeysPath,
				resolver,
				f.platform,
				f.logger,
			)

		case ConfigDriveSourceOptions:
			metadataService = NewConfigDriveMetadataService(
				resolver,
				f.platform,
				typedOpts.DiskPaths,
				typedOpts.MetaDataPath,
				typedOpts.UserDataPath,
				f.logger,
			)

		case FileSourceOptions:
			metadataService = NewFileMetadataService(
				typedOpts.MetaDataPath,
				typedOpts.UserDataPath,
				typedOpts.SettingsPath,
				f.platform.GetFs(),
				f.logger,
			)

		case CDROMSourceOptions:
			return nil, bosherr.Error("CDROM source is not supported when registry is used")

		case InstanceMetadataSourceOptions:
			return nil, bosherr.Error("Instance Metadata source is not supported when registry is used")
		}
		metadataServices = append(metadataServices, metadataService)
	}

	metadataService := NewMultiSourceMetadataService(metadataServices...)
	registryProvider := NewRegistryProvider(metadataService, f.platform, f.options.UseServerName, f.platform.GetFs(), f.logger)
	settingsSource := NewComplexSettingsSource(metadataService, registryProvider, f.logger)

	return settingsSource, nil
}

func (f SettingsSourceFactory) buildWithoutRegistry() (boshsettings.Source, error) {
	var settingsSources []boshsettings.Source

	for _, opts := range f.options.Sources {
		var settingsSource boshsettings.Source

		switch typedOpts := opts.(type) {
		case HTTPSourceOptions:
			return nil, bosherr.Error("HTTP source is not supported without registry")

		case ConfigDriveSourceOptions:
			settingsSource = NewConfigDriveSettingsSource(
				typedOpts.DiskPaths,
				typedOpts.MetaDataPath,
				typedOpts.SettingsPath,
				f.platform,
				f.logger,
			)

		case FileSourceOptions:
			return nil, bosherr.Error("File source is not supported without registry")

		case CDROMSourceOptions:
			settingsSource = NewCDROMSettingsSource(
				typedOpts.FileName,
				f.platform,
				f.logger,
			)

		case InstanceMetadataSourceOptions:
			settingsSource = NewInstanceMetadataSettingsSource(
				typedOpts.URI,
				typedOpts.Headers,
				typedOpts.SettingsPath,
				f.platform,
				f.logger,
			)
		}

		settingsSources = append(settingsSources, settingsSource)
	}

	return NewMultiSettingsSource(settingsSources...)
}

func (s *SourceOptionsSlice) UnmarshalJSON(data []byte) error {
	var maps []map[string]interface{}

	err := json.Unmarshal(data, &maps)
	if err != nil {
		return bosherr.WrapError(err, "Unmarshalling sources")
	}

	for _, m := range maps {
		if optType, ok := m["Type"]; ok {
			var err error
			var opts SourceOptions

			switch {
			case optType == "HTTP":
				var o HTTPSourceOptions
				err, opts = mapstruc.Decode(m, &o), o

			case optType == "InstanceMetadata":
				var o InstanceMetadataSourceOptions
				err, opts = mapstruc.Decode(m, &o), o

			case optType == "ConfigDrive":
				var o ConfigDriveSourceOptions
				err, opts = mapstruc.Decode(m, &o), o

			case optType == "File":
				var o FileSourceOptions
				err, opts = mapstruc.Decode(m, &o), o

			case optType == "CDROM":
				var o CDROMSourceOptions
				err, opts = mapstruc.Decode(m, &o), o

			default:
				err = bosherr.Errorf("Unknown source type '%s'", optType)
			}

			if err != nil {
				return bosherr.WrapErrorf(err, "Unmarshalling source type '%s'", optType)
			}
			*s = append(*s, opts)
		} else {
			return bosherr.Error("Missing source type")
		}
	}

	return nil
}
