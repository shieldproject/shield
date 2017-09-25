package http_test

import (
	"net/http"

	fakehttp "github.com/cloudfoundry/bosh-utils/http/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-utils/http"
)

var _ = Describe("RetryClient", func() {
	Describe("Do", func() {
		var (
			retryClient Client
			maxAttempts int
			fakeClient  *fakehttp.FakeClient
		)

		BeforeEach(func() {
			fakeClient = fakehttp.NewFakeClient()
			logger := boshlog.NewLogger(boshlog.LevelNone)
			maxAttempts = 7

			retryClient = NewRetryClient(fakeClient, uint(maxAttempts), 0, logger)
		})

		It("returns response from retryable request", func() {
			fakeClient.SetMessage("fake-response-body")
			fakeClient.StatusCode = 204

			req := &http.Request{}
			resp, err := retryClient.Do(req)
			Expect(err).ToNot(HaveOccurred())

			Expect(readString(resp.Body)).To(Equal("fake-response-body"))
			Expect(resp.StatusCode).To(Equal(204))
		})

		It("attemps once if request is successful", func() {
			fakeClient.StatusCode = 200

			req := &http.Request{}
			resp, err := retryClient.Do(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			Expect(fakeClient.CallCount).To(Equal(1))
			Expect(fakeClient.Requests).To(ContainElement(req))
		})

		It("retries for maxAttempts if request is failing", func() {
			fakeClient.StatusCode = 404

			req := &http.Request{}
			resp, err := retryClient.Do(req)
			Expect(err).To(HaveOccurred())

			Expect(resp.StatusCode).To(Equal(404))

			Expect(fakeClient.CallCount).To(Equal(maxAttempts))
			Expect(fakeClient.Requests).To(ContainElement(req))
		})
	})
})
