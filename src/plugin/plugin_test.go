package plugin

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
)

type MockPlugin struct {
	shouldFail bool
	lastOp     string
}

func (p *MockPlugin) Meta() PluginInfo {
	p.lastOp = "info"
	return PluginInfo{
		Name:    "Mock Plugin",
		Author:  "QA",
		Version: "0.0.1",
		Features: PluginFeatures{
			Target: "yes",
			Store:  "yes",
		},
	}
}
func (p *MockPlugin) op(op string) error {
	p.lastOp = op
	if p.shouldFail {
		return fmt.Errorf("Mock Plugin Failure")
	}
	return nil
}
func (p *MockPlugin) Backup(endpoint ShieldEndpoint) error {
	return p.op("backup")
}
func (p *MockPlugin) Restore(endpoint ShieldEndpoint) error {
	return p.op("restore")
}
func (p *MockPlugin) Store(endpoint ShieldEndpoint) (string, error) {
	err := p.op("store")
	return "mockfile", err
}
func (p *MockPlugin) Retrieve(endpoint ShieldEndpoint, file string) error {
	return p.op(fmt.Sprintf("retrieve %s", file))
}
func (p *MockPlugin) Purge(endpoint ShieldEndpoint, file string) error {
	return p.op(fmt.Sprintf("purge %s", file))
}
func (p *MockPlugin) Reset() {
	p.shouldFail = false
	p.lastOp = "unoperated"
}
func (p *MockPlugin) LastOp() string {
	return p.lastOp
}
func (p *MockPlugin) FailMode(mode bool) {
	p.shouldFail = mode
}

