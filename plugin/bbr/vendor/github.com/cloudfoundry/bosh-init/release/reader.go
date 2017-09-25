package release

import (
	"github.com/pivotal-golang/yaml"
	"os"
	"path"

	bireljob "github.com/cloudfoundry/bosh-init/release/job"
	birelmanifest "github.com/cloudfoundry/bosh-init/release/manifest"
	birelpkg "github.com/cloudfoundry/bosh-init/release/pkg"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type reader struct {
	tarFilePath          string
	extractedReleasePath string
	fs                   boshsys.FileSystem
	extractor            boshcmd.Compressor
}

type Reader interface {
	Read() (Release, error)
}

func NewReader(
	tarFilePath string,
	extractedReleasePath string,
	fs boshsys.FileSystem,
	extractor boshcmd.Compressor,
) Reader {
	return &reader{
		tarFilePath:          tarFilePath,
		extractedReleasePath: extractedReleasePath,
		fs:                   fs,
		extractor:            extractor,
	}
}

func (r *reader) Read() (Release, error) {
	err := r.extractor.DecompressFileToDir(r.tarFilePath, r.extractedReleasePath, boshcmd.CompressorOptions{})
	if err != nil {
		return nil, bosherr.WrapError(err, "Extracting release")
	}

	releaseManifestPath := path.Join(r.extractedReleasePath, "release.MF")
	releaseManifestBytes, err := r.fs.ReadFile(releaseManifestPath)
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Reading release manifest '%s'", releaseManifestPath)
	}

	var manifest birelmanifest.Manifest
	err = yaml.Unmarshal(releaseManifestBytes, &manifest)
	if err != nil {
		return nil, bosherr.WrapError(err, "Parsing release manifest")
	}

	release, err := r.newReleaseFromManifest(manifest)
	if err != nil {
		return nil, bosherr.WrapError(err, "Constructing release from manifest")
	}

	return release, nil
}

func (r *reader) newReleaseFromManifest(releaseManifest birelmanifest.Manifest) (Release, error) {
	errors := []error{}
	packages, isCompiledRelease, err := r.newPackagesFromManifestPackages(releaseManifest)
	if err != nil {
		errors = append(errors, bosherr.WrapError(err, "Constructing packages from manifest"))
	}

	jobs, err := r.newJobsFromManifestJobs(packages, releaseManifest.Jobs)
	if err != nil {
		errors = append(errors, bosherr.WrapError(err, "Constructing jobs from manifest"))
	}

	if len(errors) > 0 {
		return nil, bosherr.NewMultiError(errors...)
	}

	release := &release{
		name:    releaseManifest.Name,
		version: releaseManifest.Version,

		jobs:     jobs,
		packages: packages,

		extractedPath: r.extractedReleasePath,
		fs:            r.fs,
		isCompiled:    isCompiledRelease,
	}

	return release, nil
}

func (r *reader) newJobsFromManifestJobs(packages []*birelpkg.Package, manifestJobs []birelmanifest.JobRef) ([]bireljob.Job, error) {
	jobs := []bireljob.Job{}
	errors := []error{}
	for _, manifestJob := range manifestJobs {
		extractedJobPath := path.Join(r.extractedReleasePath, "extracted_jobs", manifestJob.Name)
		err := r.fs.MkdirAll(extractedJobPath, os.ModeDir|0700)
		if err != nil {
			errors = append(errors, bosherr.WrapError(err, "Creating extracted job path"))
			continue
		}

		jobArchivePath := path.Join(r.extractedReleasePath, "jobs", manifestJob.Name+".tgz")
		jobReader := bireljob.NewReader(jobArchivePath, extractedJobPath, r.extractor, r.fs)
		job, err := jobReader.Read()
		if err != nil {
			errors = append(errors, bosherr.WrapErrorf(err, "Reading job '%s' from archive", manifestJob.Name))
			continue
		}

		job.Fingerprint = manifestJob.Fingerprint
		job.SHA1 = manifestJob.SHA1
		for _, pkgName := range job.PackageNames {
			pkg, found := r.findPackageByName(packages, pkgName)
			if !found {
				return []bireljob.Job{}, bosherr.Errorf("Package not found: '%s'", pkgName)
			}
			job.Packages = append(job.Packages, pkg)
		}

		jobs = append(jobs, job)
	}

	if len(errors) > 0 {
		return []bireljob.Job{}, bosherr.NewMultiError(errors...)
	}

	return jobs, nil
}

func (r *reader) findPackageByName(packages []*birelpkg.Package, pkgName string) (*birelpkg.Package, bool) {
	for _, pkg := range packages {
		if pkg.Name == pkgName {
			return pkg, true
		}
	}
	return nil, false
}

func (r *reader) newPackagesFromManifestPackages(releaseManifest birelmanifest.Manifest) ([]*birelpkg.Package, bool, error) {

	manifestPackages := releaseManifest.Packages
	isCompiledPackage := false

	if len(releaseManifest.Packages) > 0 && len(releaseManifest.CompiledPackages) > 0 {
		return []*birelpkg.Package{}, isCompiledPackage, bosherr.Errorf("Release '%s' contains compiled and non-compiled pacakges", releaseManifest.Name)
	} else if len(releaseManifest.CompiledPackages) > 0 {
		manifestPackages = releaseManifest.CompiledPackages
		isCompiledPackage = true
	}

	packages := []*birelpkg.Package{}
	errors := []error{}
	packageRepo := &birelpkg.PackageRepo{}

	packagesDirectory := "packages"
	if isCompiledPackage {
		packagesDirectory = "compiled_packages"
	}

	for _, manifestPackage := range manifestPackages {
		pkg := packageRepo.FindOrCreatePackage(manifestPackage.Name)

		extractedPackagePath := path.Join(r.extractedReleasePath, "extracted_packages", manifestPackage.Name)
		err := r.fs.MkdirAll(extractedPackagePath, os.ModeDir|0700)
		if err != nil {
			errors = append(errors, bosherr.WrapError(err, "Creating extracted package path"))
			continue
		}

		packageArchivePath := path.Join(r.extractedReleasePath, packagesDirectory, manifestPackage.Name+".tgz")
		err = r.extractor.DecompressFileToDir(packageArchivePath, extractedPackagePath, boshcmd.CompressorOptions{})
		if err != nil {
			errors = append(errors, bosherr.WrapErrorf(err, "Extracting package '%s'", manifestPackage.Name))
			continue
		}

		pkg.Fingerprint = manifestPackage.Fingerprint
		pkg.SHA1 = manifestPackage.SHA1
		pkg.ExtractedPath = extractedPackagePath
		pkg.ArchivePath = packageArchivePath

		if isCompiledPackage {
			pkg.Stemcell = manifestPackage.Stemcell
		}

		pkg.Dependencies = []*birelpkg.Package{}
		for _, manifestPackageName := range manifestPackage.Dependencies {
			pkg.Dependencies = append(pkg.Dependencies, packageRepo.FindOrCreatePackage(manifestPackageName))
		}

		packages = append(packages, pkg)
	}

	if len(errors) > 0 {
		return []*birelpkg.Package{}, isCompiledPackage, bosherr.NewMultiError(errors...)
	}

	return packages, isCompiledPackage, nil
}
