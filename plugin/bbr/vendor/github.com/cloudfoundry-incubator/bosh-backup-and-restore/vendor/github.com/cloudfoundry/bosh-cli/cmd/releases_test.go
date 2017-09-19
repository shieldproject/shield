package cmd_test

import (
	"errors"

	semver "github.com/cppforlife/go-semi-semantic/version"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
	boshtbl "github.com/cloudfoundry/bosh-cli/ui/table"
)

var _ = Describe("ReleasesCmd", func() {
	var (
		ui       *fakeui.FakeUI
		director *fakedir.FakeDirector
		command  ReleasesCmd
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		director = &fakedir.FakeDirector{}
		command = NewReleasesCmd(ui, director)
	})

	Describe("Run", func() {
		act := func() error { return command.Run() }

		It("lists releases", func() {
			releases := []boshdir.Release{
				&fakedir.FakeRelease{
					NameStub: func() string { return "rel1" },
					VersionStub: func() semver.Version {
						return semver.MustNewVersionFromString("rel1-ver1")
					},
					VersionMarkStub:        func(mark string) string { return "rel1-ver1-mark" + mark },
					CommitHashWithMarkStub: func(mark string) string { return "rel1-hash1" + mark },
				},
				&fakedir.FakeRelease{
					NameStub: func() string { return "rel1" },
					VersionStub: func() semver.Version {
						return semver.MustNewVersionFromString("rel1-ver2")
					},
					VersionMarkStub:        func(mark string) string { return "rel1-ver2-mark" + mark },
					CommitHashWithMarkStub: func(mark string) string { return "rel1-hash2" + mark },
				},
				&fakedir.FakeRelease{
					NameStub: func() string { return "rel2" },
					VersionStub: func() semver.Version {
						return semver.MustNewVersionFromString("rel2-ver1")
					},
					VersionMarkStub:        func(string) string { return "rel2-ver1-mark" },
					CommitHashWithMarkStub: func(string) string { return "rel2-hash1" },
				},
				&fakedir.FakeRelease{
					NameStub: func() string { return "rel2" },
					VersionStub: func() semver.Version {
						return semver.MustNewVersionFromString("rel2-ver2")
					},
					VersionMarkStub:        func(string) string { return "rel2-ver2-mark" },
					CommitHashWithMarkStub: func(string) string { return "rel2-hash2" },
				},
			}

			director.ReleasesReturns(releases, nil)

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(ui.Table).To(Equal(boshtbl.Table{
				Content: "releases",

				Header: []boshtbl.Header{
					boshtbl.NewHeader("Name"),
					boshtbl.NewHeader("Version"),
					boshtbl.NewHeader("Commit Hash"),
				},

				SortBy: []boshtbl.ColumnSort{
					{Column: 0, Asc: true},
					{Column: 1},
				},

				Rows: [][]boshtbl.Value{
					{
						boshtbl.NewValueString("rel1"),
						boshtbl.NewValueSuffix(
							boshtbl.NewValueVersion(semver.MustNewVersionFromString("rel1-ver1")),
							"rel1-ver1-mark*",
						),
						boshtbl.NewValueString("rel1-hash1+"),
					},
					{
						boshtbl.NewValueString("rel1"),
						boshtbl.NewValueSuffix(
							boshtbl.NewValueVersion(semver.MustNewVersionFromString("rel1-ver2")),
							"rel1-ver2-mark*",
						),
						boshtbl.NewValueString("rel1-hash2+"),
					},
					{
						boshtbl.NewValueString("rel2"),
						boshtbl.NewValueSuffix(
							boshtbl.NewValueVersion(semver.MustNewVersionFromString("rel2-ver1")),
							"rel2-ver1-mark",
						),
						boshtbl.NewValueString("rel2-hash1"),
					},
					{
						boshtbl.NewValueString("rel2"),
						boshtbl.NewValueSuffix(
							boshtbl.NewValueVersion(semver.MustNewVersionFromString("rel2-ver2")),
							"rel2-ver2-mark",
						),
						boshtbl.NewValueString("rel2-hash2"),
					},
				},

				Notes: []string{
					"(*) Currently deployed",
					"(+) Uncommitted changes",
				},
			}))
		})

		It("returns error if releases cannot be retrieved", func() {
			director.ReleasesReturns(nil, errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
