package supervisor_test

import (
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
			Ω(s.Port).Should(Equal("8888"))
			Ω(s.PrivateKeyFile).Should(Equal("/etc/shield/ssh/server.key"))
			Ω(s.Workers).Should(Equal(uint(5)))
			Expect(s.PurgeAgent).Should(Equal("localhost:5444"))
		})

		It("handles YAML files with all the directives", func() {
			Ω(s.ReadConfig("test/etc/valid.yml")).Should(Succeed())
			Ω(s.Database.Driver).Should(Equal("my-driver"))
			Ω(s.Database.DSN).Should(Equal("my:dsn=database"))
			Ω(s.Port).Should(Equal("8988"))
			Ω(s.PrivateKeyFile).Should(Equal("/etc/priv.key"))
			Expect(s.PurgeAgent).Should(Equal("remotehost:5444"))
		})

		It("autovivifies the supervisor database object", func() {
			s.Database = nil
			Ω(s.ReadConfig("test/etc/valid.yml")).Should(Succeed())
			Ω(s.Database).ShouldNot(BeNil())
		})
	})
})
