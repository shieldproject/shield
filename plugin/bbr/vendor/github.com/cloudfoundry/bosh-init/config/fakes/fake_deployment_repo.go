package fakes

type FakeDeploymentRepo struct {
	UpdateCurrentManifestSHA1 string
	UpdateCurrentErr          error

	findCurrentOutput deploymentRepoFindCurrentOutput
}

type deploymentRepoFindCurrentOutput struct {
	manifestSHA1 string
	found        bool
	err          error
}

func NewFakeDeploymentRepo() *FakeDeploymentRepo {
	return &FakeDeploymentRepo{}
}

func (r *FakeDeploymentRepo) UpdateCurrent(manifestSHA1 string) error {
	r.UpdateCurrentManifestSHA1 = manifestSHA1
	return r.UpdateCurrentErr
}

func (r *FakeDeploymentRepo) FindCurrent() (manifestSHA1 string, found bool, err error) {
	return r.findCurrentOutput.manifestSHA1, r.findCurrentOutput.found, r.findCurrentOutput.err
}

func (r *FakeDeploymentRepo) SetFindCurrentBehavior(manifestSHA1 string, found bool, err error) {
	r.findCurrentOutput = deploymentRepoFindCurrentOutput{
		manifestSHA1: manifestSHA1,
		found:        found,
		err:          err,
	}
}
