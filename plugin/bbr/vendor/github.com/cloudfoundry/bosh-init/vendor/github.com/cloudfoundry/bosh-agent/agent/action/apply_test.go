package action_test

import (
	"errors"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agent/action"
	boshas "github.com/cloudfoundry/bosh-agent/agent/applier/applyspec"
	fakeas "github.com/cloudfoundry/bosh-agent/agent/applier/applyspec/fakes"
	fakeappl "github.com/cloudfoundry/bosh-agent/agent/applier/fakes"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshdir "github.com/cloudfoundry/bosh-agent/settings/directories"
	fakesettings "github.com/cloudfoundry/bosh-agent/settings/fakes"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

func init() {
	Describe("ApplyAction", func() {
		var (
			applier         *fakeappl.FakeApplier
			specService     *fakeas.FakeV1Service
			settingsService *fakesettings.FakeSettingsService
			dirProvider     boshdir.Provider
			action          ApplyAction
			fs              boshsys.FileSystem
		)

		BeforeEach(func() {
			applier = fakeappl.NewFakeApplier()
			specService = fakeas.NewFakeV1Service()
			settingsService = &fakesettings.FakeSettingsService{}
			dirProvider = boshdir.NewProvider("/var/vcap")
			fs = fakesys.NewFakeFileSystem()
			action = NewApply(applier, specService, settingsService, dirProvider.InstanceDir(), fs)
		})

		It("apply should be asynchronous", func() {
			Expect(action.IsAsynchronous()).To(BeTrue())
		})

		It("is not persistent", func() {
			Expect(action.IsPersistent()).To(BeFalse())
		})

		Describe("Run", func() {
			settings := boshsettings.Settings{AgentID: "fake-agent-id"}

			BeforeEach(func() {
				settingsService.Settings = settings
			})

			Context("when desired spec has configuration hash", func() {
				currentApplySpec := boshas.V1ApplySpec{ConfigurationHash: "fake-current-config-hash"}
				desiredApplySpec := boshas.V1ApplySpec{ConfigurationHash: "fake-desired-config-hash"}
				populatedDesiredApplySpec := boshas.V1ApplySpec{
					ConfigurationHash: "fake-populated-desired-config-hash",
				}

				Context("when current spec can be retrieved", func() {
					BeforeEach(func() {
						specService.Spec = currentApplySpec
					})

					It("populates dynamic networks in desired spec", func() {
						_, err := action.Run(desiredApplySpec)
						Expect(err).ToNot(HaveOccurred())
						Expect(specService.PopulateDHCPNetworksSpec).To(Equal(desiredApplySpec))
						Expect(specService.PopulateDHCPNetworksSettings).To(Equal(settings))
					})

					Context("when resolving dynamic networks succeeds", func() {
						BeforeEach(func() {
							specService.PopulateDHCPNetworksResultSpec = populatedDesiredApplySpec
						})

						It("runs applier with populated desired spec", func() {
							_, err := action.Run(desiredApplySpec)
							Expect(err).ToNot(HaveOccurred())
							Expect(applier.Applied).To(BeTrue())
							Expect(applier.ApplyCurrentApplySpec).To(Equal(currentApplySpec))
							Expect(applier.ApplyDesiredApplySpec).To(Equal(populatedDesiredApplySpec))
						})

						Context("when applier succeeds applying desired spec", func() {
							Context("when saving desires spec as current spec succeeds", func() {
								It("returns 'applied' after setting populated desired spec as current spec", func() {
									value, err := action.Run(desiredApplySpec)
									Expect(err).ToNot(HaveOccurred())
									Expect(value).To(Equal("applied"))

									Expect(specService.Spec).To(Equal(populatedDesiredApplySpec))
								})

								Context("desired spec has id, instance name, deployment name, and az", func() {

									BeforeEach(func() {
										desiredApplySpec = boshas.V1ApplySpec{ConfigurationHash: "fake-desired-config-hash", NodeID: "node-id01-123f-r2344", AvailabilityZone: "ex-az", Deployment: "deployment-name", Name: "instance-name"}
										specService.PopulateDHCPNetworksResultSpec = desiredApplySpec
									})

									It("returns 'applied' and writes the id, instance name, deployment name, and az to files in the instance directory", func() {
										value, err := action.Run(desiredApplySpec)
										Expect(err).ToNot(HaveOccurred())
										Expect(value).To(Equal("applied"))

										instanceDir := dirProvider.InstanceDir()

										id, err := fs.ReadFileString(path.Join(instanceDir, "id"))
										Expect(err).ToNot(HaveOccurred())
										Expect(id).To(Equal(desiredApplySpec.NodeID))

										az, err := fs.ReadFileString(path.Join(instanceDir, "az"))
										Expect(err).ToNot(HaveOccurred())
										Expect(az).To(Equal(desiredApplySpec.AvailabilityZone))

										instanceName, err := fs.ReadFileString(path.Join(instanceDir, "name"))
										Expect(err).ToNot(HaveOccurred())
										Expect(instanceName).To(Equal(desiredApplySpec.Name))

										deploymentName, err := fs.ReadFileString(path.Join(instanceDir, "deployment"))
										Expect(err).ToNot(HaveOccurred())
										Expect(deploymentName).To(Equal(desiredApplySpec.Deployment))
									})
								})
							})

							Context("when saving populated desires spec as current spec fails", func() {
								It("returns error because agent was not able to remember that is converged to desired spec", func() {
									specService.SetErr = errors.New("fake-set-error")

									_, err := action.Run(desiredApplySpec)
									Expect(err).To(HaveOccurred())
									Expect(err.Error()).To(ContainSubstring("fake-set-error"))
								})
							})
						})

						Context("when applier fails applying desired spec", func() {
							BeforeEach(func() {
								applier.ApplyError = errors.New("fake-apply-error")
							})

							It("returns error", func() {
								_, err := action.Run(desiredApplySpec)
								Expect(err).To(HaveOccurred())
								Expect(err.Error()).To(ContainSubstring("fake-apply-error"))
							})

							It("does not save desired spec as current spec", func() {
								_, err := action.Run(desiredApplySpec)
								Expect(err).To(HaveOccurred())
								Expect(specService.Spec).To(Equal(currentApplySpec))
							})
						})
					})

					Context("when resolving dynamic networks fails", func() {
						BeforeEach(func() {
							specService.PopulateDHCPNetworksErr = errors.New("fake-populate-dynamic-networks-err")
						})

						It("returns error", func() {
							_, err := action.Run(desiredApplySpec)
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(ContainSubstring("fake-populate-dynamic-networks-err"))
						})

						It("does not apply desired spec as current spec", func() {
							_, err := action.Run(desiredApplySpec)
							Expect(err).To(HaveOccurred())
							Expect(applier.Applied).To(BeFalse())
						})

						It("does not save desired spec as current spec", func() {
							_, err := action.Run(desiredApplySpec)
							Expect(err).To(HaveOccurred())
							Expect(specService.Spec).To(Equal(currentApplySpec))
						})
					})
				})

				Context("when current spec cannot be retrieved", func() {
					BeforeEach(func() {
						specService.Spec = currentApplySpec
						specService.GetErr = errors.New("fake-get-error")
					})

					It("returns error and does not apply desired spec", func() {
						_, err := action.Run(desiredApplySpec)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("fake-get-error"))
					})

					It("does not run applier with desired spec", func() {
						_, err := action.Run(desiredApplySpec)
						Expect(err).To(HaveOccurred())
						Expect(applier.Applied).To(BeFalse())
					})

					It("does not save desired spec as current spec", func() {
						_, err := action.Run(desiredApplySpec)
						Expect(err).To(HaveOccurred())
						Expect(specService.Spec).To(Equal(currentApplySpec))
					})
				})
			})

			Context("when desired spec does not have a configuration hash", func() {
				desiredApplySpec := boshas.V1ApplySpec{
					JobSpec: boshas.JobSpec{
						Template: "fake-job-template",
					},
				}

				populatedDesiredApplySpec := boshas.V1ApplySpec{
					JobSpec: boshas.JobSpec{
						Template: "fake-populated-job-template",
					},
				}

				It("populates dynamic networks in desired spec", func() {
					_, err := action.Run(desiredApplySpec)
					Expect(err).ToNot(HaveOccurred())
					Expect(specService.PopulateDHCPNetworksSpec).To(Equal(desiredApplySpec))
					Expect(specService.PopulateDHCPNetworksSettings).To(Equal(settings))
				})

				Context("when resolving dynamic networks succeeds", func() {
					BeforeEach(func() {
						specService.PopulateDHCPNetworksResultSpec = populatedDesiredApplySpec
					})

					Context("when saving desires spec as current spec succeeds", func() {
						It("returns 'applied' after setting desired spec as current spec", func() {
							value, err := action.Run(desiredApplySpec)
							Expect(err).ToNot(HaveOccurred())
							Expect(value).To(Equal("applied"))

							Expect(specService.Spec).To(Equal(populatedDesiredApplySpec))
						})

						It("does not try to apply desired spec since it does not have jobs and packages", func() {
							_, err := action.Run(desiredApplySpec)
							Expect(err).ToNot(HaveOccurred())
							Expect(applier.Applied).To(BeFalse())
						})
					})

					Context("when saving desires spec as current spec fails", func() {
						BeforeEach(func() {
							specService.SetErr = errors.New("fake-set-error")
						})

						It("returns error because agent was not able to remember that is converged to desired spec", func() {
							_, err := action.Run(desiredApplySpec)
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(ContainSubstring("fake-set-error"))
						})

						It("does not try to apply desired spec since it does not have jobs and packages", func() {
							_, err := action.Run(desiredApplySpec)
							Expect(err).To(HaveOccurred())
							Expect(applier.Applied).To(BeFalse())
						})
					})
				})

				Context("when resolving dynamic networks fails", func() {
					BeforeEach(func() {
						specService.PopulateDHCPNetworksErr = errors.New("fake-populate-dynamic-networks-err")
					})

					It("returns error", func() {
						_, err := action.Run(desiredApplySpec)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("fake-populate-dynamic-networks-err"))
					})

					It("does not apply desired spec as current spec", func() {
						_, err := action.Run(desiredApplySpec)
						Expect(err).To(HaveOccurred())
						Expect(applier.Applied).To(BeFalse())
					})

					It("does not save desired spec as current spec", func() {
						_, err := action.Run(desiredApplySpec)
						Expect(err).To(HaveOccurred())
						Expect(specService.Spec).ToNot(Equal(desiredApplySpec))
					})
				})
			})
		})
	})
}
