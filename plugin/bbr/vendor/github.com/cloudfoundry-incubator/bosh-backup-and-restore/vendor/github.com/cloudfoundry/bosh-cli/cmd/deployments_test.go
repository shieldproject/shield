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

var _ = Describe("DeploymentsCmd", func() {
	var (
		ui       *fakeui.FakeUI
		director *fakedir.FakeDirector
		command  DeploymentsCmd
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		director = &fakedir.FakeDirector{}
		command = NewDeploymentsCmd(ui, director)
	})

	Describe("Run", func() {
		act := func() error { return command.Run() }

		It("lists deployments", func() {
			deployments := []boshdir.Deployment{
				&fakedir.FakeDeployment{
					NameStub:        func() string { return "dep1" },
					CloudConfigStub: func() (string, error) { return "cloud-config", nil },

					TeamsStub: func() ([]string, error) { return []string{"team1", "team2"}, nil },

					ReleasesStub: func() ([]boshdir.Release, error) {
						return []boshdir.Release{
							&fakedir.FakeRelease{
								NameStub:    func() string { return "rel1" },
								VersionStub: func() semver.Version { return semver.MustNewVersionFromString("rel1-ver") },
							},
							&fakedir.FakeRelease{
								NameStub:    func() string { return "rel2" },
								VersionStub: func() semver.Version { return semver.MustNewVersionFromString("rel2-ver") },
							},
						}, nil
					},

					StemcellsStub: func() ([]boshdir.Stemcell, error) {
						return []boshdir.Stemcell{
							&fakedir.FakeStemcell{
								NameStub:    func() string { return "stemcell1" },
								VersionStub: func() semver.Version { return semver.MustNewVersionFromString("stemcell1-ver") },
							},
							&fakedir.FakeStemcell{
								NameStub:    func() string { return "stemcell2" },
								VersionStub: func() semver.Version { return semver.MustNewVersionFromString("stemcell2-ver") },
							},
						}, nil
					},
				},
			}

			director.DeploymentsReturns(deployments, nil)

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(ui.Table).To(Equal(boshtbl.Table{
				Content: "deployments",

				Header: []boshtbl.Header{
					boshtbl.NewHeader("Name"),
					boshtbl.NewHeader("Release(s)"),
					boshtbl.NewHeader("Stemcell(s)"),
					boshtbl.NewHeader("Team(s)"),
					boshtbl.NewHeader("Cloud Config"),
				},

				SortBy: []boshtbl.ColumnSort{{Column: 0, Asc: true}},

				Rows: [][]boshtbl.Value{
					{
						boshtbl.NewValueString("dep1"),
						boshtbl.NewValueStrings([]string{"rel1/rel1-ver", "rel2/rel2-ver"}),
						boshtbl.NewValueStrings([]string{"stemcell1/stemcell1-ver", "stemcell2/stemcell2-ver"}),
						boshtbl.NewValueStrings([]string{"team1", "team2"}),
						boshtbl.NewValueString("cloud-config"),
					},
				},
			}))
		})

		It("returns error if deployments cannot be retrieved", func() {
			director.DeploymentsReturns(nil, errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
