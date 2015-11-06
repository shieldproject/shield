package plugin_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"plugin"
)

var _ = Describe("Plugin Framework", func() {
	Describe("GenUUID()", func() {
		It("Returns a UUID", func() {
			uuidRegex := `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`
			uuid := plugin.GenUUID()
			Expect(uuid).Should(MatchRegexp(uuidRegex))
			uuid2 := plugin.GenUUID()
			Expect(uuid2).Should(MatchRegexp(uuidRegex))
			Expect(uuid).ShouldNot(Equal(uuid2))
		})
	})
	Describe("Pugin Execution", func() {
		It("Provides usage when bad commands/flags are given", func() {
			Skip("Test not implemented yet :( PRs welcome ;)")
		})
		It("Provides help when requested via flags", func() {
			Skip("Test not implemented yet :( PRs welcome ;)")
		})
		Describe("info", func() {
			It("Exits non-zero and returns an error if it could not json encode the plugin info", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
			It("Exits zero and outputs a JSON string of plugin info on success", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
		})
		Describe("backup", func() {
			It("Exits non-zero and errors when the --endpoint arg is not set", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
			It("Exits non-zero and errors when the --endpoint arg is not valid json", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
			It("Exits zero and outputs backup data on success", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
			It("Exits non-zero on backup failure", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
		})
		Describe("restore", func() {
			It("Exits non-zero and errors when the --endpoint arg is not set", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
			It("Exits non-zero and errors when the --endpoint arg is not valid json", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
			It("Dispatches and performs a restore, exiting 0", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
			It("Exits non-zero on restore failure", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
		})
		Describe("store", func() {
			It("Exits non-zero and errors when the --endoint arg is not set", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
			It("Exits non-zero and errors when the --endpoint arg is not valid json", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
			It("Exits zero and outputs JSON of the key the backup was stored under", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
			It("Exits non-zero on storage failure", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
			It("Exits non-zero on failure to encode key as JSON", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
		})
		Describe("retrieve", func() {
			It("Exits non-zero and errors when the --endpoint arg is not set", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
			It("Exits non-zero and errors when the --endpoint arg is not valid json", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
			It("Exits non-zero and errors when the --key arg is not set", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
			It("Exits zero and outputs data from successful retrieval", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
			It("Exits non-zero on retrieval failure", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
		})
		Describe("purge", func() {
			It("Exits non-zero and errors when the --endpoint arg is not set", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
			It("Exits non-zero and errors when the --endpiont arg is not valid json", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
			It("Exits non-zero and errors when the --key arg is not set", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
			It("Exits zero on successful purge", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
			It("Exits non-zero on failed purge", func() {
				Skip("Test not implemented yet :( PRs welcome ;)")
			})
		})
	})
	Describe("DEBUG()", func() {
		It("Prints output to stderr", func() {
			Skip("Test not implemented yet :( PRs welcome ;)")
		})
	})
})
