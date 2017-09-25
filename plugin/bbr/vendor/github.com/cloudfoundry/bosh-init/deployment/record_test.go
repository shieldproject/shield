package deployment_test

import (
	"errors"

	biconfig "github.com/cloudfoundry/bosh-init/config"
	fakebiconfig "github.com/cloudfoundry/bosh-init/config/fakes"
	fakebicrypto "github.com/cloudfoundry/bosh-init/crypto/fakes"
	"github.com/cloudfoundry/bosh-init/release"
	fakebirel "github.com/cloudfoundry/bosh-init/release/fakes"
	bistemcell "github.com/cloudfoundry/bosh-init/stemcell"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-init/deployment"
)

var _ = Describe("Record", rootDesc)

func rootDesc() {
	var (
		fakeRelease        *fakebirel.FakeRelease
		stemcell           bistemcell.ExtractedStemcell
		deploymentRepo     *fakebiconfig.FakeDeploymentRepo
		releaseRepo        *fakebiconfig.FakeReleaseRepo
		stemcellRepo       *fakebiconfig.FakeStemcellRepo
		fakeSHA1Calculator *fakebicrypto.FakeSha1Calculator
		deploymentRecord   Record
		releases           []release.Release
	)

	BeforeEach(func() {
		fakeRelease = &fakebirel.FakeRelease{
			ReleaseName:    "fake-release-name",
			ReleaseVersion: "fake-release-version",
		}
		releases = []release.Release{fakeRelease}
		fakeFS := fakesys.NewFakeFileSystem()
		stemcell = bistemcell.NewExtractedStemcell(
			bistemcell.Manifest{
				Name:    "fake-stemcell-name",
				Version: "fake-stemcell-version",
			},
			"fake-extracted-path",
			fakeFS,
		)
		deploymentRepo = fakebiconfig.NewFakeDeploymentRepo()
		releaseRepo = &fakebiconfig.FakeReleaseRepo{}
		stemcellRepo = fakebiconfig.NewFakeStemcellRepo()
		fakeSHA1Calculator = fakebicrypto.NewFakeSha1Calculator()
		deploymentRecord = NewRecord(deploymentRepo, releaseRepo, stemcellRepo, fakeSHA1Calculator)
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
			fakeSHA1Calculator.SetCalculateBehavior(map[string]fakebicrypto.CalculateInput{
				"fake-manifest-path": fakebicrypto.CalculateInput{
					Sha1: "fake-manifest-sha1",
					Err:  nil,
				},
			})
		})

		Context("when the stemcell and manifest do not change", func() {
			Context("when no release is currently deployed", func() {
				BeforeEach(func() {
					releaseRepo.ListReturns([]biconfig.ReleaseRecord{}, nil)
				})

				It("returns false", func() {
					isDeployed, err := deploymentRecord.IsDeployed("fake-manifest-path", releases, stemcell)
					Expect(err).ToNot(HaveOccurred())
					Expect(isDeployed).To(BeFalse())
				})
			})

			Context("when the same release is currently deployed", func() {
				BeforeEach(func() {
					releaseRecords := []biconfig.ReleaseRecord{{
						ID:      "fake-release-id",
						Name:    fakeRelease.Name(),
						Version: fakeRelease.Version(),
					}}
					releaseRepo.ListReturns(releaseRecords, nil)
				})

				It("returns true", func() {
					isDeployed, err := deploymentRecord.IsDeployed("fake-manifest-path", releases, stemcell)
					Expect(err).ToNot(HaveOccurred())
					Expect(isDeployed).To(BeTrue())
				})
			})

			Context("when a different version of the same release is currently deployed", func() {
				BeforeEach(func() {
					Expect("other-version").ToNot(Equal(fakeRelease.Version()))
					releaseRecords := []biconfig.ReleaseRecord{{
						ID:      "fake-release-id-2",
						Name:    fakeRelease.Name(),
						Version: "other-version",
					}}
					releaseRepo.ListReturns(releaseRecords, nil)
				})

				It("returns false", func() {
					isDeployed, err := deploymentRecord.IsDeployed("fake-manifest-path", releases, stemcell)
					Expect(err).ToNot(HaveOccurred())
					Expect(isDeployed).To(BeFalse())
				})
			})

			Context("when a same version of a different release is currently deployed", func() {
				BeforeEach(func() {
					Expect("other-release").ToNot(Equal(fakeRelease.Name()))
					releaseRecords := []biconfig.ReleaseRecord{{
						ID:      "fake-release-id-2",
						Name:    "other-release",
						Version: fakeRelease.Version(),
					}}
					releaseRepo.ListReturns(releaseRecords, nil)
				})

				It("returns false", func() {
					isDeployed, err := deploymentRecord.IsDeployed("fake-manifest-path", releases, stemcell)
					Expect(err).ToNot(HaveOccurred())
					Expect(isDeployed).To(BeFalse())
				})
			})

			Context("when deploying multiple releases", func() {
				var otherFakeRelease *fakebirel.FakeRelease

				BeforeEach(func() {
					otherFakeRelease = &fakebirel.FakeRelease{
						ReleaseName:    "other-fake-release-name",
						ReleaseVersion: "other-fake-release-version",
					}
					releaseRecords := []biconfig.ReleaseRecord{
						{
							ID:      "fake-release-id-1",
							Name:    fakeRelease.Name(),
							Version: fakeRelease.Version(),
						},
						{
							ID:      "other-fake-release-id-1",
							Name:    otherFakeRelease.Name(),
							Version: otherFakeRelease.Version(),
						},
					}
					releaseRepo.ListReturns(releaseRecords, nil)
				})

				Context("when the same releases are currently deployed", func() {
					Context("(in the same order)", func() {
						BeforeEach(func() {
							releases = []release.Release{
								fakeRelease,
								otherFakeRelease,
							}
						})

						It("returns true", func() {
							isDeployed, err := deploymentRecord.IsDeployed("fake-manifest-path", releases, stemcell)
							Expect(err).ToNot(HaveOccurred())
							Expect(isDeployed).To(BeTrue())
						})
					})

					Context("(in a different order)", func() {
						BeforeEach(func() {
							releases = []release.Release{
								otherFakeRelease,
								fakeRelease,
							}
						})

						It("returns true", func() {
							isDeployed, err := deploymentRecord.IsDeployed("fake-manifest-path", releases, stemcell)
							Expect(err).ToNot(HaveOccurred())
							Expect(isDeployed).To(BeTrue())
						})
					})

					Context("when a superset of releases is currently deployed", func() {
						BeforeEach(func() {
							releases = []release.Release{
								fakeRelease,
							}
						})

						It("returns false", func() {
							isDeployed, err := deploymentRecord.IsDeployed("fake-manifest-path", releases, stemcell)
							Expect(err).ToNot(HaveOccurred())
							Expect(isDeployed).To(BeFalse())
						})
					})
				})
			})
		})

		Context("when getting current deployment manifest sha1 fails", func() {
			BeforeEach(func() {
				deploymentRepo.SetFindCurrentBehavior("fake-manifest-path", true, errors.New("fake-find-error"))
			})

			It("returns an error", func() {
				_, err := deploymentRecord.IsDeployed("fake-manifest-path", releases, stemcell)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-find-error"))
			})
		})

		Context("when no deployment is set", func() {
			BeforeEach(func() {
				deploymentRepo.SetFindCurrentBehavior("", false, nil)
			})

			It("returns false", func() {
				isDeployed, err := deploymentRecord.IsDeployed("fake-manifest-path", releases, stemcell)
				Expect(err).ToNot(HaveOccurred())
				Expect(isDeployed).To(BeFalse())
			})
		})

		Context("when calculating the deployment manifest sha1 fails", func() {
			BeforeEach(func() {
				fakeSHA1Calculator.SetCalculateBehavior(map[string]fakebicrypto.CalculateInput{
					"fake-manifest-path": fakebicrypto.CalculateInput{
						Sha1: "",
						Err:  errors.New("fake-calculate-error"),
					},
				})
			})

			It("returns an error", func() {
				_, err := deploymentRecord.IsDeployed("fake-manifest-path", releases, stemcell)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-calculate-error"))
			})
		})

		Context("when a different deployment manifest is currently deployed", func() {
			BeforeEach(func() {
				deploymentRepo.SetFindCurrentBehavior("fake-manifest-sha1-2", true, nil)
			})

			It("returns false", func() {
				isDeployed, err := deploymentRecord.IsDeployed("fake-manifest-path", releases, stemcell)
				Expect(err).ToNot(HaveOccurred())
				Expect(isDeployed).To(BeFalse())
			})
		})

		Context("when finding the currently deployed stemcell fails", func() {
			BeforeEach(func() {
				stemcellRepo.SetFindCurrentBehavior(biconfig.StemcellRecord{}, false, errors.New("fake-find-error"))
			})

			It("returns an error", func() {
				_, err := deploymentRecord.IsDeployed("fake-manifest-path", releases, stemcell)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-find-error"))
			})
		})

		Context("when no stemcell is currently deployed", func() {
			BeforeEach(func() {
				stemcellRepo.SetFindCurrentBehavior(biconfig.StemcellRecord{}, false, nil)
			})

			It("returns false", func() {
				isDeployed, err := deploymentRecord.IsDeployed("fake-manifest-path", releases, stemcell)
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
				isDeployed, err := deploymentRecord.IsDeployed("fake-manifest-path", releases, stemcell)
				Expect(err).ToNot(HaveOccurred())
				Expect(isDeployed).To(BeFalse())
			})
		})

		Context("when finding the currently deployed release fails", func() {
			BeforeEach(func() {
				releaseRepo.ListReturns([]biconfig.ReleaseRecord{}, errors.New("fake-find-error"))
			})

			It("returns an error", func() {
				_, err := deploymentRecord.IsDeployed("fake-manifest-path", releases, stemcell)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-find-error"))
			})
		})
	})

	Describe("Update", func() {
		BeforeEach(func() {
			fakeSHA1Calculator.SetCalculateBehavior(map[string]fakebicrypto.CalculateInput{
				"fake-manifest-path": fakebicrypto.CalculateInput{
					Sha1: "fake-manifest-sha1",
					Err:  nil,
				},
			})
		})

		It("calculates and updates sha1 of currently deployed manifest", func() {
			err := deploymentRecord.Update("fake-manifest-path", releases)
			Expect(err).ToNot(HaveOccurred())
			Expect(deploymentRepo.UpdateCurrentManifestSHA1).To(Equal("fake-manifest-sha1"))
		})

		It("passes the releases to the release repo", func() {
			err := deploymentRecord.Update("fake-manifest-path", releases)
			Expect(err).ToNot(HaveOccurred())
			Expect(releaseRepo.UpdateCallCount()).To(Equal(1))
			Expect(releaseRepo.UpdateArgsForCall(0)).To(Equal(releases))
		})

		Context("when calculating the deployment manifest sha1 fails", func() {
			BeforeEach(func() {
				fakeSHA1Calculator.SetCalculateBehavior(map[string]fakebicrypto.CalculateInput{
					"fake-manifest-path": fakebicrypto.CalculateInput{
						Sha1: "",
						Err:  errors.New("fake-calculate-error"),
					},
				})
			})

			It("returns an error", func() {
				err := deploymentRecord.Update("fake-manifest-path", releases)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-calculate-error"))
			})

			It("does not update the release records", func() {
				deploymentRecord.Update("fake-manifest-path", releases)
				Expect(releaseRepo.UpdateCallCount()).To(Equal(0))
			})
		})

		Context("when updating currently deployed manifest sha1 fails", func() {
			BeforeEach(func() {
				deploymentRepo.UpdateCurrentErr = errors.New("fake-update-error")
			})

			It("returns an error", func() {
				err := deploymentRecord.Update("fake-manifest-path", releases)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-update-error"))
			})

			It("does not update the release records", func() {
				deploymentRecord.Update("fake-manifest-path", releases)
				Expect(releaseRepo.UpdateCallCount()).To(Equal(0))
			})
		})

		Context("when updating release records fails", func() {
			BeforeEach(func() {
				releaseRepo.UpdateReturns(errors.New("fake-update-error"))
			})

			It("returns an error", func() {
				err := deploymentRecord.Update("fake-manifest-path", releases)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-update-error"))
			})
		})
	})

	Describe("Clear", func() {
		It("clears manifest hash", func() {
			deploymentRepo.UpdateCurrentManifestSHA1 = "initial-sha1"

			err := deploymentRecord.Clear()
			Expect(err).ToNot(HaveOccurred())
			Expect(deploymentRepo.UpdateCurrentManifestSHA1).To(Equal(""))
		})

		It("clears releases list", func() {
			err := deploymentRecord.Clear()
			Expect(err).ToNot(HaveOccurred())
			Expect(releaseRepo.UpdateCallCount()).To(Equal(1))
			Expect(releaseRepo.UpdateArgsForCall(0)).To(Equal([]release.Release{}))
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
}
