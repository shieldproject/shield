package cmd_test

import (
	. "github.com/cloudfoundry/bosh-cli/cmd"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ColumnOpt", func() {

	It("should keyify column", func() {
		var columnOpt ColumnOpt
		columnOpt.UnmarshalFlag("Header1")

		Expect(columnOpt.Key).To(Equal("header1"))
		Expect(columnOpt.Hidden).To(BeFalse())
	})
})
