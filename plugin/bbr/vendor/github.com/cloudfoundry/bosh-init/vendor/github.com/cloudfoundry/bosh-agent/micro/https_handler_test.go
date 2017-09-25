package micro_test

import (
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	. "github.com/cloudfoundry/bosh-agent/micro"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshhandler "github.com/cloudfoundry/bosh-agent/handler"
	boshdir "github.com/cloudfoundry/bosh-agent/settings/directories"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("HTTPSHandler", func() {
	var (
		serverURL       string
		handler         HTTPSHandler
		fs              *fakesys.FakeFileSystem
		receivedRequest boshhandler.Request
		httpClient      http.Client
	)

	BeforeEach(func() {
		serverURL = "https://user:pass@127.0.0.1:6900"
		mbusURL, _ := url.Parse(serverURL)
		logger := boshlog.NewLogger(boshlog.LevelNone)
		fs = fakesys.NewFakeFileSystem()
		dirProvider := boshdir.NewProvider("/var/vcap")
		handler = NewHTTPSHandler(mbusURL, logger, fs, dirProvider)

		go handler.Start(func(req boshhandler.Request) (resp boshhandler.Response) {
			receivedRequest = req
			return boshhandler.NewValueResponse("expected value")
		})

		httpTransport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		httpClient = http.Client{Transport: httpTransport}

		waitForServerToStart(serverURL, httpClient)
	})

	AfterEach(func() {
		handler.Stop()
		time.Sleep(1 * time.Millisecond)
	})

	Describe("POST /agent", func() {
		It("receives request and responds", func() {
			postBody := `{"method":"ping","arguments":["foo","bar"], "reply_to": "reply to me!"}`
			postPayload := strings.NewReader(postBody)

			httpResponse, err := httpClient.Post(serverURL+"/agent", "application/json", postPayload)
			for err != nil {
				httpResponse, err = httpClient.Post(serverURL+"/agent", "application/json", postPayload)
			}

			defer httpResponse.Body.Close()

			Expect(receivedRequest.ReplyTo).To(Equal("reply to me!"))
			Expect(receivedRequest.Method).To(Equal("ping"))
			Expect(receivedRequest.GetPayload()).To(Equal([]byte(postBody)))

			httpBody, readErr := ioutil.ReadAll(httpResponse.Body)
			Expect(readErr).ToNot(HaveOccurred())
			Expect(httpBody).To(Equal([]byte(`{"value":"expected value"}`)))
		})

		Context("when incorrect http method is used", func() {
			It("returns a 404", func() {
				httpResponse, err := httpClient.Get(serverURL + "/agent")
				Expect(err).ToNot(HaveOccurred())
				Expect(httpResponse.StatusCode).To(Equal(404))
			})
		})
	})

	Describe("blob access", func() {
		Describe("GET /blobs", func() {
			It("returns data from file system", func() {
				fs.WriteFileString("/var/vcap/micro_bosh/data/cache/123-456-789", "Some data")

				httpResponse, err := httpClient.Get(serverURL + "/blobs/a5/123-456-789")
				for err != nil {
					httpResponse, err = httpClient.Get(serverURL + "/blobs/a5/123-456-789")
				}

				defer httpResponse.Body.Close()

				httpBody, readErr := ioutil.ReadAll(httpResponse.Body)
				Expect(readErr).ToNot(HaveOccurred())
				Expect(httpResponse.StatusCode).To(Equal(200))
				Expect(httpBody).To(Equal([]byte("Some data")))
			})

			It("closes the underlying file", func() {
				blobPath := "/var/vcap/micro_bosh/data/cache/123-456-789"

				fs.WriteFileString(blobPath, "Some data")

				httpResponse, err := httpClient.Get(serverURL + "/blobs/a5/123-456-789")

				defer httpResponse.Body.Close()
				fileStats, err := fs.FindFileStats(blobPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(fileStats.Open).To(BeFalse())
			})

			Context("when incorrect http method is used", func() {
				It("returns a 404", func() {
					postBody := `{"method":"ping","arguments":["foo","bar"], "reply_to": "reply to me!"}`
					postPayload := strings.NewReader(postBody)

					httpResponse, err := httpClient.Post(serverURL+"/blobs/123", "application/json", postPayload)
					Expect(err).ToNot(HaveOccurred())

					defer httpResponse.Body.Close()

					Expect(httpResponse.StatusCode).To(Equal(404))
				})
			})

			Context("when file does not exist", func() {
				It("returns a 404", func() {
					fs.OpenFileErr = errors.New("no such file or directory")
					httpResponse, err := httpClient.Get(serverURL + "/blobs/123")
					Expect(err).ToNot(HaveOccurred())

					defer httpResponse.Body.Close()
					Expect(httpResponse.StatusCode).To(Equal(404))
				})
			})

			Context("when file does not have correct permissions", func() {
				It("returns a 500", func() {
					fs.OpenFileErr = errors.New("permission denied")
					httpResponse, err := httpClient.Get(serverURL + "/blobs/123")
					Expect(err).ToNot(HaveOccurred())

					defer httpResponse.Body.Close()
					Expect(httpResponse.StatusCode).To(Equal(500))
				})
			})
		})

		Describe("PUT /blobs", func() {
			It("updates the blob on the file system", func() {
				fs.WriteFileString("/var/vcap/micro_bosh/data/cache/123-456-789", "Some data")

				putBody := `Updated data`
				putPayload := strings.NewReader(putBody)

				request, err := http.NewRequest("PUT", serverURL+"/blobs/a5/123-456-789", putPayload)
				Expect(err).ToNot(HaveOccurred())

				httpResponse, err := httpClient.Do(request)
				Expect(err).ToNot(HaveOccurred())

				defer httpResponse.Body.Close()
				Expect(httpResponse.StatusCode).To(Equal(201))

				contents, err := fs.ReadFileString("/var/vcap/micro_bosh/data/cache/123-456-789")
				Expect(err).ToNot(HaveOccurred())
				Expect(contents).To(Equal("Updated data"))
			})

			Context("when an incorrect username and password is provided", func() {
				It("returns a 401", func() {
					fs.WriteFileString("/var/vcap/micro_bosh/data/cache/123-456-789", "Some data")

					putBody := `Updated data`
					putPayload := strings.NewReader(putBody)

					httpRequest, err := http.NewRequest("PUT", strings.Replace(serverURL, "pass", "wrong", -1)+"/blobs/a5/123-456-789", putPayload)
					httpResponse, err := httpClient.Do(httpRequest)
					Expect(err).ToNot(HaveOccurred())

					defer httpResponse.Body.Close()

					Expect(httpResponse.StatusCode).To(Equal(401))
					Expect(httpResponse.Header.Get("WWW-Authenticate")).To(Equal(`Basic realm=""`))
				})
			})

			Context("when manager errors", func() {
				It("returns a 500 because of openfile error", func() {
					fs.OpenFileErr = errors.New("oops")

					putBody := `Updated data`
					putPayload := strings.NewReader(putBody)

					request, err := http.NewRequest("PUT", serverURL+"/blobs/a5/123-456-789", putPayload)
					Expect(err).ToNot(HaveOccurred())

					httpResponse, err := httpClient.Do(request)
					Expect(err).ToNot(HaveOccurred())

					defer httpResponse.Body.Close()
					Expect(httpResponse.StatusCode).To(Equal(500))

					responseBody, err := ioutil.ReadAll(httpResponse.Body)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(responseBody)).To(ContainSubstring("oops"))
				})
			})
		})
	})

	Describe("routing and auth", func() {
		Context("when an incorrect uri is specificed", func() {
			It("returns a 404", func() {
				postBody := `{"method":"ping","arguments":["foo","bar"], "reply_to": "reply to me!"}`
				postPayload := strings.NewReader(postBody)
				httpResponse, err := httpClient.Post(serverURL+"/bad_url", "application/json", postPayload)
				Expect(err).ToNot(HaveOccurred())

				defer httpResponse.Body.Close()

				Expect(httpResponse.StatusCode).To(Equal(404))
			})
		})

		Context("when an incorrect username/password was provided", func() {
			It("returns a 401", func() {
				postBody := `{"method":"ping","arguments":["foo","bar"], "reply_to": "reply to me!"}`
				postPayload := strings.NewReader(postBody)

				httpResponse, err := httpClient.Post(strings.Replace(serverURL, "pass", "wrong", -1)+"/agent", "application/json", postPayload)
				Expect(err).ToNot(HaveOccurred())

				defer httpResponse.Body.Close()

				Expect(httpResponse.StatusCode).To(Equal(401))
				Expect(httpResponse.Header.Get("WWW-Authenticate")).To(Equal(`Basic realm=""`))
			})
		})
	})
})

func waitForServerToStart(serverURL string, httpClient http.Client) {
	httpResponse, err := httpClient.Get(serverURL + "/healthz")
	for err != nil {
		httpResponse, err = httpClient.Get(serverURL + "/healthz")
	}
	defer httpResponse.Body.Close()
}
