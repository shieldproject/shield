package platform_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshretry "github.com/cloudfoundry/bosh-utils/retrystrategy"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"

	. "github.com/cloudfoundry/bosh-agent/platform"
)

var _ = Describe("MonitRetryable", func() {
	var (
		cmdRunner      *fakesys.FakeCmdRunner
		monitRetryable boshretry.Retryable
	)

	BeforeEach(func() {
		cmdRunner = fakesys.NewFakeCmdRunner()
		monitRetryable = NewMonitRetryable(cmdRunner)
	})

	Describe("Attempt", func() {
		Context("when starting monit fails", func() {
			BeforeEach(func() {
				cmdRunner.AddCmdResult("sv start monit", fakesys.FakeCmdResult{
					ExitStatus: 255,
					Error:      errors.New("fake-start-monit-error"),
				})
			})

			It("is retryable and returns err", func() {
				isRetryable, err := monitRetryable.Attempt()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-start-monit-error"))
				Expect(isRetryable).To(BeTrue())
				Expect(len(cmdRunner.RunCommands)).To(Equal(1))
				Expect(cmdRunner.RunCommands[0]).To(Equal([]string{"sv", "start", "monit"}))
			})
		})

		Context("when starting succeeds", func() {
			BeforeEach(func() {
				cmdRunner.AddCmdResult("sv start monit", fakesys.FakeCmdResult{
					ExitStatus: 0,
				})
			})

			It("returns no error", func() {
				_, err := monitRetryable.Attempt()
				Expect(err).ToNot(HaveOccurred())
				Expect(len(cmdRunner.RunCommands)).To(Equal(1))
				Expect(cmdRunner.RunCommands[0]).To(Equal([]string{"sv", "start", "monit"}))
			})
		})
	})
})
