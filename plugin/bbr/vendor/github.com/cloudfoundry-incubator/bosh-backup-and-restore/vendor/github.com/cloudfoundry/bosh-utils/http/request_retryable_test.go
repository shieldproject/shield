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

	"bytes"
	"os"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type seekableReadClose struct {
	Seeked     bool
	closed     bool
	content    []byte
	readCloser io.ReadCloser
}

func NewSeekableReadClose(content []byte) *seekableReadClose {
	return &seekableReadClose{
		Seeked:     false,
		content:    content,
		readCloser: ioutil.NopCloser(bytes.NewReader(content)),
	}
}

func (s *seekableReadClose) Seek(offset int64, whence int) (ret int64, err error) {
	s.readCloser = ioutil.NopCloser(bytes.NewReader(s.content))
	s.Seeked = true
	return 0, nil
}

func (s *seekableReadClose) Read(p []byte) (n int, err error) {
	return s.readCloser.Read(p)
}

func (s *seekableReadClose) Close() error {
	if s.closed {
		return errors.New("Can not close twice")
	}

	s.closed = true
	return nil
}

var _ = Describe("RequestRetryable", func() {
	Describe("Attempt", func() {
		var (
			requestRetryable RequestRetryable
			request          *http.Request
			fakeClient       *fakehttp.FakeClient
			logger           boshlog.Logger
		)

		BeforeEach(func() {
			fakeClient = fakehttp.NewFakeClient()
			logger = boshlog.NewLogger(boshlog.LevelNone)

			request = &http.Request{
				Body: ioutil.NopCloser(strings.NewReader("fake-request-body")),
			}

			requestRetryable = NewRequestRetryable(request, fakeClient, logger, nil)
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

		Context("when the request body has a seek method", func() {
			var (
				seekableReaderCloser *seekableReadClose
			)

			It("os.File conforms to the Seekable interface", func() {
				var seekable io.ReadSeeker
				seekable, err := ioutil.TempFile(os.TempDir(), "seekable")
				Expect(err).ToNot(HaveOccurred())
				_, err = seekable.Seek(0, 0)
				Expect(err).ToNot(HaveOccurred())
			})

			BeforeEach(func() {
				seekableReaderCloser = NewSeekableReadClose([]byte("hello from seekable"))
				request = &http.Request{
					Body: seekableReaderCloser,
				}
				requestRetryable = NewRequestRetryable(request, fakeClient, logger, nil)
			})

			Context("when the response status code is success", func() {
				BeforeEach(func() {
					fakeClient.SetMessage("fake-response-body")
					fakeClient.StatusCode = 200
				})

				// It does not consume the whole body and store it in memory for future re-attempts, it seeks to the
				// beginning of the body instead
				It("seeks to the beginning of the request body uses the request body *as is*", func() {
					_, err := requestRetryable.Attempt()
					Expect(err).ToNot(HaveOccurred())
					Expect(seekableReaderCloser.Seeked).To(BeTrue())
					Expect(fakeClient.RequestBodies[0]).To(Equal("hello from seekable"))
				})

				It("closes file handles", func() {
					_, err := requestRetryable.Attempt()
					Expect(err).ToNot(HaveOccurred())
					Expect(seekableReaderCloser.closed).To(BeTrue())
				})
			})

			Context("when it returns an error checking if response can be attempted again", func() {
				BeforeEach(func() {
					seekableReaderCloser = NewSeekableReadClose([]byte("hello from seekable"))
					request = &http.Request{
						Body: seekableReaderCloser,
					}

					errOnResponseAttemptable := func(*http.Response, error) (bool, error) {
						return false, errors.New("fake-error")
					}
					requestRetryable = NewRequestRetryable(request, fakeClient, logger, errOnResponseAttemptable)
				})

				It("still closes the request body", func() {
					_, err := requestRetryable.Attempt()
					Expect(err).To(HaveOccurred())
					Expect(seekableReaderCloser.closed).To(BeTrue())
				})
			})

			Context("when the response status code is not between 200 and 300", func() {
				var (
					isRetryable bool
					err         error
				)
				BeforeEach(func() {
					fakeClient.SetMessage("fake-response-body")
					fakeClient.StatusCode = 404
					isRetryable, err = requestRetryable.Attempt()
				})

				It("is retryable", func() {
					Expect(err).To(HaveOccurred())
					Expect(isRetryable).To(BeTrue())

					resp := requestRetryable.Response()
					Expect(readString(resp.Body)).To(Equal("fake-response-body"))
					Expect(resp.StatusCode).To(Equal(404))
				})

				Context("when making another, successful, attempt", func() {
					BeforeEach(func() {
						fakeClient.SetMessage("fake-response-body")
						fakeClient.StatusCode = 200
						seekableReaderCloser.Seeked = false
						_, err = requestRetryable.Attempt()
					})

					It("seeks back to the beginning and on the original request body", func() {
						Expect(err).ToNot(HaveOccurred())

						Expect(seekableReaderCloser.Seeked).To(BeTrue())
						Expect(fakeClient.RequestBodies[1]).To(Equal("hello from seekable"))

						resp := requestRetryable.Response()
						Expect(resp.StatusCode).To(Equal(200))
						Expect(readString(resp.Body)).To(Equal("fake-response-body"))
					})

					It("closes file handles", func() {
						Expect(err).ToNot(HaveOccurred())
						Expect(seekableReaderCloser.closed).To(BeTrue())
					})
				})
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
