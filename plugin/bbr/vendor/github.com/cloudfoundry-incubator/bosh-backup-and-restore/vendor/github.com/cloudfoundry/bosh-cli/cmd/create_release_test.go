package cmd_test

import (
	"errors"

	semver "github.com/cppforlife/go-semi-semantic/version"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	boshrel "github.com/cloudfoundry/bosh-cli/release"
	fakerel "github.com/cloudfoundry/bosh-cli/release/releasefakes"
	boshreldir "github.com/cloudfoundry/bosh-cli/releasedir"
	fakereldir "github.com/cloudfoundry/bosh-cli/releasedir/releasedirfakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
	boshtbl "github.com/cloudfoundry/bosh-cli/ui/table"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("CreateReleaseCmd", func() {
	var (
		releaseReader *fakerel.FakeReader
		releaseDir    *fakereldir.FakeReleaseDir
		ui            *fakeui.FakeUI
		fakeFS        *fakesys.FakeFileSystem
		fakeWriter    *fakerel.FakeWriter
		command       CreateReleaseCmd
	)

	BeforeEach(func() {
		releaseReader = &fakerel.FakeReader{}
		releaseDir = &fakereldir.FakeReleaseDir{}

		releaseDirFactory := func(dir DirOrCWDArg) (boshrel.Reader, boshreldir.ReleaseDir) {
			Expect(dir).To(Equal(DirOrCWDArg{Path: "/dir"}))
			return releaseReader, releaseDir
		}

		fakeWriter = &fakerel.FakeWriter{}
		fakeFS = fakesys.NewFakeFileSystem()
		ui = &fakeui.FakeUI{}
		command = NewCreateReleaseCmd(releaseDirFactory, fakeWriter, fakeFS, ui)
	})

	Describe("Run", func() {
		var (
			opts    CreateReleaseOpts
			release *fakerel.FakeRelease
		)

		BeforeEach(func() {
			opts = CreateReleaseOpts{
				Directory: DirOrCWDArg{Path: "/dir"},
			}

			release = &fakerel.FakeRelease{
				NameStub:               func() string { return "rel" },
				VersionStub:            func() string { return "ver" },
				CommitHashWithMarkStub: func(string) string { return "commit" },

				SetNameStub:    func(name string) { release.NameReturns(name) },
				SetVersionStub: func(ver string) { release.VersionReturns(ver) },
			}
		})

		act := func() error {
			_, err := command.Run(opts)
			return err
		}

		Context("when manifest path is provided", func() {
			BeforeEach(func() {
				opts.Args.Manifest = FileBytesWithPathArg{Path: "/manifest-path"}

				releaseReader.ReadStub = func(path string) (boshrel.Release, error) {
					Expect(path).To(Equal("/manifest-path"))
					return release, nil
				}
			})

			It("builds release and release archive based on manifest path", func() {
				err := act()
				Expect(err).ToNot(HaveOccurred())

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

			It("returns error if reading manifest fails", func() {
				releaseReader.ReadReturns(nil, errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			Context("with tarball", func() {
				BeforeEach(func() {
					opts.Tarball = FileArg{ExpandedPath: "/tarball-destination.tgz"}
				})

				It("builds release and release archive based on manifest path", func() {
					fakeWriter.WriteStub = func(rel boshrel.Release, skipPkgs []string) (string, error) {
						Expect(rel).To(Equal(release))

						fakeFS.WriteFileString("/temp-tarball.tgz", "release content blah")
						return "/temp-tarball.tgz", nil
					}

					err := act()
					Expect(err).ToNot(HaveOccurred())

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
								boshtbl.NewValueString("/tarball-destination.tgz"),
							},
						},
						Transpose: true,
					}))

					Expect(fakeFS.FileExists("/temp-tarball.tgz")).To(BeFalse())
					content, err := fakeFS.ReadFileString("/tarball-destination.tgz")
					Expect(err).ToNot(HaveOccurred())
					Expect(content).To(Equal("release content blah"))
				})

				It("returns error if building release archive fails", func() {
					releaseReader.ReadReturns(release, nil)

					fakeWriter.WriteStub = func(rel boshrel.Release, skipPkgs []string) (string, error) {
						Expect(rel).To(Equal(release))
						return "", errors.New("fake-err")
					}

					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-err"))
				})

				It("returns error moving the archive fails", func() {
					fakeWriter.WriteStub = func(rel boshrel.Release, skipPkgs []string) (string, error) {
						fakeFS.WriteFileString("/temp-tarball.tgz", "release content blah")
						return "/temp-tarball.tgz", nil
					}

					fakeFS.RenameError = errors.New("fake-err")

					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-err"))
				})
			})
		})

		Context("when manifest path is not provided", func() {
			It("builds release with default release name and next dev version", func() {
				releaseDir.DefaultNameReturns("default-rel-name", nil)
				releaseDir.NextDevVersionReturns(semver.MustNewVersionFromString("next-dev+ver"), nil)

				releaseDir.BuildReleaseStub = func(name string, version semver.Version, force bool) (boshrel.Release, error) {
					release.SetName(name)
					release.SetVersion(version.String())
					Expect(force).To(BeFalse())
					return release, nil
				}

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(ui.Tables[0]).To(Equal(boshtbl.Table{
					Header: []boshtbl.Header{
						boshtbl.NewHeader("Name"),
						boshtbl.NewHeader("Version"),
						boshtbl.NewHeader("Commit Hash"),
					},
					Rows: [][]boshtbl.Value{
						{
							boshtbl.NewValueString("default-rel-name"),
							boshtbl.NewValueString("next-dev+ver"),
							boshtbl.NewValueString("commit"),
						},
					},
					Transpose: true,
				}))
			})

			It("builds release with custom release name and version", func() {
				opts.Name = "custom-name"
				opts.Version = VersionArg(semver.MustNewVersionFromString("custom-ver"))

				releaseDir.DefaultNameReturns("default-rel-name", nil)
				releaseDir.NextDevVersionReturns(semver.MustNewVersionFromString("1.1"), nil)

				releaseDir.BuildReleaseStub = func(name string, version semver.Version, force bool) (boshrel.Release, error) {
					release.SetName(name)
					release.SetVersion(version.String())
					Expect(force).To(BeFalse())
					return release, nil
				}

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(ui.Tables[0]).To(Equal(boshtbl.Table{
					Header: []boshtbl.Header{
						boshtbl.NewHeader("Name"),
						boshtbl.NewHeader("Version"),
						boshtbl.NewHeader("Commit Hash"),
					},
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

			It("builds release forcefully with timestamp version", func() {
				opts.TimestampVersion = true
				opts.Force = true

				releaseDir.DefaultNameReturns("default-rel-name", nil)

				releaseDir.NextDevVersionStub = func(name string, timestamp bool) (semver.Version, error) {
					Expect(name).To(Equal("default-rel-name"))
					Expect(timestamp).To(BeTrue())
					return semver.MustNewVersionFromString("ts-ver"), nil
				}

				releaseDir.BuildReleaseStub = func(name string, version semver.Version, force bool) (boshrel.Release, error) {
					release.SetName(name)
					release.SetVersion(version.String())
					Expect(force).To(BeTrue())
					return release, nil
				}

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(ui.Tables[0]).To(Equal(boshtbl.Table{
					Header: []boshtbl.Header{
						boshtbl.NewHeader("Name"),
						boshtbl.NewHeader("Version"),
						boshtbl.NewHeader("Commit Hash"),
					},
					Rows: [][]boshtbl.Value{
						{
							boshtbl.NewValueString("default-rel-name"),
							boshtbl.NewValueString("ts-ver"),
							boshtbl.NewValueString("commit"),
						},
					},
					Transpose: true,
				}))
			})

			It("builds and then finalizes release", func() {
				opts.Final = true

				releaseDir.DefaultNameReturns("default-rel-name", nil)
				releaseDir.NextDevVersionReturns(semver.MustNewVersionFromString("next-dev+ver"), nil)
				releaseDir.NextFinalVersionReturns(semver.MustNewVersionFromString("next-final+ver"), nil)

				releaseDir.BuildReleaseStub = func(name string, version semver.Version, force bool) (boshrel.Release, error) {
					release.SetName(name)
					release.SetVersion(version.String())
					Expect(force).To(BeFalse())
					return release, nil
				}

				releaseDir.FinalizeReleaseStub = func(rel boshrel.Release, force bool) error {
					Expect(rel).To(Equal(release))
					Expect(rel.Name()).To(Equal("default-rel-name"))
					Expect(rel.Version()).To(Equal("next-final+ver"))
					Expect(force).To(BeFalse())
					return nil
				}

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(ui.Tables[0]).To(Equal(boshtbl.Table{
					Header: []boshtbl.Header{
						boshtbl.NewHeader("Name"),
						boshtbl.NewHeader("Version"),
						boshtbl.NewHeader("Commit Hash"),
					},
					Rows: [][]boshtbl.Value{
						{
							boshtbl.NewValueString("default-rel-name"),
							boshtbl.NewValueString("next-final+ver"),
							boshtbl.NewValueString("commit"),
						},
					},
					Transpose: true,
				}))
			})

			It("builds and then finalizes release with custom version", func() {
				opts.Final = true
				opts.Version = VersionArg(semver.MustNewVersionFromString("custom-ver"))

				releaseDir.DefaultNameReturns("default-rel-name", nil)
				releaseDir.NextDevVersionReturns(semver.MustNewVersionFromString("1.1"), nil)

				releaseDir.BuildReleaseStub = func(name string, version semver.Version, force bool) (boshrel.Release, error) {
					release.SetName(name)
					release.SetVersion(version.String())
					Expect(force).To(BeFalse())
					return release, nil
				}

				releaseDir.FinalizeReleaseStub = func(rel boshrel.Release, force bool) error {
					Expect(rel).To(Equal(release))
					Expect(rel.Name()).To(Equal("default-rel-name"))
					Expect(rel.Version()).To(Equal("custom-ver"))
					Expect(force).To(BeFalse())
					return nil
				}

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(ui.Tables[0]).To(Equal(boshtbl.Table{
					Header: []boshtbl.Header{
						boshtbl.NewHeader("Name"),
						boshtbl.NewHeader("Version"),
						boshtbl.NewHeader("Commit Hash"),
					},
					Rows: [][]boshtbl.Value{
						{
							boshtbl.NewValueString("default-rel-name"),
							boshtbl.NewValueString("custom-ver"),
							boshtbl.NewValueString("commit"),
						},
					},
					Transpose: true,
				}))
			})

			It("builds release and archive if building archive is requested", func() {
				opts.Final = true
				opts.Tarball = FileArg{ExpandedPath: "/archive-path"}

				releaseDir.DefaultNameReturns("default-rel-name", nil)
				releaseDir.NextDevVersionReturns(semver.MustNewVersionFromString("next-dev+ver"), nil)
				releaseDir.NextFinalVersionReturns(semver.MustNewVersionFromString("next-final+ver"), nil)

				releaseDir.BuildReleaseStub = func(name string, version semver.Version, force bool) (boshrel.Release, error) {
					release.SetName(name)
					release.SetVersion(version.String())
					Expect(force).To(BeFalse())
					return release, nil
				}

				fakeWriter.WriteStub = func(rel boshrel.Release, skipPkgs []string) (string, error) {
					Expect(rel).To(Equal(release))

					fakeFS.WriteFileString("/temp-tarball.tgz", "release content blah")
					return "/temp-tarball.tgz", nil
				}

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(ui.Tables[0]).To(Equal(boshtbl.Table{
					Header: []boshtbl.Header{
						boshtbl.NewHeader("Name"),
						boshtbl.NewHeader("Version"),
						boshtbl.NewHeader("Commit Hash"),
						boshtbl.NewHeader("Archive"),
					},
					Rows: [][]boshtbl.Value{
						{
							boshtbl.NewValueString("default-rel-name"),
							boshtbl.NewValueString("next-final+ver"),
							boshtbl.NewValueString("commit"),
							boshtbl.NewValueString("/archive-path"),
						},
					},
					Transpose: true,
				}))

				Expect(fakeFS.FileExists("/temp-tarball.tgz")).To(BeFalse())
				content, err := fakeFS.ReadFileString("/archive-path")
				Expect(err).ToNot(HaveOccurred())
				Expect(content).To(Equal("release content blah"))
			})

			It("returns error if retrieving default release name fails", func() {
				releaseDir.DefaultNameReturns("", errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("returns error if retrieving next dev version fails", func() {
				releaseDir.NextDevVersionReturns(semver.Version{}, errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("returns error if retrieving next final version fails", func() {
				opts.Final = true

				releaseDir.BuildReleaseReturns(release, nil)
				releaseDir.NextFinalVersionReturns(semver.Version{}, errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("returns error if building release archive fails", func() {
				opts.Tarball = FileArg{ExpandedPath: "/tarball/dest/path.tgz"}

				fakeWriter.WriteStub = func(rel boshrel.Release, skipPkgs []string) (string, error) {
					return "", errors.New("fake-err")
				}

				releaseDir.DefaultNameReturns("default-rel-name", nil)
				releaseDir.NextDevVersionReturns(semver.MustNewVersionFromString("next-dev+ver"), nil)

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})
		})
	})
})
