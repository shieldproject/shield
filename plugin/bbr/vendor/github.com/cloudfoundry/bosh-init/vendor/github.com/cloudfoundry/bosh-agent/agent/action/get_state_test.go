package action_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agent/action"
	boshas "github.com/cloudfoundry/bosh-agent/agent/applier/applyspec"
	fakeas "github.com/cloudfoundry/bosh-agent/agent/applier/applyspec/fakes"
	boshjobsuper "github.com/cloudfoundry/bosh-agent/jobsupervisor"
	fakejobsuper "github.com/cloudfoundry/bosh-agent/jobsupervisor/fakes"
	boshntp "github.com/cloudfoundry/bosh-agent/platform/ntp"
	fakentp "github.com/cloudfoundry/bosh-agent/platform/ntp/fakes"
	boshvitals "github.com/cloudfoundry/bosh-agent/platform/vitals"
	fakevitals "github.com/cloudfoundry/bosh-agent/platform/vitals/fakes"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	fakesettings "github.com/cloudfoundry/bosh-agent/settings/fakes"
	boshassert "github.com/cloudfoundry/bosh-utils/assert"
)

var _ = Describe("GetState", func() {
	var (
		settingsService *fakesettings.FakeSettingsService
		specService     *fakeas.FakeV1Service
		jobSupervisor   *fakejobsuper.FakeJobSupervisor
		vitalsService   *fakevitals.FakeService
		action          GetStateAction
	)

	BeforeEach(func() {
		settingsService = &fakesettings.FakeSettingsService{}
		jobSupervisor = fakejobsuper.NewFakeJobSupervisor()
		specService = fakeas.NewFakeV1Service()
		vitalsService = fakevitals.NewFakeService()
		ntpService := &fakentp.FakeService{
			GetOffsetNTPOffset: boshntp.Info{
				Offset:    "0.34958",
				Timestamp: "12 Oct 17:37:58",
			},
		}
		action = NewGetState(settingsService, specService, jobSupervisor, vitalsService, ntpService)
	})

	It("get state should be synchronous", func() {
		Expect(action.IsAsynchronous()).To(BeFalse())
	})

	It("is not persistent", func() {
		Expect(action.IsPersistent()).To(BeFalse())
	})

	Describe("Run", func() {
		Context("when current spec can be retrieved", func() {
			Context("when vitals can be retrieved", func() {
				It("returns state", func() {
					settingsService.Settings.AgentID = "my-agent-id"
					settingsService.Settings.VM.Name = "vm-abc-def"

					jobSupervisor.StatusStatus = "running"

					specService.Spec = boshas.V1ApplySpec{
						Deployment: "fake-deployment",
					}

					expectedSpec := GetStateV1ApplySpec{
						V1ApplySpec: boshas.V1ApplySpec{
							NetworkSpecs:      map[string]boshas.NetworkSpec{},
							ResourcePoolSpecs: map[string]interface{}{},
							PackageSpecs:      map[string]boshas.PackageSpec{},
						},
						AgentID:      "my-agent-id",
						JobState:     "running",
						BoshProtocol: "1",
						VM:           boshsettings.VM{Name: "vm-abc-def"},
						Ntp: boshntp.Info{
							Offset:    "0.34958",
							Timestamp: "12 Oct 17:37:58",
						},
					}
					expectedSpec.Deployment = "fake-deployment"

					state, err := action.Run()
					Expect(err).ToNot(HaveOccurred())

					Expect(state.AgentID).To(Equal(expectedSpec.AgentID))
					Expect(state.JobState).To(Equal(expectedSpec.JobState))
					Expect(state.Deployment).To(Equal(expectedSpec.Deployment))
					boshassert.LacksJSONKey(GinkgoT(), state, "vitals")

					Expect(state).To(Equal(expectedSpec))
				})

				It("returns state in full format", func() {
					settingsService.Settings.AgentID = "my-agent-id"
					settingsService.Settings.VM.Name = "vm-abc-def"

					jobSupervisor.StatusStatus = "running"
					jobSupervisor.ProcessesStatus = []boshjobsuper.Process{
						boshjobsuper.Process{
							Name:  "fake-process-name-1",
							State: "running",
						},
						boshjobsuper.Process{
							Name:  "fake-process-name-2",
							State: "failing",
						},
					}

					specService.Spec = boshas.V1ApplySpec{
						Deployment: "fake-deployment",
					}

					expectedVitals := boshvitals.Vitals{
						Load: []string{"foo", "bar", "baz"},
					}

					vitalsService.GetVitals = expectedVitals
					expectedVM := map[string]interface{}{"name": "vm-abc-def"}

					expectedProcesses := []boshjobsuper.Process{
						boshjobsuper.Process{
							Name:  "fake-process-name-1",
							State: "running",
						},
						boshjobsuper.Process{
							Name:  "fake-process-name-2",
							State: "failing",
						},
					}

					state, err := action.Run("full")
					Expect(err).ToNot(HaveOccurred())

					boshassert.MatchesJSONString(GinkgoT(), state.AgentID, `"my-agent-id"`)
					boshassert.MatchesJSONString(GinkgoT(), state.JobState, `"running"`)
					boshassert.MatchesJSONString(GinkgoT(), state.Deployment, `"fake-deployment"`)
					Expect(*state.Vitals).To(Equal(expectedVitals))
					Expect(state.Processes).To(Equal(expectedProcesses))
					boshassert.MatchesJSONMap(GinkgoT(), state.VM, expectedVM)
				})

				Describe("non-populated field formatting", func() {
					It("returns network as empty hash if not set", func() {
						specService.Spec = boshas.V1ApplySpec{NetworkSpecs: nil}
						state, err := action.Run("full")
						Expect(err).ToNot(HaveOccurred())
						boshassert.MatchesJSONString(GinkgoT(), state.NetworkSpecs, `{}`)

						// Non-empty NetworkSpecs
						specService.Spec = boshas.V1ApplySpec{
							NetworkSpecs: map[string]boshas.NetworkSpec{
								"fake-net-name": boshas.NetworkSpec{
									Fields: map[string]interface{}{"ip": "fake-ip"},
								},
							},
						}
						state, err = action.Run("full")
						Expect(err).ToNot(HaveOccurred())
						boshassert.MatchesJSONString(GinkgoT(), state.NetworkSpecs, `{"fake-net-name":{"ip":"fake-ip"}}`)
					})

					It("returns resource_pool as empty hash if not set", func() {
						specService.Spec = boshas.V1ApplySpec{ResourcePoolSpecs: nil}
						state, err := action.Run("full")
						Expect(err).ToNot(HaveOccurred())
						boshassert.MatchesJSONString(GinkgoT(), state.ResourcePoolSpecs, `{}`)

						// Non-empty ResourcePoolSpecs
						specService.Spec = boshas.V1ApplySpec{ResourcePoolSpecs: "fake-resource-pool"}
						state, err = action.Run("full")
						Expect(err).ToNot(HaveOccurred())
						boshassert.MatchesJSONString(GinkgoT(), state.ResourcePoolSpecs, `"fake-resource-pool"`)
					})

					It("returns packages as empty hash if not set", func() {
						specService.Spec = boshas.V1ApplySpec{PackageSpecs: nil}
						state, err := action.Run("full")
						Expect(err).ToNot(HaveOccurred())
						boshassert.MatchesJSONString(GinkgoT(), state.PackageSpecs, `{}`)

						// Non-empty PackageSpecs
						specService.Spec = boshas.V1ApplySpec{PackageSpecs: map[string]boshas.PackageSpec{}}
						state, err = action.Run("full")
						Expect(err).ToNot(HaveOccurred())
						boshassert.MatchesJSONString(GinkgoT(), state.PackageSpecs, `{}`)
					})
				})
			})

			Context("when vitals cannot be retrieved", func() {
				It("returns error", func() {
					vitalsService.GetErr = errors.New("fake-vitals-get-error")

					_, err := action.Run("full")
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-vitals-get-error"))
				})
			})
		})

		Context("when current spec cannot be retrieved", func() {
			It("without current spec", func() {
				specService.GetErr = errors.New("fake-spec-get-error")

				_, err := action.Run()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-spec-get-error"))
			})
		})
	})
})
