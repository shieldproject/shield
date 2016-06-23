package api_test

import (
	"fmt"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/starkandwayne/shield/api"
)

var _ = Describe("API Config", func() {
	Describe("When loading configs", func() {
		defaultCfg := &Config{Backends: map[string]string{}, Aliases: map[string]string{}}
		BeforeEach(func() {
			os.Chmod("test/etc/unreadable.yml", 0200)
		})
		AfterEach(func() {
			os.Chmod("test/etc/unreadable.yml", 0644)
		})
		It("Throws an error on invalid yaml files", func() {
			Expect(LoadConfig("test/etc/invalid.yml")).ShouldNot(Succeed())
			defaultCfg.Path = "test/etc/invalid.yml"
			Expect(Cfg).Should(Equal(defaultCfg))
		})

		It("Throws an error on unreadable files", func() {
			if os.Geteuid() == 0 {
				Skip("Cannot test unreadable files when euid = 0")
			}
			Expect(LoadConfig("test/etc/unreadable.yml")).ShouldNot(Succeed())
			defaultCfg.Path = "test/etc/unreadable.yml"
			Expect(Cfg).Should(Equal(defaultCfg))
		})
		It("Succeeds if no config was found", func() {
			Expect(LoadConfig("test/etc/missing.yml")).Should(Succeed())
			defaultCfg.Path = "test/etc/missing.yml"
			Expect(Cfg).Should(Equal(defaultCfg))
		})
		It("Reads configs and sets up the api.Cfg variable if config was valid", func() {
			Expect(LoadConfig("test/etc/valid.yml")).Should(Succeed())

			valid := &Config{
				Backends: map[string]string{
					"http://first":  "basic mytoken1",
					"http://second": "basic mytoken2",
				},
				Aliases: map[string]string{
					"first":  "http://first",
					"second": "http://second",
				},
				Backend: "first",
				Path:    "test/etc/valid.yml",
			}
			Expect(Cfg).Should(Equal(valid))
		})
	})
	Describe("When saving configs", func() {
		It("Throws an error when failing to write data", func() {
			if os.Geteuid() == 0 {
				Skip("Cannot test unwritable files when euid = 0. golang will create the path structure for you")
			}
			cfg := &Config{Backend: "default", Path: "/path/to/nowhere"}
			Expect(cfg.Save()).ShouldNot(Succeed())
		})
		It("Successfully writes the config to disk", func() {
			tempFile, err := ioutil.TempFile("", "shield-test-cfg") // get default tmpdir for OS + supply a prefix
			Expect(err).ShouldNot(HaveOccurred())
			tempFile.Close()
			cfg := &Config{Backend: "default", Path: tempFile.Name()}
			expectedCfg := `backend: default
backends: {}
aliases: {}
`

			Expect(cfg.Save()).Should(Succeed())

			data, err := ioutil.ReadFile(tempFile.Name())
			Expect(err).ShouldNot(HaveOccurred())
			Expect(string(data)).Should(Equal(expectedCfg))

			err = os.Remove(tempFile.Name())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error cleaning up temporary test file (%s): %s\n", tempFile.Name(), err)
			}
		})
	})
	Describe("When retrieving the URI of the current backend", func() {
		var cfg *Config
		BeforeEach(func() {
			cfg = &Config{
				Backend: "",
				Backends: map[string]string{
					"http://localhost":      "basic token",
					"http://localhost:8080": "bearer token",
				},
				Aliases: map[string]string{
					"shield1": "http://localhost",
					"shield2": "http://localhost:8080",
					"invalid": "http://google.com",
				},
			}
		})
		It("Returns empty string if no current backend set", func() {
			Expect(cfg.BackendURI()).Should(Equal(""))
		})
		It("Returns an empty string if current backend is an alias which is set to an invalid backend", func() {
			cfg.Backend = "invalid"
			Expect(cfg.BackendURI()).Should(Equal(""))

		})
		It("Returns an empty string if current backend is an invalid alias", func() {
			cfg.Backend = "google"
			Expect(cfg.BackendURI()).Should(Equal(""))
		})
		It("Returns an empty string if the current backend is an invalid backend", func() {
			cfg.Backend = "http://google.com"
			Expect(cfg.BackendURI()).Should(Equal(""))
		})
		It("Returns the current URI if a valid alias is set", func() {
			cfg.Backend = "shield2"
			Expect(cfg.BackendURI()).Should(Equal("http://localhost:8080"))
		})
		It("Returns the current URI if a valid backend is set", func() {
			cfg.Backend = "http://localhost"
			Expect(cfg.BackendURI()).Should(Equal("http://localhost"))
		})
	})
	Describe("When retrieving the Token of the current backend", func() {
		var cfg *Config
		BeforeEach(func() {
			cfg = &Config{
				Backend: "",
				Backends: map[string]string{
					"http://localhost":      "basic token",
					"http://localhost:8080": "bearer token",
				},
				Aliases: map[string]string{
					"shield1": "http://localhost",
					"shield2": "http://localhost:8080",
					"invalid": "http://google.com",
				},
			}
		})
		It("Returns empty tring if no current backend set", func() {
			Expect(cfg.BackendToken()).Should(Equal(""))
		})
		It("Returns an empty string if current backend is an alias is set to an invalid backend", func() {
			cfg.Backend = "invalid"
			Expect(cfg.BackendToken()).Should(Equal(""))

		})
		It("Returns an empty string if current backend is an invalid alias", func() {
			cfg.Backend = "google"
			Expect(cfg.BackendToken()).Should(Equal(""))
		})
		It("Returns an empty string if the current backend is an invalid backend", func() {
			cfg.Backend = "http://google.com"
			Expect(cfg.BackendToken()).Should(Equal(""))
		})
		It("Returns the current Token if a valid alias is set", func() {
			cfg.Backend = "shield2"
			Expect(cfg.BackendToken()).Should(Equal("bearer token"))
		})
		It("Returns the current Token if a valid backend is set", func() {
			cfg.Backend = "http://localhost"
			Expect(cfg.BackendToken()).Should(Equal("basic token"))
		})
	})
	Describe("When resolving aliases", func() {
		var cfg *Config
		BeforeEach(func() {
			cfg = &Config{
				Backend: "",
				Backends: map[string]string{
					"http://localhost":      "basic token",
					"http://localhost:8080": "bearer token",
				},
				Aliases: map[string]string{
					"shield1": "http://localhost",
					"shield2": "http://localhost:8080",
					"invalid": "http://google.com",
				},
			}
		})
		It("Returns an empty string if alias was not found", func() {
			Expect(cfg.ResolveAlias("google")).Should(Equal(""))
		})
		It("Returns an empty string if a backend was not found", func() {
			Expect(cfg.ResolveAlias("http://google.com")).Should(Equal(""))
		})
		It("Returns an empty string if alias pointed to a bad backend", func() {
			Expect(cfg.ResolveAlias("invalid")).Should(Equal(""))
		})
		It("Returns the URI for a valid alias", func() {
			Expect(cfg.ResolveAlias("shield2")).Should(Equal("http://localhost:8080"))
		})
		It("Returns the URI for a valid backend", func() {
			Expect(cfg.ResolveAlias("http://localhost")).Should(Equal("http://localhost"))
		})
	})
	Describe("When updating backends", func() {
		var cfg *Config
		BeforeEach(func() {
			cfg = &Config{
				Backends: map[string]string{
					"http://localhost": "basic token",
				},
				Aliases: map[string]string{
					"shield1": "http://localhost",
				},
			}
		})
		It("Saves the token to the backend for a valid host/alias", func() {
			Expect(cfg.UpdateBackend("shield1", "bearer token")).Should(Succeed())
			Expect(cfg.Backends).Should(Equal(map[string]string{"http://localhost": "bearer token"}))
		})
		It("Fails to save the token if the backend is invalid", func() {
			Expect(cfg.UpdateBackend("invalid", "bearer token")).ShouldNot(Succeed())
			Expect(cfg.Backends).Should(Equal(map[string]string{"http://localhost": "basic token"}))
		})
	})
	Describe("When updating the current backend", func() {
		var cfg *Config
		BeforeEach(func() {
			cfg = &Config{
				Backends: map[string]string{
					"http://localhost": "basic token",
				},
				Aliases: map[string]string{
					"shield1": "http://localhost",
				},
			}
		})
		It("fails to save anything if the current backend is invalid", func() {
			Expect(cfg.UpdateCurrentBackend("bearer token")).ShouldNot(Succeed())
			Expect(cfg.Backends).Should(Equal(map[string]string{"http://localhost": "basic token"}))
		})
		It("saves the token to the current backend if current backend is set", func() {
			cfg.Backend = "shield1"
			Expect(cfg.UpdateCurrentBackend("bearer token")).Should(Succeed())
			Expect(cfg.Backends).Should(Equal(map[string]string{"http://localhost": "bearer token"}))
		})
	})
	Describe("When adding a backend", func() {
		var cfg *Config
		initialAliases := map[string]string{"shield": "http://localhost"}
		initialBackends := map[string]string{"http://localhost": "basic token"}
		BeforeEach(func() {
			cfg = &Config{
				Backends: initialBackends,
				Aliases:  initialAliases,
			}
		})
		It("fails if the URL is bad", func() {
			Expect(cfg.AddBackend("not a url", "willFail")).ShouldNot(Succeed())
			Expect(cfg.Aliases).Should(Equal(initialAliases))
			Expect(cfg.Backends).Should(Equal(initialBackends))
		})
		It("Fails if the URL doesnt exist", func() {
			Expect(cfg.AddBackend("", "alias")).ShouldNot(Succeed())
			Expect(cfg.Aliases).Should(Equal(initialAliases))
			Expect(cfg.Backends).Should(Equal(initialBackends))
		})
		It("Adds a new alias, and vivifies the backend if it doesn't exist", func() {
			Expect(cfg.AddBackend("http://localhost:8080", "shield-2")).Should(Succeed())
			Expect(cfg.Backends["http://localhost:8080"]).Should(Equal(""))
			Expect(cfg.Aliases["shield-2"]).Should(Equal("http://localhost:8080"))
		})
		It("Adds a new alias but doesn't overwrite existing backend token values", func() {
			Expect(cfg.AddBackend("http://localhost", "shield-2")).Should(Succeed())
			Expect(cfg.Backends["http://localhost"]).Should(Equal("basic token"))
			Expect(cfg.Aliases["shield-2"]).Should(Equal("http://localhost"))
		})
		It("Updates alias mappings, auto-vivifying if needed", func() {
			Expect(cfg.AddBackend("http://localhost:8080", "shield")).Should(Succeed())
			Expect(cfg.Backends["http://localhost:8080"]).Should(Equal(""))
			Expect(cfg.Aliases["shield"]).Should(Equal("http://localhost:8080"))
		})
		It("Updates alias mappings, not overwriting existing backend token values", func() {
			Expect(cfg.AddBackend("http://localhost", "shield")).Should(Succeed())
			Expect(cfg.Backends["http://localhost"]).Should(Equal("basic token"))
			Expect(cfg.Aliases["shield"]).Should(Equal("http://localhost"))
		})
	})
	Describe("When selecting a backend to use", func() {
		var cfg *Config
		initialAliases := map[string]string{"shield": "http://localhost"}
		initialBackends := map[string]string{"http://localhost": "basic token"}
		BeforeEach(func() {
			cfg = &Config{
				Backends: initialBackends,
				Aliases:  initialAliases,
			}
		})
		It("Errors for invalid backend/aliases", func() {
			Expect(cfg.UseBackend("invalid")).ShouldNot(Succeed())
			Expect(cfg.Backend).Should(Equal(""))
		})
		It("Succeeds for valid backend/aliases", func() {
			Expect(cfg.UseBackend("shield")).Should(Succeed())
			Expect(cfg.Backend).Should(Equal("shield"))
		})
	})
	Describe("When generating an HTTP Basic Authentication token", func() {
		It("Returns a base64 encoded copy of 'user:password' prefixed with 'Basic '", func() {
			Expect(BasicAuthToken("user", "password")).Should(Equal("Basic dXNlcjpwYXNzd29yZA=="))
		})
	})
})
