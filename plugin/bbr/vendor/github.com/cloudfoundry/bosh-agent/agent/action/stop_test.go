package action_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agent/action"
	fakejobsuper "github.com/cloudfoundry/bosh-agent/jobsupervisor/fakes"
)

var _ = Describe("Stop", func() {
	var (
		jobSupervisor *fakejobsuper.FakeJobSupervisor
		action        StopAction
	)

	BeforeEach(func() {
		jobSupervisor = fakejobsuper.NewFakeJobSupervisor()
		action = NewStop(jobSupervisor)
	})

	AssertActionIsAsynchronous(action)
	AssertActionIsNotPersistent(action)
	AssertActionIsLoggable(action)

	AssertActionIsNotResumable(action)
	AssertActionIsNotCancelable(action)

	It("returns stopped", func() {
		stopped, err := action.Run(ProtocolVersion(2))
		Expect(err).ToNot(HaveOccurred())
		Expect(stopped).To(Equal("stopped"))
	})

	It("stops job supervisor services", func() {
		_, err := action.Run(ProtocolVersion(2))
		Expect(err).ToNot(HaveOccurred())
		Expect(jobSupervisor.Stopped).To(BeTrue())
	})

	It("stops when protocol version is 2", func() {
		_, err := action.Run(ProtocolVersion(2))
		Expect(err).ToNot(HaveOccurred())
		Expect(jobSupervisor.StoppedAndWaited).ToNot(BeTrue())
	})

	It("stops and waits when protocol version is greater than 2", func() {
		_, err := action.Run(ProtocolVersion(3))
		Expect(err).ToNot(HaveOccurred())
		Expect(jobSupervisor.StoppedAndWaited).To(BeTrue())
	})
})
