package action_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agent/action"
	fakeplatform "github.com/cloudfoundry/bosh-agent/platform/fakes"
)

var _ = Describe("ReleaseApplySpec", func() {
	var (
		platform *fakeplatform.FakePlatform
		action   ReleaseApplySpecAction
	)

	BeforeEach(func() {
		platform = fakeplatform.NewFakePlatform()
		action = NewReleaseApplySpec(platform)
	})

	AssertActionIsNotAsynchronous(action)
	AssertActionIsNotPersistent(action)
	AssertActionIsLoggable(action)

	AssertActionIsNotResumable(action)
	AssertActionIsNotCancelable(action)

	It("run", func() {
		err := platform.GetFs().WriteFileString("/var/vcap/micro/apply_spec.json", `{"json":["objects"]}`)
		Expect(err).ToNot(HaveOccurred())

		value, err := action.Run()
		Expect(err).ToNot(HaveOccurred())

		Expect(value).To(Equal(map[string]interface{}{"json": []interface{}{"objects"}}))
	})
})
