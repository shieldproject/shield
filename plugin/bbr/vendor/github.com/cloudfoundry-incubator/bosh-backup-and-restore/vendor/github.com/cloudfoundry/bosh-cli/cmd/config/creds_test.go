package config_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd/config"
)

var _ = Describe("Creds", func() {
	Describe("IsBasicComplete", func() {
		It("returns true if both username and password are non-empty", func() {
			Expect(Creds{Client: "", ClientSecret: ""}.IsBasicComplete()).To(BeFalse())
			Expect(Creds{Client: "user", ClientSecret: ""}.IsBasicComplete()).To(BeFalse())
			Expect(Creds{Client: "", ClientSecret: "pass"}.IsBasicComplete()).To(BeFalse())
			Expect(Creds{Client: "user", ClientSecret: "pass"}.IsBasicComplete()).To(BeTrue())
		})
	})

	Describe("IsUAAClient", func() {
		It("returns true if client is non-empty", func() {
			Expect(Creds{Client: ""}.IsUAAClient()).To(BeFalse())
			Expect(Creds{Client: "cli"}.IsUAAClient()).To(BeTrue())
		})
	})

	Describe("IsUAA", func() {
		It("returns true if client or refresh token is non-empty", func() {
			Expect(Creds{Client: "foo", RefreshToken: ""}.IsUAA()).To(BeTrue())
			Expect(Creds{Client: "", RefreshToken: "bar"}.IsUAA()).To(BeTrue())
			Expect(Creds{Client: "", RefreshToken: ""}.IsUAA()).To(BeFalse())
		})
	})

	Describe("Description", func() {
		var (
			token = "xxx.eyJhdWQiOlsiYm9zaF9jbGkiLCJvcGVuaWQiLCJib3NoIl0sImNpZCI6ImJvc2hfY2xpIiwiY2xpZW50X2lkIjoiYm9zaF9jbGkiLCJleHAiOjE0NTI1NjI3NTYsImdyYW50X3R5cGUiOiJwYXNzd29yZCIsImlhdCI6MTQ1MjQ3NjM1NiwiaXNzIjoiaHR0cHM6Ly8xMC4yNDQuMy4yOjg0NDMvb2F1dGgvdG9rZW4iLCJqdGkiOiI2N2QyYjcyNS1kZjdkLTRlZjEtYjExYy02YzA0MDliYjYxM2ItciIsIm9yaWdpbiI6InVhYSIsInJldl9zaWciOiI1MmFhZGE2ZCIsInNjb3BlIjpbIm9wZW5pZCIsImJvc2guYWRtaW4iXSwic3ViIjoiOTE2NGRkMmEtZmU1ZS00OTRkLWJmZWUtMWFhM2ZhYTZhNmEyIiwidXNlcl9pZCI6IjkxNjRkZDJhLWZlNWUtNDk0ZC1iZmVlLTFhYTNmYWE2YTZhMiIsInVzZXJfbmFtZSI6ImFkbWluIiwiemlkIjoidWFhIn0.xxx"
		)

		It("returns description", func() {
			Expect(Creds{}.Description()).To(Equal("anonymous user"))
			Expect(Creds{RefreshToken: "token"}.Description()).To(Equal("'?'"))
			Expect(Creds{RefreshToken: token}.Description()).To(Equal("user 'admin' (openid, bosh.admin)"))
			Expect(Creds{Client: "cli"}.Description()).To(Equal("client 'cli'"))
		})
	})
})
