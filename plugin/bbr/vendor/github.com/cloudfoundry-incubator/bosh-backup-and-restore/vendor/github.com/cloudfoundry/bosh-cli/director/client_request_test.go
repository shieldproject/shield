package director_test

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	boshhttp "github.com/cloudfoundry/bosh-utils/httpclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "github.com/cloudfoundry/bosh-cli/director"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	"github.com/cloudfoundry/bosh-cli/ui"
)

var _ = Describe("ClientRequest", func() {
	var (
		server *ghttp.Server
		resp   []string

		buildReq func(FileReporter) ClientRequest
		req      ClientRequest

		logger   fakedir.Logger
		logCalls []fakedir.LogCallArgs

		locationHeader http.Header
	)

	BeforeEach(func() {
		_, server = BuildServer()
		logCalls = []fakedir.LogCallArgs{}
		logger = fakedir.NewFakeLogger(&logCalls)

		buildReq = func(fileReporter FileReporter) ClientRequest {
			httpTransport := &http.Transport{
				TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
				TLSHandshakeTimeout: 10 * time.Second,
			}

			rawClient := &http.Client{Transport: httpTransport}
			httpClient := boshhttp.NewHTTPClient(rawClient, logger)
			return NewClientRequest(server.URL(), httpClient, fileReporter, logger)
		}

		resp = nil
		req = buildReq(NewNoopFileReporter())

		locationHeader = http.Header{}
		locationHeader.Add("Location", "/redirect")
	})

	AfterEach(func() {
		server.Close()
	})

	successCodes := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusPartialContent,
	}

	Describe("Get", func() {
		act := func() error { return req.Get("/path", &resp) }

		for _, code := range successCodes {
			code := code

			Describe(fmt.Sprintf("'%d' response", code), func() {
				It("makes request, succeeds and unmarshals response", func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/path", ""),
							ghttp.RespondWith(code, `["val"]`),
						),
					)

					err := act()
					Expect(err).ToNot(HaveOccurred())
					Expect(resp).To(Equal([]string{"val"}))
				})

				It("returns error if cannot be unmarshalled", func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/path"),
							ghttp.RespondWith(code, ""),
						),
					)

					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Unmarshaling Director response"))
				})
			})
		}

		Describe("'302' response", func() {
			It("makes request, follows redirect, succeeds and unmarshals response", func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/path", ""),
						ghttp.RespondWith(http.StatusFound, "", locationHeader),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/redirect"),
						ghttp.RespondWith(http.StatusOK, `["val"]`),
					),
				)

				err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(resp).To(Equal([]string{"val"}))
			})

			It("returns error if redirect response cannot be unmarshalled", func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/path"),
						ghttp.RespondWith(http.StatusFound, "", locationHeader),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/redirect"),
						ghttp.RespondWith(http.StatusOK, `-`),
					),
				)

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Unmarshaling Director response"))
			})
		})

		It("returns error if response in non-successful response code", func() {
			AppendBadRequest(ghttp.VerifyRequest("GET", "/path"), server)

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Director responded with non-successful status code"))
		})
	})

	Describe("RawGet", func() {
		BeforeEach(func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/path"),
					ghttp.RespondWith(http.StatusOK, "body"),
				),
			)
		})

		Context("when custom writer is not set", func() {
			It("returns full response body", func() {
				body, resp, err := req.RawGet("/path", nil, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(body).To(Equal([]byte("body")))
				Expect(resp).ToNot(BeNil())
			})

			It("does not track downloading", func() {
				fileReporter := &fakedir.FakeFileReporter{}
				req = buildReq(fileReporter)

				_, _, err := req.RawGet("/path", nil, nil)
				Expect(err).ToNot(HaveOccurred())

				Expect(fileReporter.TrackDownloadCallCount()).To(Equal(0))
			})
		})

		Context("when custom writer is set", func() {
			It("returns response body", func() {
				buf := bytes.NewBufferString("")

				body, resp, err := req.RawGet("/path", buf, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(body).To(BeEmpty())
				Expect(resp).ToNot(BeNil())

				Expect(buf.String()).To(Equal("body"))
			})

			It("tracks downloading based on content length", func() {
				buf := bytes.NewBufferString("")
				otherBuf := bytes.NewBufferString("")

				fileReporter := &fakedir.FakeFileReporter{
					TrackDownloadStub: func(size int64, out io.Writer) io.Writer {
						Expect(size).To(Equal(int64(4)))
						Expect(out).To(Equal(buf))
						return otherBuf
					},
				}

				req = buildReq(fileReporter)

				_, _, err := req.RawGet("/path", buf, nil)
				Expect(err).ToNot(HaveOccurred())

				Expect(otherBuf.String()).To(Equal("body"))
			})

			Context("when context id is not set", func() {
				It("does not set a X-Bosh-Context-Id header", func() {
					verifyContextIdNotSet := func(_ http.ResponseWriter, req *http.Request) {
						_, found := req.Header["X-Bosh-Context-Id"]
						Expect(found).To(BeFalse())
					}

					server.WrapHandler(0, verifyContextIdNotSet)

					_, _, err := req.RawGet("/path", nil, nil)
					Expect(err).ToNot(HaveOccurred())
				})
			})

			Context("when context id set", func() {
				It("does set a X-Bosh-Context-Id header", func() {
					contextId := "example-context-id"
					req = req.WithContext(contextId)
					server.WrapHandler(0, ghttp.VerifyHeaderKV("X-Bosh-Context-Id", contextId))
					_, _, err := req.RawGet("/path", nil, nil)
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})

		Describe("Request logging", func() {
			It("Sanitizes requests for logging", func() {

				_, resp, err := req.RawGet("/path", nil, func(r *http.Request) {
					r.Header.Add("Authorization", "basic=")
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())

				host := resp.Request.Host
				expectedLogCallArgs := fakedir.LogCallArgs{
					LogLevel: "Debug",
					Tag:      "director.clientRequest",
					Msg:      "Dumping Director client request:\n%s",
					Args: []string{
						fmt.Sprintf("GET /path HTTP/1.1\r\nHost: %s\r\nAuthorization: [removed]\r\n\r\n", host),
					},
				}
				actualLogCallArgs := (*logger.LogCallArgs)[1]

				Expect(expectedLogCallArgs.LogLevel).To(Equal(actualLogCallArgs.LogLevel))
				Expect(expectedLogCallArgs.Tag).To(Equal(actualLogCallArgs.Tag))
				Expect(expectedLogCallArgs.Msg).To(Equal(actualLogCallArgs.Msg))
				Expect(expectedLogCallArgs.Args[0]).To(Equal(actualLogCallArgs.Args[0]))
			})
		})
	})

	Describe("Post", func() {
		act := func() error { return req.Post("/path", []byte("req-body"), nil, &resp) }

		for _, code := range successCodes {
			code := code

			Describe(fmt.Sprintf("'%d' response", code), func() {
				It("makes request, succeeds and unmarshals response", func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("POST", "/path", ""),
							ghttp.VerifyBody([]byte("req-body")),
							ghttp.RespondWith(code, `["val"]`),
						),
					)

					err := act()
					Expect(err).ToNot(HaveOccurred())
					Expect(resp).To(Equal([]string{"val"}))
				})

				It("returns error if cannot be unmarshalled", func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("POST", "/path"),
							ghttp.VerifyBody([]byte("req-body")),
							ghttp.RespondWith(code, ""),
						),
					)

					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Unmarshaling Director response"))
				})
			})
		}

		Describe("'302' response", func() {
			It("makes request, follows redirect, succeeds and unmarshals response", func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/path", ""),
						ghttp.VerifyBody([]byte("req-body")),
						ghttp.RespondWith(http.StatusFound, "", locationHeader),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/redirect"),
						ghttp.RespondWith(http.StatusOK, `["val"]`),
					),
				)

				err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(resp).To(Equal([]string{"val"}))
			})

			It("returns error if redirect response cannot be unmarshalled", func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/path"),
						ghttp.RespondWith(http.StatusFound, "", locationHeader),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/redirect"),
						ghttp.RespondWith(http.StatusOK, `-`),
					),
				)

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Unmarshaling Director response"))
			})
		})

		It("returns error if response in non-successful response code", func() {
			AppendBadRequest(ghttp.VerifyRequest("POST", "/path"), server)

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Director responded with non-successful status code"))
		})
	})

	Describe("RawPost", func() {
		Context("when request body is 'application/x-compressed'", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/path"),
						ghttp.VerifyBody([]byte("req-body")),
						ghttp.VerifyHeader(http.Header{"Content-Type": []string{"application/x-compressed"}}),
						ghttp.RespondWith(http.StatusOK, "body"),
					),
				)
			})

			setHeaders := func(req *http.Request) {
				req.Header.Add("Content-Type", "application/x-compressed")
				req.Body = ioutil.NopCloser(bytes.NewBufferString("req-body"))
				req.ContentLength = 8
			}

			It("uploads request body and returns response", func() {
				body, resp, err := req.RawPost("/path", nil, setHeaders)
				Expect(err).ToNot(HaveOccurred())
				Expect(body).To(Equal([]byte("body")))
				Expect(resp).ToNot(BeNil())
			})

			It("tracks uploading", func() {
				fileReporter := &fakedir.FakeFileReporter{
					TrackUploadStub: func(size int64, reader io.ReadCloser) ui.ReadSeekCloser {
						Expect(size).To(Equal(int64(8)))
						Expect(ioutil.ReadAll(reader)).To(Equal([]byte("req-body")))
						return NoopReadSeekCloser{ioutil.NopCloser(bytes.NewBufferString("req-body"))}
					},
				}
				req = buildReq(fileReporter)

				_, _, err := req.RawPost("/path", nil, setHeaders)
				Expect(err).ToNot(HaveOccurred())

				Expect(fileReporter.TrackUploadCallCount()).To(Equal(1))
			})
		})

		Context("when request body is not 'application/x-compressed'", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/path"),
						ghttp.VerifyBody([]byte("req-body")),
						ghttp.VerifyHeader(http.Header{"Content-Type": []string{"application/json"}}),
						ghttp.RespondWith(http.StatusOK, "body"),
					),
				)
			})

			setHeaders := func(req *http.Request) {
				req.Header.Add("Content-Type", "application/json")
				req.Body = ioutil.NopCloser(bytes.NewBufferString("req-body"))
				req.ContentLength = 8
			}

			It("uploads request body and returns response", func() {
				body, resp, err := req.RawPost("/path", nil, setHeaders)
				Expect(err).ToNot(HaveOccurred())
				Expect(body).To(Equal([]byte("body")))
				Expect(resp).ToNot(BeNil())
			})

			It("does not track uploading", func() {
				fileReporter := &fakedir.FakeFileReporter{}
				req = buildReq(fileReporter)

				_, _, err := req.RawPost("/path", nil, setHeaders)
				Expect(err).ToNot(HaveOccurred())

				Expect(fileReporter.TrackUploadCallCount()).To(Equal(0))
			})
		})

		Context("when context id is not set", func() {
			It("does not set a X-Bosh-Context-Id header", func() {
				verifyContextIdNotSet := func(_ http.ResponseWriter, req *http.Request) {
					_, found := req.Header["X-Bosh-Context-Id"]
					Expect(found).To(BeFalse())
				}

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/path"),
						verifyContextIdNotSet,
						ghttp.RespondWith(http.StatusOK, "body"),
					),
				)

				_, _, err := req.RawPost("/path", nil, nil)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when context id set", func() {
			It("does set a X-Bosh-Context-Id header", func() {
				contextId := "example-context-id"
				req = req.WithContext(contextId)
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/path"),
						ghttp.VerifyHeaderKV("X-Bosh-Context-Id", contextId),
						ghttp.RespondWith(http.StatusOK, "body"),
					),
				)
				_, _, err := req.RawPost("/path", nil, nil)
				Expect(err).ToNot(HaveOccurred())
			})
		})

	})

	Describe("Put", func() {
		act := func() error { return req.Put("/path", []byte("req-body"), nil, &resp) }

		for _, code := range successCodes {
			code := code

			Describe(fmt.Sprintf("'%d' response", code), func() {
				It("makes request, succeeds and unmarshals response", func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("PUT", "/path", ""),
							ghttp.VerifyBody([]byte("req-body")),
							ghttp.RespondWith(code, `["val"]`),
						),
					)

					err := act()
					Expect(err).ToNot(HaveOccurred())
					Expect(resp).To(Equal([]string{"val"}))
				})

				It("returns error if cannot be unmarshalled", func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("PUT", "/path"),
							ghttp.VerifyBody([]byte("req-body")),
							ghttp.RespondWith(code, ""),
						),
					)

					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Unmarshaling Director response"))
				})

				Context("when context id is not set", func() {
					verifyContextIdNotSet := func(_ http.ResponseWriter, req *http.Request) {
						_, found := req.Header["X-Bosh-Context-Id"]
						Expect(found).To(BeFalse())
					}

					It("does not set a X-Bosh-Context-Id header", func() {
						server.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.VerifyRequest("PUT", "/path"),
								ghttp.VerifyBody([]byte("req-body")),
								verifyContextIdNotSet,
								ghttp.RespondWith(code, `["val"]`),
							),
						)

						err := act()
						Expect(err).ToNot(HaveOccurred())
					})
				})

				Context("when context id set", func() {
					contextId := "example-context-id"
					BeforeEach(func() {
						req = req.WithContext(contextId)
					})

					It("makes request with correct header", func() {
						server.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.VerifyRequest("PUT", "/path"),
								ghttp.VerifyBody([]byte("req-body")),
								ghttp.VerifyHeaderKV("X-Bosh-Context-Id", contextId),
								ghttp.RespondWith(code, `["val"]`),
							),
						)

						err := act()
						Expect(err).ToNot(HaveOccurred())
					})
				})
			})
		}

		Describe("'302' response", func() {
			It("makes request, follows redirect, succeeds and unmarshals response", func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("PUT", "/path", ""),
						ghttp.VerifyBody([]byte("req-body")),
						ghttp.RespondWith(http.StatusFound, "", locationHeader),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/redirect"),
						ghttp.RespondWith(http.StatusOK, `["val"]`),
					),
				)

				err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(resp).To(Equal([]string{"val"}))
			})

			It("returns error if redirect response cannot be unmarshalled", func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("PUT", "/path"),
						ghttp.RespondWith(http.StatusFound, "", locationHeader),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/redirect"),
						ghttp.RespondWith(http.StatusOK, `-`),
					),
				)

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Unmarshaling Director response"))
			})
		})

		It("returns error if response in non-successful response code", func() {
			AppendBadRequest(ghttp.VerifyRequest("PUT", "/path"), server)

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Director responded with non-successful status code"))
		})
	})

	Describe("Delete", func() {
		act := func() error { return req.Delete("/path", &resp) }

		for _, code := range successCodes {
			code := code

			Describe(fmt.Sprintf("'%d' response", code), func() {
				It("makes request, succeeds and unmarshals response", func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("DELETE", "/path", ""),
							ghttp.VerifyBody([]byte("")),
							ghttp.RespondWith(code, `["val"]`),
						),
					)

					err := act()
					Expect(err).ToNot(HaveOccurred())
					Expect(resp).To(Equal([]string{"val"}))
				})

				It("returns error if cannot be unmarshalled", func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("DELETE", "/path"),
							ghttp.VerifyBody([]byte("")),
							ghttp.RespondWith(code, ""),
						),
					)

					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Unmarshaling Director response"))
				})
			})
		}

		Describe("'302' response", func() {
			It("makes request, follows redirect, succeeds and unmarshals response", func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("DELETE", "/path", ""),
						ghttp.VerifyBody([]byte("")),
						ghttp.RespondWith(http.StatusFound, "", locationHeader),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/redirect"),
						ghttp.RespondWith(http.StatusOK, `["val"]`),
					),
				)

				err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(resp).To(Equal([]string{"val"}))
			})

			It("returns error if redirect response cannot be unmarshalled", func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("DELETE", "/path"),
						ghttp.RespondWith(http.StatusFound, "", locationHeader),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/redirect"),
						ghttp.RespondWith(http.StatusOK, `-`),
					),
				)

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Unmarshaling Director response"))
			})
		})

		It("returns error if response in non-successful response code", func() {
			AppendBadRequest(ghttp.VerifyRequest("DELETE", "/path"), server)

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Director responded with non-successful status code"))
		})
	})
})
