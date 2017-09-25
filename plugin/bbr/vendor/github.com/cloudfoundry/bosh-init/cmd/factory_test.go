package cmd_test

import (
	. "github.com/cloudfoundry/bosh-init/cmd"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	biui "github.com/cloudfoundry/bosh-init/ui"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
	"github.com/pivotal-golang/clock"

	fakebiui "github.com/cloudfoundry/bosh-init/ui/fakes"
)

var _ = Describe("cmd.Factory", func() {
	var (
		factory       Factory
		fs            boshsys.FileSystem
		ui            biui.UI
		logger        boshlog.Logger
		uuidGenerator *fakeuuid.FakeGenerator
	)

	BeforeEach(func() {
		logger = boshlog.NewLogger(boshlog.LevelNone)
		fs = fakesys.NewFakeFileSystem()
		ui = &fakebiui.FakeUI{}
		uuidGenerator = &fakeuuid.FakeGenerator{}

		factory = NewFactory(
			fs,
			ui,
			clock.NewClock(),
			logger,
			uuidGenerator,
			"/fake-path",
		)
	})

	It("creates a new factory", func() {
		Expect(factory).ToNot(BeNil())
	})

	Context("known command name", func() {
		Describe("deploy command", func() {
			It("returns deploy command", func() {
				cmd, err := factory.CreateCommand("deploy")
				Expect(err).ToNot(HaveOccurred())
				Expect(cmd.Name()).To(Equal("deploy"))
			})
		})

		Describe("delete command", func() {
			It("returns delete command", func() {
				cmd, err := factory.CreateCommand("delete")
				Expect(err).ToNot(HaveOccurred())
				Expect(cmd.Name()).To(Equal("delete"))
			})
		})
	})

	Context("unknown command name", func() {
		It("returns error", func() {
			_, err := factory.CreateCommand("bogus-cmd-name")
			Expect(err).To(HaveOccurred())
		})
	})
})
