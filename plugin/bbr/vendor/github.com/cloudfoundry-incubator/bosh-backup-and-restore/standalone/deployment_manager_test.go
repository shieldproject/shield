package standalone_test

import (
	"errors"
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/standalone"

	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	instancefakes "github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance/fakes"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"
	sshfakes "github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh/fakes"
)

var _ = Describe("DeploymentManager", func() {
	var deploymentManager DeploymentManager
	var deploymentName = "bosh"
	var artifact *fakes.FakeBackup
	var logger *fakes.FakeLogger
	var hostName = "hostname"
	var username = "username"
	var privateKey string
	var fakeJobFinder *instancefakes.FakeJobFinder
	var fakeConnFactory *sshfakes.FakeSSHConnectionFactory
	var fakeSSHConnection *sshfakes.FakeSSHConnection

	BeforeEach(func() {
		privateKey = createTempFile("privateKey")
		logger = new(fakes.FakeLogger)
		artifact = new(fakes.FakeBackup)
		fakeConnFactory = new(sshfakes.FakeSSHConnectionFactory)
		fakeJobFinder = new(instancefakes.FakeJobFinder)
		fakeSSHConnection = new(sshfakes.FakeSSHConnection)

		deploymentManager = NewDeploymentManager(logger, hostName, username, privateKey, fakeJobFinder, fakeConnFactory.Spy)
	})

	AfterEach(func() {
		os.Remove(privateKey)
	})

	Describe("Find", func() {
		var actualDeployment orchestrator.Deployment
		var actualError error
		var fakeJobs instance.Jobs

		JustBeforeEach(func() {
			actualDeployment, actualError = deploymentManager.Find(deploymentName)
		})

		Context("success", func() {
			BeforeEach(func() {
				fakeJobs = instance.Jobs{instance.NewJob(nil, "", nil, instance.BackupAndRestoreScripts{"foo"}, instance.Metadata{})}
				fakeConnFactory.Returns(fakeSSHConnection, nil)
				fakeJobFinder.FindJobsReturns(fakeJobs, nil)
			})
			It("does not fail", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("invokes connection creator", func() {
				Expect(fakeConnFactory.CallCount()).To(Equal(1))
			})

			It("invokes job finder", func() {
				Expect(fakeJobFinder.FindJobsCallCount()).To(Equal(1))
			})

			It("returns a deployment", func() {
				Expect(actualDeployment).To(Equal(orchestrator.NewDeployment(logger, []orchestrator.Instance{
					NewDeployedInstance("bosh", fakeSSHConnection, logger, fakeJobs, false),
				})))
			})
		})

		Context("can't read private key", func() {
			BeforeEach(func() {
				os.Remove(privateKey)
			})

			It("should fail", func() {
				Expect(actualError).To(MatchError(ContainSubstring("failed reading private key")))
			})

			It("should not invoke connection creator", func() {
				Expect(fakeConnFactory.CallCount()).To(BeZero())
			})
		})

		Context("can't create SSH connection", func() {
			connError := fmt.Errorf("error")

			BeforeEach(func() {
				fakeConnFactory.Returns(nil, connError)
			})

			It("should fail", func() {
				Expect(actualError).To(MatchError(connError))
			})

			It("should invoke connection creator", func() {
				Expect(fakeConnFactory.CallCount()).To(Equal(1))
			})

			It("should not invoke job finder", func() {
				Expect(fakeJobFinder.FindJobsCallCount()).To(BeZero())
			})

		})

		Context("can't find jobs", func() {
			findJobsErr := fmt.Errorf("error")

			BeforeEach(func() {
				fakeConnFactory.Returns(fakeSSHConnection, nil)
				fakeJobFinder.FindJobsReturns(nil, findJobsErr)
			})

			It("should fail", func() {
				Expect(actualError).To(MatchError(findJobsErr))
			})

			It("should invoke connection creator", func() {
				Expect(fakeConnFactory.CallCount()).To(Equal(1))
			})

			It("should not invoke job finder", func() {
				Expect(fakeJobFinder.FindJobsCallCount()).To(Equal(1))
			})
		})

	})

	Describe("SaveManifest", func() {
		It("does nothing", func() {
			err := deploymentManager.SaveManifest(deploymentName, artifact)
			Expect(err).NotTo(HaveOccurred())
		})
	})

})

var _ = Describe("DeployedInstance", func() {
	var logger *fakes.FakeLogger
	var fakeSSHConnection *sshfakes.FakeSSHConnection
	var inst DeployedInstance
	var artifactDirCreated bool

	BeforeEach(func() {
		logger = new(fakes.FakeLogger)
		fakeSSHConnection = new(sshfakes.FakeSSHConnection)
	})

	Describe("Cleanup", func() {
		var err error

		JustBeforeEach(func() {
			inst = NewDeployedInstance("group", fakeSSHConnection, logger, []instance.Job{}, artifactDirCreated)
			err = inst.Cleanup()
		})

		BeforeEach(func() {
			artifactDirCreated = true
		})

		It("does not fail", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("removes the artifact directory", func() {
			Expect(fakeSSHConnection.RunCallCount()).To(Equal(1))
			Expect(fakeSSHConnection.RunArgsForCall(0)).To(Equal(
				"sudo rm -rf /var/vcap/store/bbr-backup",
			))
		})

		Context("when the artifact directory was not created this time", func() {
			BeforeEach(func() {
				artifactDirCreated = false
			})

			It("does not remove the artifact directory", func() {
				Expect(fakeSSHConnection.RunCallCount()).To(Equal(0))
			})
		})

		Context("when cleanup fails", func() {
			BeforeEach(func() {
				fakeSSHConnection.RunReturns(nil, nil, 5, nil)
			})

			It("returns an error", func() {
				Expect(err).To(MatchError("Unable to clean up backup artifact"))
			})
		})

		Context("when ssh connection fails", func() {
			BeforeEach(func() {
				fakeSSHConnection.RunReturns(nil, nil, 0, errors.New("fool!"))
			})

			It("returns the error", func() {
				Expect(err).To(MatchError(ContainSubstring("fool!")))
			})
		})
	})

	Describe("CleanupPrevious", func() {
		var err error

		JustBeforeEach(func() {
			inst = NewDeployedInstance("group", fakeSSHConnection, logger, []instance.Job{}, artifactDirCreated)
			err = inst.CleanupPrevious()
		})

		BeforeEach(func() {
			artifactDirCreated = true
		})

		It("does not fail", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("removes the artifact directory", func() {
			Expect(fakeSSHConnection.RunCallCount()).To(Equal(1))
			Expect(fakeSSHConnection.RunArgsForCall(0)).To(Equal(
				"sudo rm -rf /var/vcap/store/bbr-backup",
			))
		})

		Context("when the artifact directory was not created this time", func() {
			BeforeEach(func() {
				artifactDirCreated = false
			})

			It("does remove the artifact directory", func() {
				Expect(fakeSSHConnection.RunCallCount()).To(Equal(1))
				Expect(fakeSSHConnection.RunArgsForCall(0)).To(Equal(
					"sudo rm -rf /var/vcap/store/bbr-backup",
				))
			})
		})

		Context("when cleanup fails", func() {
			BeforeEach(func() {
				fakeSSHConnection.RunReturns(nil, nil, 5, nil)
			})

			It("returns an error", func() {
				Expect(err).To(MatchError("Unable to clean up backup artifact"))
			})
		})

		Context("when ssh connection fails", func() {
			BeforeEach(func() {
				fakeSSHConnection.RunReturns(nil, nil, 0, errors.New("fool!"))
			})

			It("returns the error", func() {
				Expect(err).To(MatchError(ContainSubstring("fool!")))
			})
		})
	})
})

func createTempFile(contents string) string {
	tempFile, err := ioutil.TempFile("", "")
	Expect(err).NotTo(HaveOccurred())
	tempFile.Write([]byte(contents))
	tempFile.Close()
	return tempFile.Name()
}
