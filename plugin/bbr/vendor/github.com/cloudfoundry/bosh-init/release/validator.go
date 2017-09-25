package release

import (
	"errors"
	"fmt"
	"path"
	"sort"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type Validator interface {
	Validate(release Release) error
}

type validator struct {
	fs boshsys.FileSystem
}

func NewValidator(fs boshsys.FileSystem) Validator {
	return &validator{fs: fs}
}

func (v *validator) Validate(release Release) error {
	errs := []error{}

	err := v.validateReleaseName(release)
	if err != nil {
		errs = append(errs, bosherr.WrapError(err, "Validating release name"))
	}

	err = v.validateReleaseVersion(release)
	if err != nil {
		errs = append(errs, bosherr.WrapError(err, "Validating release version"))
	}

	err = v.validateReleaseJobs(release)
	if err != nil {
		errs = append(errs, bosherr.WrapError(err, "Validating release jobs"))
	}

	err = v.validateReleasePackages(release)
	if err != nil {
		errs = append(errs, bosherr.WrapError(err, "Validating release packages"))
	}

	if len(errs) > 0 {
		return bosherr.NewMultiError(errs...)
	}

	return nil
}

func (v *validator) validateReleaseName(release Release) error {
	if release.Name() == "" {
		return errors.New("Release name is missing")
	}

	return nil
}

func (v *validator) validateReleaseVersion(release Release) error {
	if release.Version() == "" {
		return errors.New("Release version is missing")
	}

	return nil
}

func (v *validator) validateReleaseJobs(release Release) error {
	errs := []error{}
	for _, job := range release.Jobs() {
		if job.Name == "" {
			errs = append(errs, errors.New("Job name is missing"))
		}

		if job.Fingerprint == "" {
			errs = append(errs, fmt.Errorf("Job '%s' fingerprint is missing", job.Name))
		}

		if job.SHA1 == "" {
			errs = append(errs, fmt.Errorf("Job '%s' sha1 is missing", job.Name))
		}

		monitPath := path.Join(job.ExtractedPath, "monit")
		if !v.fs.FileExists(monitPath) {
			errs = append(errs, fmt.Errorf("Job '%s' is missing monit file", job.Name))
		}

		for template := range job.Templates {
			templatePath := path.Join(job.ExtractedPath, "templates", template)
			if !v.fs.FileExists(templatePath) {
				errs = append(errs, fmt.Errorf("Job '%s' is missing template '%s'", job.Name, templatePath))
			}
		}

		for _, pkgName := range job.PackageNames {
			found := false
			for _, releasePackage := range release.Packages() {
				if releasePackage.Name == pkgName {
					found = true
					break
				}
			}
			if !found {
				errs = append(errs, fmt.Errorf("Job '%s' requires '%s' which is not in the release", job.Name, pkgName))
			}
		}
	}

	if len(errs) > 0 {
		return bosherr.NewMultiError(errs...)
	}

	return nil
}

func (v *validator) validateReleasePackages(release Release) error {
	errs := []error{}
	stemcells := map[string]string{}

	for _, pkg := range release.Packages() {
		if pkg.Name == "" {
			errs = append(errs, errors.New("Package name is missing"))
		}

		if pkg.Fingerprint == "" {
			errs = append(errs, fmt.Errorf("Package '%s' fingerprint is missing", pkg.Name))
		}

		if pkg.SHA1 == "" {
			errs = append(errs, fmt.Errorf("Package '%s' sha1 is missing", pkg.Name))
		}

		if release.IsCompiled() {
			if pkg.Stemcell == "" {
				errs = append(errs, fmt.Errorf("Compiled package '%s' stemcell is missing", pkg.Name))
			} else {
				stemcells[pkg.Stemcell] = pkg.Stemcell
			}
		}
	}

	if release.IsCompiled() && len(stemcells) > 1 {
		keys := []string{}
		for k := range stemcells {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		errs = append(errs, fmt.Errorf("Packages were compiled against different stemcells: %v", keys))
	}

	if len(errs) > 0 {
		return bosherr.NewMultiError(errs...)
	}

	return nil
}
