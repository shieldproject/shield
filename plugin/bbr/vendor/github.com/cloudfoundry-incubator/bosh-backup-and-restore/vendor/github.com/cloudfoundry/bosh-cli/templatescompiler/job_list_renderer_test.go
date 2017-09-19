package templatescompiler_test

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshreljob "github.com/cloudfoundry/bosh-cli/release/job"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
	. "github.com/cloudfoundry/bosh-cli/templatescompiler"
	mock_template "github.com/cloudfoundry/bosh-cli/templatescompiler/mocks"
)

var _ = Describe("JobListRenderer", func() {
	var mockCtrl *gomock.Controller

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	var (
		logger boshlog.Logger

		mockJobRenderer *mock_template.MockJobRenderer

		releaseJobs          []boshreljob.Job
		releaseJobProperties map[string]*biproperty.Map
		jobProperties        biproperty.Map
		globalProperties     biproperty.Map
		deploymentName       string
		address              string

		renderedJobs []*mock_template.MockRenderedJob

		jobListRenderer JobListRenderer

		expectRender1 *gomock.Call
	)

	BeforeEach(func() {
		logger = boshlog.NewLogger(boshlog.LevelNone)
		mockJobRenderer = mock_template.NewMockJobRenderer(mockCtrl)

		// release jobs are just passed through to JobRenderer.Render, so they do not need real contents
		releaseJobs = []boshreljob.Job{
			*boshreljob.NewJob(NewResource("fake-release-job-name-0", "", nil)),
			*boshreljob.NewJob(NewResource("fake-release-job-name-1", "", nil)),
		}

		releaseJobProperties = map[string]*biproperty.Map{
			"fake-release-job-name-0": &biproperty.Map{
				"fake-template-property": "fake-template-property-value",
			},
			"fake-release-job-name-1": &biproperty.Map{},
		}

		jobProperties = biproperty.Map{
			"fake-key": "fake-job-value",
		}

		globalProperties = biproperty.Map{
			"fake-key": "fake-global-value",
		}

		deploymentName = "fake-deployment-name"
		address = "1.2.3.4"

		renderedJobs = []*mock_template.MockRenderedJob{
			mock_template.NewMockRenderedJob(mockCtrl),
			mock_template.NewMockRenderedJob(mockCtrl),
		}

		jobListRenderer = NewJobListRenderer(mockJobRenderer, logger)
	})

	JustBeforeEach(func() {
		mockJobRenderer.EXPECT().Render(releaseJobs[0], releaseJobProperties[releaseJobs[0].Name()], jobProperties, globalProperties, deploymentName, address).Return(renderedJobs[0], nil)
		expectRender1 = mockJobRenderer.EXPECT().Render(releaseJobs[1], releaseJobProperties[releaseJobs[1].Name()], jobProperties, globalProperties, deploymentName, address).Return(renderedJobs[1], nil)
	})

	Describe("Render", func() {
		It("returns a new RenderedJobList with all the RenderedJobs", func() {
			renderedJobList, err := jobListRenderer.Render(releaseJobs, releaseJobProperties, jobProperties, globalProperties, deploymentName, address)
			Expect(err).ToNot(HaveOccurred())
			Expect(renderedJobList.All()).To(Equal([]RenderedJob{
				renderedJobs[0],
				renderedJobs[1],
			}))
		})

		Context("when rendering a job fails", func() {
			JustBeforeEach(func() {
				expectRender1.Return(nil, bosherr.Error("fake-render-error"))
			})

			It("returns an error and cleans up any sucessfully rendered jobs", func() {
				renderedJobs[0].EXPECT().DeleteSilently()

				_, err := jobListRenderer.Render(releaseJobs, releaseJobProperties, jobProperties, globalProperties, deploymentName, address)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-render-error"))
			})
		})
	})

})
