package backup

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	sha256 "crypto/sha256"
	"time"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/pkg/errors"
)

const tag = "[artifact]"
const timestampFormat = "2006/01/02 15:04:05 MST"

type BackupDirectory struct {
	orchestrator.Logger
	baseDirName string
}

func (backupDirectory *BackupDirectory) logAndReturn(err error, message string, args ...interface{}) error {
	message = fmt.Sprintf(message, args...)
	backupDirectory.Debug(tag, "%s: %v", message, err)
	return errors.Wrap(err, message)
}

func (backupDirectory *BackupDirectory) DeploymentMatches(deployment string, instances []orchestrator.Instance) (bool, error) {
	_, err := backupDirectory.metadataExistsAndIsReadable()
	if err != nil {
		return false, backupDirectory.logAndReturn(err, "Error checking metadata file")
	}
	meta, err := readMetadata(backupDirectory.metadataFilename())
	if err != nil {
		return false, backupDirectory.logAndReturn(err, "Error reading metadata file")
	}

	for _, inst := range meta.MetadataForEachInstance {
		present := backupDirectory.backupInstanceIsPresent(inst, instances)
		if present != true {
			backupDirectory.Debug(tag, "Instance %v/%v not found in %v", inst.Name, inst.Index, instances)
			return false, nil
		}
	}

	return true, nil
}

func (backupDirectory *BackupDirectory) CreateArtifact(artifactIdentifier orchestrator.ArtifactIdentifier) (io.WriteCloser, error) {
	backupDirectory.Debug(tag, "Trying to create file %s", fileName(artifactIdentifier))

	file, err := os.Create(path.Join(backupDirectory.baseDirName, fileName(artifactIdentifier)))
	if err != nil {
		return nil, backupDirectory.logAndReturn(err, "Error creating file %s", fileName(artifactIdentifier))

	}

	return file, err
}

func (backupDirectory *BackupDirectory) ReadArtifact(artifactIdentifier orchestrator.ArtifactIdentifier) (io.ReadCloser, error) {
	filename := backupDirectory.instanceFilename(artifactIdentifier)
	backupDirectory.Debug(tag, "Trying to open %s", filename)
	file, err := os.Open(filename)
	if err != nil {
		backupDirectory.Debug(tag, "Error reading artifact file %s", filename)
		return nil, backupDirectory.logAndReturn(err, "Error reading artifact file %s", filename)
	}

	return file, nil
}

func (backupDirectory *BackupDirectory) FetchChecksum(artifactIdentifier orchestrator.ArtifactIdentifier) (orchestrator.BackupChecksum, error) {
	metadata, err := readMetadata(backupDirectory.metadataFilename())

	if err != nil {
		return nil, backupDirectory.logAndReturn(err, "Error reading metadata from %s", backupDirectory.metadataFilename())
	}

	if artifactIdentifier.HasCustomName() {
		for _, customArtifactInMetadata := range metadata.MetadataForEachArtifact {
			if customArtifactInMetadata.Name == artifactIdentifier.Name() {
				return customArtifactInMetadata.Checksum, nil
			}
		}
	} else {
		for _, instanceInMetadata := range metadata.MetadataForEachInstance {
			if instanceInMetadata.Index == artifactIdentifier.InstanceIndex() && instanceInMetadata.Name == artifactIdentifier.InstanceName() {
				for _, artifact := range instanceInMetadata.Artifacts {
					if artifact.Name == artifactIdentifier.Name() {
						return artifact.Checksum, nil
					}
				}
			}
		}
	}

	backupDirectory.Warn(tag, "Checksum for %s not found in artifact", logName(artifactIdentifier))
	return nil, nil
}

func logName(artifactIdentifer orchestrator.ArtifactIdentifier) string {
	if artifactIdentifer.HasCustomName() {
		return fmt.Sprintf("%s", artifactIdentifer.Name())
	}
	return fmt.Sprintf("%s/%s", artifactIdentifer.Name(), artifactIdentifer.InstanceIndex())
}

func (backupDirectory *BackupDirectory) CalculateChecksum(artifactIdentifier orchestrator.ArtifactIdentifier) (orchestrator.BackupChecksum, error) {
	file, err := backupDirectory.ReadArtifact(artifactIdentifier)
	if err != nil {
		return nil, backupDirectory.logAndReturn(err, "Error opening artifact file %v", artifactIdentifier)
	}
	defer file.Close()

	tarReader := tar.NewReader(file)
	checksum := map[string]string{}
	for {
		tarHeader, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, backupDirectory.logAndReturn(err, "Error reading tar for %s", logName(artifactIdentifier))
		}
		if tarHeader.FileInfo().IsDir() || tarHeader.FileInfo().Name() == "./" {
			continue
		}

		fileShasum := sha256.New()
		if _, err := io.Copy(fileShasum, tarReader); err != nil {
			return nil, backupDirectory.logAndReturn(err, "Error calculating sha for %s", logName(artifactIdentifier))
		}
		backupDirectory.Logger.Debug(tag, "Calculating shasum for local file %s", tarHeader.Name)
		checksum[tarHeader.Name] = fmt.Sprintf("%x", fileShasum.Sum(nil))
	}

	return checksum, nil
}

