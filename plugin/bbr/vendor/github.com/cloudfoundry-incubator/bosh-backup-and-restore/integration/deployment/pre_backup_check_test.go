package deployment

import (
	"io/ioutil"
	"os"

	"github.com/pivotal-cf-experimental/cf-webmock/mockbosh"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/testcluster"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pre-backup checks", func() {
	var director *mockhttp.Server
	var backupWorkspace string
	var session *gexec.Session
	var deploymentName string

	BeforeEach(func() {
		deploymentName = "my-little-deployment"
		director = mockbosh.NewTLS()
		director.ExpectedBasicAuth("admin", "admin")
		var err error
		backupWorkspace, err = ioutil.TempDir(".", "backup-workspace-")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(backupWorkspace)).To(Succeed())
		director.VerifyMocks()
	})

	JustBeforeEach(func() {
		session = binary.Run(
			backupWorkspace,
			[]string{"BOSH_CLIENT_SECRET=admin"},
			"deployment",
			"--ca-cert", sslCertPath,
			"--username", "admin",
			"--target", director.URL,
			"--deployment", deploymentName,
			"pre-backup-check",
		)
	})

	Context("When there is a deployment which has one instance", func() {
		var instance1 *testcluster.Instance

		singleInstanceResponse := func(instanceGroupName string) []mockbosh.VMsOutput {
			return []mockbosh.VMsOutput{
				{
					IPs:     []string{"10.0.0.1"},
					JobName: instanceGroupName,
				},
			}
		}

		BeforeEach(func() {
			instance1 = testcluster.NewInstance()
		})

		AfterEach(func() {
			instance1.DieInBackground()
		})

		Context("and there is a backup script", func() {
			BeforeEach(func() {
				By("creating a dummy backup script")

				MockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					CleanupSSH(deploymentName, "redis-dedicated-node"),
				)

				instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/backup", `#!/usr/bin/env sh
set -u
printf "backupcontent1" > $BBR_ARTIFACT_DIRECTORY/backupdump1
printf "backupcontent2" > $BBR_ARTIFACT_DIRECTORY/backupdump2
`)

			})

			It("exits zero", func() {
				Expect(session.ExitCode()).To(BeZero())
			})

			It("outputs a log message saying the deployment can be backed up", func() {
				Expect(string(session.Out.Contents())).To(ContainSubstring("Deployment '" + deploymentName + "' can be backed up."))
			})

			Context("but the backup artifact directory already exists", func() {
				BeforeEach(func() {
					instance1.CreateDir("/var/vcap/store/bbr-backup")
				})

				It("returns exit code 1", func() {
					Expect(session.ExitCode()).To(Equal(1))
				})

				It("prints an error", func() {
					Expect(string(session.Out.Contents())).To(ContainSubstring("Deployment '" + deploymentName + "' cannot be backed up."))
					Expect(string(session.Err.Contents())).To(ContainSubstring("Directory /var/vcap/store/bbr-backup already exists on instance redis-dedicated-node/fake-uuid"))
					Expect(string(session.Err.Contents())).NotTo(ContainSubstring("main.go"))
				})

				It("writes the stack trace", func() {
					files, err := filepath.Glob(filepath.Join(backupWorkspace, "bbr-*.err.log"))
					Expect(err).NotTo(HaveOccurred())
					logFilePath := files[0]
					_, err = os.Stat(logFilePath)
					Expect(os.IsNotExist(err)).To(BeFalse())
					stackTrace, err := ioutil.ReadFile(logFilePath)
					Expect(err).ToNot(HaveOccurred())
					Expect(gbytes.BufferWithBytes(stackTrace)).To(gbytes.Say("main.go"))
				})
			})

		})

		Context("if there are no backup scripts", func() {
			BeforeEach(func() {
				MockDirectorWith(director,
					mockbosh.Info().WithAuthTypeBasic(),
					VmsForDeployment(deploymentName, singleInstanceResponse("redis-dedicated-node")),
					SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
					CleanupSSH(deploymentName, "redis-dedicated-node"),
				)

				instance1.CreateFiles(
					"/var/vcap/jobs/redis/bin/not-a-backup-script",
				)
			})

			It("returns exit code 1", func() {
				Expect(session.ExitCode()).To(Equal(1))
			})

			It("prints an error", func() {
				Expect(string(session.Out.Contents())).To(ContainSubstring("Deployment '" + deploymentName + "' cannot be backed up."))
				Expect(string(session.Err.Contents())).To(ContainSubstring("Deployment '" + deploymentName + "' has no backup scripts"))
				Expect(string(session.Err.Contents())).NotTo(ContainSubstring("main.go"))
			})

			It("writes the stack trace", func() {
				files, err := filepath.Glob(filepath.Join(backupWorkspace, "bbr-*.err.log"))
				Expect(err).NotTo(HaveOccurred())
				logFilePath := files[0]
				_, err = os.Stat(logFilePath)
				Expect(os.IsNotExist(err)).To(BeFalse())
				stackTrace, err := ioutil.ReadFile(logFilePath)
				Expect(err).ToNot(HaveOccurred())
				Expect(gbytes.BufferWithBytes(stackTrace)).To(gbytes.Say("main.go"))
			})

		})
	})

	Context("When deployment does not exist", func() {
		BeforeEach(func() {
			deploymentName = "my-non-existent-deployment"
			director.VerifyAndMock(
				mockbosh.Info().WithAuthTypeBasic(),
				mockbosh.VMsForDeployment(deploymentName).NotFound(),
			)
		})

		It("returns exit code 1", func() {
			Expect(session.ExitCode()).To(Equal(1))
		})

		It("prints an error", func() {
			Expect(string(session.Out.Contents())).To(ContainSubstring("Deployment '" + deploymentName + "' cannot be backed up."))
			Expect(string(session.Err.Contents())).To(ContainSubstring("Director responded with non-successful status code"))
		})
	})

	Context("When the director is unreachable", func() {
		BeforeEach(func() {
			deploymentName = "my-director-is-broken"
			director.VerifyAndMock(
				AppendBuilders(
					InfoWithBasicAuth(),
					VmsForDeploymentFails(deploymentName),
				)...,
			)
		})

		It("returns exit code 1", func() {
			Expect(session.ExitCode()).To(Equal(1))
		})

		It("prints an error", func() {
			Expect(string(session.Out.Contents())).To(ContainSubstring("Deployment '" + deploymentName + "' cannot be backed up."))
			Expect(string(session.Err.Contents())).To(ContainSubstring("Director responded with non-successful status code"))
		})
	})
})
