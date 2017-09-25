package instance_test

import (
	"errors"
	"fmt"
	"log"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh/fakes"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var _ = Describe("DeployedInstance", func() {
	var sshConnection *fakes.FakeSSHConnection
	var boshLogger boshlog.Logger
	var stdout, stderr *gbytes.Buffer
	var jobName, jobIndex, jobID, expectedStdout, expectedStderr string
	var backupAndRestoreScripts []instance.Script
	var jobs instance.Jobs
	var metadata map[string]instance.Metadata

	var deployedInstance *instance.DeployedInstance
	BeforeEach(func() {
		sshConnection = new(fakes.FakeSSHConnection)
		jobName = "job-name"
		jobIndex = "job-index"
		jobID = "job-id"
		expectedStdout = "i'm a stdout"
		expectedStderr = "i'm a stderr"
		stdout = gbytes.NewBuffer()
		stderr = gbytes.NewBuffer()
		boshLogger = boshlog.New(boshlog.LevelDebug, log.New(stdout, "[bosh-package] ", log.Lshortfile), log.New(stderr, "[bosh-package] ", log.Lshortfile))
		backupAndRestoreScripts = []instance.Script{}
		metadata = map[string]instance.Metadata{}
	})

	JustBeforeEach(func() {
		sshConnection.UsernameReturns("sshUsername")
		jobs = instance.NewJobs(sshConnection, jobName+"/"+jobID, boshLogger, backupAndRestoreScripts, metadata)
		deployedInstance = instance.NewDeployedInstance(
			jobIndex,
			jobName,
			jobID,
			false,
			sshConnection,
			boshLogger,
			jobs)
	})

	Describe("HasBackupScript", func() {
		var actualBackupable bool

		JustBeforeEach(func() {
			actualBackupable = deployedInstance.IsBackupable()
		})

		Describe("there are backup scripts in the job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/dave/bin/bbr/backup",
				}
			})

			It("returns true", func() {
				Expect(actualBackupable).To(BeTrue())
			})
		})

		Describe("there are no backup scripts in the job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/dave/bin/foo",
				}
			})

			It("returns false", func() {
				Expect(actualBackupable).To(BeFalse())
			})
		})
	})

	Describe("ArtifactDirExists", func() {
		var sshExitCode int
		var sshError error

		var dirExists bool
		var dirError error

		JustBeforeEach(func() {
			sshConnection.RunReturns([]byte{}, []byte{}, sshExitCode, sshError)
			dirExists, dirError = deployedInstance.ArtifactDirExists()
		})

		BeforeEach(func() {
			sshExitCode = 1
		})

		Context("when artifact directory does not exist", func() {
			It("calls the ssh connection", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("stat /var/vcap/store/bbr-backup"))
			})

			It("returns false", func() {
				Expect(dirExists).To(BeFalse())
			})
		})

		Context("when artifact directory exists", func() {
			BeforeEach(func() {
				sshExitCode = 0
			})

			It("calls the ssh connection", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("stat /var/vcap/store/bbr-backup"))
			})

			It("returns true", func() {
				Expect(dirExists).To(BeTrue())
			})
		})

		Context("when ssh connection error occurs", func() {
			BeforeEach(func() {
				sshError = fmt.Errorf("argh!")
			})

			It("returns the error", func() {
				Expect(dirError).To(MatchError("argh!"))
			})
		})
	})

	Describe("IsRestorable", func() {
		var actualRestorable bool

		JustBeforeEach(func() {
			actualRestorable = deployedInstance.IsRestorable()
		})

		Describe("there are restore scripts in the job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/dave/bin/bbr/restore",
				}
			})

			It("returns true", func() {
				Expect(actualRestorable).To(BeTrue())
			})
		})

		Describe("there are no restore scripts", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/dave/bin/foo",
				}
			})

			It("returns false", func() {
				Expect(actualRestorable).To(BeFalse())
			})
		})
	})

	Describe("CustomBackupArtifactNames", func() {
		Context("when the instance has custom artifact names defined", func() {
			BeforeEach(func() {
				metadata = map[string]instance.Metadata{
					"dave": {BackupName: "foo"},
				}
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/dave/bin/foo",
				}
			})

			It("returns a list of the instance's custom artifact names", func() {
				Expect(deployedInstance.CustomBackupArtifactNames()).To(ConsistOf("foo"))
			})
		})

	})

	Describe("CustomRestoreArtifactNames", func() {
		Context("when the instance has custom restore artifact names defined", func() {
			BeforeEach(func() {
				metadata = map[string]instance.Metadata{
					"dave": {RestoreName: "foo"},
				}
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/dave/bin/foo",
				}
			})

			It("returns a list of the instance's custom restore artifact names", func() {
				Expect(deployedInstance.CustomRestoreArtifactNames()).To(ConsistOf("foo"))
			})
		})

	})

	Describe("PreBackupLock", func() {
		var err error

		JustBeforeEach(func() {
			err = deployedInstance.PreBackupLock()
		})

		Context("when there is one pre-backup-lock script in the job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{"/var/vcap/jobs/bar/bin/bbr/pre-backup-lock"}
			})

			It("uses the ssh connection to run the pre-backup-lock script", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal(
					"sudo /var/vcap/jobs/bar/bin/bbr/pre-backup-lock",
				))
			})

			It("logs the paths to the scripts being run", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/bar/bin/bbr/pre-backup-lock`))
				Expect(string(stdout.Contents())).NotTo(ContainSubstring("> \n"))
			})

			It("logs the job being locked", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Locking bar on %s/%s",
					jobName,
					jobID,
				)))
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Done")))
			})

			It("succeeds", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when there are multiple backup scripts in multiple job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/pre-backup-lock",
					"/var/vcap/jobs/bar/bin/bbr/pre-backup-lock",
					"/var/vcap/jobs/baz/bin/bbr/pre-backup-lock",
				}
			})

			It("uses the ssh connection to run each of the pre-backup-lock scripts", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(3))
				Expect([]string{
					sshConnection.RunArgsForCall(0),
					sshConnection.RunArgsForCall(1),
					sshConnection.RunArgsForCall(2),
				}).To(ConsistOf(
					"sudo /var/vcap/jobs/foo/bin/bbr/pre-backup-lock",
					"sudo /var/vcap/jobs/bar/bin/bbr/pre-backup-lock",
					"sudo /var/vcap/jobs/baz/bin/bbr/pre-backup-lock",
				))
			})

			It("logs the paths to the scripts being run", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/foo/bin/bbr/pre-backup-lock`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/bar/bin/bbr/pre-backup-lock`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/baz/bin/bbr/pre-backup-lock`))
				Expect(string(stdout.Contents())).NotTo(ContainSubstring("> \n"))
			})

			It("logs that it is locking the job on the instance", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Locking foo on %s/%s",
					jobName,
					jobID,
				)))
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Done")))
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Locking bar on %s/%s",
					jobName,
					jobID,
				)))
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Done")))
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Locking baz on %s/%s",
					jobName,
					jobID,
				)))
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Done")))
			})

			It("logs Done.", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring("Done."))
			})

			It("succeeds", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when there are several scripts and one of them fails to run pre backup lock while another one causes an error", func() {
			expectedStdout := "some stdout"
			expectedStderr := "some stderr"
			expectedError := errors.New("Errororororor")

			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/pre-backup-lock",
					"/var/vcap/jobs/bar/bin/bbr/pre-backup-lock",
					"/var/vcap/jobs/baz/bin/bbr/pre-backup-lock",
				}
				sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
					if strings.Contains(cmd, "jobs/bar") {
						return []byte(expectedStdout), []byte(expectedStderr), 1, nil
					}
					if strings.Contains(cmd, "jobs/baz") {
						return []byte("not relevant"), []byte("not relevant"), 0, expectedError
					}
					return []byte("not relevant"), []byte("not relevant"), 0, nil
				}
			})

			It("fails", func() {
				Expect(err).To(HaveOccurred())
			})

			It("returns an error including the failure for the failed script", func() {
				Expect(err.Error()).To(ContainSubstring(
					fmt.Sprintf("pre backup lock script for job bar failed on %s/%s", jobName, jobID),
				))
			})

			It("logs the failures related to the failed script", func() {
				Expect(string(stderr.Contents())).To(ContainSubstring(
					fmt.Sprintf("pre backup lock script for job bar failed on %s/%s", jobName, jobID),
				))
			})

			It("returns an error without a message related to the script which passed", func() {
				Expect(err.Error()).NotTo(ContainSubstring(
					fmt.Sprintf("pre backup lock script for job foo failed on %s/%s", jobName, jobID),
				))
			})

			It("prints stdout from the failing job", func() {
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Stdout: %s", expectedStdout)))
			})

			It("prints stderr from the failing job", func() {
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Stderr: %s", expectedStderr)))
			})

			It("returns an error including the error from running the command", func() {
				Expect(err.Error()).To(ContainSubstring(expectedError.Error()))
			})

			It("logs the error caused when running the command", func() {
				Expect(string(stderr.Contents())).To(ContainSubstring(fmt.Sprintf(
					"Error attempting to run pre backup lock script for job baz on %s/%s. Error: %s",
					jobName,
					jobID,
					expectedError.Error(),
				)))
			})
		})

	})

	Describe("Backup", func() {
		var err error

		JustBeforeEach(func() {
			err = deployedInstance.Backup()
		})

		Context("when there are multiple backup scripts in multiple job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/backup",
					"/var/vcap/jobs/bar/bin/bbr/backup",
					"/var/vcap/jobs/baz/bin/bbr/backup",
				}
			})

			It("uses the ssh connection to create each job's backup folder and run each backup script providing the correct ARTIFACT_DIRECTORY and BBR_ARTIFACT_DIRECTORY", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(3))
				Expect([]string{
					sshConnection.RunArgsForCall(0),
					sshConnection.RunArgsForCall(1),
					sshConnection.RunArgsForCall(2),
				}).To(ConsistOf(
					"sudo mkdir -p /var/vcap/store/bbr-backup/foo && sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/foo/ ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/foo/ /var/vcap/jobs/foo/bin/bbr/backup",
					"sudo mkdir -p /var/vcap/store/bbr-backup/bar && sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/bar/ ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/bar/ /var/vcap/jobs/bar/bin/bbr/backup",
					"sudo mkdir -p /var/vcap/store/bbr-backup/baz && sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/baz/ ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/baz/ /var/vcap/jobs/baz/bin/bbr/backup",
				))
			})

			It("logs the paths to the scripts being run", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/foo/bin/bbr/backup`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/bar/bin/bbr/backup`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/baz/bin/bbr/backup`))
				Expect(string(stdout.Contents())).NotTo(ContainSubstring("> \n"))
			})

			It("logs that it is backing up the job on the instance", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Backing up foo on %s/%s",
					jobName,
					jobID,
				)))
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Backing up bar on %s/%s",
					jobName,
					jobID,
				)))
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Backing up baz on %s/%s",
					jobName,
					jobID,
				)))
			})

			It("logs Done.", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring("Done."))
			})

			It("marks the instance as having had its artifact directory created", func() {
				Expect(deployedInstance.ArtifactDirCreated()).To(BeTrue())
			})

			It("succeeds", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when there are multiple backup scripts and one of them is named", func() {
			BeforeEach(func() {
				metadata = map[string]instance.Metadata{
					"baz": {BackupName: "special-backup"},
				}
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/backup",
					"/var/vcap/jobs/bar/bin/bbr/backup",
					"/var/vcap/jobs/baz/bin/bbr/backup",
				}
			})

			It("uses the ssh connection to create each job's backup folder and run each backup script providing the correct BBR_ARTIFACT_DIRECTORY and ARTIFACT_DIRECTORY", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(3))
				Expect([]string{
					sshConnection.RunArgsForCall(0),
					sshConnection.RunArgsForCall(1),
					sshConnection.RunArgsForCall(2),
				}).To(ConsistOf(
					"sudo mkdir -p /var/vcap/store/bbr-backup/foo && sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/foo/ ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/foo/ /var/vcap/jobs/foo/bin/bbr/backup",
					"sudo mkdir -p /var/vcap/store/bbr-backup/bar && sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/bar/ ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/bar/ /var/vcap/jobs/bar/bin/bbr/backup",
					"sudo mkdir -p /var/vcap/store/bbr-backup/special-backup && sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/special-backup/ ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/special-backup/ /var/vcap/jobs/baz/bin/bbr/backup",
				))
			})
		})

		Context("when there are multiple jobs with no backup scripts", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/restore",
					"/var/vcap/jobs/bar/bin/bbr/restore",
				}
			})
			It("doesn't make calls to the instance over the ssh connection", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(0))
			})
		})

		Context("when there are several scripts and one of them fails to run backup while another one causes an error", func() {
			expectedStdout := "some stdout"
			expectedStderr := "some stderr"
			expectedError := fmt.Errorf("I have a problem with your code")

			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/backup",
					"/var/vcap/jobs/bar/bin/bbr/backup",
					"/var/vcap/jobs/baz/bin/bbr/backup",
				}
				sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
					if strings.Contains(cmd, "jobs/bar") {
						return []byte(expectedStdout), []byte(expectedStderr), 1, nil
					}
					if strings.Contains(cmd, "jobs/baz") {
						return []byte("not relevant"), []byte("not relevant"), 0, expectedError
					}
					return []byte("not relevant"), []byte("not relevant"), 0, nil
				}
			})

			It("fails", func() {
				Expect(err).To(HaveOccurred())
			})

			It("returns an error including the failure for the failed script", func() {
				Expect(err.Error()).To(ContainSubstring(
					fmt.Sprintf("backup script for job bar failed on %s/%s", jobName, jobID),
				))
			})

			It("logs the failures related to the failed script", func() {
				Expect(string(stderr.Contents())).To(ContainSubstring(
					fmt.Sprintf("backup script for job bar failed on %s/%s", jobName, jobID),
				))
			})

			It("returns an error without a message related to the script which passed", func() {
				Expect(err.Error()).NotTo(ContainSubstring(
					fmt.Sprintf("backup script for job foo failed on %s/%s", jobName, jobID),
				))
			})

			It("prints stdout from the failing job", func() {
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Stdout: %s", expectedStdout)))
			})

			It("prints stderr from the failing job", func() {
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Stderr: %s", expectedStderr)))
			})

			It("returns an error including the error from running the command", func() {
				Expect(err.Error()).To(ContainSubstring(expectedError.Error()))
			})

			It("logs the error caused when running the command", func() {
				Expect(string(stderr.Contents())).To(ContainSubstring(fmt.Sprintf(
					"Error attempting to run backup script for job baz on %s/%s. Error: %s",
					jobName,
					jobID,
					expectedError.Error(),
				)))
			})

		})
	})

	Describe("PostBackupUnlock", func() {
		var err error

		JustBeforeEach(func() {
			err = deployedInstance.PostBackupUnlock()
		})

		Context("when there are multiple post-backup-unlock scripts in multiple job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/post-backup-unlock",
					"/var/vcap/jobs/bar/bin/bbr/post-backup-unlock",
					"/var/vcap/jobs/baz/bin/bbr/post-backup-unlock",
				}
			})

			It("uses the ssh connection to run each post-backup-unlock script", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(3))
				Expect([]string{
					sshConnection.RunArgsForCall(0),
					sshConnection.RunArgsForCall(1),
					sshConnection.RunArgsForCall(2),
				}).To(ConsistOf(
					"sudo /var/vcap/jobs/foo/bin/bbr/post-backup-unlock",
					"sudo /var/vcap/jobs/bar/bin/bbr/post-backup-unlock",
					"sudo /var/vcap/jobs/baz/bin/bbr/post-backup-unlock",
				))
			})

			It("logs the paths to the scripts being run", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/foo/bin/bbr/post-backup-unlock`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/bar/bin/bbr/post-backup-unlock`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/baz/bin/bbr/post-backup-unlock`))
				Expect(string(stdout.Contents())).NotTo(ContainSubstring("> \n"))
			})

			It("logs that it is backing up the job on the instance", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Unlocking foo on %s/%s",
					jobName,
					jobID,
				)))

				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Unlocking bar on %s/%s",
					jobName,
					jobID,
				)))

				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Unlocking baz on %s/%s",
					jobName,
					jobID,
				)))
			})

			It("logs Done.", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring("Done."))
			})

			It("succeeds", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when there are several scripts and one of them fails to run post-backup-unlock while another one causes an error", func() {
			expectedStdout := "some stdout"
			expectedStderr := "some stderr"
			sshConnectionError := fmt.Errorf("I still have a problem with your code")

			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/post-backup-unlock",
					"/var/vcap/jobs/bar/bin/bbr/post-backup-unlock",
					"/var/vcap/jobs/baz/bin/bbr/post-backup-unlock",
				}
				sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
					if strings.Contains(cmd, "jobs/bar") {
						return []byte(expectedStdout), []byte(expectedStderr), 1, nil
					}
					if strings.Contains(cmd, "jobs/baz") {
						return []byte("not relevant"), []byte("not relevant"), 0, sshConnectionError
					}
					return []byte("not relevant"), []byte("not relevant"), 0, nil
				}
			})

			It("fails", func() {
				Expect(err).To(HaveOccurred())
			})

			It("returns an error including the failure for the failed script", func() {
				Expect(err.Error()).To(ContainSubstring(
					fmt.Sprintf("unlock script for job bar failed on %s/%s", jobName, jobID),
				))
			})

			It("logs the failures related to the failed script", func() {
				Expect(string(stderr.Contents())).To(ContainSubstring(
					fmt.Sprintf("unlock script for job bar failed on %s/%s", jobName, jobID),
				))
			})

			It("returns an error without a message related to the script which passed", func() {
				Expect(err.Error()).NotTo(ContainSubstring(
					fmt.Sprintf("unlock script for job foo failed on %s/%s", jobName, jobID),
				))
			})

			It("prints stdout from the failing job", func() {
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Stdout: %s", expectedStdout)))
			})

			It("prints stderr from the failing job", func() {
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Stderr: %s", expectedStderr)))
			})

			It("returns an error including the error from running the command", func() {
				Expect(err.Error()).To(ContainSubstring(sshConnectionError.Error()))
			})

			It("logs the error caused when running the command", func() {
				Expect(string(stderr.Contents())).To(ContainSubstring(fmt.Sprintf(
					"Error attempting to run unlock script for job baz on %s/%s. Error: %s",
					jobName,
					jobID,
					sshConnectionError.Error(),
				)))
			})

		})
	})

	Describe("PostRestoreUnlock", func() {
		var postRestoreUnlockError error

		JustBeforeEach(func() {
			postRestoreUnlockError = deployedInstance.PostRestoreUnlock()
		})

		Context("when there are multiple post-restore-unlock scripts in multiple job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/post-restore-unlock",
					"/var/vcap/jobs/bar/bin/bbr/post-restore-unlock",
					"/var/vcap/jobs/baz/bin/bbr/post-restore-unlock",
				}
			})

			It("uses the ssh connection to run each post-restore-unlock script", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(3))
				Expect([]string{
					sshConnection.RunArgsForCall(0),
					sshConnection.RunArgsForCall(1),
					sshConnection.RunArgsForCall(2),
				}).To(ConsistOf(
					"sudo /var/vcap/jobs/foo/bin/bbr/post-restore-unlock",
					"sudo /var/vcap/jobs/bar/bin/bbr/post-restore-unlock",
					"sudo /var/vcap/jobs/baz/bin/bbr/post-restore-unlock",
				))
			})

			It("logs the paths to the scripts being run", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/foo/bin/bbr/post-restore-unlock`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/bar/bin/bbr/post-restore-unlock`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/baz/bin/bbr/post-restore-unlock`))
				Expect(string(stdout.Contents())).NotTo(ContainSubstring("> \n"))
			})

			It("logs that it is unlocking the job on the instance", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Unlocking foo on %s/%s",
					jobName,
					jobID,
				)))

				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Unlocking bar on %s/%s",
					jobName,
					jobID,
				)))

				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"INFO - Unlocking baz on %s/%s",
					jobName,
					jobID,
				)))
			})

			It("logs Done.", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring("Done."))
			})

			It("succeeds", func() {
				Expect(postRestoreUnlockError).NotTo(HaveOccurred())
			})
		})

		Context("when there are several scripts and one of them fails to run post-restore-unlock while another one causes an error", func() {

			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/post-restore-unlock",
					"/var/vcap/jobs/bar/bin/bbr/post-restore-unlock",
					"/var/vcap/jobs/baz/bin/bbr/post-restore-unlock",
				}

				sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
					if strings.Contains(cmd, "jobs/bar") {
						return []byte("stdout_bar"), []byte("stderr_bar"), 1, nil
					}

					if strings.Contains(cmd, "jobs/baz") {
						return []byte("not relevant"), []byte("not relevant"), 0, fmt.Errorf("connection failed, script not run on baz")
					}

					return []byte("not relevant"), []byte("not relevant"), 0, nil
				}
			})

			It("fails", func() {
				Expect(postRestoreUnlockError).To(HaveOccurred())
			})

			It("correctly reports a script failure", func() {
				By("describing the script failure in the returned error", func() {
					Expect(postRestoreUnlockError.Error()).To(ContainSubstring(
						fmt.Sprintf("unlock script for job bar failed on %s/%s", jobName, jobID),
					))
				})

				By("including the script stdout in the returned error", func() {
					Expect(postRestoreUnlockError.Error()).To(ContainSubstring("Stdout: stdout_bar"))
				})

				By("including the script stderr in the returned error", func() {
					Expect(postRestoreUnlockError.Error()).To(ContainSubstring("Stderr: stderr_bar"))
				})

				By("logging to stderr the failures related to the failed script", func() {
					Expect(string(stderr.Contents())).To(ContainSubstring(
						fmt.Sprintf("unlock script for job bar failed on %s/%s", jobName, jobID),
					))
				})
			})

			It("correctly reports failing to run a script (e.g. ssh connection failure)", func() {
				By("including the ssh error message in the returned error ", func() {
					Expect(postRestoreUnlockError.Error()).To(ContainSubstring("connection failed, script not run on baz"))
				})

				By("including the error message in the logs", func() {
					Expect(string(stderr.Contents())).To(ContainSubstring(fmt.Sprintf(
						"Error attempting to run post-restore-unlock script for job baz on %s/%s. Error: %s",
						jobName,
						jobID,
						"connection failed, script not run on baz",
					)))
				})
			})

		})

		Context("When there are some jobs without post-restore-unlock scripts", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/job-has-unlock-script/bin/bbr/post-restore-unlock",
					"/var/vcap/jobs/job-only-has-backup/bin/bbr/backup",
					"/var/vcap/jobs/job-only-has-restore/bin/bbr/restore",
				}
			})

			It("Only invokes post-restore-unlock on those jobs which have that script", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal("sudo /var/vcap/jobs/job-has-unlock-script/bin/bbr/post-restore-unlock"))
			})
		})
	})

	Describe("Restore", func() {
		var actualError error

		JustBeforeEach(func() {
			actualError = deployedInstance.Restore()
		})

		Context("when there are multiple restore scripts in multiple job directories", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/restore",
					"/var/vcap/jobs/bar/bin/bbr/restore",
					"/var/vcap/jobs/baz/bin/bbr/restore",
				}
			})

			It("uses the ssh connection to run each restore script providing the correct ARTIFACT_DIRECTORTY", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(3))
				Expect([]string{
					sshConnection.RunArgsForCall(0),
					sshConnection.RunArgsForCall(1),
					sshConnection.RunArgsForCall(2),
				}).To(ConsistOf(
					"sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/foo/ ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/foo/ /var/vcap/jobs/foo/bin/bbr/restore",
					"sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/bar/ ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/bar/ /var/vcap/jobs/bar/bin/bbr/restore",
					"sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/baz/ ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/baz/ /var/vcap/jobs/baz/bin/bbr/restore",
				))
			})

			It("logs the paths to the scripts being run", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/foo/bin/bbr/restore`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/bar/bin/bbr/restore`))
				Expect(string(stdout.Contents())).To(ContainSubstring(`> /var/vcap/jobs/baz/bin/bbr/restore`))
				Expect(string(stdout.Contents())).NotTo(ContainSubstring("> \n"))
			})

			It("logs that it is restoring a job on the instance", func() {
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"Restoring foo on %s/%s",
					jobName,
					jobID,
				)))
				Expect(string(stdout.Contents())).To(ContainSubstring("Done."))

				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"Restoring bar on %s/%s",
					jobName,
					jobID,
				)))
				Expect(string(stdout.Contents())).To(ContainSubstring("Done."))
				Expect(string(stdout.Contents())).To(ContainSubstring(fmt.Sprintf(
					"Restoring baz on %s/%s",
					jobName,
					jobID,
				)))
				Expect(string(stdout.Contents())).To(ContainSubstring("Done."))

			})

			It("succeeds", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})
		})

		Context("when there are multiple restore scripts and one of them is named", func() {
			BeforeEach(func() {
				metadata = map[string]instance.Metadata{
					"baz": {RestoreName: "special-backup"},
				}
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/restore",
					"/var/vcap/jobs/bar/bin/bbr/restore",
					"/var/vcap/jobs/baz/bin/bbr/restore",
				}
			})
			It("succeeds", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})
			It("uses the ssh connection to create each job's backup folder and run each backup script providing the correct BBR_ARTIFACT_DIRECTORY and ARTIFACT_DIRECTORY", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(3))
				Expect([]string{
					sshConnection.RunArgsForCall(0),
					sshConnection.RunArgsForCall(1),
					sshConnection.RunArgsForCall(2),
				}).To(ConsistOf(
					"sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/foo/ ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/foo/ /var/vcap/jobs/foo/bin/bbr/restore",
					"sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/bar/ ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/bar/ /var/vcap/jobs/bar/bin/bbr/restore",
					"sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/special-backup/ ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/special-backup/ /var/vcap/jobs/baz/bin/bbr/restore",
				))
			})
		})

		Context("when there are several scripts and one of them fails to run restore while another one causes an error", func() {
			expectedStdout := "some stdout"
			expectedStderr := "some stderr"
			expectedError := fmt.Errorf("foo bar baz error")

			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/restore",
					"/var/vcap/jobs/bar/bin/bbr/restore",
					"/var/vcap/jobs/baz/bin/bbr/restore",
				}
				sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
					if strings.Contains(cmd, "jobs/bar") {
						return []byte(expectedStdout), []byte(expectedStderr), 1, nil
					}
					if strings.Contains(cmd, "jobs/baz") {
						return []byte("not relevant"), []byte("not relevant"), 0, expectedError
					}
					return []byte("not relevant"), []byte("not relevant"), 0, nil
				}
			})

			It("fails", func() {
				Expect(actualError).To(HaveOccurred())
			})

			It("returns an error including the failure for the failed script", func() {
				Expect(actualError.Error()).To(ContainSubstring(
					fmt.Sprintf("restore script for job bar failed on %s/%s", jobName, jobID),
				))
			})

			It("logs the failures related to the failed script", func() {
				Expect(string(stderr.Contents())).To(ContainSubstring(
					fmt.Sprintf("restore script for job bar failed on %s/%s", jobName, jobID),
				))
			})

			It("returns an error without a message related to the script which passed", func() {
				Expect(actualError.Error()).NotTo(ContainSubstring(
					fmt.Sprintf("restore script for job foo failed on %s/%s", jobName, jobID),
				))
			})

			It("prints stdout from the failing job", func() {
				Expect(actualError.Error()).To(ContainSubstring(fmt.Sprintf("Stdout: %s", expectedStdout)))
			})

			It("prints stderr from the failing job", func() {
				Expect(actualError.Error()).To(ContainSubstring(fmt.Sprintf("Stderr: %s", expectedStderr)))
			})

			It("returns an error including the error from running the command", func() {
				Expect(actualError.Error()).To(ContainSubstring(expectedError.Error()))
			})

			It("logs the error caused when running the command", func() {
				Expect(string(stderr.Contents())).To(ContainSubstring(fmt.Sprintf(
					"Error attempting to run restore script for job baz on %s/%s. Error: %s",
					jobName,
					jobID,
					expectedError.Error(),
				)))
			})

		})
	})

	Describe("Name", func() {
		It("returns the instance name", func() {
			Expect(deployedInstance.Name()).To(Equal("job-name"))
		})
	})

	Describe("Index", func() {
		It("returns the instance Index", func() {
			Expect(deployedInstance.Index()).To(Equal("job-index"))
		})
	})

	Describe("ArtifactsToBackup", func() {
		var backupArtifacts []orchestrator.BackupArtifact

		JustBeforeEach(func() {
			backupArtifacts = deployedInstance.ArtifactsToBackup()
		})

		Context("Has no named backup artifacts", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/backup",
					"/var/vcap/jobs/bar/bin/bbr/backup",
				}
			})
			It("returns artifacts with default names", func() {
				Expect(backupArtifacts).To(ConsistOf(
					instance.NewBackupArtifact(
						instance.NewJob(sshConnection,
							"",
							boshLogger,
							[]instance.Script{backupAndRestoreScripts[0]},
							instance.Metadata{}),
						deployedInstance,
						sshConnection,
						boshLogger),
					instance.NewBackupArtifact(
						instance.NewJob(sshConnection,
							"",
							boshLogger,
							[]instance.Script{backupAndRestoreScripts[1]},
							instance.Metadata{}),
						deployedInstance,
						sshConnection,
						boshLogger),
				))
			})
		})

		Context("Has a named backup artifact and a default artifact", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/backup",
					"/var/vcap/jobs/job-name/bin/bbr/backup",
				}
				metadata = map[string]instance.Metadata{
					"job-name": {BackupName: "my-artifact"},
				}
			})

			It("returns the named artifact and the default artifact", func() {
				Expect(backupArtifacts).To(ConsistOf(
					instance.NewBackupArtifact(instance.NewJob(sshConnection, "", boshLogger, []instance.Script{backupAndRestoreScripts[0]}, instance.Metadata{}), deployedInstance, sshConnection, boshLogger),
					instance.NewBackupArtifact(instance.NewJob(sshConnection, "", boshLogger, []instance.Script{backupAndRestoreScripts[1]}, instance.Metadata{BackupName: "my-artifact"}), deployedInstance, sshConnection, boshLogger),
				))
			})
		})

		Context("Has only a named backup artifact", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/job-name/bin/bbr/backup",
				}
				metadata = map[string]instance.Metadata{
					"job-name": {BackupName: "my-artifact"},
				}
			})

			It("returns the named artifact and the default artifact", func() {
				Expect(backupArtifacts).To(Equal(
					[]orchestrator.BackupArtifact{
						instance.NewBackupArtifact(instance.NewJob(sshConnection, "", boshLogger, backupAndRestoreScripts, instance.Metadata{BackupName: "my-artifact"}), deployedInstance, sshConnection, boshLogger),
					},
				))
			})
		})
	})

	Describe("ArtifactsToRestore", func() {
		var restoreArtifacts []orchestrator.BackupArtifact

		JustBeforeEach(func() {
			restoreArtifacts = deployedInstance.ArtifactsToRestore()
		})

		Context("Has no named restore artifacts", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/foo/bin/bbr/restore",
					"/var/vcap/jobs/bar/bin/bbr/restore",
				}
			})
			It("returns the default artifacts", func() {
				Expect(restoreArtifacts).To(ConsistOf(
					instance.NewRestoreArtifact(instance.NewJob(sshConnection, "", boshLogger, []instance.Script{backupAndRestoreScripts[0]}, instance.Metadata{}), deployedInstance, sshConnection, boshLogger),
					instance.NewRestoreArtifact(instance.NewJob(sshConnection, "", boshLogger, []instance.Script{backupAndRestoreScripts[1]}, instance.Metadata{}), deployedInstance, sshConnection, boshLogger),
				))
			})
		})

		Context("Has a named restore artifact", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/job-name-2/bin/bbr/restore",
					"/var/vcap/jobs/job-name/bin/bbr/restore",
				}
				metadata = map[string]instance.Metadata{
					"job-name": {RestoreName: "my-artifact"},
				}
			})

			It("returns the named artifact and the default artifact", func() {
				Expect(restoreArtifacts).To(ConsistOf(
					instance.NewRestoreArtifact(instance.NewJob(sshConnection, "", boshLogger, []instance.Script{backupAndRestoreScripts[0]}, instance.Metadata{}), deployedInstance, sshConnection, boshLogger),
					instance.NewRestoreArtifact(instance.NewJob(sshConnection, "", boshLogger, []instance.Script{backupAndRestoreScripts[1]}, instance.Metadata{RestoreName: "my-artifact"}), deployedInstance, sshConnection, boshLogger),
				))
			})
		})

		Context("has only named restore artifacts", func() {
			BeforeEach(func() {
				backupAndRestoreScripts = []instance.Script{
					"/var/vcap/jobs/job-name/bin/bbr/restore",
				}
				metadata = map[string]instance.Metadata{
					"job-name": {RestoreName: "my-artifact"},
				}
			})

			It("returns only the named artifact", func() {
				Expect(restoreArtifacts).To(Equal(
					[]orchestrator.BackupArtifact{
						instance.NewRestoreArtifact(instance.NewJob(
							sshConnection, "", boshLogger,
							backupAndRestoreScripts, instance.Metadata{RestoreName: "my-artifact"},
						), deployedInstance, sshConnection, boshLogger),
					},
				))
			})
		})
	})

})
