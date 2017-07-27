package cmd_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
)

var _ = Describe("TrimmedSpaceArgs", func() {
	Describe("AsStrings", func() {
		It("returns strings list with trimmed spaces", func() {
			args := TrimmedSpaceArgs([]string{" prefix", "suffix ", " prefix suffix "})
			Expect(args.AsStrings()).To(Equal([]string{"prefix", "suffix", "prefix suffix"}))

			args = TrimmedSpaceArgs([]string{"    prefix", "suffix    ", "    prefix    suffix    "})
			Expect(args.AsStrings()).To(Equal([]string{"prefix", "suffix", "prefix    suffix"}))

			args = TrimmedSpaceArgs([]string{"\nprefix", "suffix\r", " \nprefix    suffix\r "})
			Expect(args.AsStrings()).To(Equal([]string{"prefix", "suffix", "prefix    suffix"}))
		})
	})
})
