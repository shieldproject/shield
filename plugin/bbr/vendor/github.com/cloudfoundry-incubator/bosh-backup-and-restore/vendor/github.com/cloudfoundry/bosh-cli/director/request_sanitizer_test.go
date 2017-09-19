package director_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/director"
	"net/http"
)

var _ = Describe("RequestSanitizer", func() {
	Describe("SanitizeRequest", func() {
		Context("with authorization", func() {
			It("sanitizes the 'Host' and 'Authorization' fields", func() {
				header := http.Header{}
				header.Add("Location", "/redirect")
				header.Add("Authorization", "Basic foo=")

				req := http.Request{Header: header, Host: "1.2.3.4:25555"}
				requestSanitizer := RequestSanitizer{Request: req}

				req, err := requestSanitizer.SanitizeRequest()

				Expect(err).ToNot(HaveOccurred())
				Expect(req.Host).To(Equal("1.2.3.4:25555"))
				Expect(req.Header["Location"]).To(Equal([]string{"/redirect"}))
				Expect(req.Header["Authorization"]).To(Equal([]string{"[removed]"}))
			})
		})

		Context("without authorization", func() {
			It("doesn't modify request", func() {
				header := http.Header{}
				header.Add("Location", "/redirect")

				req := http.Request{Header: header, Host: "some_host"}
				requestSanitizer := RequestSanitizer{Request: req}

				newReq, err := requestSanitizer.SanitizeRequest()

				Expect(err).ToNot(HaveOccurred())
				Expect(req).To(Equal(newReq))
			})
		})

	})

})
