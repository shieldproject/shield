package agent_test

import (
	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agent"

	boshalert "github.com/cloudfoundry/bosh-agent/agent/alert"
	boshas "github.com/cloudfoundry/bosh-agent/agent/applier/applyspec"
	fakeas "github.com/cloudfoundry/bosh-agent/agent/applier/applyspec/fakes"
	fakeagent "github.com/cloudfoundry/bosh-agent/agent/fakes"
	boshhandler "github.com/cloudfoundry/bosh-agent/handler"
	fakejobsuper "github.com/cloudfoundry/bosh-agent/jobsupervisor/fakes"
	fakembus "github.com/cloudfoundry/bosh-agent/mbus/fakes"
	fakeplatform "github.com/cloudfoundry/bosh-agent/platform/fakes"
	boshvitals "github.com/cloudfoundry/bosh-agent/platform/vitals"
	fakesettings "github.com/cloudfoundry/bosh-agent/settings/fakes"
	boshsyslog "github.com/cloudfoundry/bosh-agent/syslog"
	fakesyslog "github.com/cloudfoundry/bosh-agent/syslog/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
	"github.com/pivotal-golang/clock/fakeclock"
)

func init() {
	Describe("Agent", func() {
		var (
			logger           boshlog.Logger
			handler          *fakembus.FakeHandler
			platform         *fakeplatform.FakePlatform
			actionDispatcher *fakeagent.FakeActionDispatcher
			jobSupervisor    *fakejobsuper.FakeJobSupervisor
			specService      *fakeas.FakeV1Service
			syslogServer     *fakesyslog.FakeServer
			settingsService  *fakesettings.FakeSettingsService
			uuidGenerator    *fakeuuid.FakeGenerator
			timeService      *fakeclock.FakeClock
			agent            Agent
		)

		BeforeEach(func() {
			logger = boshlog.NewLogger(boshlog.LevelNone)
			handler = &fakembus.FakeHandler{}
			platform = fakeplatform.NewFakePlatform()
			actionDispatcher = &fakeagent.FakeActionDispatcher{}
			jobSupervisor = fakejobsuper.NewFakeJobSupervisor()
			specService = fakeas.NewFakeV1Service()
			syslogServer = &fakesyslog.FakeServer{}
			settingsService = &fakesettings.FakeSettingsService{}
			uuidGenerator = &fakeuuid.FakeGenerator{}
			timeService = fakeclock.NewFakeClock(time.Now())
			agent = New(
				logger,
				handler,
				platform,
				actionDispatcher,
				jobSupervisor,
				specService,
				syslogServer,
				5*time.Millisecond,
				settingsService,
				uuidGenerator,
				timeService,
			)
		})

		Describe("Run", func() {
			It("lets dispatcher handle requests arriving via handler", func() {
				err := agent.Run()
				Expect(err).ToNot(HaveOccurred())

				expectedResp := boshhandler.NewValueResponse("pong")
				actionDispatcher.DispatchResp = expectedResp

				req := boshhandler.NewRequest("fake-reply", "fake-action", []byte("fake-payload"))
				resp := handler.RunFunc(req)

				Expect(actionDispatcher.DispatchReq).To(Equal(req))
				Expect(resp).To(Equal(expectedResp))
			})

			It("resumes persistent actions *before* dispatching new requests", func() {
				resumedBeforeStartingToDispatch := false
				handler.RunCallBack = func() {
					resumedBeforeStartingToDispatch = actionDispatcher.ResumedPreviouslyDispatchedTasks
				}

				err := agent.Run()
				Expect(err).ToNot(HaveOccurred())
				Expect(resumedBeforeStartingToDispatch).To(BeTrue())
			})

			Context("when heartbeats can be sent", func() {
				BeforeEach(func() {
					handler.KeepOnRunning()
				})

				BeforeEach(func() {
					jobName := "fake-job"
					nodeID := "node-id"
					jobIndex := 1
					specService.Spec = boshas.V1ApplySpec{
						Deployment: "FakeDeployment",
						JobSpec:    boshas.JobSpec{Name: &jobName},
						Index:      &jobIndex,
						NodeID:     nodeID,
					}

					jobSupervisor.StatusStatus = "fake-state"

					platform.FakeVitalsService.GetVitals = boshvitals.Vitals{
						Load: []string{"a", "b", "c"},
					}
				})

				expectedJobName := "fake-job"
				expectedJobIndex := 1
				expectedNodeID := "node-id"
				expectedHb := Heartbeat{
					Deployment: "FakeDeployment",
					Job:        &expectedJobName,
					Index:      &expectedJobIndex,
					JobState:   "fake-state",
					NodeID:     expectedNodeID,
					Vitals:     boshvitals.Vitals{Load: []string{"a", "b", "c"}},
				}

				It("sends initial heartbeat", func() {
					// Configure periodic heartbeat every 5 hours
					// so that we are sure that we will not receive it
					agent = New(
						logger,
						handler,
						platform,
						actionDispatcher,
						jobSupervisor,
						specService,
						syslogServer,
						5*time.Hour,
						settingsService,
						uuidGenerator,
						timeService,
					)

					// Immediately exit after sending initial heartbeat
					handler.SendErr = errors.New("stop")

					err := agent.Run()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("stop"))

					Expect(handler.SendInputs()).To(Equal([]fakembus.SendInput{
						{
							Target:  boshhandler.HealthMonitor,
							Topic:   boshhandler.Heartbeat,
							Message: expectedHb,
						},
					}))
				})

				It("sends periodic heartbeats", func() {
					sentRequests := 0
					handler.SendCallback = func(_ fakembus.SendInput) {
						sentRequests++
						if sentRequests == 3 {
							handler.SendErr = errors.New("stop")
						}
					}

					err := agent.Run()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("stop"))

					Expect(handler.SendInputs()).To(Equal([]fakembus.SendInput{
						{
							Target:  boshhandler.HealthMonitor,
							Topic:   boshhandler.Heartbeat,
							Message: expectedHb,
						},
						{
							Target:  boshhandler.HealthMonitor,
							Topic:   boshhandler.Heartbeat,
							Message: expectedHb,
						},
						{
							Target:  boshhandler.HealthMonitor,
							Topic:   boshhandler.Heartbeat,
							Message: expectedHb,
						},
					}))
				})
			})

			Context("when the agent fails to get job spec for a heartbeat", func() {
				BeforeEach(func() {
					specService.GetErr = errors.New("fake-spec-service-error")
					handler.KeepOnRunning()
				})

				It("returns the error", func() {
					err := agent.Run()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-spec-service-error"))
				})
			})

			Context("when the agent fails to get vitals for a heartbeat", func() {
				BeforeEach(func() {
					platform.FakeVitalsService.GetErr = errors.New("fake-vitals-service-error")
					handler.KeepOnRunning()
				})

				It("returns the error", func() {
					err := agent.Run()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-vitals-service-error"))
				})
			})

			It("sends job monitoring alerts to health manager", func() {
				handler.KeepOnRunning()

				monitAlert := boshalert.MonitAlert{
					ID:          "fake-monit-alert",
					Service:     "fake-service",
					Event:       "fake-event",
					Action:      "fake-action",
					Date:        "Sun, 22 May 2011 20:07:41 +0500",
					Description: "fake-description",
				}
				jobSupervisor.JobFailureAlert = &monitAlert

				// Fail the first time handler.Send is called for an alert (ignore heartbeats)
				handler.SendCallback = func(input fakembus.SendInput) {
					if input.Topic == boshhandler.Alert {
						handler.SendErr = errors.New("stop")
					}
				}

				err := agent.Run()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("stop"))

				expectedAlert := boshalert.Alert{
					ID:        "fake-monit-alert",
					Severity:  boshalert.SeverityDefault,
					Title:     "fake-service - fake-event - fake-action",
					Summary:   "fake-description",
					CreatedAt: int64(1306076861),
				}

				Expect(handler.SendInputs()).To(ContainElement(fakembus.SendInput{
					Target:  boshhandler.HealthMonitor,
					Topic:   boshhandler.Alert,
					Message: expectedAlert,
				}))
			})

			It("sends ssh alerts to health manager", func() {
				handler.KeepOnRunning()

				syslogMsg := boshsyslog.Msg{Content: "disconnected by user"}
				syslogServer.StartFirstSyslogMsg = &syslogMsg

				uuidGenerator.GeneratedUUID = "fake-uuid"

				// Fail the first time handler.Send is called for an alert (ignore heartbeats)
				handler.SendCallback = func(input fakembus.SendInput) {
					if input.Topic == boshhandler.Alert {
						handler.SendErr = errors.New("stop")
					}
				}

				err := agent.Run()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("stop"))

				expectedAlert := boshalert.Alert{
					ID:        "fake-uuid",
					Severity:  boshalert.SeverityWarning,
					Title:     "SSH Logout",
					Summary:   "disconnected by user",
					CreatedAt: timeService.Now().Unix(),
				}

				Expect(handler.SendInputs()).To(ContainElement(fakembus.SendInput{
					Target:  boshhandler.HealthMonitor,
					Topic:   boshhandler.Alert,
					Message: expectedAlert,
				}))
			})
		})
	})
}