func (backupDirectory *BackupDirectory) AddChecksum(artifactIdentifier orchestrator.ArtifactIdentifier, shasum orchestrator.BackupChecksum) error {
	if exists, err := backupDirectory.metadataExistsAndIsReadable(); !exists {
		return backupDirectory.logAndReturn(err, "unable to load metadata")
	}

	metadata, err := readMetadata(backupDirectory.metadataFilename())
	if err != nil {
		return backupDirectory.logAndReturn(err, "Error reading metadata from %s", backupDirectory.metadataFilename())
	}

	if artifactIdentifier.HasCustomName() {
		metadata.MetadataForEachArtifact = append(metadata.MetadataForEachArtifact, artifactMetadata{
			Name:     artifactIdentifier.Name(),
			Checksum: shasum,
		})
	} else {
		instanceMetadata := metadata.findOrCreateInstanceMetadata(artifactIdentifier.InstanceName(), artifactIdentifier.InstanceIndex())
		instanceMetadata.Artifacts = append(instanceMetadata.Artifacts, artifactMetadata{
			Name:     artifactIdentifier.Name(),
			Checksum: shasum,
		})
	}

	return metadata.save(backupDirectory.metadataFilename())
}

func (backupDirectory *BackupDirectory) CreateMetadataFileWithStartTime(startTime time.Time) error {
	exists, _ := backupDirectory.metadataExistsAndIsReadable()
	if exists {
		message := "metadata file already exists"
		backupDirectory.Debug(tag, "%s: %v", message, nil)
		return errors.New(message)
	}

	metadata := metadata{
		MetadataForBackupActivity: backupActivityMetadata{
			StartTime: startTime.Format(timestampFormat),
		},
	}
	metadata.save(backupDirectory.metadataFilename())

	return nil
}

func (backupDirectory *BackupDirectory) AddFinishTime(finishTime time.Time) error {
	metadata, err := readMetadata(backupDirectory.metadataFilename())
	if err != nil {
		message := "unable to load metadata"
		backupDirectory.Debug(tag, "%s: %v", message, nil)
		return backupDirectory.logAndReturn(err, message)
	}

	metadata.MetadataForBackupActivity.FinishTime = finishTime.Format(timestampFormat)
	metadata.save(backupDirectory.metadataFilename())

	return nil
}

func (backupDirectory *BackupDirectory) SaveManifest(manifest string) error {
	return errors.Wrap(ioutil.WriteFile(backupDirectory.manifestFilename(), []byte(manifest), 0666), "failed to save manifest")
}

func (backupDirectory *BackupDirectory) Valid() (bool, error) {
	meta, err := readMetadata(backupDirectory.metadataFilename())
	if err != nil {
		return false, backupDirectory.logAndReturn(err, "Error reading metadata from %s", backupDirectory.metadataFilename())
	}

	for _, artifact := range meta.MetadataForEachArtifact {
		actualArtifactChecksum, _ := backupDirectory.CalculateChecksum(makeCustomArtifactIdentifier(artifact))
		if !actualArtifactChecksum.Match(artifact.Checksum) {
			return false, backupDirectory.logAndReturn(err, "Can't match checksums for %s, in metadata: %v, in actual file: %v", artifact.Name, actualArtifactChecksum, artifact.Checksum)
		}
	}

	for _, inst := range meta.MetadataForEachInstance {
		for _, artifact := range inst.Artifacts {

			actualInstanceChecksum, err := backupDirectory.CalculateChecksum(makeDefaultArtifactIdentifier(artifact, inst))
			if err != nil {
				return false, backupDirectory.logAndReturn(err, "Error calculating checksum for artifact")
			}

			if !actualInstanceChecksum.Match(artifact.Checksum) {
				return false, backupDirectory.logAndReturn(err, "Can't match checksums for %s/%s %s, in metadata: %v, in actual file: %v", inst.Name, inst.Index, artifact.Name, actualInstanceChecksum, artifact.Checksum)
			}
		}
	}

	return true, nil
}

func (backupDirectory *BackupDirectory) backupInstanceIsPresent(backupInstance *instanceMetadata, instances []orchestrator.Instance) bool {
	for _, inst := range instances {
		if inst.Index() == backupInstance.Index && inst.Name() == backupInstance.Name {
			return true
		}
	}
	return false
}

func (backupDirectory *BackupDirectory) instanceFilename(artifactIdentifier orchestrator.ArtifactIdentifier) string {
	return path.Join(backupDirectory.baseDirName, fileName(artifactIdentifier))
}

func (backupDirectory *BackupDirectory) metadataFilename() string {
	return path.Join(backupDirectory.baseDirName, "metadata")
}

func (backupDirectory *BackupDirectory) manifestFilename() string {
	return path.Join(backupDirectory.baseDirName, "manifest.yml")
}

func (backupDirectory *BackupDirectory) metadataExistsAndIsReadable() (bool, error) {
	_, err := os.Stat(backupDirectory.metadataFilename())
	if err != nil {
		return false, backupDirectory.logAndReturn(err, "Error checking metadata exists and is readable")
	}
	return true, nil
}

func fileName(artifactIdentifier orchestrator.ArtifactIdentifier) string {
	if artifactIdentifier.HasCustomName() {
		return customArtifactFileName(artifactIdentifier.Name())
	}

	return instanceArtifactFileName(artifactIdentifier.InstanceName(), artifactIdentifier.InstanceIndex(), artifactIdentifier.Name())
}

func instanceArtifactFileName(instanceName string, instanceIndex string, name string) string {
	return instanceName + "-" + instanceIndex + "-" + name + ".tar"
}

func customArtifactFileName(artifactName string) string {
	return artifactName + ".tar"
}
