package http_test

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	. "github.com/cloudfoundry/bosh-utils/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	fakehttp "github.com/cloudfoundry/bosh-utils/http/fakes"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var _ = Describe("RequestRetryable", func() {
	Describe("Attempt", func() {
		var (
			requestRetryable RequestRetryable
			request          *http.Request
			fakeClient       *fakehttp.FakeClient
		)

		BeforeEach(func() {
			fakeClient = fakehttp.NewFakeClient()
			logger := boshlog.NewLogger(boshlog.LevelNone)

			request = &http.Request{
				Body: ioutil.NopCloser(strings.NewReader("fake-request-body")),
			}

			requestRetryable = NewRequestRetryable(request, fakeClient, logger)
		})

		It("calls Do on the delegate", func() {
			fakeClient.SetMessage("fake-response-body")
			fakeClient.StatusCode = 200

			_, err := requestRetryable.Attempt()
			Expect(err).ToNot(HaveOccurred())

			resp := requestRetryable.Response()
			Expect(readString(resp.Body)).To(Equal("fake-response-body"))
			Expect(resp.StatusCode).To(Equal(200))

			Expect(fakeClient.CallCount).To(Equal(1))
			Expect(fakeClient.Requests).To(ContainElement(request))
		})

		Context("when request returns an error", func() {
			BeforeEach(func() {
				fakeClient.Error = errors.New("fake-response-error")
			})

			It("is retryable", func() {
				isRetryable, err := requestRetryable.Attempt()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-response-error"))
				Expect(isRetryable).To(BeTrue())
			})
		})

		Context("when response status code is not between 200 and 300", func() {
			BeforeEach(func() {
				fakeClient.SetMessage("fake-response-body")
				fakeClient.StatusCode = 404
			})

			It("is retryable", func() {
				isRetryable, err := requestRetryable.Attempt()
				Expect(err).To(HaveOccurred())
				Expect(isRetryable).To(BeTrue())

				resp := requestRetryable.Response()
				Expect(readString(resp.Body)).To(Equal("fake-response-body"))
				Expect(resp.StatusCode).To(Equal(404))
			})

			It("re-populates the request body on subsequent attempts", func() {
				_, err := requestRetryable.Attempt()
				Expect(err).To(HaveOccurred())

				_, err = requestRetryable.Attempt()
				Expect(err).To(HaveOccurred())

				resp := requestRetryable.Response()
				Expect(readString(resp.Body)).To(Equal("fake-response-body"))
				Expect(resp.StatusCode).To(Equal(404))

				Expect(fakeClient.RequestBodies[0]).To(Equal("fake-request-body"))
				Expect(fakeClient.RequestBodies[1]).To(Equal("fake-request-body"))
			})

			It("closes the previous response body on subsequent attempts", func() {
				type ClosedChecker interface {
					io.ReadCloser
					Closed() bool
				}
				_, err := requestRetryable.Attempt()
				Expect(err).To(HaveOccurred())
				originalResp := requestRetryable.Response()
				Expect(originalResp.Body.(ClosedChecker).Closed()).To(BeFalse())

				_, err = requestRetryable.Attempt()
				Expect(err).To(HaveOccurred())
				Expect(originalResp.Body.(ClosedChecker).Closed()).To(BeTrue())
				Expect(requestRetryable.Response().Body.(ClosedChecker).Closed()).To(BeFalse())
			})

			It("fully reads the previous response body on subsequent attempts", func() {
				// go1.5+ fails the next request with `request canceled` if you do not fully read the
				// prior requests body; ref https://marc.ttias.be/golang-nuts/2016-02/msg00256.php
				type readLengthCloser interface {
					ReadLength() int
				}

				_, err := requestRetryable.Attempt()
				Expect(err).To(HaveOccurred())
				originalRespBody := requestRetryable.Response().Body.(readLengthCloser)
				Expect(originalRespBody.ReadLength()).To(Equal(0))

				_, err = requestRetryable.Attempt()
				Expect(err).To(HaveOccurred())
				Expect(originalRespBody.ReadLength()).To(Equal(18))
				Expect(requestRetryable.Response().Body.(readLengthCloser).ReadLength()).To(Equal(0))
			})
		})
	})
})

func readString(body io.ReadCloser) string {
	content, err := ReadAndClose(body)
	Expect(err).ToNot(HaveOccurred())
	return string(content)
}
