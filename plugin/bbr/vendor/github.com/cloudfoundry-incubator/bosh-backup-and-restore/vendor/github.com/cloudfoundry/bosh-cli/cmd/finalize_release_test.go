package cmd_test

import (
	"errors"

	semver "github.com/cppforlife/go-semi-semantic/version"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	boshrel "github.com/cloudfoundry/bosh-cli/release"
	fakerel "github.com/cloudfoundry/bosh-cli/release/releasefakes"
	fakereldir "github.com/cloudfoundry/bosh-cli/releasedir/releasedirfakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
	boshtbl "github.com/cloudfoundry/bosh-cli/ui/table"
)

var _ = Describe("FinalizeReleaseCmd", func() {
	var (
		releaseReader *fakerel.FakeReader
		releaseDir    *fakereldir.FakeReleaseDir
		ui            *fakeui.FakeUI
		command       FinalizeReleaseCmd
	)

	BeforeEach(func() {
		releaseReader = &fakerel.FakeReader{}
		releaseDir = &fakereldir.FakeReleaseDir{}
		ui = &fakeui.FakeUI{}
		command = NewFinalizeReleaseCmd(releaseReader, releaseDir, ui)
	})

	Describe("Run", func() {
		var (
			opts    FinalizeReleaseOpts
			release *fakerel.FakeRelease
		)

		BeforeEach(func() {
			opts = FinalizeReleaseOpts{
				Args: FinalizeReleaseArgs{Path: "/archive-path"},
			}

			release = &fakerel.FakeRelease{
				NameStub:               func() string { return "rel" },
				VersionStub:            func() string { return "ver" },
				CommitHashWithMarkStub: func(string) string { return "commit" },

				SetNameStub:    func(name string) { release.NameReturns(name) },
				SetVersionStub: func(ver string) { release.VersionReturns(ver) },
			}
		})

		act := func() error { return command.Run(opts) }

		It("finalizes release based on path, picking next final version", func() {
			releaseReader.ReadStub = func(path string) (boshrel.Release, error) {
				Expect(path).To(Equal("/archive-path"))
				return release, nil
			}

			releaseDir.NextFinalVersionStub = func(name string) (semver.Version, error) {
				Expect(name).To(Equal("rel"))
				return semver.MustNewVersionFromString("next-final+ver"), nil
			}

			releaseDir.FinalizeReleaseStub = func(rel boshrel.Release, force bool) error {
				Expect(rel).To(Equal(release))
				Expect(rel.Name()).To(Equal("rel"))
				Expect(rel.Version()).To(Equal("next-final+ver"))
				Expect(force).To(BeFalse())
				return nil
			}

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(ui.Tables[0]).To(Equal(boshtbl.Table{
				Header: []boshtbl.Header{boshtbl.NewHeader("Name"), boshtbl.NewHeader("Version"), boshtbl.NewHeader("Commit Hash")},
				Rows: [][]boshtbl.Value{
					{
						boshtbl.NewValueString("rel"),
						boshtbl.NewValueString("next-final+ver"),
						boshtbl.NewValueString("commit"),
					},
				},
				Transpose: true,
			}))
		})

		It("finalizes release based on path, using custom name and version", func() {
			opts.Name = "custom-name"
			opts.Version = VersionArg(semver.MustNewVersionFromString("custom-ver"))

			releaseReader.ReadStub = func(path string) (boshrel.Release, error) {
				Expect(path).To(Equal("/archive-path"))
				return release, nil
			}

			releaseDir.NextFinalVersionStub = func(name string) (semver.Version, error) {
				Expect(name).To(Equal("custom-name"))
				return semver.MustNewVersionFromString("custom-ver"), nil
			}

			releaseDir.FinalizeReleaseStub = func(rel boshrel.Release, force bool) error {
				Expect(rel).To(Equal(release))
				Expect(rel.Name()).To(Equal("custom-name"))
				Expect(rel.Version()).To(Equal("custom-ver"))
				Expect(force).To(BeFalse())
				return nil
			}

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(ui.Tables[0]).To(Equal(boshtbl.Table{
				Header: []boshtbl.Header{boshtbl.NewHeader("Name"), boshtbl.NewHeader("Version"), boshtbl.NewHeader("Commit Hash")},
				Rows: [][]boshtbl.Value{
					{
						boshtbl.NewValueString("custom-name"),
						boshtbl.NewValueString("custom-ver"),
						boshtbl.NewValueString("commit"),
					},
				},
				Transpose: true,
			}))
		})

		It("returns error if reading path fails", func() {
			releaseReader.ReadReturns(nil, errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("returns error if retrieving next final version fails", func() {
			releaseReader.ReadReturns(release, nil)
			releaseDir.NextFinalVersionReturns(semver.Version{}, errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("returns error if finalizing release fails", func() {
			releaseReader.ReadReturns(release, nil)
			releaseDir.FinalizeReleaseReturns(errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
