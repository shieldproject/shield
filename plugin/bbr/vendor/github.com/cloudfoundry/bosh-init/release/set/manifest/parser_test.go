package manifest_test

import (
	"errors"

	birelmanifest "github.com/cloudfoundry/bosh-init/release/manifest"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/bosh-init/release/set/manifest"
	"github.com/cloudfoundry/bosh-init/release/set/manifest/fakes"
)

var _ = Describe("Parser", func() {
	comboManifestPath := "/path/to/manifest/fake-deployment-manifest"
	var (
		fakeFs        *fakesys.FakeFileSystem
		parser        manifest.Parser
		fakeValidator *fakes.FakeValidator
	)

	BeforeEach(func() {
		fakeFs = fakesys.NewFakeFileSystem()
		logger := boshlog.NewLogger(boshlog.LevelNone)
		fakeValidator = fakes.NewFakeValidator()
		fakeValidator.SetValidateBehavior([]fakes.ValidateOutput{
			{Err: nil},
		})
		parser = manifest.NewParser(fakeFs, logger, fakeValidator)
		fakeFs.WriteFileString(comboManifestPath, `
---
releases:
- name: fake-release-name-1
  url: file://~/absolute-path/fake-release-1.tgz
  sha1: fake-sha1
- name: fake-release-name-2
  url: file:///absolute-path/fake-release-2.tgz
  sha1: fake-sha2
- name: fake-release-name-3
  url: file://relative-path/fake-release-3.tgz
  sha1: fake-sha3
- name: fake-release-name-4
  url: http://fake-url/fake-release-4.tgz
  sha1: fake-sha4
name: unknown-keys-are-ignored
`)
	})

	Context("when combo manifest path does not exist", func() {
		BeforeEach(func() {
			err := fakeFs.RemoveAll(comboManifestPath)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error", func() {
			_, err := parser.Parse(comboManifestPath)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when parser fails to read the combo manifest file", func() {
		BeforeEach(func() {
			fakeFs.ReadFileError = errors.New("fake-read-file-error")
		})

		It("returns an error", func() {
			_, err := parser.Parse(comboManifestPath)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when release url points to a local file", func() {
		Context("when release file path begins with 'file://~' or 'file:///'", func() {
			BeforeEach(func() {
				fakeFs.WriteFileString(comboManifestPath, `
---
releases:
- name: fake-release-name-1
  url: file://~/absolute-path/fake-release-1.tgz
  sha1: fake-sha1
- name: fake-release-name-2
  url: file:///absolute-path/fake-release-2.tgz
  sha1: fake-sha2
`)
			})

			It("does not change release url", func() {
				deploymentManifest, err := parser.Parse(comboManifestPath)
				Expect(err).ToNot(HaveOccurred())

				Expect(deploymentManifest).To(Equal(manifest.Manifest{
					Releases: []birelmanifest.ReleaseRef{
						{
							Name: "fake-release-name-1",
							URL:  "file://~/absolute-path/fake-release-1.tgz",
							SHA1: "fake-sha1",
						},
						{
							Name: "fake-release-name-2",
							URL:  "file:///absolute-path/fake-release-2.tgz",
							SHA1: "fake-sha2",
						},
					},
				}))
			})
		})

		Context("when release file path does not begin with 'file://~' or 'file:///'", func() {
			BeforeEach(func() {
				fakeFs.WriteFileString(comboManifestPath, `
---
releases:
- name: fake-release-name-3
  url: file://relative-path/fake-release-3.tgz
  sha1: fake-sha3
`)
			})

			It("changes release url to include absolute path to manifest directory", func() {
				deploymentManifest, err := parser.Parse(comboManifestPath)
				Expect(err).ToNot(HaveOccurred())

				Expect(deploymentManifest).To(Equal(manifest.Manifest{
					Releases: []birelmanifest.ReleaseRef{
						{
							Name: "fake-release-name-3",
							URL:  "file:///path/to/manifest/relative-path/fake-release-3.tgz",
							SHA1: "fake-sha3",
						},
					},
				}))
			})
		})
	})
	Context("when release url points to an http url", func() {
		BeforeEach(func() {
			fakeFs.WriteFileString(comboManifestPath, `
---
releases:
- name: fake-release-name-4
  url: http://fake-url/fake-release-4.tgz
  sha1: fake-sha4
`)
		})

		It("does not change the release url", func() {
			deploymentManifest, err := parser.Parse(comboManifestPath)
			Expect(err).ToNot(HaveOccurred())

			Expect(deploymentManifest).To(Equal(manifest.Manifest{
				Releases: []birelmanifest.ReleaseRef{
					{
						Name: "fake-release-name-4",
						URL:  "http://fake-url/fake-release-4.tgz",
						SHA1: "fake-sha4",
					},
				},
			}))
		})
	})

	It("parses release set manifest from combo manifest file", func() {
		deploymentManifest, err := parser.Parse(comboManifestPath)
		Expect(err).ToNot(HaveOccurred())

		Expect(deploymentManifest).To(Equal(manifest.Manifest{
			Releases: []birelmanifest.ReleaseRef{
				{
					Name: "fake-release-name-1",
					URL:  "file://~/absolute-path/fake-release-1.tgz",
					SHA1: "fake-sha1",
				},
				{
					Name: "fake-release-name-2",
					URL:  "file:///absolute-path/fake-release-2.tgz",
					SHA1: "fake-sha2",
				},
				{
					Name: "fake-release-name-3",
					URL:  "file:///path/to/manifest/relative-path/fake-release-3.tgz",
					SHA1: "fake-sha3",
				},
				{
					Name: "fake-release-name-4",
					URL:  "http://fake-url/fake-release-4.tgz",
					SHA1: "fake-sha4",
				},
			},
		}))
	})

	It("handles errors validating the release set manifest", func() {
		fakeValidator.SetValidateBehavior([]fakes.ValidateOutput{
			{Err: errors.New("couldn't validate that")},
		})

		_, err := parser.Parse(comboManifestPath)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Validating release set manifest: couldn't validate that"))
	})
})
