package cmd_test

import (
	"errors"

	"github.com/cppforlife/go-patch/patch"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	boshtpl "github.com/cloudfoundry/bosh-cli/director/template"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("UpdateCPIConfigCmd", func() {
	var (
		ui       *fakeui.FakeUI
		director *fakedir.FakeDirector
		command  UpdateCPIConfigCmd
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		director = &fakedir.FakeDirector{}
		command = NewUpdateCPIConfigCmd(ui, director)
	})

	Describe("Run", func() {
		var (
			opts UpdateCPIConfigOpts
		)

		BeforeEach(func() {
			opts = UpdateCPIConfigOpts{
				Args: UpdateCPIConfigArgs{
					CPIConfig: FileBytesArg{Bytes: []byte("cpi-config")},
				},
			}
		})

		act := func() error { return command.Run(opts) }

		It("updates cpi config", func() {
			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(director.UpdateCPIConfigCallCount()).To(Equal(1))

			bytes := director.UpdateCPIConfigArgsForCall(0)
			Expect(bytes).To(Equal([]byte("cpi-config\n")))
		})

		It("updates templated cpi config", func() {
			opts.Args.CPIConfig = FileBytesArg{
				Bytes: []byte("name: ((name))\ntype: ((type))"),
			}

			opts.VarKVs = []boshtpl.VarKV{
				{Name: "name", Value: "val1-from-kv"},
			}

			opts.VarsFiles = []boshtpl.VarsFileArg{
				{Vars: boshtpl.StaticVariables(map[string]interface{}{"name": "val1-from-file"})},
				{Vars: boshtpl.StaticVariables(map[string]interface{}{"type": "val2-from-file"})},
			}

			opts.OpsFiles = []OpsFileArg{
				{
					Ops: patch.Ops([]patch.Op{
						patch.ReplaceOp{Path: patch.MustNewPointerFromString("/xyz?"), Value: "val"},
					}),
				},
			}

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(director.UpdateCPIConfigCallCount()).To(Equal(1))

			bytes := director.UpdateCPIConfigArgsForCall(0)
			Expect(bytes).To(Equal([]byte("name: val1-from-kv\ntype: val2-from-file\nxyz: val\n")))
		})

		It("does not stop if confirmation is rejected", func() {
			ui.AskedConfirmationErr = errors.New("stop")

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("stop"))

			Expect(director.UpdateCPIConfigCallCount()).To(Equal(0))
		})

		It("returns error if updating failed", func() {
			director.UpdateCPIConfigReturns(errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
