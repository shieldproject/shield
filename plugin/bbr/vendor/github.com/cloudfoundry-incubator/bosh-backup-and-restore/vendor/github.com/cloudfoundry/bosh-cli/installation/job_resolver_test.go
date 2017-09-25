package installation_test

import (
	"github.com/cloudfoundry/bosh-cli/installation"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	mock_deployment_release "github.com/cloudfoundry/bosh-cli/deployment/release/mocks"
	biinstallmanifest "github.com/cloudfoundry/bosh-cli/installation/manifest"
	bireljob "github.com/cloudfoundry/bosh-cli/release/job"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
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
		mockReleaseJobResolver *mock_deployment_release.MockJobResolver
		resolver               installation.JobResolver
		releaseJob             bireljob.Job
		manifest               biinstallmanifest.Manifest
		expectJobResolve       *gomock.Call
	)

	BeforeEach(func() {
		mockReleaseJobResolver = mock_deployment_release.NewMockJobResolver(mockCtrl)

		manifest = biinstallmanifest.Manifest{
			Name: "fake-installation-name",
			Template: biinstallmanifest.ReleaseJobRef{
				Name:    "fake-cpi-job-name",
				Release: "fake-cpi-release-name",
			},
			Properties: biproperty.Map{
				"fake-installation-property": "fake-installation-property-value",
			},
		}

		job := bireljob.NewJob(NewResource("cpi", "fake-release-job-fingerprint", nil))
		releaseJob = *job
	})

	JustBeforeEach(func() {
		resolver = installation.NewJobResolver(mockReleaseJobResolver)
		expectJobResolve = mockReleaseJobResolver.EXPECT().Resolve("fake-cpi-job-name", "fake-cpi-release-name").Return(releaseJob, nil).AnyTimes()
	})

	Describe("From", func() {
		It("when the release does contain a 'cpi' job returns release jobs", func() {
			jobs, err := resolver.From(manifest)
			Expect(err).ToNot(HaveOccurred())
			Expect(jobs).To(Equal([]bireljob.Job{releaseJob}))
		})

		It("when the release does not contain a 'cpi' job returns an error", func() {
			expectJobResolve.Return(bireljob.Job{}, bosherr.Error("fake-job-resolve-error")).Times(1)
			_, err := resolver.From(manifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-job-resolve-error"))
		})
	})
})
