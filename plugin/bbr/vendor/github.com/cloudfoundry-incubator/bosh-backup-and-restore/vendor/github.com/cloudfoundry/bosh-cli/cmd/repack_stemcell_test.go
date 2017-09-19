package cmd_test

import (
	biproperty "github.com/cloudfoundry/bosh-utils/property"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"errors"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	"github.com/cloudfoundry/bosh-cli/stemcell/stemcellfakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("RepackStemcellCmd", func() {
	var (
		fs        *fakesys.FakeFileSystem
		ui        *fakeui.FakeUI
		command   RepackStemcellCmd
		extractor *stemcellfakes.FakeExtractor
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		ui = &fakeui.FakeUI{}

		extractor = stemcellfakes.NewFakeExtractor()
		command = NewRepackStemcellCmd(ui, fs, extractor)
	})

	Describe("Run", func() {
		var (
			opts RepackStemcellOpts
		)

		BeforeEach(func() {
			opts = RepackStemcellOpts{}
		})

		act := func() error { return command.Run(opts) }

		Context("when stemcell path is a local file", func() {
			var (
				extractedStemcell *stemcellfakes.FakeExtractedStemcell
				err               error
			)

			BeforeEach(func() {
				opts.Args.PathToStemcell = "some-stemcell.tgz"
				opts.Args.PathToResult = FileArg{ExpandedPath: "repacked-stemcell.tgz"}
				extractedStemcell = &stemcellfakes.FakeExtractedStemcell{}
			})

			Context("when no flags are passed", func() {
				BeforeEach(func() {
					extractor.SetExtractBehavior("some-stemcell.tgz", extractedStemcell, nil)

					extractedStemcell.PackReturns(nil)
					err = act()
				})

				It("duplicates the stemcell and saves to PathToResult", func() {
					Expect(err).ToNot(HaveOccurred())

					Expect(len(extractor.ExtractInputs)).To(Equal(1))
					Expect(extractor.ExtractInputs[0].TarballPath).To(Equal("some-stemcell.tgz"))

					Expect(extractedStemcell.PackCallCount()).To(Equal(1))
					Expect(extractedStemcell.PackArgsForCall(0)).To(Equal("repacked-stemcell.tgz"))
					extractedStemcell.PackReturns(nil)
				})

				It("should NOT set empty name", func() {
					Expect(err).ToNot(HaveOccurred())

					Expect(extractedStemcell.SetNameCallCount()).To(BeZero())
				})

				It("should NOT set empty version", func() {
					Expect(err).ToNot(HaveOccurred())

					Expect(extractedStemcell.SetVersionCallCount()).To(BeZero())
				})

				It("should NOT set empty cloud_properties", func() {
					Expect(err).ToNot(HaveOccurred())

					Expect(extractedStemcell.SetCloudPropertiesCallCount()).To(BeZero())
				})
			})

			Context("and --name is specfied", func() {
				It("overrides the stemcell name", func() {
					opts.Name = "new-name"
					extractor.SetExtractBehavior("some-stemcell.tgz", extractedStemcell, nil)

					extractedStemcell.PackReturns(nil)
					err = act()
					Expect(err).ToNot(HaveOccurred())

					Expect(extractedStemcell.SetNameCallCount()).To(Equal(1))
					Expect(extractedStemcell.SetNameArgsForCall(0)).To(Equal("new-name"))

					Expect(extractedStemcell.PackCallCount()).To(Equal(1))
				})
			})

			Context("and --version is specfied", func() {
				It("overrides the stemcell version", func() {
					opts.Version = "new-version"
					extractor.SetExtractBehavior("some-stemcell.tgz", extractedStemcell, nil)

					extractedStemcell.PackReturns(nil)
					err = act()
					Expect(err).ToNot(HaveOccurred())

					Expect(extractedStemcell.SetVersionCallCount()).To(Equal(1))
					Expect(extractedStemcell.SetVersionArgsForCall(0)).To(Equal("new-version"))

					Expect(extractedStemcell.PackCallCount()).To(Equal(1))
				})
			})

			Context("and --properties are specfied", func() {
				var (
					err error
				)

				BeforeEach(func() {
					opts.CloudProperties = "new_property: new_value"
					extractor.SetExtractBehavior("some-stemcell.tgz", extractedStemcell, nil)
					extractedStemcell.PackReturns(nil)
				})

				It("overrides the stemcell version", func() {
					err = act()
					Expect(err).ToNot(HaveOccurred())

					Expect(extractedStemcell.SetCloudPropertiesCallCount()).To(Equal(1))
					Expect(extractedStemcell.SetCloudPropertiesArgsForCall(0)).To(Equal(biproperty.Map{
						"new_property": "new_value",
					}))

					Expect(extractedStemcell.PackCallCount()).To(Equal(1))
				})

				Context("and properties are not valid YAML", func() {
					BeforeEach(func() {
						opts.CloudProperties = "not-valid-yaml"
					})

					It("should return an error", func() {
						err = act()
						Expect(err).To(HaveOccurred())
					})
				})
			})

			Context("when error ocurrs", func() {
				Context("when it's NOT able to extract stemcell", func() {
					BeforeEach(func() {
						extractor.SetExtractBehavior("some-stemcell.tgz", nil, errors.New("fake-error"))
						err = act()
					})

					It("returns an error", func() {
						Expect(err).To(HaveOccurred())
					})
				})

				Context("when it's NOT able to create new stemcell", func() {
					BeforeEach(func() {
						extractor.SetExtractBehavior("some-stemcell.tgz", extractedStemcell, nil)
						extractedStemcell.PackReturns(errors.New("fake-error"))
						err = act()
					})

					It("returns an error", func() {
						Expect(err).To(HaveOccurred())

						Expect(len(extractor.ExtractInputs)).To(Equal(1))
						Expect(extractor.ExtractInputs[0].TarballPath).To(Equal("some-stemcell.tgz"))

						Expect(extractedStemcell.PackCallCount()).To(Equal(1))
					})
				})
			})
		})
	})
})
