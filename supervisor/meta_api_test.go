package supervisor_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/starkandwayne/shield/supervisor"
)

var _ = Describe("/v1/meta API", func() {
	var API http.Handler

	BeforeEach(func() {
		API = MetaAPI{
			PrivateKeyFile: "test/ssh/id_rsa",
		}
	})

	It("should respond with the public key", func() {
		res := GET(API, "/v1/meta/pubkey")
		Ω(res.Code).Should(Equal(200))
		// note: comment is ignored.
		Ω(res.Body.String()).Should(Equal(
			"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDDVvaJ7VrZAN9fn0SYqNAo2qR4rcjy1VxPcchN/TdUGo1OZn4TLILkwUb1gUQzgOkSQSZ26+mm92xT8/b0BYxLZSxL0gq1vx9pZ9/Umf9+8ESk+ZXnIjQwIhnqCcAalOVOiRFLdc6qPzkGmX0050BMgVai1pIqSdLiB1RqaCuYqy+xKEST+OHl91Y1MMQ0UaqFHIsxzQKoklqbgaHVmPGcK2Jo8uvYFeDOU68LxFRP/shj18y74ph9Dz0SnOrBOqZR+NgOxulLBV9+mCqfp7GrAG3ajRK3YoTnM/I7XBJNix/dloMeqROrbTtnvQK5qedLHao56RInH8zG2pCq02Ar\n",
		))
	})

	It("ignores other HTTP methods", func() {
		for _, method := range []string{"PUT", "DELETE", "PATCH", "OPTIONS", "TRACE"} {
			NotImplemented(API, method, "/v1/meta/pubkey", nil)
		}

		for _, method := range []string{"GET", "HEAD", "POST", "PATCH", "OPTIONS", "TRACE"} {
			NotImplemented(API, method, "/v1/meta/nything/else", nil)
		}
	})

})
