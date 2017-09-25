package jobsupervisor_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshalert "github.com/cloudfoundry/bosh-agent/agent/alert"
	boshhandler "github.com/cloudfoundry/bosh-agent/handler"
	. "github.com/cloudfoundry/bosh-agent/jobsupervisor"
	fakembus "github.com/cloudfoundry/bosh-agent/mbus/fakes"
)

var _ = Describe("dummyNatsJobSupervisor", func() {
	var (
		dummyNats JobSupervisor
		handler   *fakembus.FakeHandler
	)

	BeforeEach(func() {
		handler = &fakembus.FakeHandler{}
		dummyNats = NewDummyNatsJobSupervisor(handler)
	})

	Describe("MonitorJobFailures", func() {
		It("monitors job status", func() {
			dummyNats.MonitorJobFailures(func(boshalert.MonitAlert) error { return nil })
			Expect(handler.RegisteredAdditionalFunc).ToNot(BeNil())
		})
	})

	Describe("Status", func() {
		BeforeEach(func() {
			dummyNats.MonitorJobFailures(func(boshalert.MonitAlert) error { return nil })
		})

		It("returns the received status", func() {
			statusMessage := boshhandler.NewRequest("", "set_dummy_status", []byte(`{"status":"failing"}`))
			handler.RegisteredAdditionalFunc(statusMessage)
			Expect(dummyNats.Status()).To(Equal("failing"))
		})

		It("returns running as a default value", func() {
			Expect(dummyNats.Status()).To(Equal("running"))
		})

		It("does not change the status given other messages", func() {
			statusMessage := boshhandler.NewRequest("", "some_other_message", []byte(`{"status":"failing"}`))
			handler.RegisteredAdditionalFunc(statusMessage)
			Expect(dummyNats.Status()).To(Equal("running"))
		})
	})

	Describe("Start", func() {
		BeforeEach(func() {
			dummyNats.MonitorJobFailures(func(boshalert.MonitAlert) error { return nil })
		})

		Context("When set_task_fail flag is sent in messagae", func() {
			It("raises an error", func() {
				statusMessage := boshhandler.NewRequest("", "set_task_fail", []byte(`{"status":"fail_task"}`))
				handler.RegisteredAdditionalFunc(statusMessage)
				err := dummyNats.Start()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-task-fail-error"))
			})
		})

		Context("when set_task_fail flag is not sent in message", func() {
			It("raises an error", func() {
				statusMessage := boshhandler.NewRequest("", "set_task_fail", []byte(`{"status":"something_else"}`))
				handler.RegisteredAdditionalFunc(statusMessage)
				err := dummyNats.Start()
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
