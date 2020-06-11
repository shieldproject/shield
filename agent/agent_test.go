package agent_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/shieldproject/shield/agent"
)

var _ = Describe("Agent", func() {
	Describe("Authorized Keys Loader", func() {
		It("throws an error when loading authorized keys from a non-existent file", func() {
			_, err := LoadAuthorizedKeysFromFile("test/enoent")
			Ω(err).Should(HaveOccurred())
		})

		It("can load authorized keys from a file", func() {
			keys, err := LoadAuthorizedKeysFromFile("test/authorized_keys")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(len(keys)).Should(Equal(2))
		})

		It("ignores malformed keys in the authorized keys file", func() {
			keys, err := LoadAuthorizedKeysFromFile("test/authorized_keys.malformed")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(len(keys)).Should(Equal(2))
		})
	})

	Describe("SSH Server Configurator", func() {
		It("returns a ServerConfig when given a valid host key", func() {
			key, err := LoadPrivateKeyFromFile("test/identities/server/id_rsa")
			Ω(err).ShouldNot(HaveOccurred())
			keys, err := LoadAuthorizedKeysFromFile("test/authorized_keys")
			Ω(err).ShouldNot(HaveOccurred())
			config, err := ConfigureSSHServer(key, keys, nil)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(config).ShouldNot(BeNil())
		})
	})

	Describe("SSH Client Configurator", func() {
		It("throws an error when given a badprivate host key path", func() {
			_, err := ConfigureSSHClient("test/enoent")
			Ω(err).Should(HaveOccurred())
		})

		It("throws an error when given a malformed host key", func() {
			_, err := ConfigureSSHClient("test/identities/bad/malformed")
			Ω(err).Should(HaveOccurred())
		})

		It("returns a ClientConfig when given a valid host key", func() {
			config, err := ConfigureSSHClient("test/identities/a/id_rsa")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(config).ShouldNot(BeNil())
		})
	})

	Describe("SSH Command Parser", func() {
		It("errors for an empty payload", func() {
			_, err := ParseCommand([]byte(""))
			Ω(err).Should(HaveOccurred())
		})

		It("errors for an non-JSON payload", func() {
			_, err := ParseCommand([]byte("not json"))
			Ω(err).Should(HaveOccurred())
		})

		It("errors for a payload missing required 'operation' field", func() {
			_, err := ParseCommand([]byte(`
				{
					"task_uuid"       : "d9b66d82-b016-4e4a-8d7a-800ef9699112",
					"target_plugin"   : "plugin",
					"target_endpoint" : "endpoint",
					"stream":{"url":"http://ssg:8080", "id":"f00", "token":"t0ken", "path":"ssg://foo/bar/file"}
				}
			`))
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(MatchRegexp(`missing required 'operation' `))
		})

		It("errors for a payload missing required 'target_plugin' field", func() {
			_, err := ParseCommand([]byte(`
				{
					"task_uuid"       : "d9b66d82-b016-4e4a-8d7a-800ef9699112",
					"operation"       : "backup",
					"target_endpoint" : "endpoint",
					"stream":{"url":"http://ssg:8080", "id":"f00", "token":"t0ken", "path":"ssg://foo/bar/file"}
				}
			`))
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(MatchRegexp(`missing required 'target_plugin' `))
		})

		It("errors for a payload missing required 'target_endpoint' field", func() {
			_, err := ParseCommand([]byte(`
				{
					"task_uuid"      : "d9b66d82-b016-4e4a-8d7a-800ef9699112",
					"operation"      : "backup",
					"target_plugin"  : "plugin",
					"stream":{"url":"http://ssg:8080", "id":"f00", "token":"t0ken", "path":"ssg://foo/bar/file"}
				}
			`))
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(MatchRegexp(`missing required 'target_endpoint' `))
		})

		It("errors for a payload with unsupported 'operation' field", func() {
			_, err := ParseCommand([]byte(`
				{
					"operation":"XYZZY",
					"target_plugin":"plugin",
					"target_endpoint":"endpoint",
					"stream":{"url":"http://ssg:8080", "id":"f00", "token":"t0ken", "path":"ssg://foo/bar/file"}
				}
			`))
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(MatchRegexp(`unsupported operation.*XYZZY`))
		})

		It("returns a Command object for a valid backup operation", func() {
			cmd, err := ParseCommand([]byte(`
				{
					"task_uuid"       : "d9b66d82-b016-4e4a-8d7a-800ef9699112",
					"operation"       : "backup",
					"target_plugin"   : "t.plugin",
					"target_endpoint" : "t.endpoint",
					"stream":{"url":"http://ssg:8080", "id":"f00", "token":"t0ken", "path":"ssg://foo/bar/file"}
				}
			`))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(cmd).ShouldNot(BeNil())
			Ω(cmd.Op).Should(Equal("backup"))
			Ω(cmd.TargetPlugin).Should(Equal("t.plugin"))
			Ω(cmd.TargetEndpoint).Should(Equal("t.endpoint"))
		})

		It("returns a Command object for a valid restore operation", func() {
			cmd, err := ParseCommand([]byte(`
				{
					"task_uuid"       : "d9b66d82-b016-4e4a-8d7a-800ef9699112",
					"operation"       : "restore",
					"target_plugin"   : "t.plugin",
					"target_endpoint" : "t.endpoint",
					"stream":{"url":"http://ssg:8080", "id":"f00", "token":"t0ken", "path":"ssg://foo/bar/file"}
				}
			`))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(cmd).ShouldNot(BeNil())
			Ω(cmd.Op).Should(Equal("restore"))
			Ω(cmd.TargetPlugin).Should(Equal("t.plugin"))
			Ω(cmd.TargetEndpoint).Should(Equal("t.endpoint"))
		})
	})

	Describe("Agent configuration file", func() {
		It("Requires an authorized_keys_file", func() {
			ag := NewAgent()
			err := ag.ReadConfig("test/auth_key_test.conf")
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(Equal("no authorized keys supplied"))
		})
		It("Requires a listen_address", func() {
			ag := NewAgent()
			err := ag.ReadConfig("test/listen_test.conf")
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(Equal("no listen address and/or port supplied"))
		})
		It("Requires a plugin_paths", func() {
			ag := NewAgent()
			err := ag.ReadConfig("test/plugin_path_test.conf")
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(Equal("no plugin path supplied"))
		})
		It("Requires a non-empty plugin path", func() {
			ag := NewAgent()
			err := ag.ReadConfig("test/plugin_empty_test.conf")
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(Equal("no plugin path supplied"))
		})
		It("Requires a list, not a scalar, of plugin paths", func() {
			ag := NewAgent()
			err := ag.ReadConfig("test/plugin_scalar_test.conf")
			Ω(err).Should(HaveOccurred())
		})
	})
})
