package cmd_test

import (
	"errors"

	"github.com/cppforlife/go-patch/patch"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	boshtpl "github.com/cloudfoundry/bosh-cli/director/template"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("UpdateCloudConfigCmd", func() {
	var (
		ui       *fakeui.FakeUI
		director *fakedir.FakeDirector
		command  UpdateCloudConfigCmd
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		director = &fakedir.FakeDirector{}
		command = NewUpdateCloudConfigCmd(ui, director)
	})

	Describe("Run", func() {
		var (
			opts UpdateCloudConfigOpts
		)

		BeforeEach(func() {
			opts = UpdateCloudConfigOpts{
				Args: UpdateCloudConfigArgs{
					CloudConfig: FileBytesArg{Bytes: []byte("cloud-config")},
				},
			}
		})

		act := func() error { return command.Run(opts) }

		It("updates cloud config", func() {
			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(director.UpdateCloudConfigCallCount()).To(Equal(1))

			bytes := director.UpdateCloudConfigArgsForCall(0)
			Expect(bytes).To(Equal([]byte("cloud-config\n")))
		})

		It("updates templated cloud config", func() {
			opts.Args.CloudConfig = FileBytesArg{
				Bytes: []byte("name1: ((name1))\nname2: ((name2))"),
			}

			opts.VarKVs = []boshtpl.VarKV{
				{Name: "name1", Value: "val1-from-kv"},
			}

			opts.VarsFiles = []boshtpl.VarsFileArg{
				{Vars: boshtpl.StaticVariables(map[string]interface{}{"name1": "val1-from-file"})},
				{Vars: boshtpl.StaticVariables(map[string]interface{}{"name2": "val2-from-file"})},
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

			Expect(director.UpdateCloudConfigCallCount()).To(Equal(1))

			bytes := director.UpdateCloudConfigArgsForCall(0)
			Expect(bytes).To(Equal([]byte("name1: val1-from-kv\nname2: val2-from-file\nxyz: val\n")))
		})

		It("returns an error if diffing failed", func() {
			director.DiffCloudConfigReturns(boshdir.CloudConfigDiff{}, errors.New("Fetching diff result"))

			err := act()
			Expect(err).To(HaveOccurred())
		})

		It("gets the diff from the deployment", func() {
			diff := [][]interface{}{
				[]interface{}{"some line that stayed", nil},
				[]interface{}{"some line that was added", "added"},
				[]interface{}{"some line that was removed", "removed"},
			}

			expectedDiff := boshdir.NewCloudConfigDiff(diff)
			director.DiffCloudConfigReturns(expectedDiff, nil)
			err := act()
			Expect(err).ToNot(HaveOccurred())
			Expect(director.DiffCloudConfigCallCount()).To(Equal(1))
			Expect(ui.Said).To(ContainElement("  some line that stayed\n"))
			Expect(ui.Said).To(ContainElement("+ some line that was added\n"))
			Expect(ui.Said).To(ContainElement("- some line that was removed\n"))
		})

		It("does not stop if confirmation is rejected", func() {
			ui.AskedConfirmationErr = errors.New("stop")

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("stop"))

			Expect(director.UpdateCloudConfigCallCount()).To(Equal(0))
		})

		It("returns error if updating failed", func() {
			director.UpdateCloudConfigReturns(errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
