package backup

import (
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type backupActivityMetadata struct {
	StartTime  string `yaml:"start_time"`
	FinishTime string `yaml:"finish_time,omitempty"`
}

type instanceMetadata struct {
	Name      string             `yaml:"name"`
	Index     string             `yaml:"index"`
	Artifacts []artifactMetadata `yaml:"artifacts"`
}

type artifactMetadata struct {
	Name     string            `yaml:"name"`
	Checksum map[string]string `yaml:"checksums"`
}

type metadata struct {
	MetadataForEachInstance   []*instanceMetadata    `yaml:"instances,omitempty"`
	MetadataForEachArtifact   []artifactMetadata     `yaml:"custom_artifacts,omitempty"`
	MetadataForBackupActivity backupActivityMetadata `yaml:"backup_activity"`
}

func readMetadata(filename string) (metadata, error) {
	metadata := metadata{}

	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return metadata, errors.Wrap(err, "failed to read metadata")
	}

	if err := yaml.Unmarshal(contents, &metadata); err != nil {
		return metadata, errors.Wrap(err, "failed to unmarshal metadata")
	}
	return metadata, nil
}

func (data *metadata) save(filename string) error {
	contents, err := yaml.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to marshal metadata")
	}

	return ioutil.WriteFile(filename, contents, 0666)
}

func (data *metadata) findOrCreateInstanceMetadata(name, index string) *instanceMetadata {
	for _, instanceMetadata := range data.MetadataForEachInstance {
		if instanceMetadata.Name == name && instanceMetadata.Index == index {
			return instanceMetadata
		}
	}
	newInstanceMetadata := &instanceMetadata{
		Name:  name,
		Index: index,
	}
	data.MetadataForEachInstance = append(data.MetadataForEachInstance, newInstanceMetadata)
	return newInstanceMetadata
}
