package applyspec

import (
	models "github.com/cloudfoundry/bosh-agent/agent/applier/models"
)

type JobTemplateSpec struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (s *JobTemplateSpec) AsJob() models.Job {
	return models.Job{
		Name:    s.Name,
		Version: s.Version,
	}
}
