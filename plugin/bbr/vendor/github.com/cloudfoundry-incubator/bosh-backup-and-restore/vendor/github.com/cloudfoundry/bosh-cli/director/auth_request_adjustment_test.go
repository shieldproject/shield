package director_test

import (
	"errors"
	"net/http"
	gourl "net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/director"
)

var _ = Describe("AuthRequestAdjustment", func() {
	var (
		authFuncCalled bool
		authFunc       func(bool) (string, error)
		username       string
		password       string
		adjustment     AuthRequestAdjustment
	)

	BeforeEach(func() {
		authFuncCalled = false
		authFunc = nil
		username = ""
		password = ""
	})

	JustBeforeEach(func() {
		adjustment = NewAuthRequestAdjustment(authFunc, username, password)
	})

	Describe("NeedsReadjustment", func() {
		It("returns true if resp is unauthorized", func() {
			resp := &http.Response{StatusCode: 401}
			Expect(adjustment.NeedsReadjustment(resp)).To(BeTrue())
		})

		It("returns false if resp is something else", func() {
			resp := &http.Response{StatusCode: 200}
			Expect(adjustment.NeedsReadjustment(resp)).To(BeFalse())
		})
	})

	Describe("Adjust", func() {
		var (
			req *http.Request
		)

		BeforeEach(func() {
			req = &http.Request{
				URL:    &gourl.URL{},
				Header: http.Header(map[string][]string{}),
			}
		})

		Context("when username is set but auth func is not", func() {
			BeforeEach(func() {
				username = "username"
				password = "password"
			})

			It("sets the Authorization header", func() {
				Expect(adjustment.Adjust(req, false)).ToNot(HaveOccurred())
				Expect(req.Header.Get("Authorization")).To(Equal("Basic dXNlcm5hbWU6cGFzc3dvcmQ="))
			})

			It("does not set Userinfo on the URL", func() {
				Expect(adjustment.Adjust(req, false)).ToNot(HaveOccurred())
				Expect(req.URL.User).To(BeNil())
			})
		})

		Context("when username and auth func are set", func() {
			BeforeEach(func() {
				username = "username"
				password = "password"
				authFuncCalled = false
				authFunc = func(retried bool) (string, error) {
					authFuncCalled = true
					return "", nil
				}
			})

			It("sets the Authorization header", func() {
				Expect(adjustment.Adjust(req, false)).ToNot(HaveOccurred())
				Expect(req.Header.Get("Authorization")).To(Equal("Basic dXNlcm5hbWU6cGFzc3dvcmQ="))
			})

			It("does not set Userinfo on the URL", func() {
				Expect(adjustment.Adjust(req, false)).ToNot(HaveOccurred())
				Expect(req.URL.User).To(BeNil())
			})

			It("does not call auth func", func() {
				Expect(adjustment.Adjust(req, false)).ToNot(HaveOccurred())
				Expect(authFuncCalled).To(BeFalse())
			})
		})

		Context("when username is not set but auth func is", func() {
			var (
				authFuncError   error
				authFuncRetried bool
			)

			BeforeEach(func() {
				authFuncCalled = false
				authFuncError = nil
				authFuncRetried = false
				authFunc = func(retried bool) (string, error) {
					authFuncCalled = true
					authFuncRetried = retried
					return "auth-header", authFuncError
				}
			})

			It("does not set basic auth", func() {
				Expect(adjustment.Adjust(req, false)).ToNot(HaveOccurred())
				Expect(req.URL.User).To(BeNil())
			})

			It("sets Authorization header with with auth header", func() {
				Expect(adjustment.Adjust(req, false)).ToNot(HaveOccurred())
				Expect(req.Header.Get("Authorization")).To(Equal("auth-header"))
			})

			It("retrieves auth header with retried flag", func() {
				Expect(adjustment.Adjust(req, true)).ToNot(HaveOccurred())
				Expect(authFuncRetried).To(BeTrue())

				Expect(adjustment.Adjust(req, false)).ToNot(HaveOccurred())
				Expect(authFuncRetried).To(BeFalse())
			})

			It("returns an error if failed to retrieve auth header", func() {
				authFuncError = errors.New("fake-err")

				err := adjustment.Adjust(req, false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))

				Expect(req.Header.Get("Authorization")).To(BeEmpty())
			})
		})

		Context("when username or auth func are not set", func() {
			It("does not set basic auth", func() {
				Expect(adjustment.Adjust(req, false)).ToNot(HaveOccurred())
				Expect(req.URL.User).To(BeNil())
			})

			It("does not set Authorization header", func() {
				Expect(adjustment.Adjust(req, false)).ToNot(HaveOccurred())
				Expect(req.Header.Get("Authorization")).To(BeEmpty())
			})
		})
	})
})
