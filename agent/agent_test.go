package agent_test

import (
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/shieldproject/shield/agent"
)

var _ = Describe("Agent", func() {
	pathify := func(rel string) string {
		cwd, err := os.Getwd()
		Ω(err).ShouldNot(HaveOccurred())

		return fmt.Sprintf("%s/%s", cwd, rel)
	}

	Describe("Plugin Loader", func() {
		var a *Agent

		BeforeEach(func() {
			a = &Agent{
				PluginPaths: []string{"test/plugins/dir", "test/plugins"},
			}
		})

		It("throws an error if the plugin is not found", func() {
			_, err := a.ResolveBinary("enoent")
			Ω(err).Should(HaveOccurred())
		})

		It("skips directories implicitly", func() {
			_, err := a.ResolveBinary("dir")
			Ω(err).Should(HaveOccurred())
		})

		It("skips non-executable files implicitly", func() {
			_, err := a.ResolveBinary("regular")
			Ω(err).Should(HaveOccurred())
		})

		It("finds the executable script", func() {
			path, err := a.ResolveBinary("executable")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(path).Should(Equal("test/plugins/executable"))
		})

		It("finds the first executable script, if there are multiple", func() {
			path, err := a.ResolveBinary("common")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(path).Should(Equal("test/plugins/dir/common"))
		})
	})

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
					"store_plugin"    : "plugin",
					"store_endpoint"  : "endpoint"
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
					"store_plugin"    : "plugin",
					"store_endpoint"  : "endpoint"
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
					"store_plugin"   : "plugin",
					"store_endpoint" : "endpoint"
				}
			`))
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(MatchRegexp(`missing required 'target_endpoint' `))
		})

		It("errors for a payload missing required 'store_plugin' field", func() {
			_, err := ParseCommand([]byte(`
				{
					"task_uuid"       : "d9b66d82-b016-4e4a-8d7a-800ef9699112",
					"operation"       : "backup",
					"target_plugin"   : "plugin",
					"target_endpoint" : "endpoint",
					"store_endpoint"  : "endpoint"
				}
			`))
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(MatchRegexp(`missing required 'store_plugin' `))
		})

		It("errors for a payload missing required 'store_endpoint' field", func() {
			_, err := ParseCommand([]byte(`
				{
					"task_uuid"       : "d9b66d82-b016-4e4a-8d7a-800ef9699112",
					"operation"       : "backup",
					"target_plugin"   : "plugin",
					"target_endpoint" : "endpoint",
					"store_plugin"    : "plugin"
				}
			`))
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(MatchRegexp(`missing required 'store_endpoint' `))
		})

		It("errors for a restore payload missing required 'restore_key' field", func() {
			_, err := ParseCommand([]byte(`
				{
					"task_uuid"       : "d9b66d82-b016-4e4a-8d7a-800ef9699112",
					"operation"       : "restore",
					"target_plugin"   : "plugin",
					"target_endpoint" : "endpoint",
					"store_plugin"    : "plugin",
					"store_endpoint"  : "endpoint"
				}
			`))
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(MatchRegexp(`missing required 'restore_key'`))
		})

		It("errors for a purge payload missing required 'restore_key' field", func() {
			_, err := ParseCommand([]byte(`
				{
					"task_uuid"      : "d9b66d82-b016-4e4a-8d7a-800ef9699112",
					"operation"      : "purge",
					"store_plugin"   : "plugin",
					"store_endpoint" : "endpoint"
				}
			`))
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(MatchRegexp(`missing required 'restore_key'`))
		})

		It("errors for a payload with unsupported 'operation' field", func() {
			_, err := ParseCommand([]byte(`
				{
					"operation":"XYZZY",
					"target_plugin":"plugin",
					"target_endpoint":"endpoint",
					"store_plugin":"plugin",
					"store_endpoint":"endpoint"
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
					"store_plugin"    : "s.plugin",
					"store_endpoint"  : "s.endpoint"
				}
			`))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(cmd).ShouldNot(BeNil())
			Ω(cmd.Op).Should(Equal("backup"))
			Ω(cmd.TargetPlugin).Should(Equal("t.plugin"))
			Ω(cmd.TargetEndpoint).Should(Equal("t.endpoint"))
			Ω(cmd.StorePlugin).Should(Equal("s.plugin"))
			Ω(cmd.StoreEndpoint).Should(Equal("s.endpoint"))
			Ω(cmd.RestoreKey).Should(Equal(""))
		})

		It("returns a Command object for a valid restore operation", func() {
			cmd, err := ParseCommand([]byte(`
				{
					"task_uuid"       : "d9b66d82-b016-4e4a-8d7a-800ef9699112",
					"operation"       : "restore",
					"target_plugin"   : "t.plugin",
					"target_endpoint" : "t.endpoint",
					"store_plugin"    : "s.plugin",
					"store_endpoint"  : "s.endpoint",
					"restore_key"     : "r.key"
				}
			`))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(cmd).ShouldNot(BeNil())
			Ω(cmd.Op).Should(Equal("restore"))
			Ω(cmd.TargetPlugin).Should(Equal("t.plugin"))
			Ω(cmd.TargetEndpoint).Should(Equal("t.endpoint"))
			Ω(cmd.StorePlugin).Should(Equal("s.plugin"))
			Ω(cmd.StoreEndpoint).Should(Equal("s.endpoint"))
			Ω(cmd.RestoreKey).Should(Equal("r.key"))
		})
	})

	Describe("Command Runner", func() {
		var cmd *Command
		var a *Agent

		BeforeEach(func() {
			var err error
			Ω(err).ShouldNot(HaveOccurred())

			cmd, err = ParseCommand([]byte(`{
				"task_uuid"       : "d9b66d82-b016-4e4a-8d7a-800ef9699112",
				"operation"       : "backup",
				"compression"     : "bzip2",
				"target_plugin"   : "` + pathify("test/bin/dummy") + `",
				"target_endpoint" : "{mode:target,endpoint:config}",
				"store_plugin"    : "` + pathify("test/bin/dummy") + `",
				"store_endpoint"  : "{mode:store,endpoint:config}"
			}`))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(cmd).ShouldNot(BeNil())

			a = &Agent{
				PluginPaths: []string{"test/plugins/dir", "test/plugins"},
			}
		})

		collect := func(out chan string, in chan string) {
			var stdout []string
			var stderr []string
			var other []string

			for {
				s, ok := <-in
				if !ok {
					break
				}
				switch s[:2] {
				case "O:":
					stdout = append(stdout, s[2:])
				case "E:":
					stderr = append(stderr, s[2:])
				default:
					other = append(other, s)
				}
			}
			out <- strings.Join(stdout, "")
			out <- strings.Join(stderr, "")
			out <- strings.Join(other, "")
			close(out)
		}

		It("works", func() {
			out := make(chan string)
			c := make(chan string)

			go collect(out, c)
			err := a.Execute(cmd, c)
			Ω(err).ShouldNot(HaveOccurred())

			var s string
			Eventually(out).Should(Receive(&s)) // stdout
			Eventually(out).Should(Receive(&s)) // stderr
			Eventually(out).Should(Receive(&s)) // misc
		})

		It("works with purge commands", func() {
			cmd.Op = "purge"
			cmd.RestoreKey = "fakeKey"

			out := make(chan string)
			c := make(chan string)

			go collect(out, c)

			err := a.Execute(cmd, c)
			Expect(err).ShouldNot(HaveOccurred())

			var s string
			Eventually(out).Should(Receive(&s)) // stdout
			Expect(s).Should(Equal(""))

			Eventually(out).Should(Receive(&s)) // stderr
			Expect(s).Should(MatchRegexp(`\Q(dummy) purge:  starting up...\E`))
			Expect(s).Should(MatchRegexp(`\Q(dummy) purge:  purging data at key [fakeKey]\E`))
			Expect(s).Should(MatchRegexp(`\Q(dummy) purge:  shutting down...\E`))

			Eventually(out).Should(Receive(&s)) //misc
			Expect(s).Should(Equal(""))
		})

		It("collects output from the command pipeline", func() {
			out := make(chan string)
			c := make(chan string)

			go collect(out, c)
			err := a.Execute(cmd, c)
			Ω(err).ShouldNot(HaveOccurred())

			var s string
			Eventually(out).Should(Receive(&s)) // stdout
			// sha1sum value depends on bzip2 compression
			Ω(s).Should(MatchJSON(`{"compression":"bzip2","key":"9ea61fef3024caadf35dd65d466a41fb51a3c152"}`))

			Eventually(out).Should(Receive(&s)) // stderr
			Ω(s).Should(MatchRegexp(`\Q(dummy) store:  starting up...\E`))
			Ω(s).Should(MatchRegexp(`\Q(dummy) backup:  starting up...\E`))
			Ω(s).Should(MatchRegexp(`\Q(dummy) backup:  shutting down...\E`))
			Ω(s).Should(MatchRegexp(`\Q(dummy) store:  shutting down...\E`))

			Eventually(out).Should(Receive(&s)) // misc
			Ω(s).Should(Equal(""))
		})

		It("handles backup operations with large output", func() {
			out := make(chan string)
			c := make(chan string)

			go collect(out, c)

			// big_dummy outputs > 16384 bytes of data
			cmd.TargetPlugin = "test/bin/big_dummy"

			err := a.Execute(cmd, c)
			Ω(err).ShouldNot(HaveOccurred())

			var s string
			Eventually(out).Should(Receive(&s)) // stdout
			// sha1sum value depends on bzip2 compression
			Ω(s).Should(MatchJSON(`{"compression":"bzip2","key":"acfd124b56584c471d7e03572fe62222ee4862e9"}`))

			Eventually(out).Should(Receive(&s)) // stderr
			Eventually(out).Should(Receive(&s)) // misc
			Ω(s).Should(Equal(""))
		})

		It("handles restore operations with large output", func() {
			out := make(chan string)
			c := make(chan string)

			go collect(out, c)

			// big_dummy outputs > 16384 bytes of data
			cmd.TargetPlugin = "test/bin/big_dummy"
			cmd.Op = "restore"
			cmd.RestoreKey = "some.key"

			err := a.Execute(cmd, c)
			Ω(err).ShouldNot(HaveOccurred())

			var s string
			Eventually(out).Should(Receive(&s)) // stdout
			// sha1sum value depends on bzip2 compression
			Ω(s).Should(Equal("SHA1SUM of restored data: 5736538c1c1fcae2a7aac709e195c709735b90a7\n"))

			Eventually(out).Should(Receive(&s)) // stderr
			Eventually(out).Should(Receive(&s)) // misc
			Ω(s).Should(Equal(""))
		})

		It("handles non-existent plugin commands for both target and store", func() {
			out := make(chan string)
			c := make(chan string)

			go collect(out, c)

			cmd.TargetPlugin = pathify("test/bin/enoent")
			cmd.StorePlugin = pathify("test/bin/enoent")

			err := a.Execute(cmd, c)
			Ω(err).Should(HaveOccurred())

			var s string
			Eventually(out).Should(Receive(&s)) // stdout
			Ω(s).Should(Equal(""))
			Eventually(out).Should(Receive(&s)) // stderr
			Ω(s).ShouldNot(Equal(""))
			Eventually(out).Should(Receive(&s)) // misc
			Ω(s).Should(Equal(""))
		})

		It("handles non-existent plugin commands for just store", func() {
			out := make(chan string)
			c := make(chan string)

			go collect(out, c)

			cmd.StorePlugin = pathify("test/bin/enoent")

			err := a.Execute(cmd, c)
			Ω(err).Should(HaveOccurred())

			var s string
			Eventually(out).Should(Receive(&s)) // stdout
			Ω(s).Should(Equal(""))
			Eventually(out).Should(Receive(&s)) // stderr
			Ω(s).ShouldNot(Equal(""))
			Eventually(out).Should(Receive(&s)) // misc
			Ω(s).Should(Equal(""))
		})

		It("handles non-existent plugin commands for just target", func() {
			out := make(chan string)
			c := make(chan string)

			go collect(out, c)

			cmd.TargetPlugin = pathify("test/bin/enoent")

			err := a.Execute(cmd, c)
			Ω(err).Should(HaveOccurred())

			var s string
			Eventually(out).Should(Receive(&s)) // stdout
			Skip("bin/shield-pipe/Run() needs some more work to suppress output in case of failure?")
			Ω(s).Should(Equal(""))
			Eventually(out).Should(Receive(&s)) // stderr
			Ω(s).ShouldNot(Equal(""))
			Eventually(out).Should(Receive(&s)) // misc
			Ω(s).Should(Equal(""))
		})
	})

	Describe("Agent configuration file", func() {
		It("Requires an authorized_keys_file", func() {
			ag := NewAgent()
			err := ag.ReadConfig("test/auth_key_test.conf")
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(Equal("No authorized keys supplied."))
		})
		It("Requires a listen_address", func() {
			ag := NewAgent()
			err := ag.ReadConfig("test/listen_test.conf")
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(Equal("No listen address and/or port supplied."))
		})
		It("Requires a plugin_paths", func() {
			ag := NewAgent()
			err := ag.ReadConfig("test/plugin_path_test.conf")
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(Equal("No plugin path supplied."))
		})
		It("Requires a non-empty plugin path", func() {
			ag := NewAgent()
			err := ag.ReadConfig("test/plugin_empty_test.conf")
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(Equal("No plugin path supplied."))
		})
		It("Requires a list, not a scalar, of plugin paths", func() {
			ag := NewAgent()
			err := ag.ReadConfig("test/plugin_scalar_test.conf")
			Ω(err).Should(HaveOccurred())
		})
	})

	Describe("SSH Server", func() {
		Endpoint := "127.0.0.1:9122"
		var ag *Agent
		var client *Client

		BeforeEach(func() {
			var err error

			ag = NewAgent()
			err = ag.ReadConfig("test/test.conf")
			Ω(err).ShouldNot(HaveOccurred())
			//---STAWP
			cconfig, err := ConfigureSSHClient("test/identities/a/id_rsa")
			Ω(err).Should(BeNil())
			Ω(cconfig).ShouldNot(BeNil())
			client = NewClient(cconfig)
			Ω(client).ShouldNot(BeNil())

			go ag.ServeOne(ag.Listen, false)
		})

		collect := func(out chan string, in chan string) {
			var buf []string
			for {
				s, ok := <-in
				if !ok {
					break
				}
				buf = append(buf, s)
			}
			out <- strings.Join(buf, "")
			close(out)
		}

		It("handles valid agent-request messages across the session channel", func() {
			err := client.Dial(Endpoint)
			Ω(err).ShouldNot(HaveOccurred())
			defer client.Close()

			final := make(chan string)
			partial := make(chan string)

			go collect(final, partial)
			err = client.Run(partial, `{
				"task_uuid"       : "d9b66d82-b016-4e4a-8d7a-800ef9699112",
				"operation"       : "backup",
				"target_plugin"   : "dummy",
				"target_endpoint" : "TARGET-ENDPOINT",
				"store_plugin"    : "dummy",
				"store_endpoint"  : "STORE-ENDPOINT"
			}`)
			Ω(err).ShouldNot(HaveOccurred())

			var s string
			Eventually(final).Should(Receive(&s))
			Ω(s).ShouldNot(Equal(""))
		})
	})
})
