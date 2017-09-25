package instance_test

import (
	"bytes"
	"errors"
	"fmt"
	"log"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance/fakes"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	backuperfakes "github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"
)

var _ = Describe("artifact", func() {

	var sshConnection *fakes.FakeSSHConnection
	var boshLogger boshlog.Logger
	var testInstance *backuperfakes.FakeInstance
	var stdout, stderr *gbytes.Buffer
	var job instance.Job

	var backupArtifact orchestrator.BackupArtifact

	BeforeEach(func() {
		sshConnection = new(fakes.FakeSSHConnection)
		testInstance = new(backuperfakes.FakeInstance)
		testInstance.NameReturns("redis")
		testInstance.IDReturns("foo")
		testInstance.IndexReturns("redis-index-1")

		stdout = gbytes.NewBuffer()
		stderr = gbytes.NewBuffer()
		boshLogger = boshlog.New(boshlog.LevelDebug, log.New(stdout, "[bosh-package] ", log.Lshortfile), log.New(stderr, "[bosh-package] ", log.Lshortfile))

	})
	var ArtifactBehaviourForDirectory = func(artifactDirectory string) {
		Describe("StreamFromRemote", func() {
			var err error
			var writer = bytes.NewBufferString("dave")

			JustBeforeEach(func() {
				err = backupArtifact.StreamFromRemote(writer)
			})

			Describe("when successful", func() {
				BeforeEach(func() {
					sshConnection.StreamReturns([]byte("not relevant"), 0, nil)
				})

				It("uses the ssh connection to tar the backup and stream it to the local machine", func() {
					Expect(sshConnection.StreamCallCount()).To(Equal(1))

					cmd, returnedWriter := sshConnection.StreamArgsForCall(0)
					Expect(cmd).To(Equal("sudo tar -C " + artifactDirectory + " -c ."))
					Expect(returnedWriter).To(Equal(writer))
				})

				It("does not fail", func() {
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Describe("when there is an error tarring the backup", func() {
				BeforeEach(func() {
					sshConnection.StreamReturns([]byte("not relevant"), 1, nil)
				})

				It("uses the ssh connection to tar the backup and stream it to the local machine", func() {
					Expect(sshConnection.StreamCallCount()).To(Equal(1))

					cmd, returnedWriter := sshConnection.StreamArgsForCall(0)
					Expect(cmd).To(Equal("sudo tar -C " + artifactDirectory + " -c ."))
					Expect(returnedWriter).To(Equal(writer))
				})

				It("fails", func() {
					Expect(err).To(HaveOccurred())
				})
			})

			Describe("when there is an SSH error", func() {
				var sshError error

				BeforeEach(func() {
					sshError = fmt.Errorf("SHH causing problems here")
					sshConnection.StreamReturns([]byte("not relevant"), -1, sshError)
				})

				It("uses the ssh connection to tar the backup and stream it to the local machine", func() {
					Expect(sshConnection.StreamCallCount()).To(Equal(1))

					cmd, returnedWriter := sshConnection.StreamArgsForCall(0)
					Expect(cmd).To(Equal("sudo tar -C " + artifactDirectory + " -c ."))
					Expect(returnedWriter).To(Equal(writer))
				})

				It("fails", func() {
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring(sshError.Error())))
				})
			})
		})

		Describe("BackupChecksum", func() {
			var actualChecksum map[string]string
			var actualChecksumError error

			JustBeforeEach(func() {
				actualChecksum, actualChecksumError = backupArtifact.Checksum()
			})

			Context("triggers find & shasum as root", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("not relevant"), nil, 0, nil)
				})

				It("generates the correct request", func() {
					Expect(sshConnection.RunArgsForCall(0)).To(Equal("cd " + artifactDirectory + "; sudo sh -c 'find . -type f | xargs shasum -a 256'"))
				})
			})
			Context("can calculate checksum", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("07fc29fb3aacd99f7f7b81df9c43b13e71c56a1e  file1\n07fc29fb3aacd99f7f7b81df9c43b13e71c56a1e  file2\nn87fc29fb3aacd99f7f7b81df9c43b13e71c56a1e file3/file4"), nil, 0, nil)
				})
				It("converts the checksum to a map", func() {
					Expect(actualChecksumError).NotTo(HaveOccurred())
					Expect(actualChecksum).To(Equal(map[string]string{
						"file1":       "07fc29fb3aacd99f7f7b81df9c43b13e71c56a1e",
						"file2":       "07fc29fb3aacd99f7f7b81df9c43b13e71c56a1e",
						"file3/file4": "n87fc29fb3aacd99f7f7b81df9c43b13e71c56a1e",
					}))
				})
			})
			Context("can calculate checksum, with trailing spaces", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("07fc29fb3aacd99f7f7b81df9c43b13e71c56a1e file1\n"), nil, 0, nil)
				})
				It("converts the checksum to a map", func() {
					Expect(actualChecksumError).NotTo(HaveOccurred())
					Expect(actualChecksum).To(Equal(map[string]string{
						"file1": "07fc29fb3aacd99f7f7b81df9c43b13e71c56a1e",
					}))
				})
			})
			Context("sha output is empty", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte(""), nil, 0, nil)
				})
				It("converts an empty map", func() {
					Expect(actualChecksumError).NotTo(HaveOccurred())
					Expect(actualChecksum).To(Equal(map[string]string{}))
				})
			})
			Context("sha for a empty directory", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("da39a3ee5e6b4b0d3255bfef95601890afd80709  -"), nil, 0, nil)
				})
				It("reject '-' as a filename", func() {
					Expect(actualChecksumError).NotTo(HaveOccurred())
					Expect(actualChecksum).To(Equal(map[string]string{}))
				})
			})

			Context("fails to calculate checksum", func() {
				expectedErr := fmt.Errorf("some error")

				BeforeEach(func() {
					sshConnection.RunReturns(nil, nil, 0, expectedErr)
				})
				It("returns an error", func() {
					Expect(actualChecksumError).To(MatchError(expectedErr))
				})
			})
			Context("fails to execute the command", func() {
				BeforeEach(func() {
					sshConnection.RunReturns(nil, nil, 1, nil)
				})
				It("returns an error", func() {
					Expect(actualChecksumError).To(HaveOccurred())
				})
			})
		})

		Describe("Delete", func() {
			var err error

			JustBeforeEach(func() {
				err = backupArtifact.Delete()
			})

			It("succeeds", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("deletes only the named artifact directory on the remote", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo rm -rf " + artifactDirectory))
			})

			Context("when there is an error with the SSH connection", func() {
				var expectedErr error

				BeforeEach(func() {
					expectedErr = fmt.Errorf("nope")
					sshConnection.RunReturns([]byte("don't matter"), []byte("don't matter"), 0, expectedErr)
				})

				It("fails", func() {
					Expect(err).To(MatchError(expectedErr))
				})
			})

			Context("when the rm command returns an error", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("don't matter"), []byte("don't matter"), 1, nil)
				})

				It("fails", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(
						"Error deleting artifact on instance redis/foo. Directory name " + artifactDirectory + ". Exit code 1",
					))
				})
			})
		})

		Describe("BackupSize", func() {
			Context("when there is a backup", func() {
				var size string

				BeforeEach(func() {
					sshConnection.RunReturns([]byte("4.1G\n"), nil, 0, nil)
				})

				JustBeforeEach(func() {
					size, _ = backupArtifact.Size()
				})

				It("returns the size of the backup according to the root user, as a string", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
					Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo du -sh " + artifactDirectory + " | cut -f1"))
					Expect(size).To(Equal("4.1G"))
				})
			})

			Context("when there is no backup directory", func() {
				var err error

				BeforeEach(func() {
					sshConnection.RunReturns(nil, nil, 1, nil) // simulating file not found
				})

				JustBeforeEach(func() {
					_, err = backupArtifact.Size()
				})

				It("returns the size of the backup according to the root user, as a string", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
					Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo du -sh " + artifactDirectory + " | cut -f1"))
					Expect(err).To(HaveOccurred())
				})
			})

			Context("when an error occurs", func() {
				var err error
				var actualError = errors.New("oh noes, more errors")

				BeforeEach(func() {
					sshConnection.RunReturns(nil, nil, 0, actualError)
				})

				JustBeforeEach(func() {
					_, err = backupArtifact.Size()
				})

				It("returns the error", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
					Expect(err).To(MatchError(actualError))
				})
			})

		})

		Describe("StreamBackupToRemote", func() {
			var err error
			var reader = bytes.NewBufferString("dave")

			JustBeforeEach(func() {
				err = backupArtifact.StreamToRemote(reader)
			})

			Describe("when successful", func() {
				It("uses the ssh connection to make the backup directory on the remote machine", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
					command := sshConnection.RunArgsForCall(0)
					Expect(command).To(Equal("sudo mkdir -p " + artifactDirectory))
				})

				It("uses the ssh connection to stream files from the remote machine", func() {
					Expect(sshConnection.StreamStdinCallCount()).To(Equal(1))
					command, sentReader := sshConnection.StreamStdinArgsForCall(0)
					Expect(command).To(Equal("sudo sh -c 'tar -C " + artifactDirectory + " -x'"))
					Expect(reader).To(Equal(sentReader))
				})

				It("does not fail", func() {
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Describe("when the remote side returns an error", func() {
				BeforeEach(func() {
					sshConnection.StreamStdinReturns([]byte("not relevant"), []byte("All the pies"), 1, nil)
				})

				It("fails and return the error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("All the pies"))
				})
			})

			Describe("when there is an error running the stream", func() {
				BeforeEach(func() {
					sshConnection.StreamStdinReturns([]byte("not relevant"), []byte("not relevant"), -1, fmt.Errorf("Errorerrororororororor"))
				})

				It("fails", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Errorerrororororororor"))
				})
			})

			Describe("when creating the directory fails on the remote", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 1, nil)
				})

				It("fails and returns the error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Creating backup directory on the remote returned 1"))
				})
			})

			Describe("when creating the directory fails because of a connection error", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("not relevant"), []byte("not relevant"), 0, fmt.Errorf("I refuse to create you this directory."))
				})

				It("fails and returns the error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("I refuse to create you this directory."))
				})
			})
		})
	}

	Context("BackupArtifact", func() {
		JustBeforeEach(func() {
			backupArtifact = instance.NewBackupArtifact(job, testInstance, sshConnection, boshLogger)
		})
		Context("Named Artifact", func() {
			BeforeEach(func() {
				job = instance.NewJob(nil,
					"",
					nil,
					instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo1/start_ctl"},
					instance.Metadata{BackupName: "named-artifact-to-backup"})
			})

			It("is named with the job's custom backup name", func() {
				Expect(backupArtifact.Name()).To(Equal(job.BackupArtifactName()))
			})

			It("has a custom name", func() {
				Expect(backupArtifact.HasCustomName()).To(BeTrue())
			})

			ArtifactBehaviourForDirectory("/var/vcap/store/bbr-backup/named-artifact-to-backup")
		})
		Context("Default Artifact", func() {
			BeforeEach(func() {
				job = instance.NewJob(nil,
					"",
					nil,
					instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo1/start_ctl"},
					instance.Metadata{})
			})

			It("is named after the job", func() {
				Expect(backupArtifact.Name()).To(Equal(job.Name()))
			})

			It("does not have a custom name", func() {
				Expect(backupArtifact.HasCustomName()).To(BeFalse())
			})

			Describe("InstanceName", func() {
				It("returns the instance name", func() {
					Expect(backupArtifact.InstanceName()).To(Equal(testInstance.Name()))
				})
			})

			Describe("InstanceIndex", func() {
				It("returns the instance index", func() {
					Expect(backupArtifact.InstanceIndex()).To(Equal(testInstance.Index()))
				})
			})

			ArtifactBehaviourForDirectory("/var/vcap/store/bbr-backup/foo1")
		})
	})

	Context("RestoreArtifact", func() {
		JustBeforeEach(func() {
			backupArtifact = instance.NewRestoreArtifact(job, testInstance, sshConnection, boshLogger)
		})
		Context("Named Artifact", func() {
			BeforeEach(func() {
				job = instance.NewJob(nil,
					"",
					nil,
					instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo1/start_ctl"},
					instance.Metadata{RestoreName: "named-artifact-to-restore"})
			})

			It("is named with the job's custom backup name", func() {
				Expect(backupArtifact.Name()).To(Equal(job.RestoreArtifactName()))
			})

			It("has a custom name", func() {
				Expect(backupArtifact.HasCustomName()).To(BeTrue())
			})

			ArtifactBehaviourForDirectory("/var/vcap/store/bbr-backup/named-artifact-to-restore")
		})
		Context("Default Artifact", func() {
			BeforeEach(func() {
				job = instance.NewJob(nil,
					"",
					nil,
					instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo1/start_ctl"},
					instance.Metadata{})
			})

			It("is named after the job", func() {
				Expect(backupArtifact.Name()).To(Equal(job.Name()))
			})

			It("does not have a custom name", func() {
				Expect(backupArtifact.HasCustomName()).To(BeFalse())
			})

			Describe("InstanceName", func() {
				It("returns the instance name", func() {
					Expect(backupArtifact.InstanceName()).To(Equal(testInstance.Name()))
				})
			})

			Describe("InstanceIndex", func() {
				It("returns the instance index", func() {
					Expect(backupArtifact.InstanceIndex()).To(Equal(testInstance.Index()))
				})
			})

			ArtifactBehaviourForDirectory("/var/vcap/store/bbr-backup/foo1")
		})
	})
})
