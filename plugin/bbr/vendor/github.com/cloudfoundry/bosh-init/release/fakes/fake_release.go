package fakes

import (
	bireljob "github.com/cloudfoundry/bosh-init/release/job"
	birelpkg "github.com/cloudfoundry/bosh-init/release/pkg"
)

type FakeRelease struct {
	ReleaseName       string
	ReleaseVersion    string
	ReleaseJobs       []bireljob.Job
	ReleasePackages   []*birelpkg.Package
	ReleaseIsCompiled bool
	DeleteCalled      bool
	DeleteErr         error
}

func NewFakeRelease() *FakeRelease {
	return &FakeRelease{}
}

func New(name, version string) *FakeRelease {
	return &FakeRelease{
		ReleaseName:    name,
		ReleaseVersion: version,
	}
}

func (r *FakeRelease) Name() string { return r.ReleaseName }

func (r *FakeRelease) Version() string { return r.ReleaseVersion }

func (r *FakeRelease) Jobs() []bireljob.Job { return r.ReleaseJobs }

func (r *FakeRelease) Packages() []*birelpkg.Package { return r.ReleasePackages }

func (r *FakeRelease) FindJobByName(jobName string) (bireljob.Job, bool) {
	for _, job := range r.ReleaseJobs {
		if job.Name == jobName {
			return job, true
		}
	}

	return bireljob.Job{}, false
}

func (r *FakeRelease) Delete() error {
	r.DeleteCalled = true
	return r.DeleteErr
}

func (r *FakeRelease) Exists() bool {
	return !r.DeleteCalled
}

func (r *FakeRelease) IsCompiled() bool { return r.ReleaseIsCompiled }
