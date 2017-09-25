package httpclient_test

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/http"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-utils/httpclient"
)

var _ = Describe("HTTPClient", func() {
	var (
		httpClient HTTPClient
		serv       *fakeServer
	)

	BeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		httpClient = NewHTTPClient(CreateDefaultClientInsecureSkipVerify(), logger)

		serv = newFakeServer("localhost:0")
		readyCh := make(chan error)
		go serv.Start(readyCh)

		err := <-readyCh

		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		serv.Stop()
	})

	Describe("Post/PostCustomized", func() {
		It("makes a POST request with given payload", func() {
			serv.SetResponseBody("post-response")
			serv.SetResponseStatus(200)

			url := "http://" + serv.Listener.Addr().String() + "/path"

			response, err := httpClient.Post(url, []byte("post-request"))
			Expect(err).ToNot(HaveOccurred())

			defer response.Body.Close()

			responseBody, err := ioutil.ReadAll(response.Body)
			Expect(err).ToNot(HaveOccurred())

			Expect(responseBody).To(Equal([]byte("post-response")))
			Expect(response.StatusCode).To(Equal(200))

			Expect(serv.ReceivedRequests).To(HaveLen(1))
			Expect(serv.ReceivedRequests).To(ContainElement(
				receivedRequest{
					Body:   []byte("post-request"),
					Method: "POST",
				},
			))
		})

		It("allows to override request including payload", func() {
			serv.SetResponseBody("post-response")
			serv.SetResponseStatus(200)

			url := "http://" + serv.Listener.Addr().String() + "/path"

			setHeaders := func(r *http.Request) {
				r.Header.Add("X-Custom", "custom")
				r.Body = ioutil.NopCloser(bytes.NewBufferString("post-request-override"))
				r.ContentLength = 21
			}

			response, err := httpClient.PostCustomized(url, []byte("post-request"), setHeaders)
			Expect(err).ToNot(HaveOccurred())

			defer response.Body.Close()

			responseBody, err := ioutil.ReadAll(response.Body)
			Expect(err).ToNot(HaveOccurred())

			Expect(responseBody).To(Equal([]byte("post-response")))
			Expect(response.StatusCode).To(Equal(200))

			Expect(serv.ReceivedRequests).To(HaveLen(1))
			Expect(serv.ReceivedRequests).To(ContainElement(
				receivedRequest{
					Body:   []byte("post-request-override"),
					Method: "POST",

					CustomHeader: "custom",
				},
			))
		})
	})

	Describe("Put/PutCustomized", func() {
		It("makes a PUT request with given payload", func() {
			serv.SetResponseBody("put-response")
			serv.SetResponseStatus(200)

			url := "http://" + serv.Listener.Addr().String() + "/path"

			response, err := httpClient.Put(url, []byte("put-request"))
			Expect(err).ToNot(HaveOccurred())

			defer response.Body.Close()

			responseBody, err := ioutil.ReadAll(response.Body)
			Expect(err).ToNot(HaveOccurred())

			Expect(responseBody).To(Equal([]byte("put-response")))
			Expect(response.StatusCode).To(Equal(200))

			Expect(serv.ReceivedRequests).To(HaveLen(1))
			Expect(serv.ReceivedRequests).To(ContainElement(
				receivedRequest{
					Body:   []byte("put-request"),
					Method: "PUT",
				},
			))
		})

		It("allows to override request including payload", func() {
			serv.SetResponseBody("put-response")
			serv.SetResponseStatus(200)

			url := "http://" + serv.Listener.Addr().String() + "/path"

			setHeaders := func(r *http.Request) {
				r.Header.Add("X-Custom", "custom")
				r.Body = ioutil.NopCloser(bytes.NewBufferString("put-request-override"))
				r.ContentLength = 20
			}

			response, err := httpClient.PutCustomized(url, []byte("put-request"), setHeaders)
			Expect(err).ToNot(HaveOccurred())

			defer response.Body.Close()

			responseBody, err := ioutil.ReadAll(response.Body)
			Expect(err).ToNot(HaveOccurred())

			Expect(responseBody).To(Equal([]byte("put-response")))
			Expect(response.StatusCode).To(Equal(200))

			Expect(serv.ReceivedRequests).To(HaveLen(1))
			Expect(serv.ReceivedRequests).To(ContainElement(
				receivedRequest{
					Body:   []byte("put-request-override"),
					Method: "PUT",

					CustomHeader: "custom",
				},
			))
		})
	})

	Describe("Get/GetCustomized", func() {
		It("makes a get request with given payload", func() {
			serv.SetResponseBody("get-response")
			serv.SetResponseStatus(200)

			url := "http://" + serv.Listener.Addr().String() + "/path"

			response, err := httpClient.Get(url)
			Expect(err).ToNot(HaveOccurred())

			defer response.Body.Close()

			responseBody, err := ioutil.ReadAll(response.Body)
			Expect(err).ToNot(HaveOccurred())

			Expect(responseBody).To(Equal([]byte("get-response")))
			Expect(response.StatusCode).To(Equal(200))

			Expect(serv.ReceivedRequests).To(HaveLen(1))
			Expect(serv.ReceivedRequests).To(ContainElement(
				receivedRequest{
					Body:   []byte(""),
					Method: "GET",
				},
			))
		})

		It("allows to override request", func() {
			serv.SetResponseBody("get-response")
			serv.SetResponseStatus(200)

			url := "http://" + serv.Listener.Addr().String() + "/path"

			setHeaders := func(r *http.Request) {
				r.Header.Add("X-Custom", "custom")
			}

			response, err := httpClient.GetCustomized(url, setHeaders)
			Expect(err).ToNot(HaveOccurred())

			defer response.Body.Close()

			responseBody, err := ioutil.ReadAll(response.Body)
			Expect(err).ToNot(HaveOccurred())

			Expect(responseBody).To(Equal([]byte("get-response")))
			Expect(response.StatusCode).To(Equal(200))

			Expect(serv.ReceivedRequests).To(HaveLen(1))
			Expect(serv.ReceivedRequests).To(ContainElement(
				receivedRequest{
					Body:   []byte(""),
					Method: "GET",

					CustomHeader: "custom",
				},
			))
		})
	})

})

