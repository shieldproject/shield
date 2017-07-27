package cmd_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
)

var _ = Describe("URLArg", func() {
	Describe("IsEmpty", func() {
		It("returns true if empty", func() {
			Expect(URLArg("val").IsEmpty()).To(BeFalse())
			Expect(URLArg("").IsEmpty()).To(BeTrue())
		})
	})

	Describe("IsRemote", func() {
		It("returns true if http/https scheme is used", func() {
			Expect(URLArg("https://host").IsRemote()).To(BeTrue())
			Expect(URLArg("http://host").IsRemote()).To(BeTrue())
			Expect(URLArg("other://host").IsRemote()).To(BeFalse())
		})
	})

	Describe("IsGit", func() {
		It("returns true if git/git+file/git+https scheme is used", func() {
			Expect(URLArg("git://host").IsGit()).To(BeTrue())
			Expect(URLArg("git@host").IsGit()).To(BeTrue())
			Expect(URLArg("git+file://host").IsGit()).To(BeTrue())
			Expect(URLArg("git+https://host").IsGit()).To(BeTrue())
			Expect(URLArg("git+other://host").IsGit()).To(BeTrue())
			Expect(URLArg("other://host").IsGit()).To(BeFalse())
		})
	})

	Describe("FilePath", func() {
		It("returns path without 'file://'", func() {
			Expect(URLArg("path").FilePath()).To(Equal("path"))
			Expect(URLArg("file://path").FilePath()).To(Equal("path"))
		})
	})

	Describe("GitRepo", func() {
		It("returns same value stripping off git+", func() {
			Expect(URLArg("git://host").GitRepo()).To(Equal("git://host"))
			Expect(URLArg("git@host").GitRepo()).To(Equal("git@host"))
			Expect(URLArg("git+https://host").GitRepo()).To(Equal("https://host"))
			Expect(URLArg("git+file://host").GitRepo()).To(Equal("file://host"))
		})
	})
})
