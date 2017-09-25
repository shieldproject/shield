package action_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"errors"
	. "github.com/cloudfoundry/bosh-agent/agent/action"
	fakeas "github.com/cloudfoundry/bosh-agent/agent/applier/applyspec/fakes"
	fakeappl "github.com/cloudfoundry/bosh-agent/agent/applier/fakes"
	fakejobsuper "github.com/cloudfoundry/bosh-agent/jobsupervisor/fakes"
)

func init() {
	Describe("Start", func() {
		var (
			jobSupervisor *fakejobsuper.FakeJobSupervisor
			applier       *fakeappl.FakeApplier
			specService   *fakeas.FakeV1Service
			action        StartAction
		)

		BeforeEach(func() {
			jobSupervisor = fakejobsuper.NewFakeJobSupervisor()
			applier = fakeappl.NewFakeApplier()
			specService = fakeas.NewFakeV1Service()
			action = NewStart(jobSupervisor, applier, specService)
		})

		It("is synchronous", func() {
			Expect(action.IsAsynchronous()).To(BeFalse())
		})

		It("is not persistent", func() {
			Expect(action.IsPersistent()).To(BeFalse())
		})

		It("returns started", func() {
			started, err := action.Run()
			Expect(err).ToNot(HaveOccurred())
			Expect(started).To(Equal("started"))
		})

		It("starts monitor services", func() {
			_, err := action.Run()
			Expect(err).ToNot(HaveOccurred())
			Expect(jobSupervisor.Started).To(BeTrue())
		})

		It("configures jobs", func() {
			_, err := action.Run()
			Expect(err).ToNot(HaveOccurred())
			Expect(applier.Configured).To(BeTrue())
		})

		It("apply errs if a job fails configuring", func() {
			applier.ConfiguredError = errors.New("fake error")
			_, err := action.Run()

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Configuring jobs"))
		})
	})
}
