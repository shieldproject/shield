package action_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agent/action"
	fakeplatform "github.com/cloudfoundry/bosh-agent/platform/fakes"
	boshdirs "github.com/cloudfoundry/bosh-agent/settings/directories"
	boshassert "github.com/cloudfoundry/bosh-utils/assert"
)

func buildMigrateDiskAction() (platform *fakeplatform.FakePlatform, action MigrateDiskAction) {
	platform = fakeplatform.NewFakePlatform()
	dirProvider := boshdirs.NewProvider("/foo")
	action = NewMigrateDisk(platform, dirProvider)
	return
}

var _ = Describe("Testing with Ginkgo", func() {
	var (
		action   MigrateDiskAction
		platform *fakeplatform.FakePlatform
	)

	BeforeEach(func() {
		platform, action = buildMigrateDiskAction()
	})

	AssertActionIsAsynchronous(action)
	AssertActionIsNotPersistent(action)
	AssertActionIsLoggable(action)

	AssertActionIsNotResumable(action)
	AssertActionIsNotCancelable(action)

	It("migrate disk action run", func() {
		value, err := action.Run()
		Expect(err).ToNot(HaveOccurred())
		boshassert.MatchesJSONString(GinkgoT(), value, "{}")

		Expect(platform.MigratePersistentDiskFromMountPoint).To(boshassert.MatchPath("/foo/store"))
		Expect(platform.MigratePersistentDiskToMountPoint).To(boshassert.MatchPath("/foo/store_migration_target"))
	})
})
