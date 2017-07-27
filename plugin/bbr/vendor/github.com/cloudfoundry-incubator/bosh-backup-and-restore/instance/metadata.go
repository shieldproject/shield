package instance

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Metadata struct {
	BackupName  string `yaml:"backup_name"`
	RestoreName string `yaml:"restore_name"`
}

func NewJobMetadata(data []byte) (*Metadata, error) {
	metadata := &Metadata{}
	err := yaml.Unmarshal(data, metadata)

	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal job metadata")
	}

	return metadata, nil
}
