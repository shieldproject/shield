package manifest_test

import (
	"errors"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
	"github.com/cppforlife/go-patch/patch"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshtpl "github.com/cloudfoundry/bosh-cli/director/template"
	"github.com/cloudfoundry/bosh-cli/installation/manifest"
	"github.com/cloudfoundry/bosh-cli/installation/manifest/fakes"
	birelsetmanifest "github.com/cloudfoundry/bosh-cli/release/set/manifest"
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
				_, err := parser.Parse(comboManifestPath, boshtpl.StaticVariables{}, patch.Ops{}, releaseSetManifest)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when parser fails to read the combo manifest file", func() {
			JustBeforeEach(func() {
				fakeFs.WriteFileString(comboManifestPath, fixtures.validManifest)
				fakeFs.ReadFileError = errors.New("fake-read-file-error")
			})

			It("returns an error", func() {
				_, err := parser.Parse(comboManifestPath, boshtpl.StaticVariables{}, patch.Ops{}, releaseSetManifest)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with a valid manifest", func() {
			BeforeEach(func() {
				fakeFs.WriteFileString(comboManifestPath, fixtures.validManifest)
			})

			It("parses installation from combo manifest", func() {
				installationManifest, err := parser.Parse(comboManifestPath, boshtpl.StaticVariables{}, patch.Ops{}, releaseSetManifest)
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
			Context("with raw private key", func() {
				Context("that is valid", func() {
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
    private_key: |
      -----BEGIN RSA PRIVATE KEY-----
      MIIByQIBAAJhANs/tl5Tv7CD0Gz5TYocWZbGwHIjDU8dY1oszVMb8bhybfF4y88k
      7oaFYlyZ0oZATpx1EGXZAcgDszq5XSXhYKWQL6+u0qEylWsbra7qQefm2+WbZDfh
      ugqbt+kD0F6CjQIDAQABAmBS8yDxQShGBSjnAc9XUHCIvftzc1WGuCytokOwjOMA
      ELMN59DcNzHTTUWwmTXwOwWPnz1c7PYRnFmy99dEcyWeugU0C5QS96XWwGdXcOjY
      Kr1q/yDJZh416/nWkyGlIOECMQDvT36aXqf0xZHb47aEWmeezGS9IyK1BDMEqvcD
      DNU/GK86ymoEqtIyQbnuBUqSbkUCMQDqigydhP7j1IGABdVrWXX/WFhABjAmNWrf
      YYEecgjhjdM83QSkpwu7tYCHtZjny6kCMCZO6GpXurUxJ0823ZHEUxAVkg7A4B5w
      BKa7o30GgeBu2CYmHuCOY8WNxfC3Qh+8rQIwGQIXTkR8GTbzh/8XPpcPaea1oj4G
      rExN1PvElMZ8A/DncTnv4M6fBajYx5+pai3hAjBui9LTgI1fZeOtgBEo+Q3ZLm/O
      bX621YeY03FF5+TCF6Zwk4yT/NWMwJz8Fpb9QQA=
      -----END RSA PRIVATE KEY-----
  mbus: http://fake-mbus-user:fake-mbus-password@0.0.0.0:6868
`)
						fakeUUIDGenerator.GeneratedUUID = "fake-uuid"
					})

					It("sets the raw private key field", func() {
						installationManifest, err := parser.Parse(comboManifestPath, boshtpl.StaticVariables{}, patch.Ops{}, releaseSetManifest)
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
									Host: "54.34.56.8",
									Port: 22,
									User: "fake-ssh-user",
									PrivateKey: `-----BEGIN RSA PRIVATE KEY-----
MIIByQIBAAJhANs/tl5Tv7CD0Gz5TYocWZbGwHIjDU8dY1oszVMb8bhybfF4y88k
7oaFYlyZ0oZATpx1EGXZAcgDszq5XSXhYKWQL6+u0qEylWsbra7qQefm2+WbZDfh
ugqbt+kD0F6CjQIDAQABAmBS8yDxQShGBSjnAc9XUHCIvftzc1WGuCytokOwjOMA
ELMN59DcNzHTTUWwmTXwOwWPnz1c7PYRnFmy99dEcyWeugU0C5QS96XWwGdXcOjY
Kr1q/yDJZh416/nWkyGlIOECMQDvT36aXqf0xZHb47aEWmeezGS9IyK1BDMEqvcD
DNU/GK86ymoEqtIyQbnuBUqSbkUCMQDqigydhP7j1IGABdVrWXX/WFhABjAmNWrf
YYEecgjhjdM83QSkpwu7tYCHtZjny6kCMCZO6GpXurUxJ0823ZHEUxAVkg7A4B5w
BKa7o30GgeBu2CYmHuCOY8WNxfC3Qh+8rQIwGQIXTkR8GTbzh/8XPpcPaea1oj4G
rExN1PvElMZ8A/DncTnv4M6fBajYx5+pai3hAjBui9LTgI1fZeOtgBEo+Q3ZLm/O
bX621YeY03FF5+TCF6Zwk4yT/NWMwJz8Fpb9QQA=
-----END RSA PRIVATE KEY-----
`,
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
				Context("that is invalid", func() {
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
    private_key: |
      -----BEGIN RSA PRIVATE KEY-----
      no valid private key
      -----END RSA PRIVATE KEY-----
  mbus: http://fake-mbus-user:fake-mbus-password@0.0.0.0:6868
`)
						fakeUUIDGenerator.GeneratedUUID = "fake-uuid"
					})

					It("returns an error", func() {
						_, err := parser.Parse(comboManifestPath, boshtpl.StaticVariables{}, patch.Ops{}, releaseSetManifest)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(Equal("Invalid private key for ssh tunnel"))
					})
				})
			})

			Context("with private key path", func() {
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
    private_key: /path/to/fake-ssh-key.pem
  mbus: http://fake-mbus-user:fake-mbus-password@0.0.0.0:6868
`)
						fakeUUIDGenerator.GeneratedUUID = "fake-uuid"
						fakeFs.WriteFileString("/path/to/fake-ssh-key.pem", "--- BEGIN KEY --- blah --- END KEY ---")
					})

					It("generates registry config and populates properties in manifest with absolute path for private_key", func() {
						installationManifest, err := parser.Parse(comboManifestPath, boshtpl.StaticVariables{}, patch.Ops{}, releaseSetManifest)
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
									PrivateKey: "--- BEGIN KEY --- blah --- END KEY ---",
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
						fakeFs.WriteFileString("/path/to/tmp/fake-ssh-key.pem", "--- BEGIN KEY --- blah --- END KEY ---")
					})

					It("generates registry config and populates properties in manifest with expanded path for private_key", func() {
						installationManifest, err := parser.Parse(comboManifestPath, boshtpl.StaticVariables{}, patch.Ops{}, releaseSetManifest)
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
									PrivateKey: "--- BEGIN KEY --- blah --- END KEY ---",
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
						fakeFs.WriteFileString(fakeFs.ExpandPathExpanded, "--- BEGIN KEY --- blah --- END KEY ---")
					})

					It("generates registry config and populates properties in manifest with expanded path for private_key", func() {
						installationManifest, err := parser.Parse(comboManifestPath, boshtpl.StaticVariables{}, patch.Ops{}, releaseSetManifest)
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
									PrivateKey: "--- BEGIN KEY --- blah --- END KEY ---",
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
						_, err := parser.Parse(comboManifestPath, boshtpl.StaticVariables{}, patch.Ops{}, releaseSetManifest)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(Equal("Expanding private_key path: fake-expand-error"))
					})
				})

				Context("when file does not exist", func() {
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
    private_key: /bar/fake-ssh-key.pem
  mbus: http://fake-mbus-user:fake-mbus-password@0.0.0.0:6868
`)
						fakeUUIDGenerator.GeneratedUUID = "fake-uuid"
					})
					It("returns an error", func() {
						_, err := parser.Parse(comboManifestPath, boshtpl.StaticVariables{}, patch.Ops{}, releaseSetManifest)
						Expect(err.Error()).To(ContainSubstring("Reading private key from /bar/fake-ssh-key.pem"))
					})
				})
			})

			Context("when private_key is not provided", func() {
				BeforeEach(func() {
					fakeFs.WriteFileString(comboManifestPath, fixtures.missingPrivateKeyManifest)
				})

				It("does not expand the path", func() {
					installationManifest, err := parser.Parse(comboManifestPath, boshtpl.StaticVariables{}, patch.Ops{}, releaseSetManifest)
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

			_, err := parser.Parse(comboManifestPath, boshtpl.StaticVariables{}, patch.Ops{}, releaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Validating installation manifest: nope"))
		})

		Context("when interpolating variables", func() {
			BeforeEach(func() {
				fakeUUIDGenerator.GeneratedUUID = "fake-uuid"
				fakeFs.ExpandPathExpanded = "/Users/foo/tmp/fake-ssh-key.pem"

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
    private_key: ((url))
  mbus: http://fake-mbus-user:fake-mbus-password@0.0.0.0:6868
`)
				fakeFs.WriteFileString("/Users/foo/tmp/fake-ssh-key.pem", "--- BEGIN KEY --- blah --- END KEY ---")
			})

			It("resolves their values", func() {
				vars := boshtpl.StaticVariables{"url": "~/tmp/fake-ssh-key.pem"}
				ops := patch.Ops{
					patch.ReplaceOp{Path: patch.MustNewPointerFromString("/name"), Value: "replaced-name"},
				}

				installationManifest, err := parser.Parse(comboManifestPath, vars, ops, releaseSetManifest)
				Expect(err).ToNot(HaveOccurred())

				Expect(installationManifest).To(Equal(manifest.Manifest{
					Name: "replaced-name",
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
							PrivateKey: "--- BEGIN KEY --- blah --- END KEY ---",
						},
						Host:     "127.0.0.1",
						Port:     6901,
						Username: "registry",
						Password: "fake-uuid",
					},
					Mbus: "http://fake-mbus-user:fake-mbus-password@0.0.0.0:6868",
				}))
			})

			It("returns an error if variable key is missing", func() {
				_, err := parser.Parse(comboManifestPath, boshtpl.StaticVariables{}, patch.Ops{}, releaseSetManifest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Expected to find variables: url"))
			})
		})
	})
})
