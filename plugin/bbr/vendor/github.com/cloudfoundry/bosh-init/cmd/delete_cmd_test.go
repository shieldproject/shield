package cmd_test

import (
	bicmd "github.com/cloudfoundry/bosh-init/cmd"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	mock_cmd "github.com/cloudfoundry/bosh-init/cmd/mocks"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	"github.com/golang/mock/gomock"

	fakebiui "github.com/cloudfoundry/bosh-init/ui/fakes"
	fakeui "github.com/cloudfoundry/bosh-init/ui/fakes"
)

var _ = Describe("DeleteCmd", func() {
	var mockCtrl *gomock.Controller

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Run", func() {
		var (
			mockDeploymentDeleter *mock_cmd.MockDeploymentDeleter
			fs                    *fakesys.FakeFileSystem
			logger                boshlog.Logger

			fakeUI                 *fakeui.FakeUI
			fakeStage              *fakebiui.FakeStage
			deploymentManifestPath = "/deployment-dir/fake-deployment-manifest.yml"
		)

		var newDeleteCmd = func() bicmd.Cmd {
			doGetFunc := func(deploymentManifestPath string) (bicmd.DeploymentDeleter, error) {
				Expect(deploymentManifestPath).To(Equal(deploymentManifestPath))
				return mockDeploymentDeleter, nil
			}

			return bicmd.NewDeleteCmd(fakeUI, fs, logger, doGetFunc)
		}

		var writeDeploymentManifest = func() {
			fs.WriteFileString(deploymentManifestPath, `---manifest-content`)
		}

		BeforeEach(func() {
			mockDeploymentDeleter = mock_cmd.NewMockDeploymentDeleter(mockCtrl)
			fs = fakesys.NewFakeFileSystem()
			fs.EnableStrictTempRootBehavior()
			logger = boshlog.NewLogger(boshlog.LevelNone)
			fakeUI = &fakeui.FakeUI{}
			writeDeploymentManifest()
		})

		Context("when the deployment manifest does not exist", func() {
			It("returns an error", func() {
				err := newDeleteCmd().Run(fakeStage, []string{"/garbage"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Deployment manifest does not exist at '/garbage'"))
				Expect(fakeUI.Errors).To(ContainElement("Deployment '/garbage' does not exist"))
			})
		})

		Context("when the deployment manifest exists", func() {
			It("sends the manifest on to the deleter", func() {
				mockDeploymentDeleter.EXPECT().DeleteDeployment(fakeStage).Return(nil)
				newDeleteCmd().Run(fakeStage, []string{deploymentManifestPath})
			})

			Context("when the deployment deleter returns an error", func() {
				It("sends the manifest on to the deleter", func() {
					err := bosherr.Error("boom")
					mockDeploymentDeleter.EXPECT().DeleteDeployment(fakeStage).Return(err)
					returnedErr := newDeleteCmd().Run(fakeStage, []string{deploymentManifestPath})
					Expect(returnedErr).To(Equal(err))
				})
			})
		})

		It("returns err unless exactly 1 arguments is given", func() {
			command := newDeleteCmd()

			err := command.Run(fakeStage, []string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Invalid usage"))

			err = command.Run(fakeStage, []string{"1", "2"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Invalid usage"))
		})
	})
})
