package release_test

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/deployment/release"
	bireljob "github.com/cloudfoundry/bosh-cli/release/job"
	mock_release "github.com/cloudfoundry/bosh-cli/release/mocks"
	fakerel "github.com/cloudfoundry/bosh-cli/release/releasefakes"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
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
		release            *fakerel.FakeRelease
		jobResolver        JobResolver
	)

	BeforeEach(func() {
		mockReleaseManager = mock_release.NewMockManager(mockCtrl)

		release = &fakerel.FakeRelease{
			NameStub:    func() string { return "rel-name" },
			VersionStub: func() string { return "rel-ver" },
		}

		jobResolver = NewJobResolver(mockReleaseManager)
	})

	Describe("Resolve", func() {
		It("Returns the matching release job", func() {
			job0 := bireljob.NewJob(NewResource("job0", "job0-fp", nil))
			mockReleaseManager.EXPECT().Find("rel-name").Return(release, true)
			release.FindJobByNameStub = func(name string) (bireljob.Job, bool) {
				Expect(name).To(Equal("job0"))
				return *job0, true
			}

			releaseJob, err := jobResolver.Resolve("job0", "rel-name")
			Expect(err).ToNot(HaveOccurred())
			Expect(releaseJob).To(Equal(*job0))
		})

		It("Returns an error, when the job is not in the release", func() {
			mockReleaseManager.EXPECT().Find("rel-name").Return(release, true)
			release.FindJobByNameReturns(bireljob.Job{}, false)

			_, err := jobResolver.Resolve("fake-missing-release-job-name", "rel-name")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Finding job 'fake-missing-release-job-name' in release 'rel-name'"))
		})

		It("Returns an error, when the release is not in resolvable", func() {
			mockReleaseManager.EXPECT().Find("fake-missing-release-name").Return(nil, false)

			_, err := jobResolver.Resolve("job0", "fake-missing-release-name")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Finding release 'fake-missing-release-name'"))
		})
	})
})
