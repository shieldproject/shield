package manifest_test

import (
	. "github.com/cloudfoundry/bosh-cli/installation/manifest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	birelmanifest "github.com/cloudfoundry/bosh-cli/release/manifest"
	birelsetmanifest "github.com/cloudfoundry/bosh-cli/release/set/manifest"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
)

var _ = Describe("Validator", func() {
	var (
		logger             boshlog.Logger
		releaseSetManifest birelsetmanifest.Manifest
		validator          Validator

		releases      []birelmanifest.ReleaseRef
		validManifest Manifest
	)

	BeforeEach(func() {
		logger = boshlog.NewLogger(boshlog.LevelNone)

		releases = []birelmanifest.ReleaseRef{
			{Name: "provided-valid-release-name"},
		}

		validManifest = Manifest{
			Name: "fake-installation-name",
			Template: ReleaseJobRef{
				Name:    "cpi",
				Release: "provided-valid-release-name",
			},
			Properties: biproperty.Map{
				"fake-prop-key": "fake-prop-value",
				"fake-prop-map-key": biproperty.Map{
					"fake-prop-key": "fake-prop-value",
				},
			},
		}

		releaseSetManifest = birelsetmanifest.Manifest{
			Releases: releases,
		}

		validator = NewValidator(logger)
	})

	Describe("Validate", func() {
		It("does not error if deployment is valid", func() {
			manifest := validManifest

			err := validator.Validate(manifest, releaseSetManifest)
			Expect(err).ToNot(HaveOccurred())
		})

		It("validates template must be fully specified", func() {
			manifest := Manifest{}

			err := validator.Validate(manifest, releaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cloud_provider.template.name must be provided"))
			Expect(err.Error()).To(ContainSubstring("cloud_provider.template.release must be provided"))
		})

		It("validates template.name is not blank", func() {
			manifest := Manifest{
				Template: ReleaseJobRef{
					Name: " ",
				},
			}

			err := validator.Validate(manifest, releaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cloud_provider.template.name must be provided"))
		})

		It("validates template.release is not blank", func() {
			manifest := Manifest{
				Template: ReleaseJobRef{
					Release: " ",
				},
			}

			err := validator.Validate(manifest, releaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cloud_provider.template.release must be provided"))
		})

		It("validates the release is available", func() {
			manifest := Manifest{
				Template: ReleaseJobRef{
					Release: "not-provided-valid-release-name",
				},
			}

			err := validator.Validate(manifest, releaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cloud_provider.template.release 'not-provided-valid-release-name' must refer to a release in releases"))
		})
	})
})
