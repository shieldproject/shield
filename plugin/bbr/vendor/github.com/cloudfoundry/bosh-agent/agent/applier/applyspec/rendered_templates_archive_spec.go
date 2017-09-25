package applyspec

import (
	"github.com/cloudfoundry/bosh-agent/agent/applier/models"
	"github.com/cloudfoundry/bosh-utils/crypto"

	"encoding/json"
)

type RenderedTemplatesArchiveSpec struct {
	Sha1        *crypto.MultipleDigest `json:"sha1"`
	BlobstoreID string                 `json:"blobstore_id"`
}

func (s RenderedTemplatesArchiveSpec) AsSource(job models.Job) models.Source {
	var sha1 crypto.Digest
	if s.Sha1 != nil {
		sha1 = *s.Sha1
	}
	return models.Source{
		Sha1:          sha1,
		BlobstoreID:   s.BlobstoreID,
		PathInArchive: job.Name,
	}
}

type renderedTemplatesArchiveJSONStruct struct {
	Sha1        string `json:"sha1"`
	BlobstoreID string `json:"blobstore_id"`
}

func (s *RenderedTemplatesArchiveSpec) UnmarshalJSON(data []byte) error {
	var jsonStruct renderedTemplatesArchiveJSONStruct
	err := json.Unmarshal(data, &jsonStruct)
	if err != nil {
		return err
	}

	if jsonStruct.BlobstoreID == "" && jsonStruct.Sha1 == "" {
		s = nil
		return nil
	}

	var digest crypto.MultipleDigest
	err = json.Unmarshal([]byte("\""+jsonStruct.Sha1+"\""), &digest)
	if err != nil {
		return err
	}

	*s = RenderedTemplatesArchiveSpec{
		Sha1:        &digest,
		BlobstoreID: jsonStruct.BlobstoreID,
	}

	return nil
}
