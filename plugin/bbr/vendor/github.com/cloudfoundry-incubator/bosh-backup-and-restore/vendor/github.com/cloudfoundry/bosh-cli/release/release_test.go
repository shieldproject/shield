package release_test

import (
	"errors"

	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/release"
	boshjob "github.com/cloudfoundry/bosh-cli/release/job"
	boshlic "github.com/cloudfoundry/bosh-cli/release/license"
	boshman "github.com/cloudfoundry/bosh-cli/release/manifest"
	boshpkg "github.com/cloudfoundry/bosh-cli/release/pkg"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
	fakeres "github.com/cloudfoundry/bosh-cli/release/resource/resourcefakes"
)

var _ = Describe("Release", func() {
	var (
		fs      *fakesys.FakeFileSystem
		release Release
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		release = NewRelease(
			"name",
			"version",
			"commit",
			true,
			[]*boshjob.Job{},
			[]*boshpkg.Package{},
			[]*boshpkg.CompiledPackage{},
			&boshlic.License{},
			"extracted-path",
			fs,
		)
	})

	Describe("name and version", func() {
		It("gets and sets name and version", func() {
			Expect(release.Name()).To(Equal("name"))
			Expect(release.Version()).To(Equal("version"))

			release.SetName("new-name")
			release.SetVersion("new-version")

			Expect(release.Name()).To(Equal("new-name"))
			Expect(release.Version()).To(Equal("new-version"))
		})
	})

	Describe("commit hash", func() {
		It("gets and sets commit hash", func() {
			Expect(release.CommitHashWithMark("*")).To(Equal("commit*"))

			release.SetCommitHash("new-commit")
			release.SetUncommittedChanges(false)

			Expect(release.CommitHashWithMark("*")).To(Equal("new-commit"))
		})
	})

	Describe("jobs, packages, compiled packages, and license", func() {
		It("gets", func() {
			jobs := []*boshjob.Job{
				boshjob.NewJob(NewResource("job", "", nil)),
			}

			pkgs := []*boshpkg.Package{
				boshpkg.NewPackage(NewResource("pkg", "", nil), nil),
			}

			compiledPkgs := []*boshpkg.CompiledPackage{
				boshpkg.NewCompiledPackageWithoutArchive(
					"name", "fp", "os-slug", "sha1", []string{"pkg1", "pkg2"}),
			}

			license := boshlic.NewLicense(NewResource("license", "", nil))

			release = NewRelease("", "", "", true, jobs, pkgs, compiledPkgs, license, "", fs)
			Expect(release.Jobs()).To(Equal(jobs))
			Expect(release.Packages()).To(Equal(pkgs))
			Expect(release.CompiledPackages()).To(Equal(compiledPkgs))
			Expect(release.License()).To(Equal(license))

			release = NewRelease("", "", "", true, jobs, pkgs, compiledPkgs, nil, "", fs)
			Expect(release.License()).To(BeNil())
		})
	})

	Describe("IsCompiled", func() {
		It("returns whether or not release has compiled packages", func() {
			compiledPkgs := []*boshpkg.CompiledPackage{
				boshpkg.NewCompiledPackageWithoutArchive(
					"name", "fp", "os-slug", "sha1", []string{"pkg1", "pkg2"}),
			}

			release = NewRelease("", "", "", true, nil, nil, compiledPkgs, nil, "", fs)
			Expect(release.IsCompiled()).To(BeTrue())

			release = NewRelease("", "", "", true, nil, nil, nil, nil, "", fs)
			Expect(release.IsCompiled()).To(BeFalse())
		})
	})

	Describe("FindJobByName", func() {
		It("returns the job and true when the job exists", func() {
			jobs := []*boshjob.Job{
				boshjob.NewJob(NewResource("job1", "job1-fp", nil)),
				boshjob.NewJob(NewResource("job2", "job2-fp", nil)),
			}

			release = NewRelease("", "", "", true, jobs, nil, nil, nil, "", fs)

			job, ok := release.FindJobByName("job2")
			Expect(job).To(Equal(*jobs[1]))
			Expect(ok).To(BeTrue())
		})

		It("returns nil and false when the job does not exist", func() {
			jobs := []*boshjob.Job{
				boshjob.NewJob(NewResource("job1", "job1-fp", nil)),
			}

			release = NewRelease("", "", "", true, jobs, nil, nil, nil, "", fs)

			_, ok := release.FindJobByName("job2")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("Manifest", func() {
		It("returns manifest", func() {
			jobs := []*boshjob.Job{
				boshjob.NewJob(NewExistingResource("job", "job-fp", "job-sha1")),
			}

			pkgs := []*boshpkg.Package{
				boshpkg.NewPackage(NewExistingResource("pkg", "pkg-fp", "pkg-sha1"), []string{"pkg1"}),
			}

			compiledPkgs := []*boshpkg.CompiledPackage{
				boshpkg.NewCompiledPackageWithoutArchive(
					"cp", "cp-fp", "cp-os-slug", "cp-sha1", []string{"pkg1", "pkg2"}),
			}

			license := boshlic.NewLicense(NewExistingResource("license", "lic-fp", "lic-sha1"))

			release = NewRelease("name", "ver", "commit", true, jobs, pkgs, compiledPkgs, license, "", fs)

			Expect(release.Manifest()).To(Equal(boshman.Manifest{
				Name:               "name",
				Version:            "ver",
				CommitHash:         "commit",
				UncommittedChanges: true,
				Jobs: []boshman.JobRef{
					{
						Name:        "job",
						Version:     "job-fp",
						Fingerprint: "job-fp",
						SHA1:        "job-sha1",
					},
				},
				Packages: []boshman.PackageRef{
					{
						Name:         "pkg",
						Version:      "pkg-fp",
						Fingerprint:  "pkg-fp",
						SHA1:         "pkg-sha1",
						Dependencies: []string{"pkg1"},
					},
				},
				CompiledPkgs: []boshman.CompiledPackageRef{
					{
						Name:          "cp",
						Version:       "cp-fp",
						Fingerprint:   "cp-fp",
						SHA1:          "cp-sha1",
						OSVersionSlug: "cp-os-slug",
						Dependencies:  []string{"pkg1", "pkg2"},
					},
				},
				License: &boshman.LicenseRef{
					Version:     "lic-fp",
					Fingerprint: "lic-fp",
					SHA1:        "lic-sha1",
				},
			}))
		})

		It("does not include license if it's not set", func() {
			release = NewRelease("name", "ver", "commit", true, nil, nil, nil, nil, "", fs)
			Expect(release.Manifest().License).To(BeNil())
		})
	})

	Describe("Build", func() {
		It("builds jobs, packages, and license", func() {
			jobRes := &fakeres.FakeResource{}
			jobs := []*boshjob.Job{boshjob.NewJob(jobRes)}

			pkgRes := &fakeres.FakeResource{}
			pkgs := []*boshpkg.Package{boshpkg.NewPackage(pkgRes, nil)}

			licRes := &fakeres.FakeResource{}
			lic := boshlic.NewLicense(licRes)

			release = NewRelease("", "", "", true, jobs, pkgs, nil, lic, "", fs)

			devJobs := &fakeres.FakeArchiveIndex{}
			devPkgs := &fakeres.FakeArchiveIndex{}
			devLic := &fakeres.FakeArchiveIndex{}
			devIndicies := ArchiveIndicies{devJobs, devPkgs, devLic}

			finalJobs := &fakeres.FakeArchiveIndex{}
			finalPkgs := &fakeres.FakeArchiveIndex{}
			finalLic := &fakeres.FakeArchiveIndex{}
			finalIndicies := ArchiveIndicies{finalJobs, finalPkgs, finalLic}

			Expect(release.Build(devIndicies, finalIndicies)).ToNot(HaveOccurred())

			// Use == for pointer equality
			Expect(jobRes.BuildCallCount()).To(Equal(1))
			dev, final := jobRes.BuildArgsForCall(0)
			Expect(dev == devJobs).To(BeTrue())
			Expect(final == finalJobs).To(BeTrue())

			Expect(pkgRes.BuildCallCount()).To(Equal(1))
			dev, final = pkgRes.BuildArgsForCall(0)
			Expect(dev == devPkgs).To(BeTrue())
			Expect(final == finalPkgs).To(BeTrue())

			Expect(licRes.BuildCallCount()).To(Equal(1))
			dev, final = licRes.BuildArgsForCall(0)
			Expect(dev == devLic).To(BeTrue())
			Expect(final == finalLic).To(BeTrue())
		})

		It("does nothing when there is nothing to build", func() {
			release = NewRelease("", "", "", true, nil, nil, nil, nil, "", fs)
			Expect(release.Build(ArchiveIndicies{}, ArchiveIndicies{})).ToNot(HaveOccurred())
		})

		It("returns error if job building fails", func() {
			jobRes := &fakeres.FakeResource{}
			jobs := []*boshjob.Job{boshjob.NewJob(jobRes)}

			jobRes.BuildReturns(errors.New("fake-err"))
			release = NewRelease("", "", "", true, jobs, nil, nil, nil, "", fs)

			err := release.Build(ArchiveIndicies{}, ArchiveIndicies{})
			Expect(err).To(Equal(errors.New("fake-err")))
		})

		It("returns error if package building fails", func() {
			pkgRes := &fakeres.FakeResource{}
			pkgs := []*boshpkg.Package{boshpkg.NewPackage(pkgRes, nil)}

			pkgRes.BuildReturns(errors.New("fake-err"))
			release = NewRelease("", "", "", true, nil, pkgs, nil, nil, "", fs)

			err := release.Build(ArchiveIndicies{}, ArchiveIndicies{})
			Expect(err).To(Equal(errors.New("fake-err")))
		})

		It("returns error if license building fails", func() {
			licRes := &fakeres.FakeResource{}
			lic := boshlic.NewLicense(licRes)

			licRes.BuildReturns(errors.New("fake-err"))
			release = NewRelease("", "", "", true, nil, nil, nil, lic, "", fs)

			err := release.Build(ArchiveIndicies{}, ArchiveIndicies{})
			Expect(err).To(Equal(errors.New("fake-err")))
		})
	})

	Describe("Finalize", func() {
		It("builds jobs, packages, and license", func() {
			jobRes := &fakeres.FakeResource{}
			jobs := []*boshjob.Job{boshjob.NewJob(jobRes)}

			pkgRes := &fakeres.FakeResource{}
			pkgs := []*boshpkg.Package{boshpkg.NewPackage(pkgRes, nil)}

			licRes := &fakeres.FakeResource{}
			lic := boshlic.NewLicense(licRes)

			release = NewRelease("", "", "", true, jobs, pkgs, nil, lic, "", fs)

			finalJobs := &fakeres.FakeArchiveIndex{}
			finalPkgs := &fakeres.FakeArchiveIndex{}
			finalLic := &fakeres.FakeArchiveIndex{}
			finalIndicies := ArchiveIndicies{finalJobs, finalPkgs, finalLic}

			Expect(release.Finalize(finalIndicies)).ToNot(HaveOccurred())

			// Use == for pointer equality
			Expect(jobRes.FinalizeCallCount()).To(Equal(1))
			Expect(jobRes.FinalizeArgsForCall(0) == finalJobs).To(BeTrue())

			Expect(pkgRes.FinalizeCallCount()).To(Equal(1))
			Expect(pkgRes.FinalizeArgsForCall(0) == finalPkgs).To(BeTrue())

			Expect(licRes.FinalizeCallCount()).To(Equal(1))
			Expect(licRes.FinalizeArgsForCall(0) == finalLic).To(BeTrue())
		})

		It("does nothing when there is nothing to finalize", func() {
			release = NewRelease("", "", "", true, nil, nil, nil, nil, "", fs)
			Expect(release.Finalize(ArchiveIndicies{})).ToNot(HaveOccurred())
		})

		It("returns error if job finalizing fails", func() {
			jobRes := &fakeres.FakeResource{}
			jobs := []*boshjob.Job{boshjob.NewJob(jobRes)}

			jobRes.FinalizeReturns(errors.New("fake-err"))
			release = NewRelease("", "", "", true, jobs, nil, nil, nil, "", fs)

			err := release.Finalize(ArchiveIndicies{})
			Expect(err).To(Equal(errors.New("fake-err")))
		})

		It("returns error if package finalizing fails", func() {
			pkgRes := &fakeres.FakeResource{}
			pkgs := []*boshpkg.Package{boshpkg.NewPackage(pkgRes, nil)}

			pkgRes.FinalizeReturns(errors.New("fake-err"))
			release = NewRelease("", "", "", true, nil, pkgs, nil, nil, "", fs)

			err := release.Finalize(ArchiveIndicies{})
			Expect(err).To(Equal(errors.New("fake-err")))
		})

		It("returns error if license finalizing fails", func() {
			licRes := &fakeres.FakeResource{}
			lic := boshlic.NewLicense(licRes)

			licRes.FinalizeReturns(errors.New("fake-err"))
			release = NewRelease("", "", "", true, nil, nil, nil, lic, "", fs)

			err := release.Finalize(ArchiveIndicies{})
			Expect(err).To(Equal(errors.New("fake-err")))
		})
	})

	Describe("CleanUp", func() {
		It("cleans up jobs, packages", func() {
			jobs := []*boshjob.Job{boshjob.NewJob(&fakeres.FakeResource{})}
			pkgs := []*boshpkg.Package{boshpkg.NewPackage(&fakeres.FakeResource{}, nil)}
			release = NewRelease("", "", "", true, jobs, pkgs, nil, nil, "", fs)
			Expect(release.CleanUp()).ToNot(HaveOccurred())
		})

		It("does nothing when there is nothing to clean up", func() {
			release = NewRelease("", "", "", true, nil, nil, nil, nil, "", fs)
			Expect(release.CleanUp()).ToNot(HaveOccurred())
		})
	})
})
