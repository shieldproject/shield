package applyspec

import models "github.com/cloudfoundry/bosh-agent/agent/applier/models"

type JobTemplateSpec struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Sha1        string `json:"sha1"`
	BlobstoreID string `json:"blobstore_id"`
}

func (s *JobTemplateSpec) AsJob() models.Job {
	return models.Job{
		Name:    s.Name,
		Version: s.Version,
		Source: models.Source{
			Sha1:        s.Sha1,
			BlobstoreID: s.BlobstoreID,
		},
	}
}
