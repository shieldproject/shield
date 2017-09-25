package applyspec

import (
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
)

type V1Service interface {
	Get() (V1ApplySpec, error)
	Set(V1ApplySpec) error
	PopulateDHCPNetworks(V1ApplySpec, boshsettings.Settings) (V1ApplySpec, error)
}
