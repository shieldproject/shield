package cmd_test

import (
	"errors"

	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	semver "github.com/cppforlife/go-semi-semantic/version"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	boshrel "github.com/cloudfoundry/bosh-cli/release"
	boshman "github.com/cloudfoundry/bosh-cli/release/manifest"
	fakerel "github.com/cloudfoundry/bosh-cli/release/releasefakes"
	boshreldir "github.com/cloudfoundry/bosh-cli/releasedir"
	fakereldir "github.com/cloudfoundry/bosh-cli/releasedir/releasedirfakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("UploadReleaseCmd", func() {
	var (
		releaseReader *fakerel.FakeReader
		releaseWriter *fakerel.FakeWriter
		releaseDir    *fakereldir.FakeReleaseDir
		director      *fakedir.FakeDirector
		cmdRunner     *fakesys.FakeCmdRunner
		fs            *fakesys.FakeFileSystem
		archive       *fakedir.FakeReleaseArchive
		ui            *fakeui.FakeUI
		command       UploadReleaseCmd
	)

	BeforeEach(func() {
		releaseReader = &fakerel.FakeReader{}
		releaseDir = &fakereldir.FakeReleaseDir{}

		releaseDirFactory := func(dir DirOrCWDArg) (boshrel.Reader, boshreldir.ReleaseDir) {
			Expect(dir).To(Equal(DirOrCWDArg{Path: "/dir"}))
			return releaseReader, releaseDir
		}

		releaseWriter = &fakerel.FakeWriter{}
		director = &fakedir.FakeDirector{}
		cmdRunner = fakesys.NewFakeCmdRunner()
		fs = fakesys.NewFakeFileSystem()

		archive = &fakedir.FakeReleaseArchive{}

		releaseArchiveFactory := func(path string) boshdir.ReleaseArchive {
			if archive.FileStub == nil {
				archive.FileStub = func() (boshdir.UploadFile, error) {
					return fakesys.NewFakeFile(path, fs), nil
				}
			}
			return archive
		}

		ui = &fakeui.FakeUI{}

		command = NewUploadReleaseCmd(releaseDirFactory, releaseWriter, director, releaseArchiveFactory, cmdRunner, fs, ui)
	})

	Describe("Run", func() {
		var (
			opts UploadReleaseOpts
		)

		BeforeEach(func() {
			opts = UploadReleaseOpts{
				Directory: DirOrCWDArg{Path: "/dir"},
			}
		})

		act := func() error { return command.Run(opts) }

		Context("when url is remote (http/https)", func() {
			BeforeEach(func() {
				opts.Args.URL = "https://some-file.tzg"
			})

			It("uploads given release", func() {
				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(director.UploadReleaseURLCallCount()).To(Equal(1))

				url, sha1, rebase, fix := director.UploadReleaseURLArgsForCall(0)
				Expect(url).To(Equal("https://some-file.tzg"))
				Expect(sha1).To(Equal(""))
				Expect(rebase).To(BeFalse())
				Expect(fix).To(BeFalse())
			})

			It("uploads given release even if reader is nil", func() {
				command = NewUploadReleaseCmd(nil, nil, director, nil, nil, nil, ui)

				err := command.Run(opts)
				Expect(err).ToNot(HaveOccurred())

				Expect(director.UploadReleaseURLCallCount()).To(Equal(1))
			})

			It("uploads given release with a fix flag without checking if release exists", func() {
				opts.Fix = true

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(director.HasReleaseCallCount()).To(Equal(0))

				Expect(director.UploadReleaseURLCallCount()).To(Equal(1))

				url, sha1, rebase, fix := director.UploadReleaseURLArgsForCall(0)
				Expect(url).To(Equal("https://some-file.tzg"))
				Expect(sha1).To(Equal(""))
				Expect(rebase).To(BeFalse())
				Expect(fix).To(BeTrue())
			})

			It("uploads given release with a specified rebase, sha1, etc.", func() {
				opts.Rebase = true
				opts.SHA1 = "sha1"

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(director.UploadReleaseURLCallCount()).To(Equal(1))

				url, sha1, rebase, fix := director.UploadReleaseURLArgsForCall(0)
				Expect(url).To(Equal("https://some-file.tzg"))
				Expect(sha1).To(Equal("sha1"))
				Expect(rebase).To(BeTrue())
				Expect(fix).To(BeFalse())
			})

			It("does not upload release if name and version match existing release", func() {
				opts.Name = "existing-name"
				opts.Version = VersionArg(semver.MustNewVersionFromString("existing-ver"))

				director.HasReleaseReturns(true, nil)

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(director.UploadReleaseURLCallCount()).To(Equal(0))

				name, version := director.HasReleaseArgsForCall(0)
				Expect(name).To(Equal("existing-name"))
				Expect(version).To(Equal("existing-ver"))

				Expect(ui.Said).To(Equal(
					[]string{"Release 'existing-name/existing-ver' already exists."}))
			})

			It("uploads release if name and version does not match existing release", func() {
				opts.Name = "existing-name"
				opts.Version = VersionArg(semver.MustNewVersionFromString("existing-ver"))

				director.HasReleaseReturns(false, nil)

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(director.UploadReleaseURLCallCount()).To(Equal(1))

				url, sha1, rebase, fix := director.UploadReleaseURLArgsForCall(0)
				Expect(url).To(Equal("https://some-file.tzg"))
				Expect(sha1).To(Equal(""))
				Expect(rebase).To(BeFalse())
				Expect(fix).To(BeFalse())

				name, version := director.HasReleaseArgsForCall(0)
				Expect(name).To(Equal("existing-name"))
				Expect(version).To(Equal("existing-ver"))

				Expect(ui.Said).To(BeEmpty())
			})

			It("returns error if checking for release existence fails", func() {
				director.HasReleaseReturns(false, errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))

				Expect(director.UploadReleaseURLCallCount()).To(Equal(0))
			})

			It("returns error if uploading release failed", func() {
				director.UploadReleaseURLReturns(errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})
		})

		Context("when url is a local file (file or no prefix)", func() {
			var (
				release *fakerel.FakeRelease
			)

			BeforeEach(func() {
				opts.Args.URL = "./some-file.tgz"

				release = &fakerel.FakeRelease{
					NameStub: func() string { return "rel" },
					ManifestStub: func() boshman.Manifest {
						return boshman.Manifest{Name: "rel"}
					},
				}
			})

			It("returns an error if reader is nil", func() {
				command = NewUploadReleaseCmd(nil, nil, director, nil, nil, nil, ui)

				err := command.Run(opts)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Cannot upload non-remote release"))
			})

			It("uploads given release", func() {
				releaseReader.ReadStub = func(path string) (boshrel.Release, error) {
					Expect(path).To(Equal("./some-file.tgz"))
					return release, nil
				}

				director.MatchPackagesStub = func(manifest interface{}, compiled bool) ([]string, error) {
					Expect(manifest).To(Equal(boshman.Manifest{Name: "rel"}))
					Expect(compiled).To(BeFalse())
					return []string{"skip-pkg1-fp"}, nil
				}

				releaseWriter.WriteStub = func(rel boshrel.Release, pkgFpsToSkip []string) (string, error) {
					Expect(rel).To(Equal(release))
					Expect(pkgFpsToSkip).To(Equal([]string{"skip-pkg1-fp"}))
					return "/archive-path", nil
				}

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(director.MatchPackagesCallCount()).To(Equal(1))
				Expect(director.UploadReleaseFileCallCount()).To(Equal(1))

				file, rebase, fix := director.UploadReleaseFileArgsForCall(0)
				Expect(file.(*fakesys.FakeFile).Name()).To(Equal("/archive-path"))
				Expect(rebase).To(BeFalse())
				Expect(fix).To(BeFalse())
			})

			It("uploads given release with a fix flag hence does not filter out any packages", func() {
				opts.Fix = true

				releaseReader.ReadStub = func(path string) (boshrel.Release, error) {
					Expect(path).To(Equal("./some-file.tgz"))
					return release, nil
				}

				releaseWriter.WriteStub = func(rel boshrel.Release, pkgFpsToSkip []string) (string, error) {
					Expect(rel).To(Equal(release))
					Expect(pkgFpsToSkip).To(BeEmpty())
					return "/archive-path", nil
				}

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(director.MatchPackagesCallCount()).To(Equal(0))
				Expect(director.UploadReleaseFileCallCount()).To(Equal(1))

				file, rebase, fix := director.UploadReleaseFileArgsForCall(0)
				Expect(file.(*fakesys.FakeFile).Name()).To(Equal("/archive-path"))
				Expect(rebase).To(BeFalse())
				Expect(fix).To(BeTrue())
			})

			It("returns error if opening file fails", func() {
				releaseReader.ReadReturns(release, nil)

				archive.FileStub = func() (boshdir.UploadFile, error) {
					return nil, errors.New("fake-err")
				}

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))

				Expect(director.UploadReleaseFileCallCount()).To(Equal(0))
			})

			It("returns error if uploading release failed", func() {
				releaseReader.ReadReturns(release, nil)
				director.UploadReleaseFileReturns(errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})
		})

		Context("when url is a git repo", func() {
			var (
				release *fakerel.FakeRelease
			)

			BeforeEach(func() {
				// Command's --dir flag is not used
				opts.Args.URL = "git://./some-repo"
				opts.Directory = DirOrCWDArg{Path: "/dir-that-does-not-matter"}

				// Destination for git clone
				fs.TempDirDir = "/dir"

				release = &fakerel.FakeRelease{
					NameStub: func() string { return "rel" },
					ManifestStub: func() boshman.Manifest {
						return boshman.Manifest{Name: "rel"}
					},
				}
			})

			It("returns an error if reader is nil", func() {
				command = NewUploadReleaseCmd(nil, nil, director, nil, cmdRunner, fs, ui)

				err := command.Run(opts)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Cannot upload non-remote release"))
			})

			It("uploads given release", func() {
				opts.Name = "rel1"
				opts.Version = VersionArg(semver.MustNewVersionFromString("1.1"))
				afterClone := false

				cmdRunner.SetCmdCallback("git clone git://./some-repo --depth 1 /dir", func() {
					afterClone = true
				})

				releaseDir.FindReleaseStub = func(name string, version semver.Version) (boshrel.Release, error) {
					Expect(afterClone).To(BeTrue())
					Expect(name).To(Equal("rel1"))
					Expect(version).To(Equal(semver.MustNewVersionFromString("1.1")))
					return release, nil
				}

				director.MatchPackagesStub = func(manifest interface{}, compiled bool) ([]string, error) {
					Expect(manifest).To(Equal(boshman.Manifest{Name: "rel"}))
					Expect(compiled).To(BeFalse())
					return []string{"skip-pkg1-fp"}, nil
				}

				releaseWriter.WriteStub = func(rel boshrel.Release, pkgFpsToSkip []string) (string, error) {
					Expect(rel).To(Equal(release))
					Expect(pkgFpsToSkip).To(Equal([]string{"skip-pkg1-fp"}))
					return "/archive-path", nil
				}

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(director.MatchPackagesCallCount()).To(Equal(1))
				Expect(director.UploadReleaseFileCallCount()).To(Equal(1))

				file, rebase, fix := director.UploadReleaseFileArgsForCall(0)
				Expect(file.(*fakesys.FakeFile).Name()).To(Equal("/archive-path"))
				Expect(rebase).To(BeFalse())
				Expect(fix).To(BeFalse())
			})

			It("uploads given release with a fix flag hence does not filter out any packages", func() {
				opts.Fix = true

				releaseDir.FindReleaseStub = func(name string, version semver.Version) (boshrel.Release, error) {
					Expect(name).To(Equal(""))
					Expect(version).To(Equal(semver.Version{}))
					return release, nil
				}

				releaseWriter.WriteStub = func(rel boshrel.Release, pkgFpsToSkip []string) (string, error) {
					Expect(rel).To(Equal(release))
					Expect(pkgFpsToSkip).To(BeEmpty())
					return "/archive-path", nil
				}

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(director.MatchPackagesCallCount()).To(Equal(0))
				Expect(director.UploadReleaseFileCallCount()).To(Equal(1))

				file, rebase, fix := director.UploadReleaseFileArgsForCall(0)
				Expect(file.(*fakesys.FakeFile).Name()).To(Equal("/archive-path"))
				Expect(rebase).To(BeFalse())
				Expect(fix).To(BeTrue())
			})

			It("does not upload release if name and version match existing release", func() {
				opts.Name = "existing-name"
				opts.Version = VersionArg(semver.MustNewVersionFromString("existing-ver"))

				director.HasReleaseReturns(true, nil)

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(director.UploadReleaseURLCallCount()).To(Equal(0))

				name, version := director.HasReleaseArgsForCall(0)
				Expect(name).To(Equal("existing-name"))
				Expect(version).To(Equal("existing-ver"))

				Expect(ui.Said).To(Equal(
					[]string{"Release 'existing-name/existing-ver' already exists."}))
			})

			It("returns error if opening file fails", func() {
				releaseDir.FindReleaseReturns(release, nil)

				archive.FileStub = func() (boshdir.UploadFile, error) {
					return nil, errors.New("fake-err")
				}

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))

				Expect(director.UploadReleaseFileCallCount()).To(Equal(0))
			})

			It("returns error if creating temporary director failed", func() {
				fs.TempDirError = errors.New("fake-err")

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("returns error if git cloning failed", func() {
				cmdRunner.AddCmdResult("git clone git://./some-repo --depth 1 /dir", fakesys.FakeCmdResult{
					Error: errors.New("fake-err"),
				})

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("returns error if uploading release failed", func() {
				releaseDir.FindReleaseReturns(release, nil)
				director.UploadReleaseFileReturns(errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})
		})

		Context("when url is empty", func() {
			var (
				release *fakerel.FakeRelease
			)

			BeforeEach(func() {
				opts.Args.URL = ""

				release = &fakerel.FakeRelease{
					NameStub: func() string { return "rel" },
					ManifestStub: func() boshman.Manifest {
						return boshman.Manifest{Name: "rel"}
					},
					IsCompiledStub: func() bool { return true },
				}
			})

			It("uploads found release based on name and version", func() {
				opts.Name = "rel1"
				opts.Version = VersionArg(semver.MustNewVersionFromString("1.1"))

				releaseDir.FindReleaseStub = func(name string, version semver.Version) (boshrel.Release, error) {
					Expect(name).To(Equal("rel1"))
					Expect(version).To(Equal(semver.MustNewVersionFromString("1.1")))
					return release, nil
				}

				director.MatchPackagesStub = func(manifest interface{}, compiled bool) ([]string, error) {
					Expect(manifest).To(Equal(boshman.Manifest{Name: "rel"}))
					Expect(compiled).To(BeTrue())
					return []string{"skip-pkg1-fp"}, nil
				}

				releaseWriter.WriteStub = func(rel boshrel.Release, pkgFpsToSkip []string) (string, error) {
					Expect(rel).To(Equal(release))
					Expect(pkgFpsToSkip).To(Equal([]string{"skip-pkg1-fp"}))
					return "/archive-path", nil
				}

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(director.MatchPackagesCallCount()).To(Equal(1))
				Expect(director.UploadReleaseFileCallCount()).To(Equal(1))

				file, rebase, fix := director.UploadReleaseFileArgsForCall(0)
				Expect(file.(*fakesys.FakeFile).Name()).To(Equal("/archive-path"))
				Expect(rebase).To(BeFalse())
				Expect(fix).To(BeFalse())
			})

			It("uploads given release with a fix flag and does not try to repack release", func() {
				opts.Fix = true

				releaseDir.FindReleaseReturns(release, nil)

				releaseWriter.WriteStub = func(rel boshrel.Release, pkgFpsToSkip []string) (string, error) {
					Expect(rel).To(Equal(release))
					Expect(pkgFpsToSkip).To(BeEmpty())
					return "/archive-path", nil
				}

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(director.MatchPackagesCallCount()).To(Equal(0))
				Expect(director.UploadReleaseFileCallCount()).To(Equal(1))

				file, rebase, fix := director.UploadReleaseFileArgsForCall(0)
				Expect(file.(*fakesys.FakeFile).Name()).To(Equal("/archive-path"))
				Expect(rebase).To(BeFalse())
				Expect(fix).To(BeTrue())
			})

			It("returns error if finding release fails", func() {
				releaseDir.FindReleaseReturns(nil, errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))

				Expect(director.UploadReleaseFileCallCount()).To(Equal(0))
			})

			It("returns error if uploading release failed", func() {
				releaseDir.FindReleaseReturns(release, nil)
				director.UploadReleaseFileReturns(errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})
		})
	})
})
