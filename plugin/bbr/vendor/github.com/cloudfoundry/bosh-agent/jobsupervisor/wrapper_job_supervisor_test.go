package jobsupervisor_test

import (
	. "github.com/cloudfoundry/bosh-agent/jobsupervisor"

	"encoding/json"
	"errors"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	"github.com/cloudfoundry/bosh-agent/agent/alert"
	"github.com/cloudfoundry/bosh-agent/jobsupervisor/fakes"
	fakemonit "github.com/cloudfoundry/bosh-agent/jobsupervisor/monit/fakes"
	boshdir "github.com/cloudfoundry/bosh-agent/settings/directories"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("WrapperJobSupervisor", func() {

	var (
		fs                    *fakesys.FakeFileSystem
		runner                *fakesys.FakeCmdRunner
		client                *fakemonit.FakeMonitClient
		logger                boshlog.Logger
		dirProvider           boshdir.Provider
		jobFailuresServerPort int
		fakeSupervisor        *fakes.FakeJobSupervisor
		wrapper               JobSupervisor
		timeService           *fakeclock.FakeClock
	)

	var jobFailureServerPort = 5000

	getJobFailureServerPort := func() int {
		jobFailureServerPort++
		return jobFailureServerPort
	}

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		fs.MkdirAll("/var/vcap/instance", 666)
		runner = fakesys.NewFakeCmdRunner()
		client = fakemonit.NewFakeMonitClient()
		logger = boshlog.NewLogger(boshlog.LevelNone)
		dirProvider = boshdir.NewProvider("/var/vcap")
		jobFailuresServerPort = getJobFailureServerPort()
		timeService = fakeclock.NewFakeClock(time.Now())

		fakeSupervisor = fakes.NewFakeJobSupervisor()

		wrapper = NewWrapperJobSupervisor(
			fakeSupervisor,
			fs,
			dirProvider,
			logger,
		)
	})

	It("Reload should delegate to the underlying job supervisor", func() {
		error := errors.New("BOOM")
		fakeSupervisor.ReloadErr = error
		err := wrapper.Reload()
		Expect(fakeSupervisor.Reloaded).To(BeTrue())
		Expect(err).To(Equal(error))
	})

	Describe("Start", func() {
		It("should delegate to the underlying job supervisor", func() {
			error := errors.New("BOOM")
			fakeSupervisor.StartErr = error
			err := wrapper.Start()
			Expect(fakeSupervisor.Started).To(BeTrue())
			Expect(err).To(Equal(error))
		})

		It("write the health json asynchronously", func() {
			fakeSupervisor.StatusStatus = "running"
			wrapper.Start()

			healthFile := filepath.Join(dirProvider.InstanceDir(), "health.json")
			healthRaw, err := fs.ReadFile(healthFile)
			Expect(err).ToNot(HaveOccurred())
			health := &Health{}
			json.Unmarshal(healthRaw, health)
			Expect(health.State).To(Equal("running"))
		})
	})

	It("Stop should delegate to the underlying job supervisor", func() {
		error := errors.New("BOOM")
		fakeSupervisor.StopErr = error
		err := wrapper.Stop()
		Expect(fakeSupervisor.Stopped).To(BeTrue())
		Expect(err).To(Equal(error))
	})

	It("StopAndWait should delegate to the underlying job supervisor", func() {
		error := errors.New("BOOM")
		fakeSupervisor.StopErr = error
		err := wrapper.StopAndWait()
		Expect(fakeSupervisor.StoppedAndWaited).To(BeTrue())
		Expect(err).To(Equal(error))
	})

	Describe("Unmointor", func() {
		It("Unmonitor should delegate to the underlying job supervisor", func() {
			error := errors.New("BOOM")
			fakeSupervisor.UnmonitorErr = error
			err := wrapper.Unmonitor()
			Expect(fakeSupervisor.Unmonitored).To(BeTrue())
			Expect(err).To(Equal(error))
		})

		It("write the health json asynchronously", func() {
			fakeSupervisor.StatusStatus = "stopped"
			_ = wrapper.Unmonitor()

			healthFile := filepath.Join(dirProvider.InstanceDir(), "health.json")
			healthRaw, err := fs.ReadFile(healthFile)
			Expect(err).ToNot(HaveOccurred())
			health := &Health{}
			json.Unmarshal(healthRaw, health)
			Expect(health.State).To(Equal("stopped"))
		})

	})

	It("Status should delegate to the underlying job supervisor", func() {
		fakeSupervisor.StatusStatus = "my-status"
		status := wrapper.Status()
		Expect(status).To(Equal(fakeSupervisor.StatusStatus))
	})

	It("Processes should delegate to the underlying job supervisor", func() {
		fakeSupervisor.ProcessesStatus = []Process{
			{},
		}
		fakeSupervisor.ProcessesError = errors.New("BOOM")
		processes, err := wrapper.Processes()
		Expect(processes).To(Equal(fakeSupervisor.ProcessesStatus))
		Expect(err).To(Equal(fakeSupervisor.ProcessesError))
	})

	It("AddJob should delegate to the underlying job supervisor", func() {
		error := errors.New("BOOM")
		fakeSupervisor.StartErr = error
		_ = wrapper.AddJob("name", 0, "path")
		Expect(fakeSupervisor.AddJobArgs).To(Equal([]fakes.AddJobArgs{
			{
				Name:       "name",
				Index:      0,
				ConfigPath: "path",
			},
		}))
	})

	It("RemoveAllJobs should delegate to the underlying job supervisor", func() {
		fakeSupervisor.RemovedAllJobsErr = errors.New("BOOM")
		err := wrapper.RemoveAllJobs()
		Expect(fakeSupervisor.RemovedAllJobs).To(BeTrue())
		Expect(err).To(Equal(fakeSupervisor.RemovedAllJobsErr))
	})

	It("MonitorJobFailures should delegate to the underlying job supervisor", func() {
		var testAlert *alert.MonitAlert

		fakeSupervisor.JobFailureAlert = &alert.MonitAlert{ID: "test-alert"}
		_ = wrapper.MonitorJobFailures(func(a alert.MonitAlert) error {
			testAlert = &a

			return nil
		})
		Expect(testAlert).To(Equal(fakeSupervisor.JobFailureAlert))
	})
})
