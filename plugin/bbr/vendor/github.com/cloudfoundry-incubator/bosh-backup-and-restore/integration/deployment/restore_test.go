package deployment

import (
	"io/ioutil"
	"os"

	"github.com/pivotal-cf-experimental/cf-webmock/mockbosh"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/testcluster"

	"archive/tar"
	"bytes"

	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Restore", func() {
	var director *mockhttp.Server
	var restoreWorkspace string

	BeforeEach(func() {
		director = mockbosh.NewTLS()
		director.ExpectedBasicAuth("admin", "admin")
		var err error
		restoreWorkspace, err = ioutil.TempDir(".", "restore-workspace-")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(restoreWorkspace)).To(Succeed())
		director.VerifyMocks()
	})

	Context("when deployment is not present", func() {
		var session *gexec.Session
		deploymentName := "my-new-deployment"

		BeforeEach(func() {
			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances: []`))

			director.VerifyAndMock(
				mockbosh.Info().WithAuthTypeBasic(),
				mockbosh.VMsForDeployment(deploymentName).NotFound(),
			)
			session = binary.Run(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
				"deployment",
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--target", director.URL,
				"--deployment", "my-new-deployment",
				"restore",
				"--artifact-path", deploymentName)

		})

		It("fails and prints an error", func() {
			By("failing", func() {
				Expect(session.ExitCode()).To(Equal(1))
			})

			By("printing an error", func() {
				Expect(string(session.Err.Contents())).To(ContainSubstring("Director responded with non-successful status code"))
			})

			By("not printing the stack trace", func() {
				Expect(string(session.Err.Contents())).NotTo(ContainSubstring("main.go"))
			})

			By("writes the stack trace", func() {
				files, err := filepath.Glob(filepath.Join(restoreWorkspace, "bbr-*.err.log"))
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

	Context("when artifact is not present", func() {
		var session *gexec.Session

		BeforeEach(func() {
			director.VerifyAndMock(mockbosh.Info().WithAuthTypeBasic())
			session = binary.Run(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
				"deployment",
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--target", director.URL,
				"--deployment", "my-new-deployment",
				"restore",
				"--artifact-path", "i-am-not-here")

		})

		It("fails and prints an error", func() {
			By("failing", func() {
				Expect(session.ExitCode()).To(Equal(1))
			})

			By("printing an error", func() {
				Expect(string(session.Err.Contents())).To(ContainSubstring("i-am-not-here: no such file or directory"))
			})

			By("not printing the stack trace", func() {
				Expect(string(session.Err.Contents())).NotTo(ContainSubstring("main.go"))
			})

			By("writes the stack trace", func() {
				files, err := filepath.Glob(filepath.Join(restoreWorkspace, "bbr-*.err.log"))
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

	Context("when the backup is corrupted", func() {
		var session *gexec.Session
		var deploymentName string

		BeforeEach(func() {
			deploymentName = "my-new-deployment"
			director.VerifyAndMock(mockbosh.Info().WithAuthTypeBasic())

			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances:
- name: redis-dedicated-node
  index: 0
  artifacts:
  - name: redis
    checksums:
      redis-backup: this-is-not-a-checksum-this-is-only-a-tribute`))

			backupContents, err := ioutil.ReadFile("../../fixtures/backup.tar")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-dedicated-node-0-redis.tar", backupContents)
			session = binary.Run(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
				"deployment",
				"--debug",
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--target", director.URL,
				"--deployment", deploymentName,
				"restore",
				"--artifact-path", deploymentName)
		})

		It("fails and prints an error", func() {
			By("failing", func() {
				Expect(session.ExitCode()).To(Equal(1))
			})

			By("printing an error", func() {
				Expect(string(session.Err.Contents())).To(ContainSubstring("Backup is corrupted"))
			})
			By("not printing the stack trace", func() {
				Expect(string(session.Err.Contents())).NotTo(ContainSubstring("main.go"))
			})

			By("writes the stack trace", func() {
				files, err := filepath.Glob(filepath.Join(restoreWorkspace, "bbr-*.err.log"))
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

	Context("when deployment has a single instance", func() {
		var session *gexec.Session
		var instance1 *testcluster.Instance
		var deploymentName string

		BeforeEach(func() {
			instance1 = testcluster.NewInstance()
			deploymentName = "my-new-deployment"
			director.VerifyAndMock(AppendBuilders(
				InfoWithBasicAuth(),
				VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
					{
						IPs:     []string{"10.0.0.1"},
						JobName: "redis-dedicated-node",
						JobID:   "fake-uuid",
					}}),
				SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
				CleanupSSH(deploymentName, "redis-dedicated-node"))...)

			instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/post-restore-unlock", `#!/usr/bin/env sh
touch /tmp/post-restore-unlock-script-was-run
`)

			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances:
- name: redis-dedicated-node
  index: 0
  artifacts:
  - name: redis
    checksums:
      ./redis/redis-backup: 8d7fa73732d6dba6f6af01621552d3a6d814d2042c959465d0562a97c3f796b0`))

			backupContents, err := ioutil.ReadFile("../../fixtures/backup.tar")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-dedicated-node-0-redis.tar", backupContents)
		})

		JustBeforeEach(func() {
			session = binary.Run(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
				"deployment",
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--debug",
				"--target", director.URL,
				"--deployment", deploymentName,
				"restore",
				"--artifact-path", deploymentName)
		})

		AfterEach(func() {
			instance1.DieInBackground()
			Expect(os.RemoveAll(deploymentName)).To(Succeed())
		})

		Context("and the restore script works", func() {
			BeforeEach(func() {
				instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/restore", `#!/usr/bin/env sh
set -u
cp -r $BBR_ARTIFACT_DIRECTORY* /var/vcap/store/redis-server
touch /tmp/restore-script-was-run`)
			})

			It("runs the restore script successfully and cleans up", func() {
				By("succeeding", func() {
					Expect(session.ExitCode()).To(Equal(0))
				})

				By("cleaning up the archive file on the remote", func() {
					Expect(instance1.FileExists("/var/vcap/store/bbr-backup/redis-backup")).To(BeFalse())
				})

				By("running the restore script on the remote", func() {
					Expect(instance1.FileExists("/var/vcap/store/redis-server/redis-backup")).To(BeTrue())
					Expect(instance1.FileExists("/tmp/restore-script-was-run")).To(BeTrue())
				})

				By("running the post-backup-unlock script on the remote", func() {
					Expect(instance1.FileExists("/tmp/post-restore-unlock-script-was-run")).To(BeTrue())
				})
			})
		})

		Context("when restore fails", func() {
			BeforeEach(func() {
				instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/restore", `#!/usr/bin/env sh
	>&2 echo "dear lord"; exit 1`)
			})

			It("fails and returns the failure", func() {
				By("failing", func() {
					Expect(session.ExitCode()).To(Equal(1))
				})

				By("returning the failure", func() {
					Expect(session.Err.Contents()).To(ContainSubstring("dear lord"))
				})
				By("not printing the stack trace", func() {
					Expect(string(session.Err.Contents())).NotTo(ContainSubstring("main.go"))
				})

				By("writes the stack trace", func() {
					files, err := filepath.Glob(filepath.Join(restoreWorkspace, "bbr-*.err.log"))
					Expect(err).NotTo(HaveOccurred())
					logFilePath := files[0]
					_, err = os.Stat(logFilePath)
					Expect(os.IsNotExist(err)).To(BeFalse())
					stackTrace, err := ioutil.ReadFile(logFilePath)
					Expect(err).ToNot(HaveOccurred())
					Expect(gbytes.BufferWithBytes(stackTrace)).To(gbytes.Say("main.go"))
				})

				By("running the post-backup-unlock script on the remote", func() {
					Expect(instance1.FileExists("/tmp/post-restore-unlock-script-was-run")).To(BeTrue())
				})
			})
		})

		Context("when the backup artifact already exists", func() {
			BeforeEach(func() {
				instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/restore", `#!/usr/bin/env sh
set -u
cp -r $BBR_ARTIFACT_DIRECTORY* /var/vcap/store/redis-server
touch /tmp/restore-script-was-run`)
				instance1.CreateDir("/var/vcap/store/bbr-backup")
			})

			It("fails, returns an error and does not delete the artifact", func() {
				By("failing", func() {
					Expect(session.ExitCode()).To(Equal(1))
				})

				By("returning the correct error", func() {
					Expect(session.Err.Contents()).To(ContainSubstring(
						"Directory /var/vcap/store/bbr-backup already exists on instance redis-dedicated-node/fake-uuid",
					))
				})

				By("not printing the stack trace", func() {
					Expect(string(session.Err.Contents())).NotTo(ContainSubstring("main.go"))
				})

				By("writes the stack trace", func() {
					files, err := filepath.Glob(filepath.Join(restoreWorkspace, "bbr-*.err.log"))
					Expect(err).NotTo(HaveOccurred())
					logFilePath := files[0]
					_, err = os.Stat(logFilePath)
					Expect(os.IsNotExist(err)).To(BeFalse())
					stackTrace, err := ioutil.ReadFile(logFilePath)
					Expect(err).ToNot(HaveOccurred())
					Expect(gbytes.BufferWithBytes(stackTrace)).To(gbytes.Say("main.go"))
				})

				By("not deleting the artifact", func() {
					Expect(instance1.FileExists("/var/vcap/store/bbr-backup")).To(BeTrue())
				})
			})
		})
	})

	Context("when deployment has a multiple instances", func() {
		var session *gexec.Session
		var instance1 *testcluster.Instance
		var instance2 *testcluster.Instance
		var deploymentName string

		BeforeEach(func() {
			instance1 = testcluster.NewInstance()
			instance2 = testcluster.NewInstance()
			deploymentName = "my-new-deployment"
			director.VerifyAndMock(AppendBuilders(
				InfoWithBasicAuth(),
				VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
					{
						IPs:     []string{"10.0.0.1"},
						JobName: "redis-dedicated-node",
						JobID:   "fake-uuid",
					},
					{
						IPs:     []string{"10.0.0.10"},
						JobName: "redis-server",
						JobID:   "fake-uuid",
					}}),
				SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
				SetupSSH(deploymentName, "redis-server", "fake-uuid", 0, instance2),
				CleanupSSH(deploymentName, "redis-dedicated-node"),
				CleanupSSH(deploymentName, "redis-server"))...)

			instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/restore", `#!/usr/bin/env sh
set -u
cp -r $BBR_ARTIFACT_DIRECTORY* /var/vcap/store/redis-server
touch /tmp/restore-script-was-run`)
			instance2.CreateScript("/var/vcap/jobs/redis/bin/bbr/restore", `#!/usr/bin/env sh
set -u
cp -r $BBR_ARTIFACT_DIRECTORY* /var/vcap/store/redis-server
touch /tmp/restore-script-was-run`)

			instance2.CreateScript("/var/vcap/jobs/redis/bin/bbr/post-restore-unlock", `#!/usr/bin/env sh
touch /tmp/post-restore-unlock-script-was-run`)

			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances:
- name: redis-dedicated-node
  index: 0
  artifacts:
  - name: redis
    checksums:
      ./redis/redis-backup: 8d7fa73732d6dba6f6af01621552d3a6d814d2042c959465d0562a97c3f796b0
- name: redis-server
  index: 0
  artifacts:
  - name: redis
    checksums:
      ./redis/redis-backup: 8d7fa73732d6dba6f6af01621552d3a6d814d2042c959465d0562a97c3f796b0`))

			backupContents, err := ioutil.ReadFile("../../fixtures/backup.tar")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-dedicated-node-0-redis.tar", backupContents)
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-server-0-redis.tar", backupContents)
		})

		JustBeforeEach(func() {
			session = binary.Run(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
				"deployment",
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--debug",
				"--target", director.URL,
				"--deployment", deploymentName,
				"restore",
				"--artifact-path", deploymentName)
		})

		AfterEach(func() {
			instance1.DieInBackground()
			instance2.DieInBackground()
			Expect(os.RemoveAll(deploymentName)).To(Succeed())
		})

		It("runs the restore script and cleans up", func() {
			By("succeeding", func() {
				Expect(session.ExitCode()).To(Equal(0))
			})

			By("cleaning up the archive file on the remote", func() {
				Expect(instance1.FileExists("/var/vcap/store/bbr-backup/redis-backup")).To(BeFalse())
				Expect(instance2.FileExists("/var/vcap/store/bbr-backup/redis-backup")).To(BeFalse())
			})

			By("running the restore script on the remote", func() {
				Expect(instance1.FileExists("/var/vcap/store/redis-server/redis-backup")).To(BeTrue())
				Expect(instance1.FileExists("/tmp/restore-script-was-run")).To(BeTrue())
				Expect(instance2.FileExists("/var/vcap/store/redis-server/redis-backup")).To(BeTrue())
				Expect(instance2.FileExists("/tmp/restore-script-was-run")).To(BeTrue())
			})
			By("running the post restore unlock script on the remote", func() {
				Expect(instance2.FileExists("/tmp/post-restore-unlock-script-was-run")).To(BeTrue())
			})
		})

	})

	Context("when deployment has named artifacts, with a default artifact", func() {
		var session *gexec.Session
		var instance1 *testcluster.Instance
		var deploymentName string

		BeforeEach(func() {
			instance1 = testcluster.NewInstance()
			deploymentName = "my-new-deployment"
			director.VerifyAndMock(AppendBuilders(
				InfoWithBasicAuth(),
				VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
					{
						IPs:     []string{"10.0.0.1"},
						JobName: "redis-dedicated-node",
						JobID:   "fake-uuid",
					}}),
				SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
				CleanupSSH(deploymentName, "redis-dedicated-node"))...)
			instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/metadata", `#!/usr/bin/env sh
echo "---
restore_name: foo
"`)
			instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/restore", `#!/usr/bin/env sh
set -u
cp -r $BBR_ARTIFACT_DIRECTORY* /var/vcap/store/redis-server
touch /tmp/restore-script-was-run`)

			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances:
- name: redis-dedicated-node
  index: 0
custom_artifacts:
- name: foo
  checksums:
    ./redis/redis-backup: 8d7fa73732d6dba6f6af01621552d3a6d814d2042c959465d0562a97c3f796b0`))

			backupContents, err := ioutil.ReadFile("../../fixtures/backup.tar")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"foo.tar", backupContents)

			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-dedicated-node-0-redis.tar", createTarWithContents(map[string]string{}))
		})

		JustBeforeEach(func() {
			session = binary.Run(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
				"deployment",
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--debug",
				"--target", director.URL,
				"--deployment", deploymentName,
				"restore",
				"--artifact-path", deploymentName)
		})

		AfterEach(func() {
			Expect(os.RemoveAll(deploymentName)).To(Succeed())
			instance1.DieInBackground()
		})

		It("runs the restore script and cleans up", func() {
			By("succeeding", func() {
				Expect(session.ExitCode()).To(Equal(0))
			})

			By("cleaning up the archive file on the remote", func() {
				Expect(instance1.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
			})

			By("running the restore script on the remote", func() {
				Expect(instance1.FileExists("/var/vcap/store/redis-server" +
					"/redis-backup")).To(BeTrue())
				Expect(instance1.FileExists("/tmp/restore-script-was-run")).To(BeTrue())
			})
		})
	})

	Context("when deployment has named artifacts, without a default artifact", func() {
		var session *gexec.Session
		var instance1 *testcluster.Instance
		var instance2 *testcluster.Instance
		var deploymentName string

		BeforeEach(func() {
			instance1 = testcluster.NewInstance()
			instance2 = testcluster.NewInstance()
			deploymentName = "my-new-deployment"
			director.VerifyAndMock(AppendBuilders(
				InfoWithBasicAuth(),
				VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
					{
						IPs:     []string{"10.0.0.1"},
						JobName: "redis-restore-node",
						JobID:   "fake-uuid",
					},
					{
						IPs:     []string{"10.0.0.2"},
						JobName: "redis-backup-node",
						JobID:   "fake-uuid",
					}}),
				SetupSSH(deploymentName, "redis-restore-node", "fake-uuid", 0, instance1),
				SetupSSH(deploymentName, "redis-backup-node", "fake-uuid", 0, instance2),
				CleanupSSH(deploymentName, "redis-restore-node"),
				CleanupSSH(deploymentName, "redis-backup-node"))...)
			instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/metadata", `#!/usr/bin/env sh
echo "---
restore_name: foo
"`)
			instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/restore", `#!/usr/bin/env sh
set -u
cp -r $BBR_ARTIFACT_DIRECTORY/* /var/vcap/store/redis-server
touch /tmp/restore-script-was-run`)
			instance2.CreateScript("/var/vcap/jobs/redis/bin/bbr/backup", `#!/usr/bin/env sh
set -u
echo "dosent matter"`)

			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances:
- name: redis-backup-node
  index: 0
custom_artifacts:
- name: foo
  checksums:
    ./redis/redis-backup: 8d7fa73732d6dba6f6af01621552d3a6d814d2042c959465d0562a97c3f796b0`))

			backupContents, err := ioutil.ReadFile("../../fixtures/backup.tar")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"foo.tar", backupContents)

			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-backup-node-0.tar", createTarWithContents(map[string]string{}))
		})

		JustBeforeEach(func() {
			session = binary.Run(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
				"deployment",
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--debug",
				"--target", director.URL,
				"--deployment", deploymentName,
				"restore",
				"--artifact-path", deploymentName)
		})

		AfterEach(func() {
			instance1.DieInBackground()
			instance2.DieInBackground()
			Expect(os.RemoveAll(deploymentName)).To(Succeed())
		})

		It("runs the restore script and cleans up", func() {
			By("succeeding", func() {
				Expect(session.ExitCode()).To(Equal(0))
			})

			By("cleaning up the archive file on the remote", func() {
				Expect(instance1.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
			})

			By("running the restore script on the remote", func() {
				Expect(instance1.FileExists("/var/vcap/store/redis-server/redis-backup")).To(BeTrue())
				Expect(instance1.FileExists("/tmp/restore-script-was-run")).To(BeTrue())
			})
		})
	})

	Context("when the backup with named artifacts on disk is corrupted", func() {
		var session *gexec.Session
		var deploymentName string

		BeforeEach(func() {
			deploymentName = "my-new-deployment"

			director.VerifyAndMock(mockbosh.Info().WithAuthTypeBasic())
			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances:
- name: redis-backup-node
  index: 0
custom_artifacts:
- name: foo
  checksums:
    ./redis/redis-backup: this-is-damn-wrong`))

			backupContents, err := ioutil.ReadFile("../../fixtures/backup.tar")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"foo.tar", backupContents)

			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-backup-node-0.tar", createTarWithContents(map[string]string{}))
		})

		JustBeforeEach(func() {
			session = binary.Run(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
				"deployment",
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--debug",
				"--target", director.URL,
				"--deployment", deploymentName,
				"restore",
				"--artifact-path", deploymentName)
		})

		It("fails", func() {
			Expect(session.ExitCode()).To(Equal(1))
		})

		It("writes the stack trace", func() {
			files, err := filepath.Glob(filepath.Join(restoreWorkspace, "bbr-*.err.log"))
			Expect(err).NotTo(HaveOccurred())
			logFilePath := files[0]
			_, err = os.Stat(logFilePath)
			Expect(os.IsNotExist(err)).To(BeFalse())
			stackTrace, err := ioutil.ReadFile(logFilePath)
			Expect(err).ToNot(HaveOccurred())
			Expect(gbytes.BufferWithBytes(stackTrace)).To(gbytes.Say("main.go"))
		})
	})

	Context("the cleanup fails", func() {
		var session *gexec.Session
		var instance1 *testcluster.Instance
		var deploymentName string

		BeforeEach(func() {
			instance1 = testcluster.NewInstance()
			deploymentName = "my-new-deployment"
			director.VerifyAndMock(AppendBuilders(
				InfoWithBasicAuth(),
				VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
					{
						IPs:     []string{"10.0.0.1"},
						JobName: "redis-dedicated-node",
					}}),
				SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
				CleanupSSHFails(deploymentName, "redis-dedicated-node", "cleanup err"))...)

			instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/restore", `#!/usr/bin/env sh
set-u
cp -r $BBR_ARTIFACT_DIRECTORY* /var/vcap/store/redis-server/
touch /tmp/restore-script-was-run`)
			instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/post-restore-unlock", `#!/usr/bin/env sh
touch /tmp/post-restore-unlock-script-was-run`)

			Expect(os.Mkdir(restoreWorkspace+"/"+deploymentName, 0777)).To(Succeed())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"metadata", []byte(`---
instances:
- name: redis-dedicated-node
  index: 0
  artifacts:
  - name: redis
    checksums:
      ./redis/redis-backup: 8d7fa73732d6dba6f6af01621552d3a6d814d2042c959465d0562a97c3f796b0`))

			backupContents, err := ioutil.ReadFile("../../fixtures/backup.tar")
			Expect(err).NotTo(HaveOccurred())
			createFileWithContents(restoreWorkspace+"/"+deploymentName+"/"+"redis-dedicated-node-0-redis.tar", backupContents)
		})

		JustBeforeEach(func() {
			session = binary.Run(
				restoreWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
				"deployment",
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--target", director.URL,
				"--deployment", deploymentName,
				"restore",
				"--artifact-path", deploymentName)
		})

		AfterEach(func() {
			instance1.DieInBackground()
			Expect(os.RemoveAll(deploymentName)).To(Succeed())
		})

		It("runs the restore script, fails and cleans up", func() {
			By("failing", func() {
				Expect(session.ExitCode()).To(Equal(16))
			})

			By("cleaning up the archive file on the remote", func() {
				Expect(instance1.FileExists("/var/vcap/store/bbr-backup/redis-backup")).To(BeFalse())
			})

			By("running the restore script on the remote", func() {
				Expect(instance1.FileExists("/var/vcap/store/redis-server/redis-backup")).To(BeTrue())
				Expect(instance1.FileExists("/tmp/restore-script-was-run")).To(BeTrue())
			})

			By("running the post-restore-unlock scripts", func() {
				Expect(instance1.FileExists("/tmp/post-restore-unlock-script-was-run")).To(BeTrue())
			})

			By("returning the failure", func() {
				Expect(session.Err.Contents()).To(ContainSubstring("cleanup err"))
			})

			By("not printing the stack trace", func() {
				Expect(string(session.Err.Contents())).NotTo(ContainSubstring("main.go"))
			})
		})
	})
})

func createFileWithContents(filePath string, contents []byte) {
	file, err := os.Create(filePath)
	Expect(err).NotTo(HaveOccurred())
	_, err = file.Write([]byte(contents))
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