var _ = Describe("Plugin Framework", func() {
	Describe("GenUUID()", func() {
		It("Returns a UUID", func() {
			uuidRegex := `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`
			uuid := GenUUID()
			Expect(uuid).Should(MatchRegexp(uuidRegex))
			uuid2 := GenUUID()
			Expect(uuid2).Should(MatchRegexp(uuidRegex))
			Expect(uuid).ShouldNot(Equal(uuid2))
		})
	})
	Describe("Pugin Execution", func() {
		var rc int
		exit = func(code int) {
			rc = code
		}
		usage = func(err error) {
			rc = USAGE
		}

		p := &MockPlugin{}

		BeforeEach(func() {
			rc = -1
			os.Args = []string{"mock"}
			p.Reset()
			stdout, _ = os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0666)
			stderr, _ = os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND, 0666)
		})
		It("Provides usage when bad commands/flags are given", func() {
			os.Args = append(os.Args, "invalid_command")
			Run(p)
			Expect(p.LastOp()).Should(Equal("unoperated"))
			Expect(rc).Should(Equal(USAGE))
		})
		It("Provides help when requested via flags", func() {
			os.Args = append(os.Args, "-h")
			Run(p)
			Expect(p.LastOp()).Should(Equal("unoperated"))
			Expect(rc).Should(Equal(USAGE))
		})
		Describe("info", func() {
			It("Exits zero and outputs a JSON string of plugin info on success", func() {
				os.Args = append(os.Args, "info")
				rStdout, wStdout, _ := os.Pipe()
				stdout = wStdout
				Run(p)
				wStdout.Close()
				Expect(p.LastOp()).Should(Equal("info"))
				Expect(rc).Should(Equal(SUCCESS))
				content, err := ioutil.ReadAll(rStdout)
				Expect(err).ShouldNot(HaveOccurred())
				rStdout.Close()
				Expect(string(content)).Should(MatchJSON(`{
					"name":"Mock Plugin",
					"author":"QA",
					"version":"0.0.1",
					"features": {
						"target": "yes",
						"store": "yes"
					}
				}`))
			})
		})
		Describe("backup", func() {
			It("Exits non-zero and errors when the --endpoint arg is not set", func() {
				os.Args = append(os.Args, "backup")
				Run(p)
				Expect(p.LastOp()).Should(Equal("unoperated"))
				Expect(rc).Should(Equal(USAGE))
			})
			It("Exits non-zero and errors when the --endpoint arg is not valid json", func() {
				os.Args = append(os.Args, "backup", "--endpoint", "{fdsa")
				Run(p)
				Expect(p.LastOp()).Should(Equal("unoperated"))
				Expect(rc).Should(Equal(JSON_FAILURE))
			})
			It("Exits zero and outputs backup data on success", func() {
				os.Args = append(os.Args, "backup", "--endpoint", "{}")
				Run(p)
				Expect(p.LastOp()).Should(Equal("backup"))
				Expect(rc).Should(Equal(SUCCESS))
			})
			It("Exits non-zero on backup failure", func() {
				os.Args = append(os.Args, "backup", "--endpoint", "{}")
				p.FailMode(true)
				Run(p)
				Expect(p.LastOp()).Should(Equal("backup"))
				Expect(rc).Should(Equal(PLUGIN_FAILURE))
			})
		})
		Describe("restore", func() {
			It("Exits non-zero and errors when the --endpoint arg is not set", func() {
				os.Args = append(os.Args, "restore")
				Run(p)
				Expect(p.LastOp()).Should(Equal("unoperated"))
				Expect(rc).Should(Equal(USAGE))
			})
			It("Exits non-zero and errors when the --endpoint arg is not valid json", func() {
				os.Args = append(os.Args, "restore", "--endpoint", "{fdsa")
				Run(p)
				Expect(p.LastOp()).Should(Equal("unoperated"))
				Expect(rc).Should(Equal(JSON_FAILURE))
			})
			It("Dispatches and performs a restore, exiting 0", func() {
				os.Args = append(os.Args, "restore", "--endpoint", "{}")
				Run(p)
				Expect(p.LastOp()).Should(Equal("restore"))
				Expect(rc).Should(Equal(SUCCESS))
			})
			It("Exits non-zero on restore failure", func() {
				os.Args = append(os.Args, "restore", "--endpoint", "{}")
				p.FailMode(true)
				Run(p)
				Expect(p.LastOp()).Should(Equal("restore"))
				Expect(rc).Should(Equal(PLUGIN_FAILURE))
			})
		})
		Describe("store", func() {
			It("Exits non-zero and errors when the --endpoint arg is not set", func() {
				os.Args = append(os.Args, "store")
				Run(p)
				Expect(p.LastOp()).Should(Equal("unoperated"))
				Expect(rc).Should(Equal(USAGE))
			})
			It("Exits non-zero and errors when the --endpoint arg is not valid json", func() {
				os.Args = append(os.Args, "store", "--endpoint", "{fdsa")
				Run(p)
				Expect(p.LastOp()).Should(Equal("unoperated"))
				Expect(rc).Should(Equal(JSON_FAILURE))
			})
			It("Exits zero and outputs JSON of the key the backup was stored under", func() {
				os.Args = append(os.Args, "store", "--endpoint", "{}")
				rStdout, wStdout, _ := os.Pipe()
				stdout = wStdout
				Run(p)
				wStdout.Close()
				Expect(p.LastOp()).Should(Equal("store"))
				Expect(rc).Should(Equal(SUCCESS))
				content, err := ioutil.ReadAll(rStdout)
				Expect(err).ShouldNot(HaveOccurred())
				rStdout.Close()
				Expect(string(content)).Should(MatchJSON(`{"key":"mockfile"}`))
			})
			It("Exits non-zero on storage failure", func() {
				os.Args = append(os.Args, "store", "--endpoint", "{}")
				p.FailMode(true)
				Run(p)
				Expect(p.LastOp()).Should(Equal("store"))
				Expect(rc).Should(Equal(PLUGIN_FAILURE))
			})
		})
		Describe("retrieve", func() {
			It("Exits non-zero and errors when the --endpoint arg is not set", func() {
				os.Args = append(os.Args, "retrieve")
				Run(p)
				Expect(p.LastOp()).Should(Equal("unoperated"))
				Expect(rc).Should(Equal(USAGE))
			})
			It("Exits non-zero and errors when the --endpoint arg is not valid json", func() {
				os.Args = append(os.Args, "retrieve", "--endpoint", "{fdsa", "--key", "abcdefg")
				Run(p)
				Expect(p.LastOp()).Should(Equal("unoperated"))
				Expect(rc).Should(Equal(JSON_FAILURE))
			})
			It("Exits non-zero and errors when the --key arg is not set", func() {
				os.Args = append(os.Args, "retrieve", "--endpoint", "{}")
				Run(p)
				Expect(p.LastOp()).Should(Equal("unoperated"))
				Expect(rc).Should(Equal(USAGE))
			})
			It("Exits zero from successful retrieval", func() {
				os.Args = append(os.Args, "retrieve", "--endpoint", "{}", "--key", "abcdefg")
				Run(p)
				Expect(p.LastOp()).Should(Equal("retrieve abcdefg"))
				Expect(rc).Should(Equal(SUCCESS))
			})
			It("Exits non-zero on retrieval failure", func() {
				os.Args = append(os.Args, "retrieve", "--endpoint", "{}", "--key", "abcdefg")
				p.FailMode(true)
				Run(p)
				Expect(p.LastOp()).Should(Equal("retrieve abcdefg"))
				Expect(rc).Should(Equal(PLUGIN_FAILURE))
			})
		})
		Describe("purge", func() {
			It("Exits non-zero and errors when the --endpoint arg is not set", func() {
				os.Args = append(os.Args, "purge")
				Run(p)
				Expect(p.LastOp()).Should(Equal("unoperated"))
				Expect(rc).Should(Equal(USAGE))
			})
			It("Exits non-zero and errors when the --endpiont arg is not valid json", func() {
				os.Args = append(os.Args, "purge", "--endpoint", "{fdsa", "--key", "abcdefg")
				Run(p)
				Expect(p.LastOp()).Should(Equal("unoperated"))
				Expect(rc).Should(Equal(JSON_FAILURE))
			})
			It("Exits non-zero and errors when the --key arg is not set", func() {
				os.Args = append(os.Args, "purge", "--endpoint", "{}")
				Run(p)
				Expect(p.LastOp()).Should(Equal("unoperated"))
				Expect(rc).Should(Equal(USAGE))
			})
			It("Exits zero on successful purge", func() {
				os.Args = append(os.Args, "purge", "--endpoint", "{}", "--key", "abcdefg")
				Run(p)
				Expect(p.LastOp()).Should(Equal("purge abcdefg"))
				Expect(rc).Should(Equal(SUCCESS))
			})
			It("Exits non-zero on failed purge", func() {
				os.Args = append(os.Args, "purge", "--endpoint", "{}", "--key", "abcdefg")
				p.FailMode(true)
				Run(p)
				Expect(p.LastOp()).Should(Equal("purge abcdefg"))
				Expect(rc).Should(Equal(PLUGIN_FAILURE))
			})
		})
	})
})
