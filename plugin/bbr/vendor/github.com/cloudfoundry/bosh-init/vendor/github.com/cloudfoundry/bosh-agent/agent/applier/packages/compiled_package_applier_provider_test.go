package packages_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshbc "github.com/cloudfoundry/bosh-agent/agent/applier/bundlecollection"
	. "github.com/cloudfoundry/bosh-agent/agent/applier/packages"
	fakeblob "github.com/cloudfoundry/bosh-utils/blobstore/fakes"
	fakecmd "github.com/cloudfoundry/bosh-utils/fileutil/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("compiledPackageApplierProvider", func() {
	var (
		blobstore  *fakeblob.FakeBlobstore
		compressor *fakecmd.FakeCompressor
		fs         *fakesys.FakeFileSystem
		logger     boshlog.Logger
		provider   ApplierProvider
	)

	BeforeEach(func() {
		blobstore = fakeblob.NewFakeBlobstore()
		compressor = fakecmd.NewFakeCompressor()
		fs = fakesys.NewFakeFileSystem()
		logger = boshlog.NewLogger(boshlog.LevelNone)
		provider = NewCompiledPackageApplierProvider(
			"fake-install-path",
			"fake-root-enable-path",
			"fake-job-specific-enable-path",
			"fake-name",
			blobstore,
			compressor,
			fs,
			logger,
		)
	})

	Describe("Root", func() {
		It("returns package applier that is configured to update system wide packages", func() {
			expected := NewCompiledPackageApplier(
				boshbc.NewFileBundleCollection(
					"fake-install-path",
					"fake-root-enable-path",
					"fake-name",
					fs,
					logger,
				),
				true,
				blobstore,
				compressor,
				fs,
				logger,
			)
			Expect(provider.Root()).To(Equal(expected))
		})
	})

	Describe("JobSpecific", func() {
		It("returns package applier that is configured to only update job specific packages", func() {
			expected := NewCompiledPackageApplier(
				boshbc.NewFileBundleCollection(
					"fake-install-path",
					"fake-job-specific-enable-path/fake-job-name",
					"fake-name",
					fs,
					logger,
				),

				// Should not operate as owner because keeping-only job specific packages
				// should not delete packages that could potentially be used by other jobs
				false,

				blobstore,
				compressor,
				fs,
				logger,
			)
			Expect(provider.JobSpecific("fake-job-name")).To(Equal(expected))
		})
	})
})
