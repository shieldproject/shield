package bosh_test

import (
	"fmt"
	"log"

	"errors"

	"github.com/cloudfoundry/bosh-cli/director"
	boshfakes "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh/fakes"
)

var _ = Describe("BoshDeployedInstance", func() {
	var sshConnection *fakes.FakeSSHConnection
	var boshDeployment *boshfakes.FakeDeployment
	var boshLogger boshlog.Logger
	var stdout, stderr *gbytes.Buffer
	var jobName, jobIndex, jobID, expectedStdout, expectedStderr string
	var backupAndRestoreScripts []instance.Script
	var jobs instance.Jobs
	var artifactMetadata map[string]instance.Metadata
	var artifactDirCreated bool
	var backuperInstance orchestrator.Instance

	BeforeEach(func() {
		sshConnection = new(fakes.FakeSSHConnection)
		boshDeployment = new(boshfakes.FakeDeployment)
		jobName = "job-name"
		jobIndex = "job-index"
		jobID = "job-id"
		expectedStdout = "i'm a stdout"
		expectedStderr = "i'm a stderr"
		stdout = gbytes.NewBuffer()
		stderr = gbytes.NewBuffer()
		boshLogger = boshlog.New(boshlog.LevelDebug, log.New(stdout, "[bosh-package] ", log.Lshortfile), log.New(stderr, "[bosh-package] ", log.Lshortfile))
		backupAndRestoreScripts = []instance.Script{}
		artifactMetadata = map[string]instance.Metadata{}
		artifactDirCreated = true
	})

	JustBeforeEach(func() {
		jobs = instance.NewJobs(sshConnection,
			"job-name/job-index",
			boshLogger,
			backupAndRestoreScripts,
			artifactMetadata)
		sshConnection.UsernameReturns("sshUsername")
		backuperInstance = bosh.NewBoshDeployedInstance(jobName, jobIndex, jobID, sshConnection, boshDeployment, artifactDirCreated, boshLogger, jobs)
	})

	Describe("Cleanup", func() {
		var actualError error
		var expectedError error

		JustBeforeEach(func() {
			actualError = backuperInstance.Cleanup()
		})

		Describe("cleans up successfully", func() {
			It("deletes the backup folder", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				cmd := sshConnection.RunArgsForCall(0)
				Expect(cmd).To(Equal("sudo rm -rf /var/vcap/store/bbr-backup"))
			})

			It("deletes session from deployment", func() {
				Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
				slug, sshOpts := boshDeployment.CleanUpSSHArgsForCall(0)
				Expect(slug).To(Equal(director.NewAllOrInstanceGroupOrInstanceSlug(jobName, jobID)))
				Expect(sshOpts).To(Equal(director.SSHOpts{
					Username: "sshUsername",
				}))
			})
		})

		Context("when the backup artifact directory was not created this time", func() {
			BeforeEach(func() {
				artifactDirCreated = false
			})

			It("does not delete the existing artifact", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(0))
			})

			It("deletes session from deployment", func() {
				Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
				slug, sshOpts := boshDeployment.CleanUpSSHArgsForCall(0)
				Expect(slug).To(Equal(director.NewAllOrInstanceGroupOrInstanceSlug(jobName, jobID)))
				Expect(sshOpts).To(Equal(director.SSHOpts{
					Username: "sshUsername",
				}))
			})
		})

		Describe("error removing the backup folder", func() {
			BeforeEach(func() {
				expectedError = fmt.Errorf("foo bar")
				sshConnection.RunReturns(nil, nil, 1, expectedError)
			})
			It("tries to cleanup ssh connection", func() {
				Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
			})
			It("returns the error", func() {
				Expect(actualError).To(MatchError(ContainSubstring(expectedError.Error())))
			})
		})

		Describe("error removing the backup folder and an error while running cleaning up the connection", func() {
			var expectedErrorWhileDeleting error
			var expectedErrorWhileCleaningUp error

			BeforeEach(func() {
				expectedErrorWhileDeleting = fmt.Errorf("error while cleaning up var/vcap/store/bbr-backup")
				expectedErrorWhileCleaningUp = fmt.Errorf("error while cleaning the ssh tunnel")
				sshConnection.RunReturns(nil, nil, 1, expectedErrorWhileDeleting)
				boshDeployment.CleanUpSSHReturns(expectedErrorWhileCleaningUp)
			})

			It("tries delete the artifact", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
			})

			It("tries to cleanup ssh connection", func() {
				Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
			})

			It("returns the aggregated error", func() {
				Expect(actualError).To(MatchError(ContainSubstring(expectedErrorWhileDeleting.Error())))
				Expect(actualError).To(MatchError(ContainSubstring(expectedErrorWhileCleaningUp.Error())))
			})
		})

		Describe("error while running cleaning up the connection", func() {
			BeforeEach(func() {
				expectedError = errors.New("werk niet")
				boshDeployment.CleanUpSSHReturns(expectedError)
			})

			It("fails", func() {
				Expect(actualError).To(MatchError(ContainSubstring(expectedError.Error())))
			})
		})
	})

	Describe("CleanupPrevious", func() {
		var actualError error
		var expectedError error

		JustBeforeEach(func() {
			actualError = backuperInstance.CleanupPrevious()
		})

		Describe("cleans up successfully", func() {
			It("deletes the backup folder", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				cmd := sshConnection.RunArgsForCall(0)
				Expect(cmd).To(Equal("sudo rm -rf /var/vcap/store/bbr-backup"))
			})

			It("deletes session from deployment", func() {
				Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
				slug, sshOpts := boshDeployment.CleanUpSSHArgsForCall(0)
				Expect(slug).To(Equal(director.NewAllOrInstanceGroupOrInstanceSlug(jobName, jobID)))
				Expect(sshOpts).To(Equal(director.SSHOpts{
					Username: "sshUsername",
				}))
			})
		})

		Context("when the backup artifact directory was not created this time", func() {
			BeforeEach(func() {
				artifactDirCreated = false
			})

			It("does attempt to delete the existing artifact", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				cmd := sshConnection.RunArgsForCall(0)
				Expect(cmd).To(Equal("sudo rm -rf /var/vcap/store/bbr-backup"))
			})

			It("deletes session from deployment", func() {
				Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
				slug, sshOpts := boshDeployment.CleanUpSSHArgsForCall(0)
				Expect(slug).To(Equal(director.NewAllOrInstanceGroupOrInstanceSlug(jobName, jobID)))
				Expect(sshOpts).To(Equal(director.SSHOpts{
					Username: "sshUsername",
				}))
			})
		})

		Describe("error removing the backup folder", func() {
			BeforeEach(func() {
				expectedError = fmt.Errorf("foo bar")
				sshConnection.RunReturns(nil, nil, 1, expectedError)
			})
			It("tries to cleanup ssh connection", func() {
				Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
			})
			It("returns the error", func() {
				Expect(actualError).To(MatchError(ContainSubstring(expectedError.Error())))
			})
		})

		Describe("error removing the backup folder and an error while running cleaning up the connection", func() {
			var expectedErrorWhileDeleting error
			var expectedErrorWhileCleaningUp error

			BeforeEach(func() {
				expectedErrorWhileDeleting = fmt.Errorf("error while cleaning up var/vcap/store/bbr-backup")
				expectedErrorWhileCleaningUp = fmt.Errorf("error while cleaning the ssh tunnel")
				sshConnection.RunReturns(nil, nil, 1, expectedErrorWhileDeleting)
				boshDeployment.CleanUpSSHReturns(expectedErrorWhileCleaningUp)
			})

			It("tries delete the artifact", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
			})

			It("tries to cleanup ssh connection", func() {
				Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
			})

			It("returns the aggregated error", func() {
				Expect(actualError).To(MatchError(ContainSubstring(expectedErrorWhileDeleting.Error())))
				Expect(actualError).To(MatchError(ContainSubstring(expectedErrorWhileCleaningUp.Error())))
			})
		})

		Describe("error while running cleaning up the connection", func() {
			BeforeEach(func() {
				expectedError = errors.New("werk niet")
				boshDeployment.CleanUpSSHReturns(expectedError)
			})

			It("fails", func() {
				Expect(actualError).To(MatchError(ContainSubstring(expectedError.Error())))
			})
		})
	})

})
