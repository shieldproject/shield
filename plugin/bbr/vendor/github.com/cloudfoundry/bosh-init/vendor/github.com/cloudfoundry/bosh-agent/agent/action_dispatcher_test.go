package agent_test

import (
	"encoding/json"
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agent"
	fakeaction "github.com/cloudfoundry/bosh-agent/agent/action/fakes"
	boshtask "github.com/cloudfoundry/bosh-agent/agent/task"
	faketask "github.com/cloudfoundry/bosh-agent/agent/task/fakes"
	boshhandler "github.com/cloudfoundry/bosh-agent/handler"
	boshassert "github.com/cloudfoundry/bosh-utils/assert"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

func init() {
	Describe("actionDispatcher", func() {
		var (
			logger        boshlog.Logger
			taskService   *faketask.FakeService
			taskManager   *faketask.FakeManager
			actionFactory *fakeaction.FakeFactory
			actionRunner  *fakeaction.FakeRunner
			dispatcher    ActionDispatcher
		)

		BeforeEach(func() {
			logger = boshlog.NewLogger(boshlog.LevelNone)
			taskService = faketask.NewFakeService()
			taskManager = faketask.NewFakeManager()
			actionFactory = fakeaction.NewFakeFactory()
			actionRunner = &fakeaction.FakeRunner{}
			dispatcher = NewActionDispatcher(logger, taskService, taskManager, actionFactory, actionRunner)
		})

		It("responds with exception when the method is unknown", func() {
			actionFactory.RegisterActionErr("fake-action", errors.New("fake-create-error"))

			req := boshhandler.NewRequest("fake-reply", "fake-action", []byte{})
			resp := dispatcher.Dispatch(req)
			boshassert.MatchesJSONString(GinkgoT(), resp, `{"exception":{"message":"unknown message fake-action"}}`)
		})

		Context("when action is synchronous", func() {
			var (
				req boshhandler.Request
			)

			BeforeEach(func() {
				req = boshhandler.NewRequest("fake-reply", "fake-action", []byte("fake-payload"))
				actionFactory.RegisterAction("fake-action", &fakeaction.TestAction{Asynchronous: false})
			})

			It("handles synchronous action", func() {
				actionRunner.RunValue = "fake-value"

				resp := dispatcher.Dispatch(req)
				Expect(req.GetPayload()).To(Equal(actionRunner.RunPayload))
				Expect(boshhandler.NewValueResponse("fake-value")).To(Equal(resp))
			})

			It("handles synchronous action when err", func() {
				actionRunner.RunErr = errors.New("fake-run-error")

				resp := dispatcher.Dispatch(req)
				expectedJSON := fmt.Sprintf("{\"exception\":{\"message\":\"Action Failed %s: fake-run-error\"}}", req.Method)
				boshassert.MatchesJSONString(GinkgoT(), resp, expectedJSON)
			})
		})

		Context("when action is asynchronous", func() {
			var (
				req    boshhandler.Request
				action *fakeaction.TestAction
			)

			BeforeEach(func() {
				req = boshhandler.NewRequest("fake-reply", "fake-action", []byte("fake-payload"))
				action = &fakeaction.TestAction{Asynchronous: true}
				actionFactory.RegisterAction("fake-action", action)
			})

			ItAllowsToCancelTask := func() {
				It("allows task to be cancelled", func() {
					dispatcher.Dispatch(req)

					err := taskService.StartedTasks["fake-generated-task-id"].Cancel()
					Expect(err).ToNot(HaveOccurred())

					Expect(action.Canceled).To(BeTrue())
				})

				It("returns error from cancelling task if canceling task fails", func() {
					action.CancelErr = errors.New("fake-cancel-err")
					dispatcher.Dispatch(req)

					err := taskService.StartedTasks["fake-generated-task-id"].Cancel()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-cancel-err"))
				})
			}

			Context("when action is not persistent", func() {
				BeforeEach(func() {
					action.Persistent = false
				})

				It("responds with task id and state", func() {
					resp := dispatcher.Dispatch(req)
					boshassert.MatchesJSONString(GinkgoT(), resp,
						`{"value":{"agent_task_id":"fake-generated-task-id","state":"running"}}`)
				})

				It("starts running created task", func() {
					dispatcher.Dispatch(req)
					Expect(len(taskService.StartedTasks)).To(Equal(1))
					Expect(taskService.StartedTasks["fake-generated-task-id"]).ToNot(BeNil())
				})

				It("returns create task error", func() {
					taskService.CreateTaskErr = errors.New("fake-create-task-error")
					resp := dispatcher.Dispatch(req)
					respJSON, err := json.Marshal(resp)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(respJSON)).To(ContainSubstring("fake-create-task-error"))
				})

				It("return run value to the task", func() {
					actionRunner.RunValue = "fake-value"
					dispatcher.Dispatch(req)

					value, err := taskService.StartedTasks["fake-generated-task-id"].Func()
					Expect(value).To(Equal("fake-value"))
					Expect(err).ToNot(HaveOccurred())

					Expect(actionRunner.RunAction).To(Equal(action))
					Expect(string(actionRunner.RunPayload)).To(Equal("fake-payload"))
				})

				It("returns run error to the task", func() {
					actionRunner.RunErr = errors.New("fake-run-error")
					dispatcher.Dispatch(req)

					value, err := taskService.StartedTasks["fake-generated-task-id"].Func()
					Expect(value).To(BeNil())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-run-error"))

					Expect(actionRunner.RunAction).To(Equal(action))
					Expect(string(actionRunner.RunPayload)).To(Equal("fake-payload"))
				})

				ItAllowsToCancelTask()

				It("does not add task to task manager since it should not be resumed if agent is restarted", func() {
					dispatcher.Dispatch(req)
					taskInfos, _ := taskManager.GetInfos()
					Expect(taskInfos).To(BeEmpty())
				})

				It("does not do anything after task finishes", func() {
					dispatcher.Dispatch(req)
					Expect(taskService.StartedTasks["fake-generated-task-id"].EndFunc).To(BeNil())
				})
			})

			Context("when action is persistent", func() {
				BeforeEach(func() {
					action.Persistent = true
				})

				It("responds with task id and state", func() {
					resp := dispatcher.Dispatch(req)
					boshassert.MatchesJSONString(GinkgoT(), resp,
						`{"value":{"agent_task_id":"fake-generated-task-id","state":"running"}}`)
				})

				It("starts running created task", func() {
					dispatcher.Dispatch(req)
					Expect(len(taskService.StartedTasks)).To(Equal(1))
					Expect(taskService.StartedTasks["fake-generated-task-id"]).ToNot(BeNil())
				})

				It("returns create task error", func() {
					taskService.CreateTaskErr = errors.New("fake-create-task-error")
					resp := dispatcher.Dispatch(req)
					respJSON, err := json.Marshal(resp)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(respJSON)).To(ContainSubstring("fake-create-task-error"))
				})

				It("return run value to the task", func() {
					actionRunner.RunValue = "fake-value"
					dispatcher.Dispatch(req)

					value, err := taskService.StartedTasks["fake-generated-task-id"].Func()
					Expect(value).To(Equal("fake-value"))
					Expect(err).ToNot(HaveOccurred())

					Expect(actionRunner.RunAction).To(Equal(action))
					Expect(string(actionRunner.RunPayload)).To(Equal("fake-payload"))
				})

				It("returns run error to the task", func() {
					actionRunner.RunErr = errors.New("fake-run-error")
					dispatcher.Dispatch(req)

					value, err := taskService.StartedTasks["fake-generated-task-id"].Func()
					Expect(value).To(BeNil())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-run-error"))

					Expect(actionRunner.RunAction).To(Equal(action))
					Expect(string(actionRunner.RunPayload)).To(Equal("fake-payload"))
				})

				ItAllowsToCancelTask()

				It("adds task to task manager before task starts so that it could be resumed if agent is restarted", func() {
					dispatcher.Dispatch(req)
					taskInfos, _ := taskManager.GetInfos()
					Expect(taskInfos).To(Equal([]boshtask.Info{
						boshtask.Info{
							TaskID:  "fake-generated-task-id",
							Method:  "fake-action",
							Payload: []byte("fake-payload"),
						},
					}))
				})

				It("removes task from task manager after task finishes", func() {
					dispatcher.Dispatch(req)
					taskService.StartedTasks["fake-generated-task-id"].EndFunc(boshtask.Task{ID: "fake-generated-task-id"})

					taskInfos, _ := taskManager.GetInfos()
					Expect(taskInfos).To(BeEmpty())
				})

				It("does not start running created task if task manager cannot add task", func() {
					taskManager.AddInfoErr = errors.New("fake-add-task-info-error")

					resp := dispatcher.Dispatch(req)
					boshassert.MatchesJSONString(GinkgoT(), resp,
						`{"exception":{"message":"Action Failed fake-action: fake-add-task-info-error"}}`)

					Expect(len(taskService.StartedTasks)).To(Equal(0))
				})
			})
		})

		Describe("ResumePreviouslyDispatchedTasks", func() {
			var firstAction, secondAction *fakeaction.TestAction

			BeforeEach(func() {
				err := taskManager.AddInfo(boshtask.Info{
					TaskID:  "fake-task-id-1",
					Method:  "fake-action-1",
					Payload: []byte("fake-task-payload-1"),
				})
				Expect(err).ToNot(HaveOccurred())

				err = taskManager.AddInfo(boshtask.Info{
					TaskID:  "fake-task-id-2",
					Method:  "fake-action-2",
					Payload: []byte("fake-task-payload-2"),
				})
				Expect(err).ToNot(HaveOccurred())

				firstAction = &fakeaction.TestAction{}
				secondAction = &fakeaction.TestAction{}
			})

			It("calls resume on each task that was saved in a task manager", func() {
				actionFactory.RegisterAction("fake-action-1", firstAction)
				actionFactory.RegisterAction("fake-action-2", secondAction)

				dispatcher.ResumePreviouslyDispatchedTasks()
				Expect(len(taskService.StartedTasks)).To(Equal(2))

				{ // Check that first task executes first action
					actionRunner.ResumeValue = "fake-resume-value-1"
					value, err := taskService.StartedTasks["fake-task-id-1"].Func()
					Expect(err).ToNot(HaveOccurred())
					Expect(value).To(Equal("fake-resume-value-1"))
					Expect(actionRunner.ResumeAction).To(Equal(firstAction))
					Expect(string(actionRunner.ResumePayload)).To(Equal("fake-task-payload-1"))
				}

				{ // Check that second task executes second action
					actionRunner.ResumeValue = "fake-resume-value-2"
					value, err := taskService.StartedTasks["fake-task-id-2"].Func()
					Expect(err).ToNot(HaveOccurred())
					Expect(value).To(Equal("fake-resume-value-2"))
					Expect(actionRunner.ResumeAction).To(Equal(secondAction))
					Expect(string(actionRunner.ResumePayload)).To(Equal("fake-task-payload-2"))
				}
			})

			It("removes tasks from task manager after each task finishes", func() {
				actionFactory.RegisterAction("fake-action-1", firstAction)
				actionFactory.RegisterAction("fake-action-2", secondAction)

				dispatcher.ResumePreviouslyDispatchedTasks()
				Expect(len(taskService.StartedTasks)).To(Equal(2))

				// Simulate all tasks ending
				taskService.StartedTasks["fake-task-id-1"].EndFunc(boshtask.Task{ID: "fake-task-id-1"})
				taskService.StartedTasks["fake-task-id-2"].EndFunc(boshtask.Task{ID: "fake-task-id-2"})

				taskInfos, err := taskManager.GetInfos()
				Expect(err).ToNot(HaveOccurred())
				Expect(taskInfos).To(BeEmpty())
			})

			It("return resume error to each task", func() {
				actionFactory.RegisterAction("fake-action-1", firstAction)
				actionFactory.RegisterAction("fake-action-2", secondAction)

				dispatcher.ResumePreviouslyDispatchedTasks()
				Expect(len(taskService.StartedTasks)).To(Equal(2))

				{ // Check that first task propagates its resume error
					actionRunner.ResumeErr = errors.New("fake-resume-error-1")
					value, err := taskService.StartedTasks["fake-task-id-1"].Func()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-resume-error-1"))
					Expect(value).To(BeNil())
					Expect(actionRunner.ResumeAction).To(Equal(firstAction))
					Expect(string(actionRunner.ResumePayload)).To(Equal("fake-task-payload-1"))
				}

				{ // Check that second task propagates its resume error
					actionRunner.ResumeErr = errors.New("fake-resume-error-2")
					value, err := taskService.StartedTasks["fake-task-id-2"].Func()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-resume-error-2"))
					Expect(value).To(BeNil())
					Expect(actionRunner.ResumeAction).To(Equal(secondAction))
					Expect(string(actionRunner.ResumePayload)).To(Equal("fake-task-payload-2"))
				}
			})

			It("ignores actions that cannot be created and removes them from task manager", func() {
				actionFactory.RegisterActionErr("fake-action-1", errors.New("fake-action-error-1"))
				actionFactory.RegisterAction("fake-action-2", secondAction)

				dispatcher.ResumePreviouslyDispatchedTasks()
				Expect(len(taskService.StartedTasks)).To(Equal(1))

				{ // Check that first action is removed from task manager
					taskInfos, err := taskManager.GetInfos()
					Expect(err).ToNot(HaveOccurred())
					Expect(taskInfos).To(Equal([]boshtask.Info{
						boshtask.Info{
							TaskID:  "fake-task-id-2",
							Method:  "fake-action-2",
							Payload: []byte("fake-task-payload-2"),
						},
					}))
				}

				{ // Check that second task executes second action
					taskService.StartedTasks["fake-task-id-2"].Func()
					Expect(actionRunner.ResumeAction).To(Equal(secondAction))
					Expect(string(actionRunner.ResumePayload)).To(Equal("fake-task-payload-2"))
				}
			})

			It("allows to cancel after resume", func() {
				actionFactory.RegisterAction("fake-action-1", firstAction)
				actionFactory.RegisterAction("fake-action-2", secondAction)

				dispatcher.ResumePreviouslyDispatchedTasks()

				err := taskService.StartedTasks["fake-task-id-1"].Cancel()
				Expect(err).ToNot(HaveOccurred())
				Expect(firstAction.Canceled).To(BeTrue())
				Expect(secondAction.Canceled).To(BeFalse())

				err = taskService.StartedTasks["fake-task-id-2"].Cancel()
				Expect(err).ToNot(HaveOccurred())
				Expect(secondAction.Canceled).To(BeTrue())
			})

			It("returns error from cancelling task when canceling resumed task fails", func() {
				actionFactory.RegisterAction("fake-action-1", firstAction)
				actionFactory.RegisterAction("fake-action-2", secondAction)

				dispatcher.ResumePreviouslyDispatchedTasks()

				firstAction.CancelErr = errors.New("fake-cancel-err-1")
				secondAction.CancelErr = errors.New("fake-cancel-err-2")

				err := taskService.StartedTasks["fake-task-id-1"].Cancel()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-cancel-err-1"))

				err = taskService.StartedTasks["fake-task-id-2"].Cancel()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-cancel-err-2"))
			})
		})
	})
}
