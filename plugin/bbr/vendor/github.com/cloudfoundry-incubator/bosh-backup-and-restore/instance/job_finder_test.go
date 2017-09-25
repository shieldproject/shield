package instance_test

import (
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance/fakes"

	"fmt"

	"bytes"
	"io"
	"log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("JobFinderFromScripts", func() {
	var logStream *bytes.Buffer
	var jobFinder *JobFinderFromScripts
	var sshConnection *fakes.FakeSSHConnection
	var jobs Jobs
	var jobsError error
	var logger Logger

	Describe("FindJobs", func() {
		BeforeEach(func() {
			logStream = bytes.NewBufferString("")

			combinedLog := log.New(io.MultiWriter(GinkgoWriter, logStream), "[instance-test] ", log.Lshortfile)

			sshConnection = new(fakes.FakeSSHConnection)
			logger = boshlog.New(boshlog.LevelDebug, combinedLog, combinedLog)
			jobFinder = NewJobFinder(logger)
		})
		JustBeforeEach(func() {
			jobs, jobsError = jobFinder.FindJobs("identifier", sshConnection)
		})

		Context("has no job metadata scripts", func() {
			Context("Finds jobs based on scripts", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("/var/vcap/jobs/consul_agent/bin/bbr/backup\n"+
						"/var/vcap/jobs/consul_agent/bin/bbr/restore"), nil, 0, nil)
				})

				It("succeeds", func() {
					Expect(jobsError).NotTo(HaveOccurred())
				})

				It("finds the scripts", func() {
					Expect(sshConnection.RunArgsForCall(0)).To(Equal("find /var/vcap/jobs/*/bin/bbr/* -type f"))
				})

				It("returns a list of jobs", func() {
					Expect(jobs).To(Equal(NewJobs(sshConnection, "identifier", logger, BackupAndRestoreScripts{
						"/var/vcap/jobs/consul_agent/bin/bbr/backup",
						"/var/vcap/jobs/consul_agent/bin/bbr/restore",
					}, map[string]Metadata{})))
				})

				It("logs the scripts found", func() {
					Expect(logStream.String()).To(ContainSubstring("identifier/consul_agent/backup"))
					Expect(logStream.String()).To(ContainSubstring("identifier/consul_agent/restore"))
				})
			})

			Context("Finds invalid jobs scripts", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("/var/vcap/jobs/consul_agent/bin/foobar"), nil, 0, nil)
				})

				It("succeeds", func() {
					Expect(jobsError).NotTo(HaveOccurred())
				})

				It("finds the scripts", func() {
					Expect(sshConnection.RunArgsForCall(0)).To(Equal("find /var/vcap/jobs/*/bin/bbr/* -type f"))
				})

				It("returns a list of jobs", func() {
					Expect(jobs).To(Equal(NewJobs(sshConnection, "identifier", logger, BackupAndRestoreScripts{}, map[string]Metadata{})))
				})
			})

			Context("there are no job scripts returned from the connection", func() {
				BeforeEach(func() {
					sshConnection.RunReturns(
						nil, []byte("find: `/var/vcap/jobs/*/bin/bbr/*': No such file or directory"), 1, nil,
					)
				})

				It("does not fail", func() {
					Expect(jobsError).NotTo(HaveOccurred())
				})

				It("returns an empty list", func() {
					Expect(jobs).To(Equal(NewJobs(sshConnection, "identifier", logger, BackupAndRestoreScripts{}, map[string]Metadata{})))
				})
			})

			Context("find fails on a unknown error", func() {
				BeforeEach(func() {
					sshConnection.RunReturns(
						nil, []byte("find: `unknown error"), 1, nil,
					)
				})

				It("runs the command once", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))

				})
				It("does fail", func() {
					Expect(jobsError).To(HaveOccurred())
				})
			})

			Context("find fails ssh connection error", func() {
				expectedError := fmt.Errorf("no!")

				BeforeEach(func() {
					sshConnection.RunReturns(
						nil, nil, 0, expectedError,
					)
				})

				It("runs the command once", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))

				})
				It("does fail", func() {
					Expect(jobsError).To(MatchError(expectedError))
				})
			})
		})

		Context("ssh connection returns a metadata script", func() {
			Context("metadata is valid", func() {
				BeforeEach(func() {
					sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
						if cmd == "/var/vcap/jobs/consul_agent/bin/bbr/metadata" {
							return []byte(`---
backup_name: consul_backup`), nil, 0, nil
						}
						return []byte("/var/vcap/jobs/consul_agent/bin/bbr/metadata"), nil, 0, nil
					}

				})
				It("succeeds", func() {
					Expect(jobsError).NotTo(HaveOccurred())
				})

				It("uses the ssh connection to get the metadata", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(2))
					Expect(sshConnection.RunArgsForCall(1)).To(Equal("/var/vcap/jobs/consul_agent/bin/bbr/metadata"))

				})

				It("returns a list of jobs with metadata", func() {
					Expect(jobs).To(Equal(NewJobs(sshConnection, "identifier", logger, BackupAndRestoreScripts{
						"/var/vcap/jobs/consul_agent/bin/bbr/metadata",
					}, map[string]Metadata{
						"consul_agent": {BackupName: "consul_backup"},
					})))
				})
			})

			Context("reading metadata failed with ssh error", func() {
				expectedError := fmt.Errorf("foo!")

				BeforeEach(func() {
					sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
						if cmd == "/var/vcap/jobs/consul_agent/bin/bbr/metadata" {
							return []byte(`---
backup_name: consul_backup`), nil, 0, expectedError
						}
						return []byte("/var/vcap/jobs/consul_agent/bin/bbr/metadata"), nil, 0, nil
					}
				})

				It("fails", func() {
					Expect(jobsError.Error()).To(ContainSubstring(expectedError.Error()))
				})

				It("uses the ssh connection to get the metadata", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(2))
					Expect(sshConnection.RunArgsForCall(1)).To(Equal("/var/vcap/jobs/consul_agent/bin/bbr/metadata"))
				})
			})

			Context("reading metadata exited with non 0 code", func() {
				BeforeEach(func() {
					sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
						if cmd == "/var/vcap/jobs/consul_agent/bin/bbr/metadata" {
							return []byte(`---
backup_name: consul_backup`), nil, 0, nil
						}
						return []byte("/var/vcap/jobs/consul_agent/bin/bbr/metadata"), nil, 1, nil
					}
				})

				It("fails", func() {
					Expect(jobsError).To(HaveOccurred())
				})

				It("uses the ssh connection to find the metadata", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(1))
				})
			})

			Context("reading metadata returned invalid yaml", func() {
				BeforeEach(func() {
					sshConnection.RunStub = func(cmd string) ([]byte, []byte, int, error) {
						if cmd == "/var/vcap/jobs/consul_agent/bin/bbr/metadata" {
							return []byte(`this is very disappointing`), nil, 0, nil
						}
						return []byte("/var/vcap/jobs/consul_agent/bin/bbr/metadata"), nil, 0, nil
					}
				})

				It("succeeds", func() {
					Expect(jobsError).To(HaveOccurred())
				})

				It("uses the ssh connection to get the metadata", func() {
					Expect(sshConnection.RunCallCount()).To(Equal(2))
					Expect(sshConnection.RunArgsForCall(1)).To(Equal("/var/vcap/jobs/consul_agent/bin/bbr/metadata"))
				})
			})
		})
	})
})
