package fakes

import (
	models "github.com/cloudfoundry/bosh-agent/agent/applier/models"
)

type FakeApplier struct {
	PreparedJobs []models.Job
	PrepareError error

	AppliedJobs []models.Job
	ApplyError  error

	ConfiguredJobs       []models.Job
	ConfiguredJobIndices []int
	ConfigureError       error

	KeepOnlyJobs []models.Job
	KeepOnlyErr  error
}

func NewFakeApplier() *FakeApplier {
	return &FakeApplier{
		AppliedJobs: []models.Job{},
	}
}

func (s *FakeApplier) Prepare(job models.Job) error {
	s.PreparedJobs = append(s.PreparedJobs, job)
	return s.PrepareError
}

func (s *FakeApplier) Apply(job models.Job) error {
	s.AppliedJobs = append(s.AppliedJobs, job)
	return s.ApplyError
}

func (s *FakeApplier) Configure(job models.Job, jobIndex int) error {
	s.ConfiguredJobs = append(s.ConfiguredJobs, job)
	s.ConfiguredJobIndices = append(s.ConfiguredJobIndices, jobIndex)
	return s.ConfigureError
}

func (s *FakeApplier) KeepOnly(jobs []models.Job) error {
	s.KeepOnlyJobs = jobs
	return s.KeepOnlyErr
}
