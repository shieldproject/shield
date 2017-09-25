package jobs

import (
	models "github.com/cloudfoundry/bosh-agent/agent/applier/models"
)

type Applier interface {
	Prepare(job models.Job) error
	Apply(job models.Job) error
	Configure(job models.Job, jobIndex int) error
	KeepOnly(jobs []models.Job) error
}
