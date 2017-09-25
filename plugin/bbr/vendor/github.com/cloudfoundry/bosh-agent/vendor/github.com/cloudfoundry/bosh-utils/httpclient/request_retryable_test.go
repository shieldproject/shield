package httpclient_test

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/cloudfoundry/bosh-utils/httpclient"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

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
			server           *ghttp.Server
			requestRetryable *httpclient.RequestRetryable
			request          *http.Request
			logger           boshlog.Logger
		)

		BeforeEach(func() {
			server = ghttp.NewServer()
			logger = boshlog.NewLogger(boshlog.LevelNone)

			var err error
			request, err = http.NewRequest("GET", server.URL(), ioutil.NopCloser(strings.NewReader("fake-request-body")))
			Expect(err).NotTo(HaveOccurred())

			requestRetryable = httpclient.NewRequestRetryable(request, httpclient.DefaultClient, logger, nil)
		})

		It("sends a request to the server", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/"),
					ghttp.RespondWith(200, "fake-response-body"),
				),
			)

			_, err := requestRetryable.Attempt()
			Expect(err).ToNot(HaveOccurred())

			resp := requestRetryable.Response()
			Expect(readString(resp.Body)).To(Equal("fake-response-body"))
			Expect(resp.StatusCode).To(Equal(200))

			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		Context("when request returns an error", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/"),
						ghttp.RespondWith(503, "fake-response-error"),
					),
				)
			})

			It("is retryable", func() {
				isRetryable, err := requestRetryable.Attempt()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("503 Service Unavailable")))
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
				request.Body = seekableReaderCloser
				requestRetryable = httpclient.NewRequestRetryable(request, httpclient.DefaultClient, logger, nil)
			})

			Context("when the response status code is success", func() {
				BeforeEach(func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/"),
							ghttp.VerifyBody([]byte("hello from seekable")),
							ghttp.RespondWith(200, "fake-response-body"),
						),
					)
				})

				// It does not consume the whole body and store it in memory for future re-attempts, it seeks to the
				// beginning of the body instead
				It("seeks to the beginning of the request body uses the request body *as is*", func() {
					_, err := requestRetryable.Attempt()
					Expect(err).ToNot(HaveOccurred())
					Expect(seekableReaderCloser.Seeked).To(BeTrue())
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
					request.Body = seekableReaderCloser

					server.AppendHandlers(ghttp.VerifyRequest("GET", "/"))

					errOnResponseAttemptable := func(*http.Response, error) (bool, error) {
						return false, errors.New("fake-error")
					}
					requestRetryable = httpclient.NewRequestRetryable(request, httpclient.DefaultClient, logger, errOnResponseAttemptable)
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
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/"),
							ghttp.RespondWith(404, "fake-response-body"),
						),
					)
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
						server.RouteToHandler("GET", "/",
							ghttp.CombineHandlers(
								ghttp.VerifyBody([]byte("hello from seekable")),
								ghttp.RespondWith(200, "fake-response-body"),
							),
						)
						seekableReaderCloser.Seeked = false
						_, err = requestRetryable.Attempt()
					})

					It("seeks back to the beginning and on the original request body", func() {
						Expect(err).ToNot(HaveOccurred())

						Expect(seekableReaderCloser.Seeked).To(BeTrue())
						Expect(server.ReceivedRequests()).To(HaveLen(2))

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
				server.RouteToHandler("GET", "/",
					ghttp.CombineHandlers(
						ghttp.VerifyBody([]byte("fake-request-body")),
						ghttp.RespondWith(404, "fake-response-body"),
					),
				)
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

				Expect(server.ReceivedRequests()).To(HaveLen(2))
			})

			It("closes the previous response body on subsequent attempts", func() {
				_, err := requestRetryable.Attempt()
				Expect(err).To(HaveOccurred())
				originalRespBody := requestRetryable.Response().Body

				_, err = requestRetryable.Attempt()
				Expect(err).To(HaveOccurred())

				_, err = originalRespBody.Read(nil)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("http: read on closed response body"))
			})

			It("fully reads the previous response body on subsequent attempts", func() {
				// go1.5+ fails the next request with `request canceled` if you do not fully read the
				// prior requests body; ref https://marc.ttias.be/golang-nuts/2016-02/msg00256.php
				// This should not be necessary when the following CL gets accepted:
				// https://go-review.googlesource.com/c/go/+/62891
				_, err := requestRetryable.Attempt()
				Expect(err).To(HaveOccurred())

				_, err = requestRetryable.Attempt()
				Expect(err).To(HaveOccurred())

				Expect(err).NotTo(MatchError("request canceled"))
				// we expect to see 404 here because we don't want to see request
				// canceled, this is to avoid having a false positive if messages
				// change in the future
				Expect(err).To(MatchError(ContainSubstring("404 Not Found")))
			})
		})
	})
})

func readString(body io.ReadCloser) string {
	defer body.Close()
	content, err := ioutil.ReadAll(body)
	Expect(err).ToNot(HaveOccurred())
	return string(content)
}
