package manifest_test

import (
	"errors"

	"github.com/cloudfoundry/bosh-init/installation/manifest"
	"github.com/cloudfoundry/bosh-init/installation/manifest/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	birelsetmanifest "github.com/cloudfoundry/bosh-init/release/set/manifest"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
)

type manifestFixtures struct {
	validManifest             string
	missingPrivateKeyManifest string
}

var _ = Describe("Parser", func() {
	comboManifestPath := "/path/to/fake-deployment-manifest"
	releaseSetManifest := birelsetmanifest.Manifest{}
	var (
		fakeFs            *fakesys.FakeFileSystem
		fakeUUIDGenerator *fakeuuid.FakeGenerator
		parser            manifest.Parser
		logger            boshlog.Logger
		fakeValidator     *fakes.FakeValidator
		fixtures          manifestFixtures
	)
	BeforeEach(func() {
		fakeValidator = fakes.NewFakeValidator()
		fakeValidator.SetValidateBehavior([]fakes.ValidateOutput{
			{Err: nil},
		})
		fakeFs = fakesys.NewFakeFileSystem()
		logger = boshlog.NewLogger(boshlog.LevelNone)
		fakeUUIDGenerator = fakeuuid.NewFakeGenerator()
		parser = manifest.NewParser(fakeFs, fakeUUIDGenerator, logger, fakeValidator)
		fixtures = manifestFixtures{
			validManifest: `
---
name: fake-deployment-name
cloud_provider:
  template:
    name: fake-cpi-job-name
    release: fake-cpi-release-name
  mbus: http://fake-mbus-user:fake-mbus-password@0.0.0.0:6868
  properties:
    fake-property-name:
      nested-property: fake-property-value
`,
			missingPrivateKeyManifest: `
---
name: fake-deployment-name
cloud_provider:
  template:
    name: fake-cpi-job-name
    release: fake-cpi-release-name
  ssh_tunnel:
    host: 54.34.56.8
    port: 22
    user: fake-ssh-user
    password: fake-password
  mbus: http://fake-mbus-user:fake-mbus-password@0.0.0.0:6868
`,
		}
	})

	Describe("#Parse", func() {
		Context("when combo manifest path does not exist", func() {
			It("returns an error", func() {
				_, err := parser.Parse(comboManifestPath, releaseSetManifest)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when parser fails to read the combo manifest file", func() {
			JustBeforeEach(func() {
				fakeFs.WriteFileString(comboManifestPath, fixtures.validManifest)
				fakeFs.ReadFileError = errors.New("fake-read-file-error")
			})

			It("returns an error", func() {
				_, err := parser.Parse(comboManifestPath, releaseSetManifest)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with a valid manifest", func() {
			BeforeEach(func() {
				fakeFs.WriteFileString(comboManifestPath, fixtures.validManifest)
			})

			It("parses installation from combo manifest", func() {
				installationManifest, err := parser.Parse(comboManifestPath, releaseSetManifest)
				Expect(err).ToNot(HaveOccurred())

				Expect(installationManifest).To(Equal(manifest.Manifest{
					Name: "fake-deployment-name",
					Template: manifest.ReleaseJobRef{
						Name:    "fake-cpi-job-name",
						Release: "fake-cpi-release-name",
					},
					Properties: biproperty.Map{
						"fake-property-name": biproperty.Map{
							"nested-property": "fake-property-value",
						},
					},
					Mbus: "http://fake-mbus-user:fake-mbus-password@0.0.0.0:6868",
				}))
			})
		})

		Context("when ssh tunnel config is present", func() {
			Context("with absolute private_key path", func() {
				BeforeEach(func() {
					fakeFs.WriteFileString(comboManifestPath, `
---
name: fake-deployment-name
cloud_provider:
  template:
    name: fake-cpi-job-name
    release: fake-cpi-release-name
  ssh_tunnel:
    host: 54.34.56.8
    port: 22
    user: fake-ssh-user
    private_key: /tmp/fake-ssh-key.pem
  mbus: http://fake-mbus-user:fake-mbus-password@0.0.0.0:6868
`)
					fakeUUIDGenerator.GeneratedUUID = "fake-uuid"
				})

				It("generates registry config and populates properties in manifest with absolute path for private_key", func() {
					installationManifest, err := parser.Parse(comboManifestPath, releaseSetManifest)
					Expect(err).ToNot(HaveOccurred())

					Expect(installationManifest).To(Equal(manifest.Manifest{
						Name: "fake-deployment-name",
						Template: manifest.ReleaseJobRef{
							Name:    "fake-cpi-job-name",
							Release: "fake-cpi-release-name",
						},
						Properties: biproperty.Map{
							"registry": biproperty.Map{
								"host":     "127.0.0.1",
								"port":     6901,
								"username": "registry",
								"password": "fake-uuid",
							},
						},
						Registry: manifest.Registry{
							SSHTunnel: manifest.SSHTunnel{
								Host:       "54.34.56.8",
								Port:       22,
								User:       "fake-ssh-user",
								PrivateKey: "/tmp/fake-ssh-key.pem",
							},
							Host:     "127.0.0.1",
							Port:     6901,
							Username: "registry",
							Password: "fake-uuid",
						},
						Mbus: "http://fake-mbus-user:fake-mbus-password@0.0.0.0:6868",
					}))
				})
			})

			Context("with relative private_key path", func() {
				BeforeEach(func() {
					fakeFs.WriteFileString(comboManifestPath, `---
name: fake-deployment-name
cloud_provider:
  template:
    name: fake-cpi-job-name
    release: fake-cpi-release-name
  ssh_tunnel:
    host: 54.34.56.8
    port: 22
    user: fake-ssh-user
    private_key: tmp/fake-ssh-key.pem
  mbus: http://fake-mbus-user:fake-mbus-password@0.0.0.0:6868
`)
					fakeUUIDGenerator.GeneratedUUID = "fake-uuid"
				})

				It("generates registry config and populates properties in manifest with expanded path for private_key", func() {
					installationManifest, err := parser.Parse(comboManifestPath, releaseSetManifest)
					Expect(err).ToNot(HaveOccurred())

					Expect(installationManifest).To(Equal(manifest.Manifest{
						Name: "fake-deployment-name",
						Template: manifest.ReleaseJobRef{
							Name:    "fake-cpi-job-name",
							Release: "fake-cpi-release-name",
						},
						Properties: biproperty.Map{
							"registry": biproperty.Map{
								"host":     "127.0.0.1",
								"port":     6901,
								"username": "registry",
								"password": "fake-uuid",
							},
						},
						Registry: manifest.Registry{
							SSHTunnel: manifest.SSHTunnel{
								Host:       "54.34.56.8",
								Port:       22,
								User:       "fake-ssh-user",
								PrivateKey: "/path/to/tmp/fake-ssh-key.pem",
							},
							Host:     "127.0.0.1",
							Port:     6901,
							Username: "registry",
							Password: "fake-uuid",
						},
						Mbus: "http://fake-mbus-user:fake-mbus-password@0.0.0.0:6868",
					}))
				})
			})

			Context("with private_key path beginning with '~'", func() {
				BeforeEach(func() {
					fakeFs.WriteFileString(comboManifestPath, `---
name: fake-deployment-name
cloud_provider:
  template:
    name: fake-cpi-job-name
    release: fake-cpi-release-name
  ssh_tunnel:
    host: 54.34.56.8
    port: 22
    user: fake-ssh-user
    private_key: ~/tmp/fake-ssh-key.pem
  mbus: http://fake-mbus-user:fake-mbus-password@0.0.0.0:6868
`)
					fakeUUIDGenerator.GeneratedUUID = "fake-uuid"
					fakeFs.ExpandPathExpanded = "/Users/foo/tmp/fake-ssh-key.pem"
				})

				It("generates registry config and populates properties in manifest with expanded path for private_key", func() {
					installationManifest, err := parser.Parse(comboManifestPath, releaseSetManifest)
					Expect(err).ToNot(HaveOccurred())

					Expect(installationManifest).To(Equal(manifest.Manifest{
						Name: "fake-deployment-name",
						Template: manifest.ReleaseJobRef{
							Name:    "fake-cpi-job-name",
							Release: "fake-cpi-release-name",
						},
						Properties: biproperty.Map{
							"registry": biproperty.Map{
								"host":     "127.0.0.1",
								"port":     6901,
								"username": "registry",
								"password": "fake-uuid",
							},
						},
						Registry: manifest.Registry{
							SSHTunnel: manifest.SSHTunnel{
								Host:       "54.34.56.8",
								Port:       22,
								User:       "fake-ssh-user",
								PrivateKey: "/Users/foo/tmp/fake-ssh-key.pem",
							},
							Host:     "127.0.0.1",
							Port:     6901,
							Username: "registry",
							Password: "fake-uuid",
						},
						Mbus: "http://fake-mbus-user:fake-mbus-password@0.0.0.0:6868",
					}))
				})
			})

			Context("when expanding to the home directory fails", func() {
				BeforeEach(func() {
					fakeFs.WriteFileString(comboManifestPath, `
---
name: fake-deployment-name
cloud_provider:
  template:
    name: fake-cpi-job-name
    release: fake-cpi-release-name
  ssh_tunnel:
    host: 54.34.56.8
    port: 22
    user: fake-ssh-user
    private_key: ~/tmp/fake-ssh-key.pem
  mbus: http://fake-mbus-user:fake-mbus-password@0.0.0.0:6868
`)
					fakeFs.ExpandPathErr = errors.New("fake-expand-error")
				})

				It("returns an error", func() {
					_, err := parser.Parse(comboManifestPath, releaseSetManifest)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("Expanding private_key path: fake-expand-error"))
				})
			})

			Context("when private_key is not provided", func() {
				BeforeEach(func() {
					fakeFs.WriteFileString(comboManifestPath, fixtures.missingPrivateKeyManifest)
				})

				It("does not expand the path", func() {
					installationManifest, err := parser.Parse(comboManifestPath, releaseSetManifest)
					Expect(err).ToNot(HaveOccurred())

					Expect(installationManifest.Registry.SSHTunnel.PrivateKey).To(Equal(""))
				})
			})
		})

		It("handles installation manifest validation errors", func() {
			fakeFs.WriteFileString(comboManifestPath, fixtures.validManifest)

			fakeValidator.SetValidateBehavior([]fakes.ValidateOutput{
				{Err: errors.New("nope")},
			})

			_, err := parser.Parse(comboManifestPath, releaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Validating installation manifest: nope"))
		})
	})
})
