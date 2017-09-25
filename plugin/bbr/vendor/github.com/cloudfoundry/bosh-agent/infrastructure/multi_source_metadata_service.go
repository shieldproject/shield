package infrastructure

import (
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
)

type MultiSourceMetadataService struct {
	Services        []MetadataService
	selectedService MetadataService
}

func NewMultiSourceMetadataService(services ...MetadataService) MetadataService {
	return &MultiSourceMetadataService{Services: services}
}

func (ms *MultiSourceMetadataService) GetPublicKey() (string, error) {
	return ms.getSelectedService().GetPublicKey()
}

func (ms *MultiSourceMetadataService) GetInstanceID() (string, error) {
	return ms.getSelectedService().GetInstanceID()
}

func (ms *MultiSourceMetadataService) GetServerName() (string, error) {
	return ms.getSelectedService().GetServerName()
}

func (ms *MultiSourceMetadataService) GetRegistryEndpoint() (string, error) {
	return ms.getSelectedService().GetRegistryEndpoint()
}

func (ms *MultiSourceMetadataService) GetNetworks() (boshsettings.Networks, error) {
	return ms.getSelectedService().GetNetworks()
}

func (ms *MultiSourceMetadataService) IsAvailable() bool {
	return true
}

func (ms *MultiSourceMetadataService) getSelectedService() MetadataService {
	if ms.selectedService == nil {
		for _, service := range ms.Services {
			if service.IsAvailable() {
				ms.selectedService = service
				break
			}
		}
	}
	return ms.selectedService
}
