package release

import (
	boshsys "github.com/cloudfoundry/bosh-utils/system"

	bireljob "github.com/cloudfoundry/bosh-cli/release/job"
	birellic "github.com/cloudfoundry/bosh-cli/release/license"
	birelman "github.com/cloudfoundry/bosh-cli/release/manifest"
	birelpkg "github.com/cloudfoundry/bosh-cli/release/pkg"
)

type release struct {
	name    string
	version string

	commitHash         string
	uncommittedChanges bool

	jobs         []*bireljob.Job
	packages     []*birelpkg.Package
	compiledPkgs []*birelpkg.CompiledPackage
	license      *birellic.License

	extractedPath string
	fs            boshsys.FileSystem
}

func NewRelease(
	name string,
	version string,
	commitHash string,
	uncommittedChanges bool,
	jobs []*bireljob.Job,
	packages []*birelpkg.Package,
	compiledPkgs []*birelpkg.CompiledPackage,
	license *birellic.License,
	extractedPath string,
	fs boshsys.FileSystem,
) Release {
	return &release{
		name:    name,
		version: version,

		commitHash:         commitHash,
		uncommittedChanges: uncommittedChanges,

		jobs:         jobs,
		packages:     packages,
		compiledPkgs: compiledPkgs,
		license:      license,

		extractedPath: extractedPath,
		fs:            fs,
	}
}

func (r *release) Name() string        { return r.name }
func (r *release) SetName(name string) { r.name = name }

func (r *release) Version() string           { return r.version }
func (r *release) SetVersion(version string) { r.version = version }

func (r *release) SetCommitHash(commitHash string)    { r.commitHash = commitHash }
func (r *release) SetUncommittedChanges(changes bool) { r.uncommittedChanges = changes }

func (r *release) CommitHashWithMark(suffix string) string {
	if r.uncommittedChanges {
		return r.commitHash + suffix
	}
	return r.commitHash
}

func (r *release) Jobs() []*bireljob.Job                         { return r.jobs }
func (r *release) Packages() []*birelpkg.Package                 { return r.packages }
func (r *release) CompiledPackages() []*birelpkg.CompiledPackage { return r.compiledPkgs }
func (r *release) License() *birellic.License                    { return r.license }

func (r *release) IsCompiled() bool { return len(r.compiledPkgs) > 0 }

func (r *release) FindJobByName(jobName string) (bireljob.Job, bool) {
	for _, job := range r.jobs {
		if job.Name() == jobName {
			return *job, true
		}
	}
	return bireljob.Job{}, false
}

func (r *release) Manifest() birelman.Manifest {
	var jobRefs []birelman.JobRef

	for _, job := range r.Jobs() {
		jobRefs = append(jobRefs, birelman.JobRef{
			Name:        job.Name(),
			Version:     job.Fingerprint(),
			Fingerprint: job.Fingerprint(),
			SHA1:        job.ArchiveSHA1(),
		})
	}

	var packageRefs []birelman.PackageRef

	for _, pkg := range r.Packages() {
		packageRefs = append(packageRefs, birelman.PackageRef{
			Name:         pkg.Name(),
			Version:      pkg.Fingerprint(),
			Fingerprint:  pkg.Fingerprint(),
			SHA1:         pkg.ArchiveSHA1(),
			Dependencies: pkg.DependencyNames(),
		})
	}

	var compiledPkgRefs []birelman.CompiledPackageRef

	for _, compiledPkg := range r.CompiledPackages() {
		compiledPkgRefs = append(compiledPkgRefs, birelman.CompiledPackageRef{
			Name:          compiledPkg.Name(),
			Version:       compiledPkg.Fingerprint(),
			Fingerprint:   compiledPkg.Fingerprint(),
			SHA1:          compiledPkg.ArchiveSHA1(),
			OSVersionSlug: compiledPkg.OSVersionSlug(),
			Dependencies:  compiledPkg.DependencyNames(),
		})
	}

	var licenseRef *birelman.LicenseRef

	lic := r.License()

	if lic != nil {
		licenseRef = &birelman.LicenseRef{
			Version:     lic.Fingerprint(),
			Fingerprint: lic.Fingerprint(),
			SHA1:        lic.ArchiveSHA1(),
		}
	}

	return birelman.Manifest{
		Name:    r.name,
		Version: r.version,

		CommitHash:         r.commitHash,
		UncommittedChanges: r.uncommittedChanges,

		Jobs:         jobRefs,
		Packages:     packageRefs,
		CompiledPkgs: compiledPkgRefs,
		License:      licenseRef,
	}
}

func (r *release) Build(devIndicies, finalIndicies ArchiveIndicies) error {
	for _, job := range r.Jobs() {
		err := job.Build(devIndicies.Jobs, finalIndicies.Jobs)
		if err != nil {
			return err
		}
	}

	for _, pkg := range r.Packages() {
		err := pkg.Build(devIndicies.Packages, finalIndicies.Packages)
		if err != nil {
			return err
		}
	}

	if r.License() != nil {
		err := r.License().Build(devIndicies.Licenses, finalIndicies.Licenses)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *release) Finalize(finalIndicies ArchiveIndicies) error {
	for _, job := range r.Jobs() {
		err := job.Finalize(finalIndicies.Jobs)
		if err != nil {
			return err
		}
	}

	for _, pkg := range r.Packages() {
		err := pkg.Finalize(finalIndicies.Packages)
		if err != nil {
			return err
		}
	}

	if r.License() != nil {
		err := r.License().Finalize(finalIndicies.Licenses)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *release) CopyWith(jobs []*bireljob.Job, packages []*birelpkg.Package, license *birellic.License, compiledPkgs []*birelpkg.CompiledPackage) Release {
	return &release{
		name:    r.name,
		version: r.version,

		commitHash:         r.commitHash,
		uncommittedChanges: r.uncommittedChanges,

		jobs:         jobs,
		packages:     packages,
		compiledPkgs: compiledPkgs,
		license:      license,

		extractedPath: r.extractedPath,
		fs:            r.fs,
	}
}

// CleanUp removes the extracted release.
func (r *release) CleanUp() error {
	var anyErr error

	for _, job := range r.Jobs() {
		err := job.CleanUp()
		if err != nil {
			anyErr = err
		}
	}

	for _, pkg := range r.Packages() {
		err := pkg.CleanUp()
		if err != nil {
			anyErr = err
		}
	}

	if r.fs != nil && len(r.extractedPath) > 0 {
		err := r.fs.RemoveAll(r.extractedPath)
		if err != nil {
			anyErr = err
		}
	}

	return anyErr
}
