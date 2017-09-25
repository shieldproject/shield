package http_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agentclient/http"

	"github.com/cloudfoundry/bosh-agent/agentclient"
	"github.com/cloudfoundry/bosh-agent/agentclient/applyspec"

	fakehttpclient "github.com/cloudfoundry/bosh-utils/httpclient/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var _ = Describe("AgentClient", func() {
	var (
		fakeHTTPClient *fakehttpclient.FakeHTTPClient
		agentClient    agentclient.AgentClient

		agentAddress        string
		agentEndpoint       string
		replyToAddress      string
		toleratedErrorCount int
	)

	BeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		fakeHTTPClient = fakehttpclient.NewFakeHTTPClient()

		agentAddress = "http://localhost:6305"
		agentEndpoint = agentAddress + "/agent"
		replyToAddress = "fake-reply-to-uuid"

		getTaskDelay := time.Duration(0)
		toleratedErrorCount = 2

		agentClient = NewAgentClient(agentAddress, replyToAddress, getTaskDelay, toleratedErrorCount, fakeHTTPClient, logger)
	})

	Describe("get_task", func() {
		Context("when the http client errors", func() {
			It("should retry", func() {
				fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)
				fakeHTTPClient.SetPostBehavior("", 0, errors.New("connection reset by peer"))
				fakeHTTPClient.SetPostBehavior("", 0, errors.New("connection reset by peer"))
				fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)
				fakeHTTPClient.SetPostBehavior(`{"value":"stopped"}`, 200, nil)

				err := agentClient.Stop()
				Expect(err).ToNot(HaveOccurred())
			})

			Context("when the http client errors more times than the error retry count", func() {
				It("should return the error", func() {
					fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)
					fakeHTTPClient.SetPostBehavior("", 0, errors.New("connection reset by peer 1"))
					fakeHTTPClient.SetPostBehavior("", 0, errors.New("connection reset by peer 2"))
					fakeHTTPClient.SetPostBehavior("", 0, errors.New("connection reset by peer 3"))

					err := agentClient.Stop()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("connection reset by peer 3"))
				})
			})

			Context("when the https client errors, recovers, and begins erroring again", func() {
				It("should reset the error count when a successful call goes through", func() {
					fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)
					fakeHTTPClient.SetPostBehavior("", 0, errors.New("connection reset by peer 1"))
					fakeHTTPClient.SetPostBehavior("", 0, errors.New("connection reset by peer 2"))
					fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)
					fakeHTTPClient.SetPostBehavior("", 0, errors.New("connection reset by peer 3"))
					fakeHTTPClient.SetPostBehavior("", 0, errors.New("connection reset by peer 4"))
					fakeHTTPClient.SetPostBehavior("", 0, errors.New("connection is bad"))

					err := agentClient.Stop()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("connection is bad"))
				})
			})
		})
	})

	Describe("Ping", func() {
		Context("when agent responds with a value", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior(`{"value":"pong"}`, 200, nil)
			})

			It("makes a POST request to the endpoint", func() {
				_, err := agentClient.Ping()
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeHTTPClient.PostInputs).To(HaveLen(1))
				Expect(fakeHTTPClient.PostInputs[0].Endpoint).To(Equal(agentEndpoint))

				var request AgentRequestMessage
				err = json.Unmarshal(fakeHTTPClient.PostInputs[0].Payload, &request)
				Expect(err).ToNot(HaveOccurred())

				Expect(request).To(Equal(AgentRequestMessage{
					Method:    "ping",
					Arguments: []interface{}{},
					ReplyTo:   replyToAddress,
				}))
			})

			It("returns the value", func() {
				responseValue, err := agentClient.Ping()
				Expect(err).ToNot(HaveOccurred())
				Expect(responseValue).To(Equal("pong"))
			})
		})

		Context("when agent does not respond with 200", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior("", http.StatusInternalServerError, nil)
			})

			It("returns an error", func() {
				_, err := agentClient.Ping()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("status code: 500"))
			})
		})

		Context("when agent responds with exception", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior(`{"exception":{"message":"bad request"}}`, 200, nil)
			})

			It("returns an error", func() {
				_, err := agentClient.Ping()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("bad request"))
			})
		})
	})

	Describe("Stop", func() {
		Context("when agent responds with a value", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)
				fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)
				fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)
				fakeHTTPClient.SetPostBehavior(`{"value":"stopped"}`, 200, nil)
			})

			It("makes a POST request to the endpoint", func() {
				err := agentClient.Stop()
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeHTTPClient.PostInputs).To(HaveLen(4))
				Expect(fakeHTTPClient.PostInputs[0].Endpoint).To(Equal(agentEndpoint))

				var request AgentRequestMessage
				err = json.Unmarshal(fakeHTTPClient.PostInputs[0].Payload, &request)
				Expect(err).ToNot(HaveOccurred())

				Expect(request).To(Equal(AgentRequestMessage{
					Method:    "stop",
					Arguments: []interface{}{},
					ReplyTo:   replyToAddress,
				}))
			})

			It("waits for the task to be finished", func() {
				err := agentClient.Stop()
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeHTTPClient.PostInputs).To(HaveLen(4))
				Expect(fakeHTTPClient.PostInputs[1].Endpoint).To(Equal(agentEndpoint))

				var request AgentRequestMessage
				err = json.Unmarshal(fakeHTTPClient.PostInputs[1].Payload, &request)
				Expect(err).ToNot(HaveOccurred())

				Expect(request).To(Equal(AgentRequestMessage{
					Method:    "get_task",
					Arguments: []interface{}{"fake-agent-task-id"},
					ReplyTo:   replyToAddress,
				}))
			})
		})

		Context("when agent does not respond with 200", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior("", http.StatusInternalServerError, nil)
			})

			It("returns an error", func() {
				err := agentClient.Stop()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("status code: 500"))
			})
		})

		Context("when agent responds with exception", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior(`{"exception":{"message":"bad request"}}`, 200, nil)
			})

			It("returns an error", func() {
				err := agentClient.Stop()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("bad request"))
			})
		})
	})

	Describe("Apply", func() {
		var (
			specJSON []byte
			spec     applyspec.ApplySpec
		)

		BeforeEach(func() {
			spec = applyspec.ApplySpec{
				Deployment: "fake-deployment-name",
			}
			var err error
			specJSON, err = json.Marshal(spec)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when agent responds with a value", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)
				fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)
				fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)
				fakeHTTPClient.SetPostBehavior(`{"value":"stopped"}`, 200, nil)
			})

			It("makes a POST request to the endpoint", func() {
				err := agentClient.Apply(spec)
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeHTTPClient.PostInputs).To(HaveLen(4))
				Expect(fakeHTTPClient.PostInputs[0].Endpoint).To(Equal(agentEndpoint))

				var request AgentRequestMessage
				err = json.Unmarshal(fakeHTTPClient.PostInputs[0].Payload, &request)
				Expect(err).ToNot(HaveOccurred())

				var specArgument interface{}
				err = json.Unmarshal(specJSON, &specArgument)
				Expect(err).ToNot(HaveOccurred())

				Expect(request).To(Equal(AgentRequestMessage{
					Method:    "apply",
					Arguments: []interface{}{specArgument},
					ReplyTo:   replyToAddress,
				}))
			})

			It("waits for the task to be finished", func() {
				err := agentClient.Apply(spec)
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeHTTPClient.PostInputs).To(HaveLen(4))
				Expect(fakeHTTPClient.PostInputs[1].Endpoint).To(Equal(agentEndpoint))

				var request AgentRequestMessage
				err = json.Unmarshal(fakeHTTPClient.PostInputs[1].Payload, &request)
				Expect(err).ToNot(HaveOccurred())

				Expect(request).To(Equal(AgentRequestMessage{
					Method:    "get_task",
					Arguments: []interface{}{"fake-agent-task-id"},
					ReplyTo:   replyToAddress,
				}))
			})
		})

		Context("when agent does not respond with 200", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior("", http.StatusInternalServerError, nil)
			})

			It("returns an error", func() {
				err := agentClient.Apply(spec)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("status code: 500"))
			})
		})

		Context("when agent responds with exception", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior(`{"exception":{"message":"bad request"}}`, 200, nil)
			})

			It("returns an error", func() {
				err := agentClient.Apply(spec)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("bad request"))
			})
		})
	})

	Describe("Start", func() {
		Context("when agent responds with a value", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior(`{"value":"started"}`, 200, nil)
			})

			It("makes a POST request to the endpoint", func() {
				err := agentClient.Start()
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeHTTPClient.PostInputs).To(HaveLen(1))
				Expect(fakeHTTPClient.PostInputs[0].Endpoint).To(Equal(agentEndpoint))

				var request AgentRequestMessage
				err = json.Unmarshal(fakeHTTPClient.PostInputs[0].Payload, &request)
				Expect(err).ToNot(HaveOccurred())

				Expect(request).To(Equal(AgentRequestMessage{
					Method:    "start",
					Arguments: []interface{}{},
					ReplyTo:   replyToAddress,
				}))
			})
		})

		Context("when agent does not respond with 200", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior("", http.StatusInternalServerError, nil)
			})

			It("returns an error", func() {
				err := agentClient.Start()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("status code: 500"))
			})
		})

		Context("when agent responds with exception", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior(`{"exception":{"message":"bad request"}}`, 200, nil)
			})

			It("returns an error", func() {
				err := agentClient.Start()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("bad request"))
			})
		})
	})

	Describe("GetState", func() {
		Context("when agent responds with a value", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior(`{"value":{"job_state":"running","networks":{"private":{"ip":"192.0.2.10"},"public":{"ip":"192.0.3.11"}}}}`, 200, nil)
			})

			It("makes a POST request to the endpoint", func() {
				stateResponse, err := agentClient.GetState()
				Expect(err).ToNot(HaveOccurred())
				Expect(stateResponse).To(Equal(agentclient.AgentState{
					JobState: "running",
					NetworkSpecs: map[string]agentclient.NetworkSpec{
						"private": {
							IP: "192.0.2.10",
						},
						"public": {
							IP: "192.0.3.11",
						},
					},
				}))

				Expect(fakeHTTPClient.PostInputs).To(HaveLen(1))
				Expect(fakeHTTPClient.PostInputs[0].Endpoint).To(Equal(agentEndpoint))

				var request AgentRequestMessage
				err = json.Unmarshal(fakeHTTPClient.PostInputs[0].Payload, &request)
				Expect(err).ToNot(HaveOccurred())

				Expect(request).To(Equal(AgentRequestMessage{
					Method:    "get_state",
					Arguments: []interface{}{},
					ReplyTo:   replyToAddress,
				}))
			})
		})

		Context("when agent does not respond with 200", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior("", http.StatusInternalServerError, nil)
				fakeHTTPClient.SetPostBehavior("", http.StatusInternalServerError, nil)
				fakeHTTPClient.SetPostBehavior("", http.StatusInternalServerError, nil)
			})

			It("returns an error", func() {
				stateResponse, err := agentClient.GetState()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("status code: 500"))
				Expect(stateResponse).To(Equal(agentclient.AgentState{}))
			})
		})

		Context("when agent responds with exception", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior(`{"exception":{"message":"bad request"}}`, 200, nil)
				fakeHTTPClient.SetPostBehavior(`{"exception":{"message":"bad request"}}`, 200, nil)
				fakeHTTPClient.SetPostBehavior(`{"exception":{"message":"bad request"}}`, 200, nil)
			})

			It("returns an error", func() {
				stateResponse, err := agentClient.GetState()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("bad request"))
				Expect(stateResponse).To(Equal(agentclient.AgentState{}))
			})
		})

		Context("when agent client errors sending the http request less times than the sendErrorCount", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior("", 0, errors.New("connection reset by peer"))
				fakeHTTPClient.SetPostBehavior("", 0, errors.New("connection reset by peer"))
				fakeHTTPClient.SetPostBehavior(`{"value":{"job_state":"running"}}`, 200, nil)
			})

			It("retries the up to error count specified", func() {
				stateResponse, err := agentClient.GetState()
				Expect(err).ToNot(HaveOccurred())
				Expect(stateResponse).To(Equal(agentclient.AgentState{JobState: "running"}))
			})
		})

		Context("when agent client errors sending the http request more times than the sendErrorCount", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior("", 0, errors.New("connection reset by peer 1"))
				fakeHTTPClient.SetPostBehavior("", 0, errors.New("connection reset by peer 2"))
				fakeHTTPClient.SetPostBehavior("", 0, errors.New("connection reset by peer 3"))
			})

			It("returns the error", func() {
				_, err := agentClient.GetState()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("connection reset by peer 3"))
			})
		})
	})

	Describe("MountDisk", func() {
		Context("when agent responds with a value", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)
				fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)
				fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)
				fakeHTTPClient.SetPostBehavior(`{"value":{}}`, 200, nil)
			})

			It("makes a POST request to the endpoint", func() {
				err := agentClient.MountDisk("fake-disk-cid")
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeHTTPClient.PostInputs).To(HaveLen(4))
				Expect(fakeHTTPClient.PostInputs[0].Endpoint).To(Equal(agentEndpoint))

				var request AgentRequestMessage
				err = json.Unmarshal(fakeHTTPClient.PostInputs[0].Payload, &request)
				Expect(err).ToNot(HaveOccurred())

				Expect(request).To(Equal(AgentRequestMessage{
					Method:    "mount_disk",
					Arguments: []interface{}{"fake-disk-cid"},
					ReplyTo:   replyToAddress,
				}))
			})

			It("waits for the task to be finished", func() {
				err := agentClient.MountDisk("fake-disk-cid")
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeHTTPClient.PostInputs).To(HaveLen(4))
				Expect(fakeHTTPClient.PostInputs[1].Endpoint).To(Equal(agentEndpoint))

				var request AgentRequestMessage
				err = json.Unmarshal(fakeHTTPClient.PostInputs[1].Payload, &request)
				Expect(err).ToNot(HaveOccurred())

				Expect(request).To(Equal(AgentRequestMessage{
					Method:    "get_task",
					Arguments: []interface{}{"fake-agent-task-id"},
					ReplyTo:   replyToAddress,
				}))
			})
		})

		Describe("UnmountDisk", func() {
			Context("when agent responds with a value", func() {
				BeforeEach(func() {
					fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)
					fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)
					fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)
					fakeHTTPClient.SetPostBehavior(`{"value":{}}`, 200, nil)
				})

				It("makes a POST request to the endpoint", func() {
					err := agentClient.UnmountDisk("fake-disk-cid")
					Expect(err).ToNot(HaveOccurred())

					Expect(fakeHTTPClient.PostInputs).To(HaveLen(4))
					Expect(fakeHTTPClient.PostInputs[0].Endpoint).To(Equal(agentEndpoint))

					var request AgentRequestMessage
					err = json.Unmarshal(fakeHTTPClient.PostInputs[0].Payload, &request)
					Expect(err).ToNot(HaveOccurred())

					Expect(request).To(Equal(AgentRequestMessage{
						Method:    "unmount_disk",
						Arguments: []interface{}{"fake-disk-cid"},
						ReplyTo:   replyToAddress,
					}))
				})

				It("waits for the task to be finished", func() {
					err := agentClient.UnmountDisk("fake-disk-cid")
					Expect(err).ToNot(HaveOccurred())

					Expect(fakeHTTPClient.PostInputs).To(HaveLen(4))
					Expect(fakeHTTPClient.PostInputs[1].Endpoint).To(Equal(agentEndpoint))

					var request AgentRequestMessage
					err = json.Unmarshal(fakeHTTPClient.PostInputs[1].Payload, &request)
					Expect(err).ToNot(HaveOccurred())

					Expect(request).To(Equal(AgentRequestMessage{
						Method:    "get_task",
						Arguments: []interface{}{"fake-agent-task-id"},
						ReplyTo:   replyToAddress,
					}))
				})
			})
		})

		Context("when agent does not respond with 200", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior("", http.StatusInternalServerError, nil)
			})

			It("returns an error", func() {
				err := agentClient.MountDisk("fake-disk-cid")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("status code: 500"))
			})
		})

		Context("when agent responds with exception", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior(`{"exception":{"message":"bad request"}}`, 200, nil)
			})

			It("returns an error", func() {
				err := agentClient.MountDisk("fake-disk-cid")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("bad request"))
			})
		})
	})

	Describe("ListDisk", func() {
		Context("when agent responds with a value", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior(`{"value":["fake-disk-1", "fake-disk-2"]}`, 200, nil)
			})

			It("makes a POST request to the endpoint", func() {
				_, err := agentClient.ListDisk()
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeHTTPClient.PostInputs).To(HaveLen(1))
				Expect(fakeHTTPClient.PostInputs[0].Endpoint).To(Equal(agentEndpoint))

				var request AgentRequestMessage
				err = json.Unmarshal(fakeHTTPClient.PostInputs[0].Payload, &request)
				Expect(err).ToNot(HaveOccurred())

				Expect(request).To(Equal(AgentRequestMessage{
					Method:    "list_disk",
					Arguments: []interface{}{},
					ReplyTo:   replyToAddress,
				}))
			})

			It("returns disks", func() {
				disks, err := agentClient.ListDisk()
				Expect(err).ToNot(HaveOccurred())
				Expect(disks).To(Equal([]string{"fake-disk-1", "fake-disk-2"}))
			})
		})

		Context("when agent does not respond with 200", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior("", http.StatusInternalServerError, nil)
			})

			It("returns an error", func() {
				_, err := agentClient.ListDisk()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("status code: 500"))
			})
		})

		Context("when agent responds with exception", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior(`{"exception":{"message":"bad request"}}`, 200, nil)
			})

			It("returns an error", func() {
				_, err := agentClient.ListDisk()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("bad request"))
			})
		})
	})

	Describe("MigrateDisk", func() {
		Context("when agent responds with a value", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)
				fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)
				fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)
				fakeHTTPClient.SetPostBehavior(`{"value":{}}`, 200, nil)
			})

			It("makes a POST request to the endpoint", func() {
				err := agentClient.MigrateDisk()
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeHTTPClient.PostInputs).To(HaveLen(4))
				Expect(fakeHTTPClient.PostInputs[0].Endpoint).To(Equal(agentEndpoint))

				var request AgentRequestMessage
				err = json.Unmarshal(fakeHTTPClient.PostInputs[0].Payload, &request)
				Expect(err).ToNot(HaveOccurred())

				Expect(request).To(Equal(AgentRequestMessage{
					Method:    "migrate_disk",
					Arguments: []interface{}{},
					ReplyTo:   replyToAddress,
				}))
			})

			It("waits for the task to be finished", func() {
				err := agentClient.MigrateDisk()
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeHTTPClient.PostInputs).To(HaveLen(4))
				Expect(fakeHTTPClient.PostInputs[1].Endpoint).To(Equal(agentEndpoint))

				var request AgentRequestMessage
				err = json.Unmarshal(fakeHTTPClient.PostInputs[1].Payload, &request)
				Expect(err).ToNot(HaveOccurred())

				Expect(request).To(Equal(AgentRequestMessage{
					Method:    "get_task",
					Arguments: []interface{}{"fake-agent-task-id"},
					ReplyTo:   replyToAddress,
				}))
			})
		})
	})

	Describe("CompilePackage", func() {
		BeforeEach(func() {
			fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)
			fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)
			fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)
			fakeHTTPClient.SetPostBehavior(`{
	"value": {
		"result": {
			"sha1": "fake-compiled-package-sha1",
			"blobstore_id": "fake-compiled-package-blobstore-id"
		}
	}
}
`, 200, nil)
		})

		It("makes a compile_package request and waits for the task to be done", func() {
			packageSource := agentclient.BlobRef{
				Name:        "fake-package-name",
				Version:     "fake-package-version",
				SHA1:        "fake-package-sha1",
				BlobstoreID: "fake-package-blobstore-id",
			}
			dependencies := []agentclient.BlobRef{
				{
					Name:        "fake-compiled-package-dep-name",
					Version:     "fake-compiled-package-dep-version",
					SHA1:        "fake-compiled-package-dep-sha1",
					BlobstoreID: "fake-compiled-package-dep-blobstore-id",
				},
			}
			_, err := agentClient.CompilePackage(packageSource, dependencies)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeHTTPClient.PostInputs).To(HaveLen(4))
			Expect(fakeHTTPClient.PostInputs[0].Endpoint).To(Equal(agentEndpoint))

			var request AgentRequestMessage
			err = json.Unmarshal(fakeHTTPClient.PostInputs[0].Payload, &request)
			Expect(err).ToNot(HaveOccurred())

			Expect(request).To(Equal(AgentRequestMessage{
				Method: "compile_package",
				Arguments: []interface{}{
					"fake-package-blobstore-id",
					"fake-package-sha1",
					"fake-package-name",
					"fake-package-version",
					map[string]interface{}{
						"fake-compiled-package-dep-name": map[string]interface{}{
							"name":         "fake-compiled-package-dep-name",
							"version":      "fake-compiled-package-dep-version",
							"sha1":         "fake-compiled-package-dep-sha1",
							"blobstore_id": "fake-compiled-package-dep-blobstore-id",
						},
					},
				},
				ReplyTo: replyToAddress,
			}))
		})
	})

	Describe("DeleteARPEntries", func() {
		var (
			ips []string
		)

		Context("when agent responds with a value", func() {
			BeforeEach(func() {
				ips = []string{"10.0.0.1", "10.0.0.2"}
				fakeHTTPClient.SetPostBehavior(`{"value":{}}`, 200, nil)
			})

			It("makes a POST request to the endpoint", func() {
				err := agentClient.DeleteARPEntries(ips)
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeHTTPClient.PostInputs).To(HaveLen(1))
				Expect(fakeHTTPClient.PostInputs[0].Endpoint).To(Equal(agentEndpoint))

				var request AgentRequestMessage
				err = json.Unmarshal(fakeHTTPClient.PostInputs[0].Payload, &request)
				Expect(err).ToNot(HaveOccurred())

				expectedIps := []interface{}{ips[0], ips[1]}
				Expect(request).To(Equal(AgentRequestMessage{
					Method:    "delete_arp_entries",
					Arguments: []interface{}{map[string]interface{}{"ips": expectedIps}},
					ReplyTo:   replyToAddress,
				}))
			})
		})

		Context("when agent does not respond with 200", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior("", http.StatusInternalServerError, nil)
			})

			It("returns an error", func() {
				err := agentClient.DeleteARPEntries(ips)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("status code: 500"))
			})
		})

		Context("when agent responds with exception", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior(`{"exception":{"message":"bad request"}}`, 200, nil)
			})

			It("returns an error", func() {
				err := agentClient.DeleteARPEntries(ips)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("bad request"))
			})
		})
	})

	Describe("RunScript", func() {
		It("sends a run_script message to the agent", func() {
			// run_script
			fakeHTTPClient.SetPostBehavior(`{"value":{"agent_task_id":"fake-agent-task-id","state":"running"}}`, 200, nil)

			// get_task
			fakeHTTPClient.SetPostBehavior(`{"value":{}}`, 200, nil)

			err := agentClient.RunScript("the-script", map[string]interface{}{})
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeHTTPClient.PostInputs).To(HaveLen(2))
			Expect(fakeHTTPClient.PostInputs[0].Endpoint).To(Equal(agentEndpoint))

			var request AgentRequestMessage
			err = json.Unmarshal(fakeHTTPClient.PostInputs[0].Payload, &request)
			Expect(err).ToNot(HaveOccurred())

			Expect(request).To(Equal(AgentRequestMessage{
				Method:    "run_script",
				Arguments: []interface{}{"the-script", map[string]interface{}{}},
				ReplyTo:   replyToAddress,
			}))
		})

		It("returns an error if an error occurs", func() {
			fakeHTTPClient.SetPostBehavior("", 0, errors.New("connection reset by peer"))

			err := agentClient.RunScript("the-script", map[string]interface{}{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("connection reset by peer"))

			Expect(fakeHTTPClient.PostInputs).To(HaveLen(1))
			Expect(fakeHTTPClient.PostInputs[0].Endpoint).To(Equal(agentEndpoint))

			var request AgentRequestMessage
			err = json.Unmarshal(fakeHTTPClient.PostInputs[0].Payload, &request)
			Expect(err).ToNot(HaveOccurred())

			Expect(request).To(Equal(AgentRequestMessage{
				Method:    "run_script",
				Arguments: []interface{}{"the-script", map[string]interface{}{}},
				ReplyTo:   replyToAddress,
			}))
		})

		It("does not return an error if the error is 'unknown message'", func() {
			fakeHTTPClient.SetPostBehavior(`{"exception":{"message":"Agent responded with error: unknown message run_script"}}`, 200, nil)

			err := agentClient.RunScript("the-script", map[string]interface{}{})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("SyncDNS", func() {
		Context("when agent successfully executes the sync_dns", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior(`{"value":"synced"}`, 200, nil)
			})

			It("makes a POST request to the endpoint", func() {
				_, err := agentClient.SyncDNS("fake-blob-store-id", "fake-blob-store-id-sha1")
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeHTTPClient.PostInputs).To(HaveLen(1))
				Expect(fakeHTTPClient.PostInputs[0].Endpoint).To(Equal("http://localhost:6305/agent"))

				var request AgentRequestMessage
				err = json.Unmarshal(fakeHTTPClient.PostInputs[0].Payload, &request)
				Expect(err).ToNot(HaveOccurred())

				Expect(request).To(Equal(AgentRequestMessage{
					Method:    "sync_dns",
					Arguments: []interface{}{"fake-blob-store-id", "fake-blob-store-id-sha1"},
					ReplyTo:   "fake-reply-to-uuid",
				}))
			})

			It("returns the synced value", func() {
				responseValue, err := agentClient.SyncDNS("fake-blob-store-id", "fake-blob-store-id-sha1")
				Expect(err).ToNot(HaveOccurred())
				Expect(responseValue).To(Equal("synced"))
			})
		})

		Context("when agent does not respond with 200", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior("", http.StatusInternalServerError, nil)
			})

			It("returns an error", func() {
				_, err := agentClient.SyncDNS("fake-blob-store-id", "fake-blob-store-id-sha1")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("status code: 500"))
			})
		})

		Context("when agent responds with exception", func() {
			BeforeEach(func() {
				fakeHTTPClient.SetPostBehavior(`{"exception":{"message":"bad request"}}`, 200, nil)
			})

			It("returns an error", func() {
				_, err := agentClient.SyncDNS("fake-blob-store-id", "fake-blob-store-id-sha1")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("bad request"))
			})
		})
	})
})
