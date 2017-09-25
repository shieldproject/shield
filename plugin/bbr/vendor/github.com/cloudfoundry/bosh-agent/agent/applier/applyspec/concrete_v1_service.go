package applyspec

import (
	"encoding/json"

	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type concreteV1Service struct {
	fs           boshsys.FileSystem
	specFilePath string
}

func NewConcreteV1Service(fs boshsys.FileSystem, specFilePath string) V1Service {
	return concreteV1Service{fs: fs, specFilePath: specFilePath}
}

// Get reads and marshals the file contents.
func (s concreteV1Service) Get() (V1ApplySpec, error) {
	var spec V1ApplySpec

	if !s.fs.FileExists(s.specFilePath) {
		return spec, nil
	}

	contents, err := s.fs.ReadFile(s.specFilePath)
	if err != nil {
		return spec, bosherr.WrapError(err, "Reading json spec file")
	}

	err = json.Unmarshal([]byte(contents), &spec)
	if err != nil {
		return spec, bosherr.WrapError(err, "Unmarshalling json spec file")
	}

	return spec, nil
}

// Set unmarshals and writes to the file.
func (s concreteV1Service) Set(spec V1ApplySpec) error {
	specBytes, err := json.Marshal(spec)
	if err != nil {
		return bosherr.WrapError(err, "Marshalling apply spec")
	}

	err = s.fs.WriteFile(s.specFilePath, specBytes)
	if err != nil {
		return bosherr.WrapError(err, "Writing spec to disk")
	}

	return nil
}

func (s concreteV1Service) PopulateDHCPNetworks(spec V1ApplySpec, settings boshsettings.Settings) (V1ApplySpec, error) {
	for networkName, networkSpec := range spec.NetworkSpecs {
		// Skip 'local' network since for vsphere/vcloud networks
		// are generated incorrectly by the bosh_cli_plugin_micro/bosh-release;
		// can be removed with new bosh micro CLI
		if networkName == "local" && networkSpec.Fields["ip"] == "127.0.0.1" {
			continue
		}

		network, ok := settings.Networks[networkName]
		if !ok {
			return V1ApplySpec{}, bosherr.Errorf("Network '%s' is not found in settings", networkName)
		}

		if !network.IsDHCP() {
			continue
		}

		spec.NetworkSpecs[networkName] = networkSpec.PopulateIPInfo(
			network.IP,
			network.Netmask,
			network.Gateway,
		)
	}

	return spec, nil
}
