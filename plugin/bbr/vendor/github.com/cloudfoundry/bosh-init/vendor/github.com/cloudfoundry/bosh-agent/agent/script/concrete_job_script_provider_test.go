package script_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	fakeaction "github.com/cloudfoundry/bosh-agent/agent/action/fakes"
	boshscript "github.com/cloudfoundry/bosh-agent/agent/script"
	boshdrain "github.com/cloudfoundry/bosh-agent/agent/script/drain"
	fakedrain "github.com/cloudfoundry/bosh-agent/agent/script/drain/fakes"
	fakescript "github.com/cloudfoundry/bosh-agent/agent/script/fakes"
	boshdir "github.com/cloudfoundry/bosh-agent/settings/directories"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("ConcreteJobScriptProvider", func() {
	var (
		logger         boshlog.Logger
		scriptProvider boshscript.ConcreteJobScriptProvider
	)

	BeforeEach(func() {
		runner := fakesys.NewFakeCmdRunner()
		fs := fakesys.NewFakeFileSystem()
		dirProvider := boshdir.NewProvider("/the/base/dir")
		logger = boshlog.NewLogger(boshlog.LevelNone)
		scriptProvider = boshscript.NewConcreteJobScriptProvider(
			runner,
			fs,
			dirProvider,
			&fakeaction.FakeClock{},
			logger,
		)
	})

	Describe("NewScript", func() {
		It("returns script with relative job paths to the base directory", func() {
			script := scriptProvider.NewScript("myjob", "the-best-hook-ever")
			Expect(script.Tag()).To(Equal("myjob"))

			expPath := "/the/base/dir/jobs/myjob/bin/the-best-hook-ever" + boshscript.ScriptExt
			Expect(script.Path()).To(Equal(expPath))
		})
	})

	Describe("NewDrainScript", func() {
		It("returns drain script", func() {
			params := &fakedrain.FakeScriptParams{}
			script := scriptProvider.NewDrainScript("foo", params)
			Expect(script.Tag()).To(Equal("foo"))

			expPath := "/the/base/dir/jobs/foo/bin/drain" + boshscript.ScriptExt
			Expect(script.Path()).To(Equal(expPath))
			Expect(script.(boshdrain.ConcreteScript).Params()).To(Equal(params))
		})
	})

	Describe("NewParallelScript", func() {
		It("returns parallel script", func() {
			scripts := []boshscript.Script{&fakescript.FakeScript{}}
			script := scriptProvider.NewParallelScript("foo", scripts)
			Expect(script).To(Equal(boshscript.NewParallelScript("foo", scripts, logger)))
		})
	})
})
