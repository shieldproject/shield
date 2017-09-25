package bundlecollection_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"path"
	"path/filepath"

	. "github.com/cloudfoundry/bosh-agent/agent/applier/bundlecollection"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("FileBundleCollection", func() {
	var (
		fs                   *fakesys.FakeFileSystem
		logger               boshlog.Logger
		fileBundleCollection FileBundleCollection
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		logger = boshlog.NewLogger(boshlog.LevelNone)
		fileBundleCollection = NewFileBundleCollection(
			`C:\fake-collection-path\data`,
			`C:\fake-collection-path`,
			`fake-collection-name`,
			fs,
			logger,
		)
	})

	Describe("Get", func() {
		It("returns the file bundle", func() {
			bundleDefinition := testBundle{
				Name:    "fake-bundle-name",
				Version: "fake-bundle-version",
			}

			fileBundle, err := fileBundleCollection.Get(bundleDefinition)
			Expect(err).NotTo(HaveOccurred())

			expectedBundle := NewFileBundle(
				`C:/fake-collection-path/data/fake-collection-name/fake-bundle-name/fake-bundle-version`,
				`C:/fake-collection-path/fake-collection-name/fake-bundle-name`,
				fs,
				logger,
			)

			Expect(fileBundle).To(Equal(expectedBundle))
		})

		Context("when definition is missing name", func() {
			It("returns error", func() {
				_, err := fileBundleCollection.Get(testBundle{Version: "fake-bundle-version"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Missing bundle name"))
			})
		})

		Context("when definition is missing version", func() {
			It("returns error", func() {
				_, err := fileBundleCollection.Get(testBundle{Name: "fake-bundle-name"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Missing bundle version"))
			})
		})
	})

	Describe("List", func() {
		installPath := `C:\fake-collection-path\data\fake-collection-name`
		enablePath := `C:\fake-collection-path\fake-collection-name`

		It("returns list of installed bundles for windows style paths", func() {
			fs.SetGlob(cleanPath(installPath+`\*\*`), []string{
				installPath + `\fake-bundle-1-name\fake-bundle-1-version-1`,
				installPath + `\fake-bundle-1-name\fake-bundle-1-version-2`,
				installPath + `\fake-bundle-1-name\fake-bundle-2-version-1`,
			})

			bundles, err := fileBundleCollection.List()
			Expect(err).ToNot(HaveOccurred())

			expectedBundles := []Bundle{
				NewFileBundle(
					cleanPath(installPath+`\fake-bundle-1-name\fake-bundle-1-version-1`),
					cleanPath(enablePath+`\fake-bundle-1-name`),
					fs,
					logger,
				),
				NewFileBundle(
					cleanPath(installPath+`\fake-bundle-1-name\fake-bundle-1-version-2`),
					cleanPath(enablePath+`\fake-bundle-1-name`),
					fs,
					logger,
				),
				NewFileBundle(
					cleanPath(installPath+`\fake-bundle-1-name\fake-bundle-2-version-1`),
					cleanPath(enablePath+`\fake-bundle-1-name`),
					fs,
					logger,
				),
			}

			Expect(bundles).To(Equal(expectedBundles))
		})
	})
})

func cleanPath(name string) string {
	return path.Clean(filepath.ToSlash(name))
}
