package cmd_test

import (
	"github.com/cppforlife/go-patch/patch"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
)

var _ = Describe("OpsFlags", func() {
	Describe("AsOps", func() {
		It("merges all ops into one in given order", func() {
			flags := OpsFlags{
				OpsFiles: []OpsFileArg{
					{
						Ops: patch.Ops([]patch.Op{
							patch.RemoveOp{Path: patch.MustNewPointerFromString("/a")},
							patch.RemoveOp{Path: patch.MustNewPointerFromString("/b")},
						}),
					},
					{
						Ops: patch.Ops([]patch.Op{
							patch.RemoveOp{Path: patch.MustNewPointerFromString("/x")},
						}),
					},
				},
			}

			Expect(flags.AsOp()).To(Equal(patch.Ops{
				patch.RemoveOp{Path: patch.MustNewPointerFromString("/a")},
				patch.RemoveOp{Path: patch.MustNewPointerFromString("/b")},
				patch.RemoveOp{Path: patch.MustNewPointerFromString("/x")},
			}))
		})
	})
})
