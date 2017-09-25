package client_test

import (
	"errors"
	"io/ioutil"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-davcli/client"
	davconf "github.com/cloudfoundry/bosh-davcli/config"
	fakehttp "github.com/cloudfoundry/bosh-utils/http/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var _ = Describe("Client", func() {
	var (
		fakeHTTPClient *fakehttp.FakeClient
		config         davconf.Config
		client         Client
		logger         boshlog.Logger
	)

	BeforeEach(func() {
		fakeHTTPClient = fakehttp.NewFakeClient()
		logger = boshlog.NewLogger(boshlog.LevelNone)
		client = NewClient(config, fakeHTTPClient, logger)
	})

	Describe("Get", func() {
		It("returns the response body from the given path", func() {
			fakeHTTPClient.StatusCode = 200
			fakeHTTPClient.SetMessage("response")

			responseBody, err := client.Get("/")
			Expect(err).NotTo(HaveOccurred())
			buf := make([]byte, 1024)
			n, _ := responseBody.Read(buf)
			Expect(string(buf[0:n])).To(Equal("response"))
		})

		Context("when the http request fails", func() {
			BeforeEach(func() {
				fakeHTTPClient.Error = errors.New("")
			})

			It("returns err", func() {
				responseBody, err := client.Get("/")
				Expect(responseBody).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Getting dav blob /"))
			})
		})

		Context("when the http response code is not 200", func() {
			BeforeEach(func() {
				fakeHTTPClient.StatusCode = 300
				fakeHTTPClient.SetMessage("response")
			})

			It("returns err", func() {
				responseBody, err := client.Get("/")
				Expect(responseBody).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Getting dav blob /: Request failed, response: Response{ StatusCode: 300, Status: '' }"))
				Expect(len(fakeHTTPClient.Requests)).To(Equal(3))
			})
		})
	})

	Describe("Put", func() {
		Context("When the put request succeeds", func() {
			itUploadsABlob := func() {
				body := ioutil.NopCloser(strings.NewReader("content"))
				err := client.Put("/", body, int64(7))
				Expect(err).NotTo(HaveOccurred())
				Expect(len(fakeHTTPClient.Requests)).To(Equal(1))
				req := fakeHTTPClient.Requests[0]
				Expect(req.ContentLength).To(Equal(int64(7)))
				Expect(fakeHTTPClient.RequestBodies).To(Equal([]string{"content"}))
			}

			It("uploads the given content if the blob does not exist", func() {
				fakeHTTPClient.StatusCode = 201
				itUploadsABlob()
			})

			It("uploads the given content if the blob exists", func() {
				fakeHTTPClient.StatusCode = 204
				itUploadsABlob()
			})
		})

		Context("when the http request fails", func() {
			BeforeEach(func() {
				fakeHTTPClient.Error = errors.New("EOF")
			})

			It("returns err", func() {
				body := ioutil.NopCloser(strings.NewReader("content"))
				err := client.Put("/", body, int64(7))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Putting dav blob /: EOF"))
				Expect(len(fakeHTTPClient.Requests)).To(Equal(3))
			})
		})

		Context("when the http response code is not 201 or 204", func() {
			BeforeEach(func() {
				fakeHTTPClient.StatusCode = 300
				fakeHTTPClient.SetMessage("response")
			})

			It("returns err", func() {
				body := ioutil.NopCloser(strings.NewReader("content"))
				err := client.Put("/", body, int64(7))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Putting dav blob /: Request failed, response: Response{ StatusCode: 300, Status: '' }"))
			})
		})
	})

	Describe("retryable count is configurable", func() {
		BeforeEach(func() {
			fakeHTTPClient.Error = errors.New("EOF")
			config = davconf.Config{RetryAttempts: 7}
			client = NewClient(config, fakeHTTPClient, logger)
		})

		It("tries the specified number of times", func() {
			body := ioutil.NopCloser(strings.NewReader("content"))
			err := client.Put("/", body, int64(7))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Putting dav blob /: EOF"))
			Expect(len(fakeHTTPClient.Requests)).To(Equal(7))
		})

	})
})
