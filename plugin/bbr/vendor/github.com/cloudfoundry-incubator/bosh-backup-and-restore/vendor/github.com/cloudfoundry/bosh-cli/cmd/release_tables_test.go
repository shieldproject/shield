package cmd_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	boshjob "github.com/cloudfoundry/bosh-cli/release/job"
	boshpkg "github.com/cloudfoundry/bosh-cli/release/pkg"
	fakerel "github.com/cloudfoundry/bosh-cli/release/releasefakes"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
	boshtbl "github.com/cloudfoundry/bosh-cli/ui/table"
)

var _ = Describe("ReleaseTables", func() {
	var (
		ui *fakeui.FakeUI
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
	})

	Describe("Print", func() {
		var (
			release *fakerel.FakeRelease
		)

		BeforeEach(func() {
			pkg1 := boshpkg.NewPackage(NewResourceWithBuiltArchive(
				"pkg1-name", "pkg1-fp", "pkg1-path", "pkg1-sha1"), nil)

			pkg2 := boshpkg.NewPackage(NewResourceWithBuiltArchive(
				"pkg2-name", "pkg2-fp", "pkg2-path", "pkg2-sha1"), []string{"pkg1-name"})

			err := pkg2.AttachDependencies([]*boshpkg.Package{pkg1})
			Expect(err).ToNot(HaveOccurred())

			job := boshjob.NewJob(NewResourceWithBuiltArchive(
				"job-name", "job-fp", "job-path", "job-sha1"))

			job.PackageNames = []string{"pkg1-name", "pkg2-name"}

			err = job.AttachPackages([]*boshpkg.Package{pkg1, pkg2})
			Expect(err).ToNot(HaveOccurred())

			release = &fakerel.FakeRelease{
				NameStub:    func() string { return "rel" },
				VersionStub: func() string { return "ver" },

				CommitHashWithMarkStub: func(string) string { return "commit" },

				JobsStub:     func() []*boshjob.Job { return []*boshjob.Job{job} },
				PackagesStub: func() []*boshpkg.Package { return []*boshpkg.Package{pkg1, pkg2} },
			}
		})

		It("shows info about release with archive path", func() {
			ReleaseTables{Release: release, ArchivePath: "/archive-path"}.Print(ui)

			Expect(ui.Tables[0]).To(Equal(boshtbl.Table{
				Header: []boshtbl.Header{
					boshtbl.NewHeader("Name"),
					boshtbl.NewHeader("Version"),
					boshtbl.NewHeader("Commit Hash"),
					boshtbl.NewHeader("Archive"),
				},
				Rows: [][]boshtbl.Value{
					{
						boshtbl.NewValueString("rel"),
						boshtbl.NewValueString("ver"),
						boshtbl.NewValueString("commit"),
						boshtbl.NewValueString("/archive-path"),
					},
				},
				Transpose: true,
			}))

			Expect(ui.Tables[1]).To(Equal(boshtbl.Table{
				Content: "jobs",
				Header: []boshtbl.Header{
					boshtbl.NewHeader("Job"),
					boshtbl.NewHeader("Digest"),
					boshtbl.NewHeader("Packages"),
				},
				SortBy: []boshtbl.ColumnSort{{Column: 0, Asc: true}},
				Rows: [][]boshtbl.Value{
					{
						boshtbl.NewValueString("job-name/job-fp"),
						boshtbl.NewValueString("job-sha1"),
						boshtbl.NewValueStrings([]string{"pkg1-name", "pkg2-name"}),
					},
				},
			}))

			Expect(ui.Tables[2]).To(Equal(boshtbl.Table{
				Content: "packages",
				Header: []boshtbl.Header{
					boshtbl.NewHeader("Package"),
					boshtbl.NewHeader("Digest"),
					boshtbl.NewHeader("Dependencies"),
				},
				SortBy: []boshtbl.ColumnSort{{Column: 0, Asc: true}},
				Rows: [][]boshtbl.Value{
					{
						boshtbl.NewValueString("pkg1-name/pkg1-fp"),
						boshtbl.NewValueString("pkg1-sha1"),
						boshtbl.NewValueStrings(nil),
					},
					{
						boshtbl.NewValueString("pkg2-name/pkg2-fp"),
						boshtbl.NewValueString("pkg2-sha1"),
						boshtbl.NewValueStrings([]string{"pkg1-name"}),
					},
				},
			}))
		})

		It("shows info about release without archive path", func() {
			ReleaseTables{Release: release}.Print(ui)

			Expect(ui.Tables[0]).To(Equal(boshtbl.Table{
				Header: []boshtbl.Header{
					boshtbl.NewHeader("Name"),
					boshtbl.NewHeader("Version"),
					boshtbl.NewHeader("Commit Hash"),
				},
				Rows: [][]boshtbl.Value{
					{
						boshtbl.NewValueString("rel"),
						boshtbl.NewValueString("ver"),
						boshtbl.NewValueString("commit"),
					},
				},
				Transpose: true,
			}))
		})
	})
})
