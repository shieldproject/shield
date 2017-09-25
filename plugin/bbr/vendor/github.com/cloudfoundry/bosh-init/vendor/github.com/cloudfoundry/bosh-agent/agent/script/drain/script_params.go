package drain

import (
	"sort"

	boshas "github.com/cloudfoundry/bosh-agent/agent/applier/applyspec"
)

//go:generate counterfeiter . ScriptParams

type ScriptParams interface {
	JobChange() (change string)
	HashChange() (change string)
	UpdatedPackages() (pkgs []string)
	JobState() (string, error)
	JobNextState() (string, error)

	// ToStatusParams derives a new set of script params that can be used to do the
	// status check call on a dynamic drain script.
	ToStatusParams() ScriptParams
}

type concreteScriptParams struct {
	jobChange       string
	hashChange      string
	updatedPackages []string

	oldSpec boshas.V1ApplySpec
	newSpec *boshas.V1ApplySpec
}

func NewShutdownParams(
	oldSpec boshas.V1ApplySpec,
	newSpec *boshas.V1ApplySpec,
) ScriptParams {
	return concreteScriptParams{
		jobChange:       "job_shutdown",
		hashChange:      "hash_unchanged",
		updatedPackages: []string{},
		oldSpec:         oldSpec,
		newSpec:         newSpec,
	}
}

func NewUpdateParams(oldSpec, newSpec boshas.V1ApplySpec) ScriptParams {
	p := concreteScriptParams{
		oldSpec: oldSpec,
		newSpec: &newSpec,
	}

	switch {
	case len(oldSpec.Jobs()) == 0:
		p.jobChange = "job_new"
	case oldSpec.JobSpec.Sha1 == newSpec.JobSpec.Sha1:
		p.jobChange = "job_unchanged"
	default:
		p.jobChange = "job_changed"
	}

	switch {
	case oldSpec.ConfigurationHash == "":
		p.hashChange = "hash_new"
	case oldSpec.ConfigurationHash == newSpec.ConfigurationHash:
		p.hashChange = "hash_unchanged"
	default:
		p.hashChange = "hash_changed"
	}

	for _, pkg := range newSpec.PackageSpecs {
		currentPkg, found := oldSpec.PackageSpecs[pkg.Name]
		switch {
		case !found:
			p.updatedPackages = append(p.updatedPackages, pkg.Name)
		case currentPkg.Sha1 != pkg.Sha1:
			p.updatedPackages = append(p.updatedPackages, pkg.Name)
		}
	}
	sort.Strings(p.updatedPackages)

	return p
}

func (p concreteScriptParams) JobChange() (change string)       { return p.jobChange }
func (p concreteScriptParams) HashChange() (change string)      { return p.hashChange }
func (p concreteScriptParams) UpdatedPackages() (pkgs []string) { return p.updatedPackages }

func (p concreteScriptParams) JobState() (string, error) {
	return newPresentedJobState(&p.oldSpec).MarshalToJSONString()
}

func (p concreteScriptParams) JobNextState() (string, error) {
	return newPresentedJobState(p.newSpec).MarshalToJSONString()
}

func (p concreteScriptParams) ToStatusParams() ScriptParams {
	return concreteScriptParams{
		jobChange:       "job_check_status",
		hashChange:      "hash_unchanged",
		updatedPackages: []string{},
		oldSpec:         p.oldSpec,
		newSpec:         nil,
	}
}
