package release_test

import (
	. "github.com/cloudfoundry/bosh-init/deployment/release"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	bireljob "github.com/cloudfoundry/bosh-init/release/job"
	"github.com/golang/mock/gomock"

	fake_release "github.com/cloudfoundry/bosh-init/release/fakes"
	mock_release "github.com/cloudfoundry/bosh-init/release/mocks"
)

var _ = Describe("JobResolver", func() {
	var mockCtrl *gomock.Controller

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	var (
		mockReleaseManager *mock_release.MockManager
		fakeRelease        *fake_release.FakeRelease

		fakeReleaseJob0 bireljob.Job
		fakeReleaseJob1 bireljob.Job

		jobResolver JobResolver
	)

	BeforeEach(func() {
		mockReleaseManager = mock_release.NewMockManager(mockCtrl)

		fakeRelease = fake_release.New("fake-release-name", "fake-release-version")

		fakeReleaseJob0 = bireljob.Job{
			Name:        "fake-release-job-name-0",
			Fingerprint: "fake-release-job-fingerprint-0",
		}
		fakeReleaseJob1 = bireljob.Job{
			Name:        "fake-release-job-name-1",
			Fingerprint: "fake-release-job-fingerprint-1",
		}
	})

	JustBeforeEach(func() {
		jobResolver = NewJobResolver(mockReleaseManager)

		fakeRelease.ReleaseJobs = []bireljob.Job{fakeReleaseJob0, fakeReleaseJob1}
	})

	Describe("Resolve", func() {
		It("Returns the matching release job", func() {
			mockReleaseManager.EXPECT().Find("fake-release-name").Return(fakeRelease, true)

			releaseJob, err := jobResolver.Resolve("fake-release-job-name-0", "fake-release-name")
			Expect(err).ToNot(HaveOccurred())
			Expect(releaseJob).To(Equal(fakeReleaseJob0))
		})

		It("Returns an error, when the job is not in the release", func() {
			mockReleaseManager.EXPECT().Find("fake-release-name").Return(fakeRelease, true)

			_, err := jobResolver.Resolve("fake-missing-release-job-name", "fake-release-name")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Finding job 'fake-missing-release-job-name' in release 'fake-release-name'"))
		})

		It("Returns an error, when the release is not in resolvable", func() {
			mockReleaseManager.EXPECT().Find("fake-missing-release-name").Return(nil, false)

			_, err := jobResolver.Resolve("fake-release-job-name-0", "fake-missing-release-name")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Finding release 'fake-missing-release-name'"))
		})
	})
})
