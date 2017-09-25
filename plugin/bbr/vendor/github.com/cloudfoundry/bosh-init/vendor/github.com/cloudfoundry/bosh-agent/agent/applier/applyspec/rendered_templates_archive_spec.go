package applyspec

import (
	"github.com/cloudfoundry/bosh-agent/agent/applier/models"
)

type RenderedTemplatesArchiveSpec struct {
	Sha1        string `json:"sha1"`
	BlobstoreID string `json:"blobstore_id"`
}

func (s RenderedTemplatesArchiveSpec) AsSource(job models.Job) models.Source {
	return models.Source{
		Sha1:          s.Sha1,
		BlobstoreID:   s.BlobstoreID,
		PathInArchive: job.Name,
	}
}
