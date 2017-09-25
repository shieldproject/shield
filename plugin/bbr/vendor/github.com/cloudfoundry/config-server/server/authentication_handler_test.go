package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/cloudfoundry/config-server/server"
	. "github.com/cloudfoundry/config-server/server/serverfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AuthenticationHandler", func() {

	var mockTokenValidator *FakeTokenValidator
	var mockNextHandler *FakeHandler
	var authHandler http.Handler

	BeforeEach(func() {
		mockTokenValidator = &FakeTokenValidator{}
		mockNextHandler = &FakeHandler{}
		authHandler = NewAuthenticationHandler(mockTokenValidator, mockNextHandler)
	})

	It("should forward request to next handler if token is valid", func() {
		mockTokenValidator.ValidateReturns(nil)

		req, _ := http.NewRequest("PUT", "/v1/data/bla", strings.NewReader("{\"value\":\"blabla\"}"))
		req.Header.Set("Authorization", "bearer fake-auth-header")

		recorder := httptest.NewRecorder()
		authHandler.ServeHTTP(recorder, req)

		Expect(mockNextHandler.ServeHTTPCallCount()).To(Equal(1))

		capturedResWriter, capturedReq := mockNextHandler.ServeHTTPArgsForCall(0)
		Expect(capturedResWriter).To(Equal(recorder))
		Expect(capturedReq).To(Equal(req))
	})

	It("should return 401 Unauthorized if token is missing from request header", func() {
		req, _ := http.NewRequest("PUT", "/v1/data/bla", strings.NewReader("{\"value\":\"blabla\"}"))

		recorder := httptest.NewRecorder()
		authHandler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
	})

	It("should return 401 Unauthorized if token with invalid format is sent", func() {
		req, _ := http.NewRequest("PUT", "/v1/data/bla", strings.NewReader("{\"value\":\"blabla\"}"))
		req.Header.Set("Authorization", "bearer fake-auth-header extra-text")

		recorder1 := httptest.NewRecorder()
		authHandler.ServeHTTP(recorder1, req)
		Expect(recorder1.Code).To(Equal(http.StatusUnauthorized))

		req.Header.Set("Authorization", "bad-prefix fake-auth-header")

		recorder2 := httptest.NewRecorder()
		authHandler.ServeHTTP(recorder2, req)
		Expect(recorder2.Code).To(Equal(http.StatusUnauthorized))
	})

})
