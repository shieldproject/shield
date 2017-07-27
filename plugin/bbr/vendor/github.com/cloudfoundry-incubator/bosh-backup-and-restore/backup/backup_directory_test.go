package backup_test

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/backup"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
)

var _ = Describe("BackupDirectory", func() {
	var backupName string
	var deploymentName string
	var backupDirectoryManager = BackupDirectoryManager{}
	var logger = boshlog.NewWriterLogger(boshlog.LevelDebug, GinkgoWriter, GinkgoWriter)
	var nowFunc = func() time.Time {
		return time.Date(2015, 10, 21, 02, 2, 3, 0, time.FixedZone("UTC+1", 3600))
	}

	BeforeEach(func() {
		deploymentName = fmt.Sprintf("my-cool-redis-%d", config.GinkgoConfig.ParallelNode)
		backupName = deploymentName + "_20151021T010203Z"
	})

	AfterEach(func() {
		Expect(os.RemoveAll(backupName)).To(Succeed())
	})

	Describe("DeploymentMatches", func() {
		var artifact orchestrator.Backup
		var instance1 *fakes.FakeInstance
		var instance2 *fakes.FakeInstance

		BeforeEach(func() {
			instance1 = new(fakes.FakeInstance)
			instance1.NameReturns("redis")
			instance1.IndexReturns("0")

			instance2 = new(fakes.FakeInstance)
			instance2.NameReturns("redis")
			instance2.IndexReturns("1")

			artifact, _ = backupDirectoryManager.Open(backupName, logger)
		})

		Context("when the backup on disk matches the current deployment", func() {
			BeforeEach(func() {
				createTestMetadata(backupName, `---
instances:
- name: redis
  index: 0
  checksum: foo
- name: redis
  index: 1
  checksum: foo
`)
			})

			It("returns true", func() {
				match, _ := artifact.DeploymentMatches(backupName, []orchestrator.Instance{instance1, instance2})
				Expect(match).To(BeTrue())
			})
		})

		Context("when the backup doesn't match the current deployment", func() {
			BeforeEach(func() {
				createTestMetadata(backupName, `---
instances:
- name: redis
  index: 0
  checksum: foo
- name: redis
  index: 1
  checksum: foo
- name: broker
  index: 2
  checksum: foo
`)
			})

			It("returns false", func() {
				match, _ := artifact.DeploymentMatches(backupName, []orchestrator.Instance{instance1, instance2})
				Expect(match).To(BeFalse())
			})
		})

		Context("when an error occurs unmarshaling the metadata", func() {
			BeforeEach(func() {
				Expect(os.Mkdir(backupName, 0777)).To(Succeed())
				file, err := os.Create(backupName + "/" + "metadata")
				Expect(err).NotTo(HaveOccurred())
				_, err = file.Write([]byte("this is not yaml"))
				Expect(err).NotTo(HaveOccurred())
				Expect(file.Close()).To(Succeed())
			})

			It("returns error", func() {
				_, err := artifact.DeploymentMatches(backupName, []orchestrator.Instance{instance1, instance2})
				Expect(err).To(MatchError(ContainSubstring("failed to unmarshal metadata")))
			})
		})

		Context("when an error occurs checking if the file exists", func() {
			It("returns error", func() {
				_, err := artifact.DeploymentMatches(backupName, []orchestrator.Instance{instance1, instance2})
				Expect(err).To(MatchError(ContainSubstring("Error checking metadata file")))
			})
		})
	})

	Describe("Valid", func() {
		var backup orchestrator.Backup
		var verifyResult bool
		var verifyError error

		JustBeforeEach(func() {
			var err error
			backup, err = backupDirectoryManager.Open(backupName, logger)
			Expect(err).NotTo(HaveOccurred())
			verifyResult, verifyError = backup.Valid()
		})

		BeforeEach(func() {
			Expect(os.Mkdir(backupName, 0777)).To(Succeed())
		})

		Context("when the default artifact sha's match metafile", func() {
			BeforeEach(func() {
				contents := createTarWithContents(map[string]string{
					"file1": "This archive contains some text files.",
					"file2": "Gopher names:\nGeorge\nGeoffrey\nGonzo",
				})

				Expect(ioutil.WriteFile(backupName+"/redis-0-broker.tar", contents, 0666)).NotTo(HaveOccurred())

				metadataContents := fmt.Sprintf(`---
instances:
- name: redis
  index: 0
  artifacts:
  - name: broker
    checksums:
      file1: %x
      file2: %x
`, sha256.Sum256([]byte("This archive contains some text files.")), sha256.Sum256([]byte("Gopher names:\nGeorge\nGeoffrey\nGonzo")))
				createTestMetadata(backupName, metadataContents)
			})

			It("returns true", func() {
				Expect(verifyError).NotTo(HaveOccurred())
				Expect(verifyResult).To(BeTrue())
			})
		})

		Context("when the named artifact sha matches the metadata file", func() {
			BeforeEach(func() {
				contents := createTarWithContents(map[string]string{
					"file1": "This archive contains some text files.",
				})

				Expect(ioutil.WriteFile(backupName+"/foo_redis.tar", contents, 0666)).NotTo(HaveOccurred())

				createTestMetadata(backupName, fmt.Sprintf(`---
custom_artifacts:
- name: foo_redis
  checksums:
    file1: %x
`, sha256.Sum256([]byte("This archive contains some text files."))))
			})

			It("returns true", func() {
				Expect(verifyError).NotTo(HaveOccurred())
				Expect(verifyResult).To(BeTrue())
			})
		})

		Context("when the named artifact sha doesn't match the metadata file", func() {
			BeforeEach(func() {
				contents := createTarWithContents(map[string]string{
					"file1": "This archive contains some text files.",
				})

				Expect(ioutil.WriteFile(backupName+"/foo_redis.tar", contents, 0666)).NotTo(HaveOccurred())

				createTestMetadata(backupName, fmt.Sprintf(`---
custom_artifacts:
- name: foo_redis
  checksums:
    file1: %x
`, sha256.Sum256([]byte("you fools!"))))
			})

			It("returns false", func() {
				Expect(verifyError).NotTo(HaveOccurred())
				Expect(verifyResult).To(BeFalse())
			})
		})

		Context("when one of the default artifact file's contents don't match the sha", func() {
			BeforeEach(func() {
				contents := createTarWithContents(map[string]string{
					"file1": "This archive contains some text files.",
					"file2": "Gopher names:\nGeorge\nGeoffrey\nGonzo",
				})
				Expect(ioutil.WriteFile(backupName+"/redis-0-broker.tar", contents, 0666)).NotTo(HaveOccurred())
				createTestMetadata(backupName, fmt.Sprintf(`---
instances:
- name: redis
  index: 0
  artifacts:
  - name: broker
    checksums:
      file1: %x
      file2: %x
`, sha256.Sum256([]byte("This archive contains some text files.")),
					sha256.Sum256([]byte("Gopher names:\nNo Goper names"))))
			})

			It("returns false", func() {
				Expect(verifyError).NotTo(HaveOccurred())
				Expect(verifyResult).To(BeFalse())
			})
		})

		Context("when there is an extra file in the backed metadata", func() {
			BeforeEach(func() {
				contents := createTarWithContents(map[string]string{
					"file1": "This archive contains some text files.",
				})
				Expect(ioutil.WriteFile(backupName+"/redis-0-broker.tar", contents, 0666)).NotTo(HaveOccurred())
				createTestMetadata(backupName, fmt.Sprintf(`---
instances:
- name: redis
  index: 0
  artifacts:
  - name: broker
    checksums:
      file1: %x
      file2: %x`, sha256.Sum256([]byte("This archive contains some text files.")),
					sha256.Sum256([]byte("Gopher names:\nNot present"))))
			})

			It("returns false", func() {
				Expect(verifyResult).To(BeFalse())
				Expect(verifyError).NotTo(HaveOccurred())
			})
		})

		Context("metadata describes a file that doesn't exist", func() {
			BeforeEach(func() {
				createTestMetadata(backupName, fmt.Sprintf(`---
instances:
- name: redis
	index: 0
	checksums:
		file1: %x
`, sha256.Sum256([]byte("This archive contains some text files."))))
			})

			It("returns false", func() {
				Expect(verifyResult).To(BeFalse())
			})
			It("returns an error", func() {
				Expect(verifyError).To(MatchError(ContainSubstring("failed to unmarshal metadata")))
			})
		})

		Context("metadata file doesn't exist", func() {
			BeforeEach(func() {
				contents := createTarWithContents(map[string]string{
					"file1": "This archive contains some text files.",
				})
				Expect(ioutil.WriteFile(backupName+"/redis-1.tar", contents, 0666)).NotTo(HaveOccurred())
			})

			It("returns false", func() {
				Expect(verifyResult).To(BeFalse())
			})
			It("returns an error", func() {
				Expect(verifyError).To(MatchError(ContainSubstring("failed to read metadata")))
			})
		})
	})

	Describe("CreateArtifact", func() {
		var artifact orchestrator.Backup
		var fileCreationError error
		var writer io.Writer
		var fakeBackupArtifact *fakes.FakeBackupArtifact

		BeforeEach(func() {
			artifact, _ = backupDirectoryManager.Create(deploymentName, logger, nowFunc)
			fakeBackupArtifact = new(fakes.FakeBackupArtifact)
			fakeBackupArtifact.InstanceNameReturns("redis-server")
			fakeBackupArtifact.InstanceIndexReturns("0")
		})
		JustBeforeEach(func() {
			writer, fileCreationError = artifact.CreateArtifact(fakeBackupArtifact)
		})
		Context("with a default backup artifact", func() {
			BeforeEach(func() {
				fakeBackupArtifact.NameReturns("redis")
				fakeBackupArtifact.HasCustomNameReturns(false)
			})
			Context("Can create a file", func() {
				It("creates a file in the artifact directory", func() {
					Expect(backupName + "/redis-server-0-redis.tar").To(BeARegularFile())
				})

				It("writer writes contents to the file", func() {
					writer.Write([]byte("lalala a file"))
					Expect(ioutil.ReadFile(backupName + "/redis-server-0-redis.tar")).To(Equal([]byte("lalala a file")))
				})

				It("does not fail", func() {
					Expect(fileCreationError).NotTo(HaveOccurred())
				})
			})
		})
		Context("with a named backup artifact", func() {
			BeforeEach(func() {
				fakeBackupArtifact.HasCustomNameReturns(true)
				fakeBackupArtifact.NameReturns("my-backup-artifact")
			})

			It("creates the named file in the artifact directory", func() {
				Expect(backupName + "/my-backup-artifact.tar").To(BeARegularFile())
			})

			It("writer writes contents to the file", func() {
				writer.Write([]byte("lalala a file"))
				Expect(ioutil.ReadFile(backupName + "/my-backup-artifact.tar")).To(Equal([]byte("lalala a file")))
			})

			It("does not fail", func() {
				Expect(fileCreationError).NotTo(HaveOccurred())
			})
		})

		Context("Cannot create file", func() {
			BeforeEach(func() {
				fakeBackupArtifact.NameReturns("foo/bar/baz")
			})
			It("fails", func() {
				Expect(fileCreationError).To(MatchError(ContainSubstring("Error creating file")))
			})
		})

	})

	Describe("SaveManifest", func() {
		var artifact orchestrator.Backup
		var saveManifestError error

		BeforeEach(func() {
			artifact, _ = backupDirectoryManager.Create(deploymentName, logger, nowFunc)
		})

		AfterEach(func() {
			Expect(os.RemoveAll(backupName)).To(Succeed())
		})

		JustBeforeEach(func() {
			saveManifestError = artifact.SaveManifest("contents")
		})
		It("does not fail", func() {
			Expect(saveManifestError).NotTo(HaveOccurred())
		})

		It("writes contents to a file", func() {
			Expect(ioutil.ReadFile(backupName + "/manifest.yml")).To(Equal([]byte("contents")))
		})
	})

	Describe("ReadArtifact", func() {
		var artifact orchestrator.Backup
		var fileReadError error
		var reader io.Reader
		var fakeBackupArtifact *fakes.FakeBackupArtifact

		BeforeEach(func() {
			artifact, _ = backupDirectoryManager.Open(backupName, logger)
			fakeBackupArtifact = new(fakes.FakeBackupArtifact)
			fakeBackupArtifact.InstanceNameReturns("redis-server")
			fakeBackupArtifact.InstanceIndexReturns("0")
		})

		Context("default artifact - file exists and is readable", func() {
			BeforeEach(func() {
				fakeBackupArtifact.NameReturns("redis")
				fakeBackupArtifact.HasCustomNameReturns(false)

				err := os.MkdirAll(backupName, 0700)
				Expect(err).NotTo(HaveOccurred())
				_, err = os.Create(backupName + "/redis-server-0-redis.tar")
				Expect(err).NotTo(HaveOccurred())
				err = ioutil.WriteFile(backupName+"/redis-server-0-redis.tar", []byte("backup-content"), 0700)
				Expect(err).NotTo(HaveOccurred())
			})

			JustBeforeEach(func() {
				reader, fileReadError = artifact.ReadArtifact(fakeBackupArtifact)
			})

			It("does not fail", func() {
				Expect(fileReadError).NotTo(HaveOccurred())
			})

			It("reads the correct file", func() {
				contents, err := ioutil.ReadAll(reader)

				Expect(err).NotTo(HaveOccurred())
				Expect(contents).To(ContainSubstring("backup-content"))
			})
		})

		Context("named artifact - file exists and is readable", func() {
			BeforeEach(func() {
				fakeBackupArtifact.HasCustomNameReturns(true)
				fakeBackupArtifact.NameReturns("foo-bar")

				err := os.MkdirAll(backupName, 0700)
				Expect(err).NotTo(HaveOccurred())
				_, err = os.Create(backupName + "/foo-bar.tar")
				Expect(err).NotTo(HaveOccurred())
				err = ioutil.WriteFile(backupName+"/foo-bar.tar", []byte("backup-content"), 0700)
				Expect(err).NotTo(HaveOccurred())
			})

			JustBeforeEach(func() {
				reader, fileReadError = artifact.ReadArtifact(fakeBackupArtifact)
			})

			It("does not fail", func() {
				Expect(fileReadError).NotTo(HaveOccurred())
			})

			It("reads the correct file", func() {
				contents, err := ioutil.ReadAll(reader)

				Expect(err).NotTo(HaveOccurred())
				Expect(contents).To(ContainSubstring("backup-content"))
			})
		})

		Context("File is not readable", func() {
			It("fails", func() {
				_, fileReadError = artifact.ReadArtifact(fakeBackupArtifact)
				Expect(fileReadError).To(MatchError(ContainSubstring("Error reading artifact file")))
			})
		})
	})

	Describe("Checksum", func() {
		var artifact orchestrator.Backup
		var fakeBackupArtifact *fakes.FakeBackupArtifact

		BeforeEach(func() {
			fakeBackupArtifact = new(fakes.FakeBackupArtifact)
			fakeBackupArtifact.InstanceNameReturns("redis-server")
			fakeBackupArtifact.InstanceIndexReturns("0")
		})
		JustBeforeEach(func() {
			artifact, _ = backupDirectoryManager.Create(deploymentName, logger, nowFunc)
		})
		Context("file exists", func() {
			Context("default artifact", func() {
				BeforeEach(func() {
					fakeBackupArtifact.NameReturns("redis")
					fakeBackupArtifact.HasCustomNameReturns(false)
				})
				JustBeforeEach(func() {
					writer, fileCreationError := artifact.CreateArtifact(fakeBackupArtifact)
					Expect(fileCreationError).NotTo(HaveOccurred())

					contents := createTarWithContents(map[string]string{
						"readme.txt": "This archive contains some text files.",
						"gopher.txt": "Gopher names:\nGeorge\nGeoffrey\nGonzo",
						"todo.txt":   "Get animal handling license.",
					})

					writer.Write(contents)
					Expect(writer.Close()).NotTo(HaveOccurred())
				})

				It("returns the checksum for the saved instance data", func() {
					Expect(artifact.CalculateChecksum(fakeBackupArtifact)).To(Equal(
						orchestrator.BackupChecksum{
							"readme.txt": fmt.Sprintf("%x", sha256.Sum256([]byte("This archive contains some text files."))),
							"gopher.txt": fmt.Sprintf("%x", sha256.Sum256([]byte("Gopher names:\nGeorge\nGeoffrey\nGonzo"))),
							"todo.txt":   fmt.Sprintf("%x", sha256.Sum256([]byte("Get animal handling license."))),
						}))
				})
			})

			Context("named artifact", func() {
				BeforeEach(func() {
					fakeBackupArtifact.NameReturns("foo")
					fakeBackupArtifact.HasCustomNameReturns(true)
				})
				JustBeforeEach(func() {
					writer, fileCreationError := artifact.CreateArtifact(fakeBackupArtifact)
					Expect(fileCreationError).NotTo(HaveOccurred())

					contents := createTarWithContents(map[string]string{
						"readme.txt": "This archive contains some text files.",
						"gopher.txt": "Gopher names:\nGeorge\nGeoffrey\nGonzo",
						"todo.txt":   "Get animal handling license.",
					})

					writer.Write(contents)
					Expect(writer.Close()).NotTo(HaveOccurred())
				})

				It("returns the checksum for the saved instance data", func() {
					Expect(artifact.CalculateChecksum(fakeBackupArtifact)).To(Equal(
						orchestrator.BackupChecksum{
							"readme.txt": fmt.Sprintf("%x", sha256.Sum256([]byte("This archive contains some text files."))),
							"gopher.txt": fmt.Sprintf("%x", sha256.Sum256([]byte("Gopher names:\nGeorge\nGeoffrey\nGonzo"))),
							"todo.txt":   fmt.Sprintf("%x", sha256.Sum256([]byte("Get animal handling license."))),
						}))
				})
			})
		})

		Context("invalid tar file", func() {
			JustBeforeEach(func() {
				writer, fileCreationError := artifact.CreateArtifact(fakeBackupArtifact)
				Expect(fileCreationError).NotTo(HaveOccurred())

				contents := []byte("this ain't a tarball")

				writer.Write(contents)
				Expect(writer.Close()).NotTo(HaveOccurred())
			})

			It("fails to read", func() {
				_, err := artifact.CalculateChecksum(fakeBackupArtifact)
				Expect(err).To(MatchError(ContainSubstring("Error reading tar")))
			})
		})

		Context("file doesn't exist", func() {
			It("fails", func() {
				_, err := artifact.CalculateChecksum(fakeBackupArtifact)
				Expect(err).To(MatchError(ContainSubstring("Error reading artifact file")))
			})
		})
	})

	Describe("AddChecksum", func() {
		var artifact orchestrator.Backup
		var addChecksumError error
		var fakeBackupArtifact *fakes.FakeBackupArtifact
		var checksum map[string]string
		var startTime time.Time

		BeforeEach(func() {
			artifact, _ = backupDirectoryManager.Create(deploymentName, logger, nowFunc)
			startTime = time.Date(2015, 10, 21, 1, 2, 3, 0, time.UTC)
			Expect(artifact.CreateMetadataFileWithStartTime(startTime)).To(Succeed())
			fakeBackupArtifact = new(fakes.FakeBackupArtifact)
			checksum = map[string]string{"filename": "foobar"}
		})

		JustBeforeEach(func() {
			addChecksumError = artifact.AddChecksum(fakeBackupArtifact, checksum)
		})

		Context("default artifacts", func() {
			BeforeEach(func() {
				fakeBackupArtifact.InstanceNameReturns("redis-server")
				fakeBackupArtifact.InstanceIndexReturns("0")
				fakeBackupArtifact.HasCustomNameReturns(false)
				fakeBackupArtifact.NameReturns("redis")
			})

			Context("when no default artifacts have been added", func() {
				It("adds the instance and the artifact", func() {
					Expect(backupName + "/metadata").To(BeARegularFile())

					expectedMetadata := `---
backup_activity:
  start_time: 2015/10/21 01:02:03 UTC
instances:
- name: redis-server
  index: "0"
  artifacts:
  - name: redis
    checksums:
      filename: foobar`
					Expect(ioutil.ReadFile(backupName + "/metadata")).To(MatchYAML(expectedMetadata))
				})
			})

			Context("when default artifacts for the same instance have been added", func() {
				BeforeEach(func() {
					anotherFakeBackupArtifact := new(fakes.FakeBackupArtifact)
					anotherFakeBackupArtifact.InstanceNameReturns("redis-server")
					anotherFakeBackupArtifact.InstanceIndexReturns("0")
					anotherFakeBackupArtifact.HasCustomNameReturns(false)
					anotherFakeBackupArtifact.NameReturns("broker")

					addChecksumError = artifact.AddChecksum(anotherFakeBackupArtifact, checksum)
				})
				It("appends the artifact to the instance", func() {
					Expect(backupName + "/metadata").To(BeARegularFile())

					expectedMetadata := `---
backup_activity:
  start_time: 2015/10/21 01:02:03 UTC
instances:
- name: redis-server
  index: "0"
  artifacts:
  - name: broker
    checksums:
      filename: foobar
  - name: redis
    checksums:
      filename: foobar`
					Expect(ioutil.ReadFile(backupName + "/metadata")).To(MatchYAML(expectedMetadata))
				})
			})

			Context("when default artifacts for another instance have been added", func() {
				BeforeEach(func() {
					anotherFakeBackupArtifact := new(fakes.FakeBackupArtifact)
					anotherFakeBackupArtifact.InstanceNameReturns("memcached-server")
					anotherFakeBackupArtifact.InstanceIndexReturns("0")
					anotherFakeBackupArtifact.HasCustomNameReturns(false)
					anotherFakeBackupArtifact.NameReturns("memcached")

					addChecksumError = artifact.AddChecksum(anotherFakeBackupArtifact, checksum)
				})
				It("adds the new instance and the artifact", func() {
					Expect(backupName + "/metadata").To(BeARegularFile())

					expectedMetadata := `---
backup_activity:
  start_time: 2015/10/21 01:02:03 UTC
instances:
- name: memcached-server
  index: "0"
  artifacts:
  - name: memcached
    checksums:
      filename: foobar
- name: redis-server
  index: "0"
  artifacts:
  - name: redis
    checksums:
      filename: foobar
`
					Expect(ioutil.ReadFile(backupName + "/metadata")).To(MatchYAML(expectedMetadata))
				})
			})
		})

		Context("named artifacts", func() {
			BeforeEach(func() {
				fakeBackupArtifact.HasCustomNameReturns(true)
				fakeBackupArtifact.NameReturns("foo")
			})

			Context("when no named artifacts have been added", func() {
				It("adds the named artifact", func() {
					Expect(backupName + "/metadata").To(BeARegularFile())

					expectedMetadata := `---
backup_activity:
  start_time: 2015/10/21 01:02:03 UTC
custom_artifacts:
- name: foo
  checksums:
    filename: foobar`
					Expect(ioutil.ReadFile(backupName + "/metadata")).To(MatchYAML(expectedMetadata))
				})
			})

			Context("when a named artifact has been added", func() {
				BeforeEach(func() {
					anotherFakeBackupArtifact := new(fakes.FakeBackupArtifact)
					anotherFakeBackupArtifact.HasCustomNameReturns(true)
					anotherFakeBackupArtifact.NameReturns("bar")

					addChecksumError = artifact.AddChecksum(anotherFakeBackupArtifact, checksum)
				})
				It("appends the new named artifact", func() {
					Expect(backupName + "/metadata").To(BeARegularFile())

					expectedMetadata := `---
backup_activity:
  start_time: 2015/10/21 01:02:03 UTC
custom_artifacts:
- name: bar
  checksums:
    filename: foobar
- name: foo
  checksums:
    filename: foobar
`
					Expect(ioutil.ReadFile(backupName + "/metadata")).To(MatchYAML(expectedMetadata))
				})
			})
		})

		Context("when the metadata file is invalid", func() {
			BeforeEach(func() {
				createTestMetadata(backupName, "not valid yaml")
			})

			It("fails", func() {
				Expect(addChecksumError).To(MatchError(ContainSubstring("failed to unmarshal metadata")))
			})
		})

		Context("when the metadata file doesn't exist", func() {
			BeforeEach(func() {
				Expect(os.Remove(backupName + "/" + "metadata")).To(Succeed())
			})

			It("fails", func() {
				Expect(addChecksumError).To(MatchError(ContainSubstring("unable to load metadata")))
			})
		})
	})

	Describe("FetchChecksum", func() {
		var artifact orchestrator.Backup
		var fetchChecksumError error
		var fakeArtifact *fakes.FakeBackupArtifact
		var checksum orchestrator.BackupChecksum
		BeforeEach(func() {
			fakeArtifact = new(fakes.FakeBackupArtifact)
		})
		JustBeforeEach(func() {
			var artifactOpenError error
			artifact, artifactOpenError = backupDirectoryManager.Open(backupName, logger)
			Expect(artifactOpenError).NotTo(HaveOccurred())

			checksum, fetchChecksumError = artifact.FetchChecksum(fakeArtifact)
		})
		Context("the named artifact is found in metadata", func() {
			BeforeEach(func() {
				fakeArtifact.HasCustomNameReturns(true)
				fakeArtifact.NameReturns("foo")

				createTestMetadata(backupName, `---
instances: []
custom_artifacts:
- name: foo
  checksums:
    filename1: orignal_checksum`)
			})

			It("doesn't fail", func() {
				Expect(fetchChecksumError).NotTo(HaveOccurred())
			})

			It("fetches the checksum", func() {
				Expect(checksum).To(Equal(orchestrator.BackupChecksum{"filename1": "orignal_checksum"}))
			})
		})

		Context("the default artifact is found in metadata", func() {
			BeforeEach(func() {
				fakeArtifact.InstanceNameReturns("foo")
				fakeArtifact.InstanceIndexReturns("bar")
				fakeArtifact.NameReturns("baz")

				createTestMetadata(backupName, `---
instances:
- name: foo
  index: "bar"
  artifacts:
  - name: baz
    checksums:
      filename1: orignal_checksum`)
			})

			It("doesn't fail", func() {
				Expect(fetchChecksumError).NotTo(HaveOccurred())
			})

			It("fetches the checksum", func() {
				Expect(checksum).To(Equal(orchestrator.BackupChecksum{"filename1": "orignal_checksum"}))
			})
		})
		//TODO: why is this required
		Context("the default artifact is not found in metadata", func() {
			BeforeEach(func() {
				fakeArtifact.NameReturns("not-baz")
				fakeArtifact.InstanceNameReturns("foo")
				fakeArtifact.InstanceIndexReturns("bar")

				createTestMetadata(backupName, `---
instances:
- name: foo
  index: "bar"
  artifacts:
  - name: baz
    checksums:
      filename1: orignal_checksum`)
			})

			It("doesn't fail", func() {
				Expect(fetchChecksumError).ToNot(HaveOccurred())
			})

			It("returns nil", func() {
				Expect(checksum).To(BeNil())
			})
		})

		Context("the default artifact's instance is not found in metadata", func() {
			BeforeEach(func() {
				fakeArtifact.NameReturns("baz")
				fakeArtifact.InstanceNameReturns("not-foo")
				fakeArtifact.InstanceIndexReturns("bar")

				createTestMetadata(backupName, `---
instances:
- name: foo
  index: "bar"
  artifacts:
  - name: baz
    checksums:
      filename1: orignal_checksum`)
			})

			It("doesn't fail", func() {
				Expect(fetchChecksumError).ToNot(HaveOccurred())
			})

			It("returns nil", func() {
				Expect(checksum).To(BeNil())
			})
		})

		Context("the named artifact is not found in metadata", func() {
			BeforeEach(func() {
				fakeArtifact.NameReturns("not-foo")
				fakeArtifact.HasCustomNameReturns(true)

				createTestMetadata(backupName, `---
instances:
- name: foo
  index: "bar"
  checksums:
    filename1: orignal_checksum`)
			})

			It("doesn't fail", func() {
				Expect(fetchChecksumError).ToNot(HaveOccurred())
			})

			It("returns nil", func() {
				Expect(checksum).To(BeNil())
			})
		})

		Context("if existing file isn't valid", func() {
			BeforeEach(func() {
				createTestMetadata(backupName, "not valid yaml")
			})

			It("fails", func() {
				Expect(fetchChecksumError).To(MatchError(ContainSubstring("failed to unmarshal metadata")))
			})
		})

	})

	Describe("CreateMetadataFileWithStartTime", func() {
		var artifact orchestrator.Backup

		BeforeEach(func() {
			var err error
			artifact, err = backupDirectoryManager.Create(deploymentName, logger, nowFunc)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when no metadata file exists", func() {
			It("creates the file containing the timestamp in the correct format", func() {
				theTime := time.Date(2015, 10, 21, 1, 2, 3, 0, time.UTC)
				Expect(artifact.CreateMetadataFileWithStartTime(theTime)).To(Succeed())

				expectedMetadata := `---
backup_activity:
  start_time: 2015/10/21 01:02:03 UTC`

				Expect(ioutil.ReadFile(backupName + "/metadata")).To(MatchYAML(expectedMetadata))
			})
		})

		Context("when the metadata file already exists", func() {
			It("returns an error", func() {
				createTestMetadata(backupName, "")
				Expect(artifact.CreateMetadataFileWithStartTime(time.Now())).To(MatchError("metadata file already exists"))
			})
		})
	})

	Describe("AddFinishTime", func() {
		var artifact orchestrator.Backup

		BeforeEach(func() {
			var err error
			artifact, err = backupDirectoryManager.Create(deploymentName, logger, nowFunc)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when no metadata file exists", func() {
			It("returns an error", func() {
				Expect(artifact.AddFinishTime(nowFunc())).To(MatchError(ContainSubstring("unable to load metadata")))
			})
		})

		Context("when the metadata file already exists", func() {
			It("adds the timestamp in the correct format", func() {
				startTime := time.Date(2015, 10, 21, 1, 2, 3, 0, time.UTC)
				finishTime := time.Date(2016, 10, 21, 4, 5, 6, 0, time.UTC)
				Expect(artifact.CreateMetadataFileWithStartTime(startTime)).To(Succeed())
				Expect(artifact.AddFinishTime(finishTime)).To(Succeed())

				expectedMetadata := `---
backup_activity:
  start_time: 2015/10/21 01:02:03 UTC
  finish_time: 2016/10/21 04:05:06 UTC`

				Expect(ioutil.ReadFile(backupName + "/metadata")).To(MatchYAML(expectedMetadata))
			})
		})
	})
})

func createTestMetadata(backupDirectory string, metadata string) {
	Expect(os.MkdirAll(backupDirectory, 0777)).To(Succeed())

	file, err := os.Create(backupDirectory + "/" + "metadata")
	Expect(err).NotTo(HaveOccurred())

	_, err = file.Write([]byte(metadata))
	Expect(err).NotTo(HaveOccurred())
	Expect(file.Close()).To(Succeed())
}

func createTarWithContents(files map[string]string) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	tarFile := tar.NewWriter(bytesBuffer)

	for filename, contents := range files {
		hdr := &tar.Header{
			Name: filename,
			Mode: 0600,
			Size: int64(len(contents)),
		}
		if err := tarFile.WriteHeader(hdr); err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
		if _, err := tarFile.Write([]byte(contents)); err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
	}
	if err := tarFile.Close(); err != nil {
		Expect(err).NotTo(HaveOccurred())
	}
	Expect(tarFile.Close()).NotTo(HaveOccurred())
	return bytesBuffer.Bytes()
}
