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
func init() {
	Describe("Testing with Ginkgo", func() {
		It("migrate disk should be asynchronous", func() {
			_, action := buildMigrateDiskAction()
			Expect(action.IsAsynchronous()).To(BeTrue())
		})

		It("is not persistent", func() {
			_, action := buildMigrateDiskAction()
			Expect(action.IsPersistent()).To(BeFalse())
		})

		It("migrate disk action run", func() {

			platform, action := buildMigrateDiskAction()

			value, err := action.Run()
			Expect(err).ToNot(HaveOccurred())
			boshassert.MatchesJSONString(GinkgoT(), value, "{}")

			Expect(platform.MigratePersistentDiskFromMountPoint).To(Equal("/foo/store"))
			Expect(platform.MigratePersistentDiskToMountPoint).To(Equal("/foo/store_migration_target"))
		})
	})
}
