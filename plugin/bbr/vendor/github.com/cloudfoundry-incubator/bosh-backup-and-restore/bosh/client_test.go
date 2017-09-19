package bosh_test

import (
	"log"

	"bytes"
	"io"

	"strings"

	"errors"

	"github.com/cloudfoundry/bosh-cli/director"
	boshfakes "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	instancefakes "github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance/fakes"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh/fakes"
	gossh "golang.org/x/crypto/ssh"
)

var _ = Describe("Director", func() {
	var optsGenerator *fakes.FakeSSHOptsGenerator
	var sshConnectionFactory *fakes.FakeSSHConnectionFactory
	var boshDirector *boshfakes.FakeDirector
	var boshLogger boshlog.Logger
	var boshDeployment *boshfakes.FakeDeployment
	var sshConnection *fakes.FakeSSHConnection
	var fakeJobFinder *instancefakes.FakeJobFinder

	var deploymentName = "kubernetes"

	var stdoutLogStream *bytes.Buffer
	var stderrLogStream *bytes.Buffer

	var hostsPublicKey = "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEAklOUpkDHrfHY17SbrmTIpNLTGK9Tjom/BWDSUGPl+nafzlHDTYW7hdI4yZ5ew18JH4JW9jbhUFrviQzM7xlELEVf4h9lFX5QVkbPppSwg0cda3Pbv7kOdJ/MTyBlWXFCR+HAo3FXRitBqxiX1nKhXpHAZsMciLq8V6RjsNAQwdsdMFvSlVK/7XAt3FaoJoAsncM1Q9x5+3V0Ww68/eIFmb1zuUFljQJKprrX88XypNDvjYNby6vw/Pb0rwert/EnmZ+AW4OZPnTPI89ZPmVMLuayrD2cE86Z/il8b+gw3r3+1nKatmIkjn2so1d01QraTlMqVSsbxNrRFi9wrf+M7Q== schacon@mylaptop.local"
	var hostKeyAlgorithm []string

	var b bosh.BoshClient
	JustBeforeEach(func() {
		b = bosh.NewClient(boshDirector, optsGenerator.Spy, sshConnectionFactory.Spy, boshLogger, fakeJobFinder)
	})

	BeforeEach(func() {
		optsGenerator = new(fakes.FakeSSHOptsGenerator)
		sshConnectionFactory = new(fakes.FakeSSHConnectionFactory)
		boshDirector = new(boshfakes.FakeDirector)
		boshDeployment = new(boshfakes.FakeDeployment)
		sshConnection = new(fakes.FakeSSHConnection)
		fakeJobFinder = new(instancefakes.FakeJobFinder)

		stdoutLogStream = bytes.NewBufferString("")
		stderrLogStream = bytes.NewBufferString("")

		hostPublicKey, _, _, _, err := gossh.ParseAuthorizedKey([]byte(hostsPublicKey))
		Expect(err).NotTo(HaveOccurred())
		hostKeyAlgorithm = []string{hostPublicKey.Type()}

		combinedOutLog := log.New(io.MultiWriter(GinkgoWriter, stdoutLogStream), "[bosh-package] ", log.Lshortfile)
		combinedErrLog := log.New(io.MultiWriter(GinkgoWriter, stderrLogStream), "[bosh-package] ", log.Lshortfile)
		boshLogger = boshlog.New(boshlog.LevelDebug, combinedOutLog, combinedErrLog)
	})
	Describe("FindInstances", func() {
		var (
			stubbedSshOpts  director.SSHOpts = director.SSHOpts{Username: "user"}
			actualInstances []orchestrator.Instance
			actualError     error
			expectedJobs    instance.Jobs
		)

		JustBeforeEach(func() {
			actualInstances, actualError = b.FindInstances(deploymentName)
		})

		Context("finds instances for the deployment", func() {
			BeforeEach(func() {
				boshDirector.FindDeploymentReturns(boshDeployment, nil)
				boshDeployment.VMInfosReturns([]director.VMInfo{{
					JobName: "job1",
					ID:      "jobID",
				}}, nil)
				optsGenerator.Returns(stubbedSshOpts, "private_key", nil)
				boshDeployment.SetUpSSHReturns(director.SSHResult{Hosts: []director.Host{
					{
						Username:      "username",
						Host:          "hostname",
						IndexOrID:     "jobID",
						HostPublicKey: hostsPublicKey,
					},
				}}, nil)
				sshConnectionFactory.Returns(sshConnection, nil)
				expectedJobs = instance.NewJobs(sshConnection, "", boshLogger, instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/consul_agent/bin/bbr/backup",
					"/var/vcap/jobs/consul_agent/bin/bbr/restore",
				}, map[string]instance.Metadata{})
				fakeJobFinder.FindJobsReturns(expectedJobs, nil)
			})

			It("collects the instances", func() {
				Expect(actualInstances).To(Equal([]orchestrator.Instance{bosh.NewBoshDeployedInstance(
					"job1",
					"0",
					"jobID",
					sshConnection,
					boshDeployment,
					false,
					boshLogger,
					expectedJobs,
				)}))
			})

			It("does not fail", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("fetches the deployment by name", func() {
				Expect(boshDirector.FindDeploymentCallCount()).To(Equal(1))
				Expect(boshDirector.FindDeploymentArgsForCall(0)).To(Equal(deploymentName))
			})

			It("fetchs vms for the deployment", func() {
				Expect(boshDeployment.VMInfosCallCount()).To(Equal(1))
			})

			It("generates a new ssh private key", func() {
				Expect(optsGenerator.CallCount()).To(Equal(1))
			})

			It("finds the jobs with the job finder", func() {
				Expect(fakeJobFinder.FindJobsCallCount()).To(Equal(1))
			})

			It("sets up ssh for each group found", func() {
				Expect(boshDeployment.SetUpSSHCallCount()).To(Equal(1))

				slug, opts := boshDeployment.SetUpSSHArgsForCall(0)
				Expect(slug).To(Equal(director.NewAllOrInstanceGroupOrInstanceSlug("job1", "")))
				Expect(opts).To(Equal(stubbedSshOpts))
			})

			It("creates a ssh connection to each host", func() {
				Expect(sshConnectionFactory.CallCount()).To(Equal(1))
				host, username, privateKey, _, hostPublicKeyAlgorithm, logger := sshConnectionFactory.ArgsForCall(0)
				Expect(host).To(Equal("hostname"))
				Expect(username).To(Equal("username"))
				Expect(privateKey).To(Equal("private_key"))
				Expect(hostPublicKeyAlgorithm).To(Equal(hostKeyAlgorithm))
				Expect(logger).To(Equal(boshLogger))
			})

		})

		Context("finds instances for the deployment, with port specified in host", func() {
			BeforeEach(func() {
				boshDirector.FindDeploymentReturns(boshDeployment, nil)
				boshDeployment.VMInfosReturns([]director.VMInfo{{
					JobName: "job1",
				}}, nil)
				optsGenerator.Returns(stubbedSshOpts, "private_key", nil)
				boshDeployment.SetUpSSHReturns(director.SSHResult{Hosts: []director.Host{
					{
						Username:      "username",
						Host:          "hostname:3457",
						IndexOrID:     "index",
						HostPublicKey: hostsPublicKey,
					},
				}}, nil)
				sshConnectionFactory.Returns(sshConnection, nil)
			})

			It("uses the specified port", func() {
				Expect(sshConnectionFactory.CallCount()).To(Equal(1))
				host, username, privateKey, _, hostPublicKeyAlgorithm, logger := sshConnectionFactory.ArgsForCall(0)
				Expect(host).To(Equal("hostname:3457"))
				Expect(username).To(Equal("username"))
				Expect(privateKey).To(Equal("private_key"))
				Expect(hostPublicKeyAlgorithm).To(Equal(hostKeyAlgorithm))
				Expect(logger).To(Equal(boshLogger))
			})
		})

		Context("finds instances for the deployment, having multiple instances in an instance group", func() {
			var instance0Jobs, instance1Jobs instance.Jobs
			BeforeEach(func() {
				boshDirector.FindDeploymentReturns(boshDeployment, nil)
				boshDeployment.VMInfosReturns([]director.VMInfo{
					{
						JobName: "job1",
						ID:      "id1",
					},
					{
						JobName: "job1",
						ID:      "id2",
					},
				}, nil)
				optsGenerator.Returns(stubbedSshOpts, "private_key", nil)
				boshDeployment.SetUpSSHReturns(director.SSHResult{Hosts: []director.Host{
					{
						Username:      "username",
						Host:          "hostname1",
						IndexOrID:     "id1",
						HostPublicKey: hostsPublicKey,
					},
					{
						Username:      "username",
						Host:          "hostname2",
						IndexOrID:     "id2",
						HostPublicKey: hostsPublicKey,
					},
				}}, nil)
				sshConnectionFactory.Returns(sshConnection, nil)

				instance0Jobs = instance.NewJobs(sshConnection, "", boshLogger,
					instance.BackupAndRestoreScripts{"/var/vcap/jobs/consul_agent/bin/bbr/backup"},
					map[string]instance.Metadata{},
				)

				instance1Jobs = instance.NewJobs(sshConnection, "", boshLogger,
					instance.BackupAndRestoreScripts{"/var/vcap/jobs/consul_agent/bin/bbr/backup"},
					map[string]instance.Metadata{},
				)
				fakeJobFinder.FindJobsStub = func(hostIdentifier string, connection instance.SSHConnection) (instance.Jobs, error) {
					if strings.HasPrefix(hostIdentifier, "hostname1") {
						return instance0Jobs, nil
					} else {
						return instance1Jobs, nil
					}
				}
			})
			It("collects the instances", func() {
				Expect(actualInstances).To(Equal([]orchestrator.Instance{
					bosh.NewBoshDeployedInstance(
						"job1",
						"0",
						"id1",
						sshConnection,
						boshDeployment,
						false,
						boshLogger,
						instance0Jobs,
					),
					bosh.NewBoshDeployedInstance(
						"job1",
						"1",
						"id2",
						sshConnection,
						boshDeployment,
						false,
						boshLogger,
						instance1Jobs,
					),
				}))
			})
			It("does not fail", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("fetches the deployment by name", func() {
				Expect(boshDirector.FindDeploymentCallCount()).To(Equal(1))
				Expect(boshDirector.FindDeploymentArgsForCall(0)).To(Equal(deploymentName))
			})

			It("fetchs vms for the deployment", func() {
				Expect(boshDeployment.VMInfosCallCount()).To(Equal(1))
			})

			It("generates a new ssh private key", func() {
				Expect(optsGenerator.CallCount()).To(Equal(1))
			})

			It("sets up ssh for each group found", func() {
				Expect(boshDeployment.SetUpSSHCallCount()).To(Equal(1))

				slug, opts := boshDeployment.SetUpSSHArgsForCall(0)
				Expect(slug).To(Equal(director.NewAllOrInstanceGroupOrInstanceSlug("job1", "")))
				Expect(opts).To(Equal(stubbedSshOpts))
			})

			It("creates a ssh connection to each host", func() {
				Expect(sshConnectionFactory.CallCount()).To(Equal(2))

				host, username, privateKey, _, hostPublicKeyAlgorithm, logger := sshConnectionFactory.ArgsForCall(0)
				Expect(host).To(Equal("hostname1"))
				Expect(username).To(Equal("username"))
				Expect(privateKey).To(Equal("private_key"))
				Expect(hostPublicKeyAlgorithm).To(Equal(hostKeyAlgorithm))
				Expect(logger).To(Equal(boshLogger))

				host, username, privateKey, _, hostPublicKeyAlgorithm, logger = sshConnectionFactory.ArgsForCall(1)
				Expect(host).To(Equal("hostname2"))
				Expect(username).To(Equal("username"))
				Expect(privateKey).To(Equal("private_key"))
				Expect(hostPublicKeyAlgorithm).To(Equal(hostKeyAlgorithm))
				Expect(logger).To(Equal(boshLogger))
			})
			It("finds the jobs with the job finder", func() {
				Expect(fakeJobFinder.FindJobsCallCount()).To(Equal(2))
			})
		})

		Context("finds instances for the deployment, having multiple instances in multiple instance groups", func() {
			var instanceJobs instance.Jobs
			BeforeEach(func() {
				boshDirector.FindDeploymentReturns(boshDeployment, nil)
				boshDeployment.VMInfosReturns([]director.VMInfo{
					{
						JobName: "job1",
						ID:      "id1",
					},
					{
						JobName: "job1",
						ID:      "id2",
					},
					{
						JobName: "job2",
						ID:      "id3",
					},
				}, nil)
				optsGenerator.Returns(stubbedSshOpts, "private_key", nil)
				boshDeployment.SetUpSSHStub = func(slug director.AllOrInstanceGroupOrInstanceSlug, sshOpts director.SSHOpts) (director.SSHResult, error) {
					if slug.Name() == "job1" {
						return director.SSHResult{Hosts: []director.Host{
							{
								Username:      "username",
								Host:          "hostname1",
								IndexOrID:     "id1",
								HostPublicKey: hostsPublicKey,
							},
							{
								Username:      "username",
								Host:          "hostname2",
								IndexOrID:     "id2",
								HostPublicKey: hostsPublicKey,
							},
						}}, nil
					} else {
						return director.SSHResult{Hosts: []director.Host{
							{
								Username:      "username",
								Host:          "hostname3",
								IndexOrID:     "id3",
								HostPublicKey: hostsPublicKey,
							},
						}}, nil
					}
				}
				sshConnectionFactory.Returns(sshConnection, nil)
				instanceJobs = instance.NewJobs(sshConnection, "", boshLogger,
					instance.BackupAndRestoreScripts{"/var/vcap/jobs/consul_agent/bin/bbr/backup"},
					map[string]instance.Metadata{},
				)
				fakeJobFinder.FindJobsReturns(instanceJobs, nil)
			})
			It("collects the instances", func() {
				Expect(actualInstances).To(Equal([]orchestrator.Instance{
					bosh.NewBoshDeployedInstance(
						"job1",
						"0",
						"id1",
						sshConnection,
						boshDeployment,
						false,
						boshLogger,
						instanceJobs,
					),
					bosh.NewBoshDeployedInstance(
						"job1",
						"1",
						"id2",
						sshConnection,
						boshDeployment,
						false,
						boshLogger,
						instanceJobs,
					),
					bosh.NewBoshDeployedInstance(
						"job2",
						"0",
						"id3",
						sshConnection,
						boshDeployment,
						false,
						boshLogger,
						instanceJobs,
					),
				}))
			})
			It("does not fail", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("fetches the deployment by name", func() {
				Expect(boshDirector.FindDeploymentCallCount()).To(Equal(1))
				Expect(boshDirector.FindDeploymentArgsForCall(0)).To(Equal(deploymentName))
			})

			It("fetchs vms for the deployment", func() {
				Expect(boshDeployment.VMInfosCallCount()).To(Equal(1))
			})

			It("generates a new ssh private key", func() {
				Expect(optsGenerator.CallCount()).To(Equal(1))
			})

			It("sets up ssh for each group found", func() {
				Expect(boshDeployment.SetUpSSHCallCount()).To(Equal(2))

				slug, opts := boshDeployment.SetUpSSHArgsForCall(0)
				Expect(slug).To(Equal(director.NewAllOrInstanceGroupOrInstanceSlug("job1", "")))
				Expect(opts).To(Equal(stubbedSshOpts))

				slug, opts = boshDeployment.SetUpSSHArgsForCall(1)
				Expect(slug).To(Equal(director.NewAllOrInstanceGroupOrInstanceSlug("job2", "")))
				Expect(opts).To(Equal(stubbedSshOpts))
			})

			It("creates a ssh connection to each host", func() {
				Expect(sshConnectionFactory.CallCount()).To(Equal(3))

				host, username, privateKey, _, hostPublicKeyAlgorithm, logger := sshConnectionFactory.ArgsForCall(0)
				Expect(host).To(Equal("hostname1"))
				Expect(username).To(Equal("username"))
				Expect(privateKey).To(Equal("private_key"))
				Expect(hostPublicKeyAlgorithm).To(Equal(hostKeyAlgorithm))
				Expect(logger).To(Equal(boshLogger))

				host, username, privateKey, _, hostPublicKeyAlgorithm, logger = sshConnectionFactory.ArgsForCall(1)
				Expect(host).To(Equal("hostname2"))
				Expect(username).To(Equal("username"))
				Expect(privateKey).To(Equal("private_key"))
				Expect(hostPublicKeyAlgorithm).To(Equal(hostKeyAlgorithm))
				Expect(logger).To(Equal(boshLogger))

				host, username, privateKey, _, hostPublicKeyAlgorithm, logger = sshConnectionFactory.ArgsForCall(2)
				Expect(host).To(Equal("hostname3"))
				Expect(username).To(Equal("username"))
				Expect(privateKey).To(Equal("private_key"))
				Expect(hostPublicKeyAlgorithm).To(Equal(hostKeyAlgorithm))
				Expect(logger).To(Equal(boshLogger))
			})

			It("finds the jobs with the job finder", func() {
				Expect(fakeJobFinder.FindJobsCallCount()).To(Equal(3))
			})

		})
		Context("failures", func() {
			var expectedError = "er ma gerd"

			Context("fails to find the deployment", func() {
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(nil, errors.New(expectedError))
				})

				It("does fail", func() {
					Expect(actualError).To(MatchError(ContainSubstring(expectedError)))
				})

				It("tries to fetch deployment", func() {
					Expect(boshDirector.FindDeploymentCallCount()).To(Equal(1))
					Expect(boshDirector.FindDeploymentArgsForCall(0)).To(Equal(deploymentName))
				})
			})

			Context("fails to find vms for a deployment", func() {
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)
					boshDeployment.VMInfosReturns(nil, errors.New(expectedError))
				})

				It("does fails", func() {
					Expect(actualError).To(MatchError(ContainSubstring(expectedError)))
				})
				It("tries to fetch vm infos", func() {
					Expect(boshDeployment.VMInfosCallCount()).To(Equal(1))
				})

				It("fetches deployment", func() {
					Expect(boshDirector.FindDeploymentCallCount()).To(Equal(1))
					Expect(boshDirector.FindDeploymentArgsForCall(0)).To(Equal(deploymentName))
				})
			})

			Context("fails to generate ssh opts", func() {
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)

					optsGenerator.Returns(director.SSHOpts{}, "", errors.New(expectedError))
				})
				It("does fails", func() {
					Expect(actualError).To(MatchError(ContainSubstring(expectedError)))
				})

				It("tries to generate ssh keys", func() {
					Expect(optsGenerator.CallCount()).To(Equal(1))
				})
			})

			Context("fails if a invalid job name is received", func() {
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)
					boshDeployment.VMInfosReturns([]director.VMInfo{{
						JobName: "this/is/invalid",
					}}, nil)
				})
				It("does fails", func() {
					Expect(actualError).To(MatchError(ContainSubstring("invalid instance group name")))
				})

				It("tries to fetch deployment", func() {
					Expect(boshDirector.FindDeploymentCallCount()).To(Equal(1))
					Expect(boshDirector.FindDeploymentArgsForCall(0)).To(Equal(deploymentName))
				})

				It("fetchs vms for the deployment", func() {
					Expect(boshDeployment.VMInfosCallCount()).To(Equal(1))
				})
			})

			Context("fails while setting up ssh, on the vm", func() {
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)
					boshDeployment.VMInfosReturns([]director.VMInfo{{
						JobName: "job1",
					}}, nil)
					optsGenerator.Returns(stubbedSshOpts, "private_key", nil)
					boshDeployment.SetUpSSHReturns(director.SSHResult{}, errors.New(expectedError))
				})

				It("does fails", func() {
					Expect(actualError).To(MatchError(ContainSubstring(expectedError)))
				})

				It("tries to fetch vm infos", func() {
					Expect(boshDeployment.VMInfosCallCount()).To(Equal(1))
				})

				It("fetches deployment", func() {
					Expect(boshDirector.FindDeploymentCallCount()).To(Equal(1))
					Expect(boshDirector.FindDeploymentArgsForCall(0)).To(Equal(deploymentName))
				})
				It("generates ssh opts", func() {
					Expect(optsGenerator.CallCount()).To(Equal(1))
				})
			})

			Context("fails creating a ssh connection, to the vm", func() {
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)
					boshDeployment.VMInfosReturns([]director.VMInfo{{
						JobName: "job1",
					}}, nil)
					optsGenerator.Returns(stubbedSshOpts, "private_key", nil)
					boshDeployment.SetUpSSHReturns(director.SSHResult{Hosts: []director.Host{
						{
							Username:      "username",
							Host:          "hostname",
							IndexOrID:     "index",
							HostPublicKey: hostsPublicKey,
						},
					}}, nil)
					sshConnectionFactory.Returns(nil, errors.New(expectedError))
				})

				It("does fail", func() {
					Expect(actualError).To(MatchError(ContainSubstring(expectedError)))
				})

				It("tries to connect to the vm", func() {
					Expect(sshConnectionFactory.CallCount()).To(Equal(1))
				})

				It("fetchs vm infos", func() {
					Expect(boshDeployment.VMInfosCallCount()).To(Equal(1))
				})

				It("fetches deployment", func() {
					Expect(boshDirector.FindDeploymentCallCount()).To(Equal(1))
					Expect(boshDirector.FindDeploymentArgsForCall(0)).To(Equal(deploymentName))
				})
				It("generates ssh opts", func() {
					Expect(optsGenerator.CallCount()).To(Equal(1))
				})

				It("cleanup the ssh user from the instance", func() {
					Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
				})
			})

			Context("succeeds creating ssh connections to some vms, fails others", func() {
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)
					boshDeployment.VMInfosReturns([]director.VMInfo{
						{
							JobName: "job1",
						},
						{
							JobName: "job2",
						}}, nil)
					optsGenerator.Returns(stubbedSshOpts, "private_key", nil)

					boshDeployment.SetUpSSHStub = func(slug director.AllOrInstanceGroupOrInstanceSlug, opts director.SSHOpts) (director.SSHResult, error) {
						if slug.Name() == "job1" {
							return director.SSHResult{Hosts: []director.Host{
								{
									Username:      "username",
									Host:          "hostname",
									IndexOrID:     "index",
									HostPublicKey: hostsPublicKey,
								},
							}}, nil
						} else {
							return director.SSHResult{}, errors.New(expectedError)
						}
					}
					sshConnectionFactory.Returns(sshConnection, nil)

				})

				It("fails", func() {
					Expect(actualError).To(MatchError(ContainSubstring(expectedError)))
				})

				It("cleans up the successful SSH connection", func() {
					Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
				})
			})

			Context("succeeds creating ssh connections but fails to create instance group slug", func() {
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)
					boshDeployment.VMInfosReturns([]director.VMInfo{
						{
							JobName: "job1",
						},
						{
							JobName: "job2/a/a/a",
						}}, nil)
					optsGenerator.Returns(stubbedSshOpts, "private_key", nil)

					boshDeployment.SetUpSSHReturns(director.SSHResult{Hosts: []director.Host{
						{
							Username:      "username",
							Host:          "hostname",
							IndexOrID:     "index",
							HostPublicKey: hostsPublicKey,
						},
					}}, nil)

					sshConnectionFactory.Returns(sshConnection, nil)

				})

				It("fails", func() {
					Expect(actualError).To(MatchError(ContainSubstring("invalid instance group name")))
				})

				It("cleans up the successful SSH connection", func() {
					Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(1))
				})
			})

			Context("succeeds creating ssh connections but ssh connection factory fails for a later connection", func() {
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)
					boshDeployment.VMInfosReturns([]director.VMInfo{
						{
							JobName: "job1",
						},
						{
							JobName: "job2",
						}}, nil)
					optsGenerator.Returns(stubbedSshOpts, "private_key", nil)

					boshDeployment.SetUpSSHStub = func(slug director.AllOrInstanceGroupOrInstanceSlug, opts director.SSHOpts) (director.SSHResult, error) {
						return director.SSHResult{Hosts: []director.Host{
							{
								Username:      "username",
								Host:          "hostname_" + slug.Name(),
								IndexOrID:     "index",
								HostPublicKey: hostsPublicKey,
							},
						}}, nil
					}

					sshConnectionFactory.Stub = func(host, user, privateKey string, publicKeyCallback gossh.HostKeyCallback, publicKeyAlgorithm []string, logger ssh.Logger) (ssh.SSHConnection, error) {
						if host == "hostname_job1" {
							return sshConnection, nil
						}
						return nil, errors.New(expectedError)
					}
				})

				It("fails", func() {
					Expect(actualError).To(MatchError(ContainSubstring(expectedError)))
				})

				It("cleans up the successful SSH connection", func() {
					Expect(boshDeployment.CleanUpSSHCallCount()).To(Equal(2))
				})
			})
		})
	})

	Describe("GetManifest", func() {
		var actualManifest string
		var acutalError error
		JustBeforeEach(func() {
			actualManifest, acutalError = b.GetManifest(deploymentName)
		})

		Context("gets the manifest", func() {
			BeforeEach(func() {
				boshDirector.FindDeploymentReturns(boshDeployment, nil)
				boshDeployment.ManifestReturns("a good ol manifest", nil)
			})
			It("from the deployment", func() {
				Expect(actualManifest).To(Equal("a good ol manifest"))
			})
		})
		Context("fails", func() {
			Context("to find deployment", func() {
				var findDeploymentError = "no deployment here"
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(nil, errors.New(findDeploymentError))
				})
				It("returns an error", func() {
					Expect(acutalError).To(MatchError(ContainSubstring(findDeploymentError)))
				})
			})
			Context("to download manifest", func() {
				var downloadManifestError = errors.New("I refuse to download this manifest")
				BeforeEach(func() {
					boshDirector.FindDeploymentReturns(boshDeployment, nil)
					boshDeployment.ManifestReturns("", downloadManifestError)
				})
				It("returns an error", func() {
					Expect(acutalError).To(MatchError(downloadManifestError))
				})
			})
		})
	})
})
