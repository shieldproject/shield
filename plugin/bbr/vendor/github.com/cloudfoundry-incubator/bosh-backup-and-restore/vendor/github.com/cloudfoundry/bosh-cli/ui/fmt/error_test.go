package fmt_test

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/ui/fmt"
)

var _ = Describe("MultilineError", func() {
	var (
		err error
	)

	It("returns a simple single-line message string (depth=0)", func() {
		err = bosherr.Error("omg")
		Expect(MultilineError(err)).To(Equal("omg"))
	})

	Context("when given a composite error", func() {
		It("returns a multi-line, indented message string (depth=1)", func() {
			err = bosherr.WrapError(bosherr.Error("inner omg"), "omg")
			Expect(MultilineError(err)).To(Equal("omg:\n  inner omg"))
		})

		It("returns a multi-line, indented message string (depth=2)", func() {
			err = bosherr.WrapError(bosherr.WrapError(bosherr.Error("inner omg"), "omg"), "outer omg")
			Expect(MultilineError(err)).To(Equal("outer omg:\n  omg:\n    inner omg"))
		})

		It("returns a multi-line, indented message string (depth=3)", func() {
			err = bosherr.WrapError(bosherr.WrapError(bosherr.WrapError(bosherr.Error("inner omg"), "almost inner omg"), "almost outer omg"), "outer omg")
			Expect(MultilineError(err)).To(Equal("outer omg:\n  almost outer omg:\n    almost inner omg:\n      inner omg"))
		})
	})

	Context("when given an explainable error", func() {
		It("returns a multi-line message string with sibling errors at the same indentation", func() {
			err = bosherr.NewMultiError(bosherr.Error("a"), bosherr.Error("b"))
			Expect(MultilineError(err)).To(Equal("- a\n- b"))
		})

		It("returns a multi-line message string with sibling errors at the same indentation", func() {
			complex := bosherr.WrapError(bosherr.Error("inner a"), "outer a")
			err = bosherr.NewMultiError(complex, bosherr.Error("b"))
			Expect(MultilineError(err)).To(Equal("- outer a:\n    inner a\n- b"))
		})

		It("returns a multi-line message string with sibling errors at the same indentation", func() {
			complex := bosherr.WrapError(bosherr.Error("inner b"), "outer b")
			err = bosherr.NewMultiError(bosherr.Error("a"), complex)
			Expect(MultilineError(err)).To(Equal("- a\n- outer b:\n    inner b"))
		})
	})

	Context("when given a composite err with explainable errors", func() {
		It("returns a multi-line message string with sibling errors at the same indentation", func() {
			multi := bosherr.NewMultiError(bosherr.Error("inner a"), bosherr.Error("inner b"))
			err = bosherr.WrapError(multi, "outer omg")
			Expect(MultilineError(err)).To(Equal("outer omg:\n  - inner a\n  - inner b"))
		})
	})

	Context("when given an ExecError", func() {
		It("returns a multi-line message string with the command, stdout, & stderr at the same indentation", func() {
			execErr := boshsys.NewExecError("fake-cmd --flag with some args", "some\nmultiline\nstdout", "some\nmultiline\nstderr")
			err = bosherr.WrapError(execErr, "outer omg")
			Expect(MultilineError(err)).To(Equal("outer omg:\n  Error Executing Command:\n    fake-cmd --flag with some args\n  StdOut:\n    some\n    multiline\n    stdout\n  StdErr:\n    some\n    multiline\n    stderr"))
		})
	})
})
