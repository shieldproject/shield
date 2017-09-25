package action_test

import (
	. "github.com/cloudfoundry/bosh-agent/agent/action"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func AssertActionIsSynchronousForVersion(action Action, version ProtocolVersion) {
	It("is synchronous for version", func() {
		Expect(action.IsAsynchronous(version)).To(BeFalse())
	})
}

func AssertActionIsAsynchronousForVersion(action Action, version ProtocolVersion) {
	It("is synchronous for version", func() {
		Expect(action.IsAsynchronous(version)).To(BeTrue())
	})
}

func AssertActionIsAsynchronous(action Action) {
	It("is asynchronous", func() {
		Expect(action.IsAsynchronous(ProtocolVersion(1))).To(BeTrue())
	})
}

func AssertActionIsNotAsynchronous(action Action) {
	It("is not asynchronous", func() {
		Expect(action.IsAsynchronous(ProtocolVersion(1))).To(BeFalse())
	})
}

func AssertActionIsPersistent(action Action) {
	It("is persistent", func() {
		Expect(action.IsPersistent()).To(BeTrue())
	})
}

func AssertActionIsNotPersistent(action Action) {
	It("is not persistent", func() {
		Expect(action.IsPersistent()).To(BeFalse())
	})
}

func AssertActionIsLoggable(action Action) {
	It("is loggable", func() {
		Expect(action.IsLoggable()).To(BeTrue())
	})
}

func AssertActionIsNotLoggable(action Action) {
	It("is not loggable", func() {
		Expect(action.IsLoggable()).To(BeFalse())
	})
}

func AssertActionIsNotCancelable(action Action) {
	It("cannot be cancelled", func() {
		err := action.Cancel()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("not supported"))
	})
}

func AssertActionIsResumable(action Action) {
	It("can be resumed", func() {
		value, err := action.Resume()
		Expect(value).To(Equal("ok"))
		Expect(err).ToNot(HaveOccurred())
	})
}

func AssertActionIsNotResumable(action Action) {
	It("cannot be resumed", func() {
		_, err := action.Resume()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("not supported"))
	})
}
