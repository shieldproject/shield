package settings

import (
	"encoding/json"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type Service interface {
	LoadSettings() error

	// GetSettings does not return error because without settings Agent cannot start.
	GetSettings() Settings

	PublicSSHKeyForUsername(string) (string, error)

	InvalidateSettings() error
}

const settingsServiceLogTag = "settingsService"

type settingsService struct {
	fs                     boshsys.FileSystem
	settingsPath           string
	settings               Settings
	settingsSource         Source
	defaultNetworkResolver DefaultNetworkResolver
	logger                 boshlog.Logger
}

type DefaultNetworkResolver interface {
	// Ideally we would find a network based on a MAC address
	// but current CPI implementations do not include it
	GetDefaultNetwork() (Network, error)
}

func NewService(
	fs boshsys.FileSystem,
	settingsPath string,
	settingsSource Source,
	defaultNetworkResolver DefaultNetworkResolver,
	logger boshlog.Logger,
) (service Service) {
	return &settingsService{
		fs:                     fs,
		settingsPath:           settingsPath,
		settings:               Settings{},
		settingsSource:         settingsSource,
		defaultNetworkResolver: defaultNetworkResolver,
		logger:                 logger,
	}
}

func (s *settingsService) PublicSSHKeyForUsername(username string) (string, error) {
	return s.settingsSource.PublicSSHKeyForUsername(username)
}

func (s *settingsService) LoadSettings() error {
	s.logger.Debug(settingsServiceLogTag, "Loading settings from fetcher")

	newSettings, fetchErr := s.settingsSource.Settings()
	if fetchErr != nil {
		s.logger.Error(settingsServiceLogTag, "Failed loading settings via fetcher: %v", fetchErr)

		existingSettingsJSON, readError := s.fs.ReadFile(s.settingsPath)
		if readError != nil {
			s.logger.Error(settingsServiceLogTag, "Failed reading settings from file %s", readError.Error())
			return bosherr.WrapError(fetchErr, "Invoking settings fetcher")
		}

		s.logger.Debug(settingsServiceLogTag, "Successfully read settings from file")

		err := json.Unmarshal(existingSettingsJSON, &s.settings)
		if err != nil {
			s.logger.Error(settingsServiceLogTag, "Failed unmarshalling settings from file %s", err.Error())
			return bosherr.WrapError(fetchErr, "Invoking settings fetcher")
		}

		return nil
	}

	s.logger.Debug(settingsServiceLogTag, "Successfully received settings from fetcher")
	s.settings = newSettings

	newSettingsJSON, err := json.Marshal(newSettings)
	if err != nil {
		return bosherr.WrapError(err, "Marshalling settings json")
	}

	err = s.fs.WriteFile(s.settingsPath, newSettingsJSON)
	if err != nil {
		return bosherr.WrapError(err, "Writing setting json")
	}

	return nil
}

// GetSettings returns setting even if it fails to resolve IPs for dynamic networks.
func (s *settingsService) GetSettings() Settings {
	for networkName, network := range s.settings.Networks {
		if !network.IsDHCP() {
			continue
		}

		resolvedNetwork, err := s.resolveNetwork(network)
		if err != nil {
			break
		}

		s.settings.Networks[networkName] = resolvedNetwork
	}

	return s.settings
}

func (s *settingsService) InvalidateSettings() error {
	err := s.fs.RemoveAll(s.settingsPath)
	if err != nil {
		return bosherr.WrapError(err, "Removing settings file")
	}

	return nil
}

func (s *settingsService) resolveNetwork(network Network) (Network, error) {
	// Ideally this would be GetNetworkByMACAddress(mac string)
	// Currently, we are relying that if the default network does not contain
	// the MAC adddress the InterfaceConfigurationCreator will fail.
	resolvedNetwork, err := s.defaultNetworkResolver.GetDefaultNetwork()
	if err != nil {
		s.logger.Error(settingsServiceLogTag, "Failed retrieving default network %s", err.Error())
		return Network{}, bosherr.WrapError(err, "Failed retrieving default network")
	}

	// resolvedNetwork does not have all information for a network
	network.IP = resolvedNetwork.IP
	network.Netmask = resolvedNetwork.Netmask
	network.Gateway = resolvedNetwork.Gateway
	network.Resolved = true

	return network, nil
}
