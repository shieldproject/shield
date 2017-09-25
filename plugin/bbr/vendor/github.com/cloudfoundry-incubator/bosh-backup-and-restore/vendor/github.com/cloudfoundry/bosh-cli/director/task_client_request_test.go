package director_test

import (
	"crypto/tls"
	"net/http"
	"time"

	boshhttp "github.com/cloudfoundry/bosh-utils/httpclient"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "github.com/cloudfoundry/bosh-cli/director"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
)

var _ = Describe("TaskClientRequest", func() {
	var (
		server *ghttp.Server
		resp   []string

		buildReq func(TaskReporter) TaskClientRequest
		req      TaskClientRequest
	)

	BeforeEach(func() {
		_, server = BuildServer()

		buildReq = func(taskReporter TaskReporter) TaskClientRequest {
			httpTransport := &http.Transport{
				TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
				TLSHandshakeTimeout: 10 * time.Second,
			}

			rawClient := &http.Client{Transport: httpTransport}
			logger := boshlog.NewLogger(boshlog.LevelNone)
			httpClient := boshhttp.NewHTTPClient(rawClient, logger)
			fileReporter := NewNoopFileReporter()
			clientReq := NewClientRequest(server.URL(), httpClient, fileReporter, logger)
			return NewTaskClientRequest(clientReq, taskReporter, 0*time.Second)
		}

		resp = nil
		req = buildReq(NewNoopTaskReporter())
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("GetResult", func() {
		act := func() (int, []byte, error) { return req.GetResult("/path") }

		It("waits for task to finish", func() {
			redirectHeader := http.Header{}
			redirectHeader.Add("Location", "/tasks/123")

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/path"),
					ghttp.RespondWith(http.StatusFound, nil, redirectHeader),
				),
				// followed redirect
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"done"}`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"done"}`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123/output", "type=event"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123/output", "type=result"),
					ghttp.RespondWith(http.StatusOK, "task-result"),
				),
			)

			id, resp, err := act()
			Expect(err).ToNot(HaveOccurred())
			Expect(id).To(Equal(123))
			Expect(resp).To(Equal([]byte("task-result")))
		})

		It("returns error if any request fails", func() {
			AppendBadRequest(ghttp.VerifyRequest("GET", "/path"), server)

			_, _, err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Director responded with non-successful status code '400' response"))
		})
	})

	Describe("PostResult", func() {
		act := func() ([]byte, error) {
			setHeaders := func(req *http.Request) {
				req.Header.Add("Test", "val")
			}
			return req.PostResult("/path", []byte("req-body"), setHeaders)
		}

		It("waits for task to finish", func() {
			redirectHeader := http.Header{}
			redirectHeader.Add("Location", "/tasks/123")

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/path"),
					ghttp.VerifyBody([]byte("req-body")),
					ghttp.VerifyHeader(http.Header{"Test": []string{"val"}}),
					ghttp.RespondWith(http.StatusFound, nil, redirectHeader),
				),
				// followed redirect
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"done"}`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"done"}`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123/output", "type=event"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123/output", "type=result"),
					ghttp.RespondWith(http.StatusOK, "task-result"),
				),
			)

			resp, err := act()
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).To(Equal([]byte("task-result")))
		})

		It("returns error if any request fails", func() {
			AppendBadRequest(ghttp.VerifyRequest("POST", "/path"), server)

			_, err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Director responded with non-successful status code '400' response"))
		})
	})

	Describe("PutResult", func() {
		act := func() ([]byte, error) {
			setHeaders := func(req *http.Request) {
				req.Header.Add("Test", "val")
			}
			return req.PutResult("/path", []byte("req-body"), setHeaders)
		}

		It("waits for task to finish", func() {
			redirectHeader := http.Header{}
			redirectHeader.Add("Location", "/tasks/123")

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/path"),
					ghttp.VerifyBody([]byte("req-body")),
					ghttp.VerifyHeader(http.Header{"Test": []string{"val"}}),
					ghttp.RespondWith(http.StatusFound, nil, redirectHeader),
				),
				// followed redirect
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"done"}`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"done"}`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123/output", "type=event"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123/output", "type=result"),
					ghttp.RespondWith(http.StatusOK, "task-result"),
				),
			)

			resp, err := act()
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).To(Equal([]byte("task-result")))
		})

		It("returns error if any request fails", func() {
			AppendBadRequest(ghttp.VerifyRequest("PUT", "/path"), server)

			_, err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Director responded with non-successful status code '400' response"))
		})
	})

	Describe("DeleteResult", func() {
		act := func() ([]byte, error) { return req.DeleteResult("/path") }

		It("waits for task to finish", func() {
			redirectHeader := http.Header{}
			redirectHeader.Add("Location", "/tasks/123")

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("DELETE", "/path"),
					ghttp.RespondWith(http.StatusFound, nil, redirectHeader),
				),
				// followed redirect
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"done"}`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"done"}`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123/output", "type=event"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123/output", "type=result"),
					ghttp.RespondWith(http.StatusOK, "task-result"),
				),
			)

			resp, err := act()
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).To(Equal([]byte("task-result")))
		})

		It("returns error if any request fails", func() {
			AppendBadRequest(ghttp.VerifyRequest("DELETE", "/path"), server)

			_, err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Director responded with non-successful status code '400' response"))
		})
	})

	Describe("WaitForCompletion", func() {
		var (
			taskReporter *fakedir.FakeTaskReporter
		)

		BeforeEach(func() {
			taskReporter = &fakedir.FakeTaskReporter{}
		})

		act := func() error { return req.WaitForCompletion(123, "event", taskReporter) }

		It("waits for tasks to execute and succeeds if task state is 'done'", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"queued"}`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123/output", "type=event"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
				// processing state
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"processing"}`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123/output", "type=event"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
				// cancelling state
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"cancelling"}`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123/output", "type=event"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
				// done state
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"done"}`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123/output", "type=event"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)

			Expect(act()).ToNot(HaveOccurred())

			Expect(taskReporter.TaskStartedCallCount()).To(Equal(1))
			Expect(taskReporter.TaskFinishedCallCount()).To(Equal(1))
		})

		It("returns an error if task state is finished executing", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"queued"}`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123/output", "type=event"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
				// some state that's not done or indicating that task is in progress
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"state"}`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123/output", "type=event"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Expected task '123' to succeed but state is 'state'"))

			Expect(taskReporter.TaskStartedCallCount()).To(Equal(1))
			Expect(taskReporter.TaskFinishedCallCount()).To(Equal(1))
		})

		It("notifies task reporter when task has started, progressed and finished", func() {
			server.AppendHandlers(
				// #1: not satisfiable
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"queued"}`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123/output", "type=event"),
					ghttp.VerifyHeader(http.Header{"Range": []string{"bytes=0-"}}),
					ghttp.RespondWith(http.StatusRequestedRangeNotSatisfiable, ""),
				),
				// #2
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"processing"}`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123/output", "type=event"),
					ghttp.VerifyHeader(http.Header{"Range": []string{"bytes=0-"}}),
					ghttp.RespondWith(http.StatusOK, "chunk1"),
				),
				// #3
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"processing"}`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123/output", "type=event"),
					ghttp.VerifyHeader(http.Header{"Range": []string{"bytes=6-"}}),
					ghttp.RespondWith(http.StatusOK, "chunk2"),
				),
				// #4: empty response body
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"processing"}`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123/output", "type=event"),
					ghttp.VerifyHeader(http.Header{"Range": []string{"bytes=12-"}}),
					ghttp.RespondWith(http.StatusOK, ""),
				),
				// #5
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"done"}`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123/output", "type=event"),
					ghttp.VerifyHeader(http.Header{"Range": []string{"bytes=12-"}}),
					ghttp.RespondWith(http.StatusOK, "chunk3"),
				),
			)

			err := req.WaitForCompletion(123, "event", taskReporter)
			Expect(err).ToNot(HaveOccurred())

			Expect(taskReporter.TaskStartedCallCount()).To(Equal(1))
			Expect(taskReporter.TaskStartedArgsForCall(0)).To(Equal(123))

			Expect(taskReporter.TaskFinishedCallCount()).To(Equal(1))

			id, state := taskReporter.TaskFinishedArgsForCall(0)
			Expect(id).To(Equal(123))
			Expect(state).To(Equal("done"))

			{
				Expect(taskReporter.TaskOutputChunkCallCount()).To(Equal(3))

				// Not called for StatusRequestedRangeNotSatisfiable response

				id, chunk := taskReporter.TaskOutputChunkArgsForCall(0)
				Expect(id).To(Equal(123))
				Expect(chunk).To(Equal([]byte("chunk1")))

				id, chunk = taskReporter.TaskOutputChunkArgsForCall(1)
				Expect(id).To(Equal(123))
				Expect(chunk).To(Equal([]byte("chunk2")))

				// Not called for empty response body range

				id, chunk = taskReporter.TaskOutputChunkArgsForCall(2)
				Expect(id).To(Equal(123))
				Expect(chunk).To(Equal([]byte("chunk3")))
			}
		})

		It("returns an error if getting task state fails", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusBadRequest, ""),
				),
			)

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Getting task state"))

			Expect(taskReporter.TaskStartedCallCount()).To(Equal(1))
			Expect(taskReporter.TaskFinishedCallCount()).To(Equal(1))
		})

		It("returns an error if getting task output fails", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"queued"}`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123/output", "type=event"),
					ghttp.RespondWith(http.StatusBadRequest, ""),
				),
			)

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Getting task output"))

			Expect(taskReporter.TaskStartedCallCount()).To(Equal(1))
			Expect(taskReporter.TaskFinishedCallCount()).To(Equal(1))
		})
	})
})
