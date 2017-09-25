package handler_test

import (
	"github.com/cloudfoundry/bosh-agent/handler"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
	"net/http/httptest"
)

var _ = Describe("CommonEventFormat", func() {
	cef := handler.NewCommonEventFormat()

	Context("when incoming request is an http request", func() {
		var request *http.Request

		BeforeEach(func() {
			request = httptest.NewRequest("GET", "https://user:pass@127.0.0.1:6900/blobs", nil)
			request.SetBasicAuth("username", "password")
			request.Header.Set("HTTP_HOST", "host.example.com")
			request.Header.Set("HTTP_X_REAL_IP", "12.12.34.56")
			request.Header.Set("HTTP_X_FORWARDED_FOR", "forward")
			request.Header.Set("HTTP_X_FORWARDED_PROTO", "proto")
			request.Header.Set("HTTP_USER_AGENT", "my.agent")
		})

		It("should produce CEF string", func() {
			cefLog, err := cef.ProduceHTTPRequestEventLog(request, 201, "{}")

			Expect(err).NotTo(HaveOccurred())
			Expect(cefLog).To(ContainSubstring("CEF:0|CloudFoundry|BOSH|1|agent_api|/blobs|1|duser=username requestMethod=GET"))
			Expect(cefLog).To(ContainSubstring("src="))
			Expect(cefLog).To(ContainSubstring("spt="))
			Expect(cefLog).To(ContainSubstring("shost"))
			Expect(cefLog).To(ContainSubstring("cs1=HOST=host.example.com&X_REAL_IP=12.12.34.56&X_FORWARDED_FOR=forward&X_FORWARDED_PROTO=proto&USER_AGENT=my.agent cs1Label=httpHeaders"))
			Expect(cefLog).To(ContainSubstring("cs2=basic cs2Label=authType cs3=201 cs3Label=responseStatus"))
			Expect(cefLog).NotTo(ContainSubstring("cs4Label=statusReason"))
		})

		Context("when responding with an error", func() {
			It("should produce CEF string with severity=7 and statusReason", func() {
				cefLog, err := cef.ProduceHTTPRequestEventLog(request, 400, `{"reason": "no reason"}`)

				Expect(err).NotTo(HaveOccurred())
				Expect(cefLog).To(ContainSubstring("CEF:0|CloudFoundry|BOSH|1|agent_api|/blobs|7|duser=username requestMethod=GET"))
				Expect(cefLog).To(ContainSubstring("cs1=HOST=host.example.com&X_REAL_IP=12.12.34.56&X_FORWARDED_FOR=forward&X_FORWARDED_PROTO=proto&USER_AGENT=my.agent cs1Label=httpHeaders"))
				Expect(cefLog).To(ContainSubstring(`cs2=basic cs2Label=authType cs3=400 cs3Label=responseStatus cs4={"reason": "no reason"} cs4Label=statusReason`))

			})
		})
	})

	Context("when incoming request is a NATs request", func() {
		It("should produce CEF string", func() {
			cefLog, err := cef.ProduceNATSRequestEventLog("12.12.56.78", "56734", "nats_user", "get_task", 1, "agent.agent-id", "")

			Expect(err).NotTo(HaveOccurred())
			Expect(cefLog).To(ContainSubstring("CEF:0|CloudFoundry|BOSH|1|agent_api|get_task|1|duser=nats_user"))
			Expect(cefLog).To(ContainSubstring("src="))
			Expect(cefLog).To(ContainSubstring("spt="))
			Expect(cefLog).To(ContainSubstring("shost"))
			Expect(cefLog).NotTo(ContainSubstring("cs1Label=statusReason"))
		})

		Context("when responding with an error", func() {
			It("should produce CEF string with severity=7 and statusReason", func() {
				cefLog, err := cef.ProduceNATSRequestEventLog("12.12.56.78", "56734", "director.director-id", "get_task", 7, "agent.agent-id", `{"reason": "no reason"}`)

				Expect(err).NotTo(HaveOccurred())
				Expect(cefLog).To(ContainSubstring("CEF:0|CloudFoundry|BOSH|1|agent_api|get_task|7|duser=director.director-id"))
				Expect(cefLog).To(ContainSubstring(`cs1={"reason": "no reason"} cs1Label=statusReason`))
			})
		})
	})
})
