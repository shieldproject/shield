package action_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agent/action"
	"github.com/cloudfoundry/bosh-agent/agent/applier/applyspec"
	fakeapplyspec "github.com/cloudfoundry/bosh-agent/agent/applier/applyspec/fakes"
	boshscript "github.com/cloudfoundry/bosh-agent/agent/script"
	fakescript "github.com/cloudfoundry/bosh-agent/agent/script/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var _ = Describe("RunScript", func() {
	var (
		fakeJobScriptProvider *fakescript.FakeJobScriptProvider
		specService           *fakeapplyspec.FakeV1Service
		action                RunScriptAction
	)

	BeforeEach(func() {
		fakeJobScriptProvider = &fakescript.FakeJobScriptProvider{}
		specService = fakeapplyspec.NewFakeV1Service()
		logger := boshlog.NewLogger(boshlog.LevelNone)
		action = NewRunScript(fakeJobScriptProvider, specService, logger)
	})

	It("is asynchronous", func() {
		Expect(action.IsAsynchronous()).To(BeTrue())
	})

	It("is not persistent", func() {
		Expect(action.IsPersistent()).To(BeFalse())
	})

	Describe("Run", func() {
		act := func() (map[string]string, error) { return action.Run("run-me", map[string]interface{}{}) }

		Context("when current spec can be retrieved", func() {
			var parallelScript *fakescript.FakeCancellableScript

			BeforeEach(func() {
				parallelScript = &fakescript.FakeCancellableScript{}
				fakeJobScriptProvider.NewParallelScriptReturns(parallelScript)
			})

			createFakeJob := func(jobName string) {
				spec := applyspec.JobTemplateSpec{Name: jobName}
				specService.Spec.JobSpec.JobTemplateSpecs = append(specService.Spec.JobSpec.JobTemplateSpecs, spec)
			}

			It("runs specified job scripts in parallel", func() {
				createFakeJob("fake-job-1")
				script1 := &fakescript.FakeScript{}
				script1.TagReturns("fake-job-1")

				createFakeJob("fake-job-2")
				script2 := &fakescript.FakeScript{}
				script2.TagReturns("fake-job-2")

				fakeJobScriptProvider.NewScriptStub = func(jobName, scriptName string) boshscript.Script {
					Expect(scriptName).To(Equal("run-me"))

					if jobName == "fake-job-1" {
						return script1
					} else if jobName == "fake-job-2" {
						return script2
					} else {
						panic("Non-matching script created")
					}
				}

				parallelScript.RunReturns(nil)

				results, err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(Equal(map[string]string{}))

				Expect(parallelScript.RunCallCount()).To(Equal(1))

				scriptName, scripts := fakeJobScriptProvider.NewParallelScriptArgsForCall(0)
				Expect(scriptName).To(Equal("run-me"))
				Expect(scripts).To(Equal([]boshscript.Script{script1, script2}))
			})

			It("returns an error when parallel script fails", func() {
				parallelScript.RunReturns(errors.New("fake-error"))

				results, err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-error"))
				Expect(results).To(Equal(map[string]string{}))
			})
		})

		Context("when current spec cannot be retrieved", func() {
			It("without current spec", func() {
				specService.GetErr = errors.New("fake-spec-get-error")

				results, err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-spec-get-error"))
				Expect(results).To(Equal(map[string]string{}))
			})
		})
	})
})
