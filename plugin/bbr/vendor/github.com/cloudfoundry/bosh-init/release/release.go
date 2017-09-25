package release

import (
	bireljob "github.com/cloudfoundry/bosh-init/release/job"
	birelpkg "github.com/cloudfoundry/bosh-init/release/pkg"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type release struct {
	name          string
	version       string
	jobs          []bireljob.Job
	packages      []*birelpkg.Package
	extractedPath string
	fs            boshsys.FileSystem
	isCompiled    bool
}

type Release interface {
	Name() string
	Version() string
	Jobs() []bireljob.Job
	Packages() []*birelpkg.Package
	FindJobByName(jobName string) (job bireljob.Job, found bool)
	Delete() error
	Exists() bool
	IsCompiled() bool
}

func NewRelease(
	name string,
	version string,
	jobs []bireljob.Job,
	packages []*birelpkg.Package,
	extractedPath string,
	fs boshsys.FileSystem,
	isCompiled bool,
) Release {
	return &release{
		name:          name,
		version:       version,
		jobs:          jobs,
		packages:      packages,
		extractedPath: extractedPath,
		fs:            fs,
		isCompiled:    isCompiled,
	}
}

func (r *release) Name() string { return r.name }

func (r *release) Version() string { return r.version }

func (r *release) Jobs() []bireljob.Job { return r.jobs }

func (r *release) Packages() []*birelpkg.Package { return r.packages }

func (r *release) FindJobByName(jobName string) (bireljob.Job, bool) {
	for _, job := range r.jobs {
		if job.Name == jobName {
			return job, true
		}
	}

	return bireljob.Job{}, false
}

// Delete removes the extracted release code.
// Since packages and jobs are under the same path, they will be deleted too.
func (r *release) Delete() error {
	return r.fs.RemoveAll(r.extractedPath)
}

// Exists returns false after Delete (or if extractedPath does not exist)
func (r *release) Exists() bool {
	return r.fs.FileExists(r.extractedPath)
}

func (r *release) IsCompiled() bool { return r.isCompiled }