type receivedRequestBody struct {
	Method    string
	Arguments []interface{}
	ReplyTo   string `json:"reply_to"`
}

type receivedRequest struct {
	Body   []byte
	Method string

	CustomHeader string
}

type fakeServer struct {
	Listener         net.Listener
	endpoint         string
	ReceivedRequests []receivedRequest
	responseBody     string
	responseStatus   int
}

func newFakeServer(endpoint string) *fakeServer {
	return &fakeServer{
		endpoint:         endpoint,
		responseStatus:   http.StatusOK,
		ReceivedRequests: []receivedRequest{},
	}
}

func (s *fakeServer) Start(readyErrCh chan error) {
	var err error
	s.Listener, err = net.Listen("tcp", s.endpoint)
	if err != nil {
		readyErrCh <- err
		return
	}

	readyErrCh <- nil

	httpServer := http.Server{}
	httpServer.SetKeepAlivesEnabled(false)
	mux := http.NewServeMux()
	httpServer.Handler = mux

	mux.HandleFunc("/path", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(s.responseStatus)

		requestBody, _ := ioutil.ReadAll(r.Body)
		defer r.Body.Close()

		receivedRequest := receivedRequest{
			Body:   requestBody,
			Method: r.Method,

			CustomHeader: r.Header.Get("X-Custom"),
		}

		s.ReceivedRequests = append(s.ReceivedRequests, receivedRequest)
		w.Write([]byte(s.responseBody))
	})

	httpServer.Serve(s.Listener)
}

func (s *fakeServer) Stop() {
	s.Listener.Close()
}

func (s *fakeServer) SetResponseStatus(code int) {
	s.responseStatus = code
}

func (s *fakeServer) SetResponseBody(body string) {
	s.responseBody = body
}
