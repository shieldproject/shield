package supervisor_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/starkandwayne/shield/supervisor"
)

var _ = Describe("Supervisor Configuration", func() {
	Describe("Configuration", func() {
		var s *Supervisor

		BeforeEach(func() {
			s = NewSupervisor()
			Ω(s).ShouldNot(BeNil())
		})

		It("handles missing files", func() {
			Ω(s.ReadConfig("/path/to/nowhere")).ShouldNot(Succeed())
		})

		It("handles malformed YAML files", func() {
			Ω(s.ReadConfig("test/etc/config.xml")).ShouldNot(Succeed())
		})

		It("handles YAML files with missing directives", func() {
			Ω(s.ReadConfig("test/etc/empty.yml")).Should(Succeed())
			Ω(s.Database.Driver).Should(Equal(""))
			Ω(s.Database.DSN).Should(Equal(""))
			Ω(s.Web.Addr).Should(Equal(":8888"))
			Ω(s.PrivateKeyFile).Should(Equal("/etc/shield/ssh/server.key"))
			Ω(s.Workers).Should(Equal(uint(5)))
			Expect(s.PurgeAgent).Should(Equal("localhost:5444"))
			Expect(s.Web.Auth.Basic.Password).Should(Equal("admin"))
			Expect(s.Web.Auth.Basic.User).Should(Equal("admin"))
			Expect(s.Web.Auth.OAuth.Sessions.MaxAge).Should(Equal(86400 * 30))
			// if no provider specified, base url should be empty and not fail
			Expect(s.Web.Auth.OAuth.Provider).Should(Equal(""))
			Expect(s.Web.Auth.OAuth.BaseURL).Should(Equal(""))
			// no key generation if no oauthprovider
			Expect(s.Web.Auth.OAuth.JWTPrivateKey).Should(BeNil())
			Expect(s.Web.Auth.OAuth.JWTPublicKey).Should(BeNil())
		})

		It("handles YAML files with all the directives", func() {
			Ω(s.ReadConfig("test/etc/valid.yml")).Should(Succeed())
			Ω(s.Database.Driver).Should(Equal("my-driver"))
			Ω(s.Database.DSN).Should(Equal("my:dsn=database"))
			Ω(s.Web.Addr).Should(Equal(":8988"))
			Ω(s.PrivateKeyFile).Should(Equal("/etc/priv.key"))
			Expect(s.PurgeAgent).Should(Equal("remotehost:5444"))
		})

		It("autovivifies the supervisor database object", func() {
			s.Database = nil
			Ω(s.ReadConfig("test/etc/valid.yml")).Should(Succeed())
			Ω(s.Database).ShouldNot(BeNil())
		})

		Describe("when oauth is enabled", func() {
			It("Fails if baseURL is missing", func() {
				Expect(s.ReadConfig("test/etc/oauth-no-url.yml")).ShouldNot(Succeed())
			})

			It("Fails if the signing key could not be read", func() {
				Expect(s.ReadConfig("test/etc/oauth-missing-signing-key.yml")).ShouldNot(Succeed())
			})

			It("Fails if the signing key's data could not be parsed", func() {
				Expect(s.ReadConfig("test/etc/oauth-invalid-signing-key.yml")).ShouldNot(Succeed())
			})

			It("Reads in the signing key and creates pub/priv key objects on valid keys", func() {
				Expect(s.ReadConfig("test/etc/oauth-valid-signing-key.yml")).Should(Succeed())
				Expect(s.Web.Auth.OAuth.JWTPrivateKey).ShouldNot(BeNil())
				Expect(s.Web.Auth.OAuth.JWTPublicKey).ShouldNot(BeNil())
			})

			It("Generates a random signing key if no file specified", func() {
				Expect(s.ReadConfig("test/etc/oauth-no-signing-key.yml")).Should(Succeed())
				Expect(s.Web.Auth.OAuth.JWTPrivateKey).ShouldNot(BeNil())
				Expect(s.Web.Auth.OAuth.JWTPublicKey).ShouldNot(BeNil())
			})

			It("Creates an http client with ssl verification enabled", func() {
				Expect(s.ReadConfig("test/etc/oauth-ssl-checking.yml")).Should(Succeed())
				Expect(s.Web.Auth.OAuth.Client.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify).Should(BeFalse())
			})
			It("Creates an http client with ssl verification disabled", func() {
				Expect(s.ReadConfig("test/etc/oauth-ssl-skip-checking.yml")).Should(Succeed())
				Expect(s.Web.Auth.OAuth.Client.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify).Should(BeTrue())
			})
		})
	})
})
