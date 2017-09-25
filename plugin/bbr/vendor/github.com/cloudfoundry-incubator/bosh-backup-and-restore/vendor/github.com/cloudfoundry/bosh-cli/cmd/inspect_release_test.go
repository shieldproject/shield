package cmd_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
	boshtbl "github.com/cloudfoundry/bosh-cli/ui/table"
)

var _ = Describe("InspectReleaseCmd", func() {
	var (
		ui       *fakeui.FakeUI
		director *fakedir.FakeDirector
		command  InspectReleaseCmd
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		director = &fakedir.FakeDirector{}
		command = NewInspectReleaseCmd(ui, director)
	})

	Describe("Run", func() {
		var (
			opts    InspectReleaseOpts
			release *fakedir.FakeRelease
		)

		BeforeEach(func() {
			opts = InspectReleaseOpts{
				Args: InspectReleaseArgs{
					Slug: boshdir.NewReleaseSlug("some-name", "some-version"),
				},
			}

			release = &fakedir.FakeRelease{}
			director.FindReleaseReturns(release, nil)
		})

		act := func() error { return command.Run(opts) }

		It("shows jobs and packages for specified release", func() {
			release.JobsStub = func() ([]boshdir.Job, error) {
				return []boshdir.Job{
					{
						Name:        "some-job-name",
						Fingerprint: "some-job-fingerprint",

						BlobstoreID: "some-job-blob-id",
						SHA1:        "some-job-blob-sha1",

						LinksConsumed: []boshdir.Link{
							{Name: "some-link"},
						},
						LinksProvided: []boshdir.Link{
							{Name: "some-other-link"},
						},
					},
				}, nil
			}

			release.PackagesStub = func() ([]boshdir.Package, error) {
				return []boshdir.Package{
					{
						Name:        "some-pkg1-name",
						Fingerprint: "some-pkg1-fingerprint",

						BlobstoreID: "some-pkg1-blob-id",
						SHA1:        "some-pkg1-blob-sha1",
					},
					{
						Name:        "some-pkg2-name",
						Fingerprint: "some-pkg2-fingerprint",

						BlobstoreID: "some-pkg2-blob-id",
						SHA1:        "some-pkg2-blob-sha1",

						CompiledPackages: []boshdir.CompiledPackage{
							{
								StemcellSlug: boshdir.NewStemcellSlug(
									"some-stemcell-name",
									"some-stemcell-version",
								),

								BlobstoreID: "some-compiled-pkg-blob-id",
								SHA1:        "some-compiled-pkg-blob-sha1",
							},
						},
					},
				}, nil
			}

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(director.FindReleaseArgsForCall(0)).To(Equal(
				boshdir.NewReleaseSlug("some-name", "some-version")))

			Expect(ui.Tables).To(Equal([]boshtbl.Table{
				{
					Content: "jobs",

					Header: []boshtbl.Header{
						boshtbl.NewHeader("Job"),
						boshtbl.NewHeader("Blobstore ID"),
						boshtbl.NewHeader("Digest"),
						boshtbl.NewHeader("Links Consumed"),
						boshtbl.NewHeader("Links Provided"),
					},

					SortBy: []boshtbl.ColumnSort{{Column: 0, Asc: true}},

					Rows: [][]boshtbl.Value{
						{
							boshtbl.NewValueString("some-job-name/some-job-fingerprint"),
							boshtbl.NewValueString("some-job-blob-id"),
							boshtbl.NewValueString("some-job-blob-sha1"),
							boshtbl.NewValueInterface([]boshdir.Link{{Name: "some-link"}}),
							boshtbl.NewValueInterface([]boshdir.Link{{Name: "some-other-link"}}),
						},
					},
				},
				{
					Content: "packages",

					Header: []boshtbl.Header{
						boshtbl.NewHeader("Package"),
						boshtbl.NewHeader("Compiled for"),
						boshtbl.NewHeader("Blobstore ID"),
						boshtbl.NewHeader("Digest"),
					},

					SortBy: []boshtbl.ColumnSort{{Column: 0, Asc: true}},

					Sections: []boshtbl.Section{
						{
							FirstColumn: boshtbl.NewValueString("some-pkg1-name/some-pkg1-fingerprint"),

							Rows: [][]boshtbl.Value{
								{
									boshtbl.NewValueString(""),
									boshtbl.NewValueString("(source)"),
									boshtbl.NewValueString("some-pkg1-blob-id"),
									boshtbl.NewValueString("some-pkg1-blob-sha1"),
								},
							},
						},
						{
							FirstColumn: boshtbl.NewValueString("some-pkg2-name/some-pkg2-fingerprint"),

							Rows: [][]boshtbl.Value{
								{
									boshtbl.NewValueString(""),
									boshtbl.NewValueString("(source)"),
									boshtbl.NewValueString("some-pkg2-blob-id"),
									boshtbl.NewValueString("some-pkg2-blob-sha1"),
								},
								{
									boshtbl.NewValueString(""),
									boshtbl.NewValueString("some-stemcell-name/some-stemcell-version"),
									boshtbl.NewValueString("some-compiled-pkg-blob-id"),
									boshtbl.NewValueString("some-compiled-pkg-blob-sha1"),
								},
							},
						},
					},
				},
			}))
		})

		It("returns error if jobs cannot be retrieved", func() {
			release.JobsReturns(nil, errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("returns error if jobs cannot be retrieved", func() {
			release.PackagesReturns(nil, errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("returns error if release cannot be retrieved", func() {
			director.FindReleaseReturns(nil, errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
