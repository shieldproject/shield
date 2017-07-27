package deployment_test

import (
	"errors"

	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	biconfig "github.com/cloudfoundry/bosh-cli/config"
	fakebiconfig "github.com/cloudfoundry/bosh-cli/config/fakes"
	. "github.com/cloudfoundry/bosh-cli/deployment"
	boshrel "github.com/cloudfoundry/bosh-cli/release"
	fakerel "github.com/cloudfoundry/bosh-cli/release/releasefakes"
	bistemcell "github.com/cloudfoundry/bosh-cli/stemcell"
)

var _ = Describe("Record", func() {
	var (
		release          *fakerel.FakeRelease
		stemcell         bistemcell.ExtractedStemcell
		deploymentRepo   *fakebiconfig.FakeDeploymentRepo
		releaseRepo      *fakebiconfig.FakeReleaseRepo
		stemcellRepo     *fakebiconfig.FakeStemcellRepo
		deploymentRecord Record
		releases         []boshrel.Release
	)

	BeforeEach(func() {
		release = &fakerel.FakeRelease{
			NameStub:    func() string { return "fake-release-name" },
			VersionStub: func() string { return "fake-release-version" },
		}
		releases = []boshrel.Release{release}
		fakeFS := fakesys.NewFakeFileSystem()
		stemcell = bistemcell.NewExtractedStemcell(
			bistemcell.Manifest{
				Name:    "fake-stemcell-name",
				Version: "fake-stemcell-version",
			},
			"fake-extracted-path",
			nil,
			fakeFS,
		)
		deploymentRepo = fakebiconfig.NewFakeDeploymentRepo()
		releaseRepo = &fakebiconfig.FakeReleaseRepo{}
		stemcellRepo = fakebiconfig.NewFakeStemcellRepo()
		deploymentRecord = NewRecord(deploymentRepo, releaseRepo, stemcellRepo)
	})

	Describe("IsDeployed", func() {
		BeforeEach(func() {
			stemcellRecord := biconfig.StemcellRecord{
				ID:      "fake-stemcell-id",
				Name:    "fake-stemcell-name",
				Version: "fake-stemcell-version",
				CID:     "fake-stemcell-cid",
			}
			stemcellRepo.SetFindCurrentBehavior(stemcellRecord, true, nil)

			deploymentRepo.SetFindCurrentBehavior("fake-manifest-sha1", true, nil)
		})

		Context("when the stemcell and manifest do not change", func() {
			Context("when no release is currently deployed", func() {
				BeforeEach(func() {
					releaseRepo.ListReturns([]biconfig.ReleaseRecord{}, nil)
				})

				It("returns false", func() {
					isDeployed, err := deploymentRecord.IsDeployed("fake-manifest-sha1", releases, stemcell)
					Expect(err).ToNot(HaveOccurred())
					Expect(isDeployed).To(BeFalse())
				})
			})

			Context("when the same release is currently deployed", func() {
				BeforeEach(func() {
					releaseRecords := []biconfig.ReleaseRecord{{
						ID:      "fake-release-id",
						Name:    release.Name(),
						Version: release.Version(),
					}}
					releaseRepo.ListReturns(releaseRecords, nil)
				})

				It("returns true", func() {
					isDeployed, err := deploymentRecord.IsDeployed("fake-manifest-sha1", releases, stemcell)
					Expect(err).ToNot(HaveOccurred())
					Expect(isDeployed).To(BeTrue())
				})
			})

			Context("when a different version of the same release is currently deployed", func() {
				BeforeEach(func() {
					Expect("other-version").ToNot(Equal(release.Version()))
					releaseRecords := []biconfig.ReleaseRecord{{
						ID:      "fake-release-id-2",
						Name:    release.Name(),
						Version: "other-version",
					}}
					releaseRepo.ListReturns(releaseRecords, nil)
				})

				It("returns false", func() {
					isDeployed, err := deploymentRecord.IsDeployed("fake-manifest-sha1", releases, stemcell)
					Expect(err).ToNot(HaveOccurred())
					Expect(isDeployed).To(BeFalse())
				})
			})

			Context("when a same version of a different release is currently deployed", func() {
				BeforeEach(func() {
					Expect("other-release").ToNot(Equal(release.Name()))
					releaseRecords := []biconfig.ReleaseRecord{{
						ID:      "fake-release-id-2",
						Name:    "other-release",
						Version: release.Version(),
					}}
					releaseRepo.ListReturns(releaseRecords, nil)
				})

				It("returns false", func() {
					isDeployed, err := deploymentRecord.IsDeployed("fake-manifest-sha1", releases, stemcell)
					Expect(err).ToNot(HaveOccurred())
					Expect(isDeployed).To(BeFalse())
				})
			})

			Context("when deploying multiple releases", func() {
				var otherRelease *fakerel.FakeRelease

				BeforeEach(func() {
					otherRelease = &fakerel.FakeRelease{
						NameStub:    func() string { return "other-fake-release-name" },
						VersionStub: func() string { return "other-fake-release-version" },
					}
					releaseRecords := []biconfig.ReleaseRecord{
						{
							ID:      "fake-release-id-1",
							Name:    release.Name(),
							Version: release.Version(),
						},
						{
							ID:      "other-fake-release-id-1",
							Name:    otherRelease.Name(),
							Version: otherRelease.Version(),
						},
					}
					releaseRepo.ListReturns(releaseRecords, nil)
				})

				Context("when the same releases are currently deployed", func() {
					Context("(in the same order)", func() {
						BeforeEach(func() {
							releases = []boshrel.Release{
								release,
								otherRelease,
							}
						})

						It("returns true", func() {
							isDeployed, err := deploymentRecord.IsDeployed("fake-manifest-sha1", releases, stemcell)
							Expect(err).ToNot(HaveOccurred())
							Expect(isDeployed).To(BeTrue())
						})
					})

					Context("(in a different order)", func() {
						BeforeEach(func() {
							releases = []boshrel.Release{
								otherRelease,
								release,
							}
						})

						It("returns true", func() {
							isDeployed, err := deploymentRecord.IsDeployed("fake-manifest-sha1", releases, stemcell)
							Expect(err).ToNot(HaveOccurred())
							Expect(isDeployed).To(BeTrue())
						})
					})

					Context("when a superset of releases is currently deployed", func() {
						BeforeEach(func() {
							releases = []boshrel.Release{
								release,
							}
						})

						It("returns false", func() {
							isDeployed, err := deploymentRecord.IsDeployed("fake-manifest-sha1", releases, stemcell)
							Expect(err).ToNot(HaveOccurred())
							Expect(isDeployed).To(BeFalse())
						})
					})
				})
			})
		})

		Context("when no deployment is set", func() {
			BeforeEach(func() {
				deploymentRepo.SetFindCurrentBehavior("", false, nil)
			})

			It("returns false", func() {
				isDeployed, err := deploymentRecord.IsDeployed("fake-manifest-sha1", releases, stemcell)
				Expect(err).ToNot(HaveOccurred())
				Expect(isDeployed).To(BeFalse())
			})
		})

		Context("when a different deployment manifest is currently deployed", func() {
			BeforeEach(func() {
				deploymentRepo.SetFindCurrentBehavior("fake-manifest-sha1-2", true, nil)
			})

			It("returns false", func() {
				isDeployed, err := deploymentRecord.IsDeployed("fake-manifest-sha1", releases, stemcell)
				Expect(err).ToNot(HaveOccurred())
				Expect(isDeployed).To(BeFalse())
			})
		})

		Context("when finding the currently deployed stemcell fails", func() {
			BeforeEach(func() {
				stemcellRepo.SetFindCurrentBehavior(biconfig.StemcellRecord{}, false, errors.New("fake-find-error"))
			})

			It("returns an error", func() {
				_, err := deploymentRecord.IsDeployed("fake-manifest-sha1", releases, stemcell)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-find-error"))
			})
		})

		Context("when no stemcell is currently deployed", func() {
			BeforeEach(func() {
				stemcellRepo.SetFindCurrentBehavior(biconfig.StemcellRecord{}, false, nil)
			})

			It("returns false", func() {
				isDeployed, err := deploymentRecord.IsDeployed("fake-manifest-sha1", releases, stemcell)
				Expect(err).ToNot(HaveOccurred())
				Expect(isDeployed).To(BeFalse())
			})
		})

		Context("when a different stemcell is currently deployed", func() {
			BeforeEach(func() {
				stemcellRecord := biconfig.StemcellRecord{
					ID:      "fake-stemcell-id-2",
					Name:    "fake-stemcell-name-2",
					Version: "fake-stemcell-version-2",
					CID:     "fake-stemcell-cid-2",
				}
				stemcellRepo.SetFindCurrentBehavior(stemcellRecord, true, nil)
			})

			It("returns false", func() {
				isDeployed, err := deploymentRecord.IsDeployed("fake-manifest-sha1", releases, stemcell)
				Expect(err).ToNot(HaveOccurred())
				Expect(isDeployed).To(BeFalse())
			})
		})

		Context("when finding the currently deployed release fails", func() {
			BeforeEach(func() {
				releaseRepo.ListReturns([]biconfig.ReleaseRecord{}, errors.New("fake-find-error"))
			})

			It("returns an error", func() {
				_, err := deploymentRecord.IsDeployed("fake-manifest-sha1", releases, stemcell)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-find-error"))
			})
		})
	})

	Describe("Update", func() {
		It("calculates and updates sha1 of currently deployed manifest", func() {
			err := deploymentRecord.Update("fake-manifest-sha1", releases)
			Expect(err).ToNot(HaveOccurred())
			Expect(deploymentRepo.UpdateCurrentManifestSHA).To(Equal("fake-manifest-sha1"))
		})

		It("passes the releases to the release repo", func() {
			err := deploymentRecord.Update("fake-manifest-path", releases)
			Expect(err).ToNot(HaveOccurred())
			Expect(releaseRepo.UpdateCallCount()).To(Equal(1))
			Expect(releaseRepo.UpdateArgsForCall(0)).To(Equal(releases))
		})

		Context("when updating currently deployed manifest sha1 fails", func() {
			BeforeEach(func() {
				deploymentRepo.UpdateCurrentErr = errors.New("fake-update-error")
			})

			It("returns an error", func() {
				err := deploymentRecord.Update("fake-manifest-sha1", releases)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-update-error"))
			})

			It("does not update the release records", func() {
				deploymentRecord.Update("fake-manifest-sha1", releases)
				Expect(releaseRepo.UpdateCallCount()).To(Equal(0))
			})
		})

		Context("when updating release records fails", func() {
			BeforeEach(func() {
				releaseRepo.UpdateReturns(errors.New("fake-update-error"))
			})

			It("returns an error", func() {
				err := deploymentRecord.Update("fake-manifest-sha1", releases)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-update-error"))
			})
		})
	})

	Describe("Clear", func() {
		It("clears manifest hash", func() {
			deploymentRepo.UpdateCurrentManifestSHA = "initial-sha"

			err := deploymentRecord.Clear()
			Expect(err).ToNot(HaveOccurred())
			Expect(deploymentRepo.UpdateCurrentManifestSHA).To(Equal(""))
		})

		It("clears releases list", func() {
			err := deploymentRecord.Clear()
			Expect(err).ToNot(HaveOccurred())
			Expect(releaseRepo.UpdateCallCount()).To(Equal(1))
			Expect(releaseRepo.UpdateArgsForCall(0)).To(Equal([]boshrel.Release{}))
		})

		Context("when clearing manifest hash fails", func() {
			BeforeEach(func() {
				deploymentRepo.UpdateCurrentErr = errors.New("fake-update-error")
			})

			It("returns an error", func() {
				err := deploymentRecord.Clear()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-update-error"))
			})
		})

		Context("when clearing release records fails", func() {
			BeforeEach(func() {
				releaseRepo.UpdateReturns(errors.New("fake-update-error"))
			})

			It("returns an error", func() {
				err := deploymentRecord.Clear()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-update-error"))
			})
		})
	})
})
