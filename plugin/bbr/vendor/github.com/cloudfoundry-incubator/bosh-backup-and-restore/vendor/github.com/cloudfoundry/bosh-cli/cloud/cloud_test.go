package cloud_test

import (
	"errors"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	fakebicloud "github.com/cloudfoundry/bosh-cli/cloud/fakes"

	. "github.com/cloudfoundry/bosh-cli/cloud"
)

var _ = Describe("Cloud", func() {
	var (
		cloud            Cloud
		context          CmdContext
		fakeCPICmdRunner *fakebicloud.FakeCPICmdRunner
	)

	BeforeEach(func() {
		fakeCPICmdRunner = fakebicloud.NewFakeCPICmdRunner()
		logger := boshlog.NewLogger(boshlog.LevelNone)
		cloud = NewCloud(fakeCPICmdRunner, "fake-director-id", logger)
		context = CmdContext{DirectorID: "fake-director-id"}
	})

	var itHandlesCPIErrors = func(method string, exec func() error) {
		It("returns a cloud.Error when the CPI command returns an error", func() {
			fakeCPICmdRunner.RunCmdOutput = CmdOutput{
				Error: &CmdError{
					Type:    "Bosh::Cloud::CloudError",
					Message: "fake-cpi-error-msg",
				},
			}

			err := exec()
			Expect(err).To(HaveOccurred())

			cpiError, ok := err.(Error)
			Expect(ok).To(BeTrue(), "Expected %s to implement the Error interface", cpiError)
			Expect(cpiError.Method()).To(Equal(method))
			Expect(cpiError.Type()).To(Equal("Bosh::Cloud::CloudError"))
			Expect(cpiError.Message()).To(Equal("fake-cpi-error-msg"))
			Expect(err.Error()).To(ContainSubstring("Bosh::Cloud::CloudError"))
			Expect(err.Error()).To(ContainSubstring("fake-cpi-error-msg"))
		})
	}

	Describe("CreateStemcell", func() {
		var (
			stemcellImagePath string
			cloudProperties   biproperty.Map
		)

		BeforeEach(func() {
			stemcellImagePath = "/stemcell/path"
			cloudProperties = biproperty.Map{
				"fake-key": "fake-value",
			}
		})

		Context("when the cpi successfully creates the stemcell", func() {
			BeforeEach(func() {
				fakeCPICmdRunner.RunCmdOutput = CmdOutput{
					Result: "fake-cid",
				}
			})

			It("executes the cpi job script with stemcell image path & cloud_properties", func() {
				_, err := cloud.CreateStemcell(stemcellImagePath, cloudProperties)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeCPICmdRunner.RunInputs).To(HaveLen(1))
				Expect(fakeCPICmdRunner.RunInputs[0]).To(Equal(fakebicloud.RunInput{
					Context: context,
					Method:  "create_stemcell",
					Arguments: []interface{}{
						stemcellImagePath,
						cloudProperties,
					},
				}))
			})

			It("returns the cid returned from executing the cpi script", func() {
				cid, err := cloud.CreateStemcell(stemcellImagePath, cloudProperties)
				Expect(err).NotTo(HaveOccurred())
				Expect(cid).To(Equal("fake-cid"))
			})
		})

		Context("when the result is of an unexpected type", func() {
			BeforeEach(func() {
				fakeCPICmdRunner.RunCmdOutput = CmdOutput{
					Result: 1,
				}
			})

			It("returns an error", func() {
				_, err := cloud.CreateStemcell(stemcellImagePath, cloudProperties)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Unexpected external CPI command result: '1'"))
			})
		})

		Context("when the cpi command execution fails", func() {
			BeforeEach(func() {
				fakeCPICmdRunner.RunErr = errors.New("fake-run-error")
			})

			It("returns an error", func() {
				_, err := cloud.CreateStemcell(stemcellImagePath, cloudProperties)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-run-error"))
			})
		})

		itHandlesCPIErrors("create_stemcell", func() error {
			_, err := cloud.CreateStemcell(stemcellImagePath, cloudProperties)
			return err
		})
	})

	Describe("DeleteStemcell", func() {
		It("executes the delete_stemcell method on the CPI with stemcell cid", func() {
			err := cloud.DeleteStemcell("fake-stemcell-cid")
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeCPICmdRunner.RunInputs).To(HaveLen(1))
			Expect(fakeCPICmdRunner.RunInputs[0]).To(Equal(fakebicloud.RunInput{
				Context: context,
				Method:  "delete_stemcell",
				Arguments: []interface{}{
					"fake-stemcell-cid",
				},
			}))
		})

		Context("when the cpi command execution fails", func() {
			BeforeEach(func() {
				fakeCPICmdRunner.RunErr = errors.New("fake-run-error")
			})

			It("returns an error", func() {
				err := cloud.DeleteStemcell("fake-stemcell-cid")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-run-error"))
			})
		})

		itHandlesCPIErrors("delete_stemcell", func() error {
			return cloud.DeleteStemcell("fake-stemcell-cid")
		})
	})

	Describe("HasVM", func() {
		It("return true when VM exists", func() {
			fakeCPICmdRunner.RunCmdOutput = CmdOutput{
				Result: true,
			}

			found, err := cloud.HasVM("fake-vm-cid")
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())

			Expect(fakeCPICmdRunner.RunInputs).To(Equal([]fakebicloud.RunInput{
				{
					Context:   context,
					Method:    "has_vm",
					Arguments: []interface{}{"fake-vm-cid"},
				},
			}))
		})

		It("return false when VM does not exist", func() {
			fakeCPICmdRunner.RunCmdOutput = CmdOutput{
				Result: false,
			}

			found, err := cloud.HasVM("fake-vm-cid")
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeFalse())
		})

		Context("when the cpi command execution fails", func() {
			BeforeEach(func() {
				fakeCPICmdRunner.RunErr = errors.New("fake-run-error")
			})

			It("returns an error when executing the CPI command fails", func() {
				_, err := cloud.HasVM("fake-vm-cid")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-run-error"))
			})
		})

		itHandlesCPIErrors("has_vm", func() error {
			_, err := cloud.HasVM("fake-vm-cid")
			return err
		})
	})

	Describe("CreateVM", func() {
		var (
			agentID           string
			stemcellCID       string
			cloudProperties   biproperty.Map
			networkInterfaces map[string]biproperty.Map
			env               biproperty.Map
		)

		BeforeEach(func() {
			agentID = "fake-agent-id"
			stemcellCID = "fake-stemcell-cid"
			networkInterfaces = map[string]biproperty.Map{
				"bosh": biproperty.Map{
					"type": "dynamic",
					"cloud_properties": biproperty.Map{
						"a": "b",
					},
				},
			}
			cloudProperties = biproperty.Map{
				"fake-cloud-property-key": "fake-cloud-property-value",
			}
			env = biproperty.Map{
				"fake-env-key": "fake-env-value",
			}
		})

		Context("when the cpi successfully creates the vm", func() {
			BeforeEach(func() {
				fakeCPICmdRunner.RunCmdOutput = CmdOutput{
					Result: "fake-vm-cid",
				}
			})

			It("executes the cpi job script with the director UUID and stemcell CID", func() {
				_, err := cloud.CreateVM(agentID, stemcellCID, cloudProperties, networkInterfaces, env)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeCPICmdRunner.RunInputs).To(HaveLen(1))
				Expect(fakeCPICmdRunner.RunInputs[0]).To(Equal(fakebicloud.RunInput{
					Context: context,
					Method:  "create_vm",
					Arguments: []interface{}{
						agentID,
						stemcellCID,
						cloudProperties,
						networkInterfaces,
						[]interface{}{},
						env,
					},
				}))
			})

			It("returns the cid returned from executing the cpi script", func() {
				cid, err := cloud.CreateVM(agentID, stemcellCID, cloudProperties, networkInterfaces, env)
				Expect(err).NotTo(HaveOccurred())
				Expect(cid).To(Equal("fake-vm-cid"))
			})
		})

		Context("when the result is of an unexpected type", func() {
			BeforeEach(func() {
				fakeCPICmdRunner.RunCmdOutput = CmdOutput{
					Result: 1,
				}
			})

			It("returns an error", func() {
				_, err := cloud.CreateVM(agentID, stemcellCID, cloudProperties, networkInterfaces, env)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Unexpected external CPI command result: '1'"))
			})
		})

		Context("when the cpi command execution fails", func() {
			BeforeEach(func() {
				fakeCPICmdRunner.RunErr = errors.New("fake-run-error")
			})

			It("returns an error", func() {
				_, err := cloud.CreateVM(agentID, stemcellCID, cloudProperties, networkInterfaces, env)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-run-error"))
			})
		})

		itHandlesCPIErrors("create_vm", func() error {
			_, err := cloud.CreateVM(agentID, stemcellCID, cloudProperties, networkInterfaces, env)
			return err
		})
	})

	Describe("SetVMMetadata", func() {
		It("calls the set_vm_metadata CPI method", func() {
			vmCID := "fake-vm-cid"
			metadata := VMMetadata{
				"director":   "bosh-init",
				"deployment": "some-deployment",
				"job":        "some-job",
				"index":      "0",
			}
			err := cloud.SetVMMetadata(vmCID, metadata)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeCPICmdRunner.RunInputs).To(HaveLen(1))
			Expect(fakeCPICmdRunner.RunInputs[0]).To(Equal(fakebicloud.RunInput{
				Context: context,
				Method:  "set_vm_metadata",
				Arguments: []interface{}{
					vmCID,
					metadata,
				},
			}))
		})

		It("returns the error if running fails", func() {
			fakeCPICmdRunner.RunErr = errors.New("fake-run-error")
			vmCID := "fake-vm-cid"
			metadata := VMMetadata{
				"director":   "bosh-init",
				"deployment": "some-deployment",
				"job":        "some-job",
				"index":      "0",
			}

			err := cloud.SetVMMetadata(vmCID, metadata)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("fake-run-error"))
		})

		itHandlesCPIErrors("set_vm_metadata", func() error {
			vmCID := "fake-vm-cid"
			metadata := VMMetadata{
				"director":   "bosh-init",
				"deployment": "some-deployment",
				"job":        "some-job",
				"index":      "0",
			}
			return cloud.SetVMMetadata(vmCID, metadata)
		})
	})

	Describe("CreateDisk", func() {
		var (
			size            int
			cloudProperties biproperty.Map
			instanceID      string
		)

		BeforeEach(func() {
			size = 1024
			cloudProperties = biproperty.Map{
				"fake-cloud-property-key": "fake-cloud-property-value",
			}
			instanceID = "fake-instance-id"
		})

		Context("when the cpi successfully creates the disk", func() {
			BeforeEach(func() {
				fakeCPICmdRunner.RunCmdOutput = CmdOutput{
					Result: "fake-disk-cid",
				}
			})

			It("executes the cpi job script with the correct arguments", func() {
				_, err := cloud.CreateDisk(size, cloudProperties, instanceID)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeCPICmdRunner.RunInputs).To(HaveLen(1))
				Expect(fakeCPICmdRunner.RunInputs[0]).To(Equal(fakebicloud.RunInput{
					Context: context,
					Method:  "create_disk",
					Arguments: []interface{}{
						size,
						cloudProperties,
						instanceID,
					},
				}))
			})

			It("returns the cid returned from executing the cpi script", func() {
				cid, err := cloud.CreateDisk(size, cloudProperties, instanceID)
				Expect(err).NotTo(HaveOccurred())
				Expect(cid).To(Equal("fake-disk-cid"))
			})
		})

		Context("when the result is of an unexpected type", func() {
			BeforeEach(func() {
				fakeCPICmdRunner.RunCmdOutput = CmdOutput{
					Result: 1,
				}
			})

			It("returns an error", func() {
				_, err := cloud.CreateDisk(size, cloudProperties, instanceID)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Unexpected external CPI command result: '1'"))
			})
		})

		Context("when the cpi command execution fails", func() {
			BeforeEach(func() {
				fakeCPICmdRunner.RunErr = errors.New("fake-run-error")
			})

			It("returns an error", func() {
				_, err := cloud.CreateDisk(size, cloudProperties, instanceID)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-run-error"))
			})
		})

		itHandlesCPIErrors("create_disk", func() error {
			_, err := cloud.CreateDisk(size, cloudProperties, instanceID)
			return err
		})
	})

	Describe("AttachDisk", func() {
		Context("when the cpi successfully attaches the disk", func() {
			It("executes the cpi job script with the correct arguments", func() {
				err := cloud.AttachDisk("fake-vm-cid", "fake-disk-cid")
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeCPICmdRunner.RunInputs).To(HaveLen(1))
				Expect(fakeCPICmdRunner.RunInputs[0]).To(Equal(fakebicloud.RunInput{
					Context: context,
					Method:  "attach_disk",
					Arguments: []interface{}{
						"fake-vm-cid",
						"fake-disk-cid",
					},
				}))
			})
		})

		Context("when the cpi command execution fails", func() {
			BeforeEach(func() {
				fakeCPICmdRunner.RunErr = errors.New("fake-run-error")
			})

			It("returns an error", func() {
				err := cloud.AttachDisk("fake-vm-cid", "fake-disk-cid")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-run-error"))
			})
		})

		itHandlesCPIErrors("attach_disk", func() error {
			return cloud.AttachDisk("fake-vm-cid", "fake-disk-cid")
		})
	})

	Describe("DetachDisk", func() {
		Context("when the cpi successfully detaches the disk", func() {
			It("executes the cpi job script with the correct arguments", func() {
				err := cloud.DetachDisk("fake-vm-cid", "fake-disk-cid")
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeCPICmdRunner.RunInputs).To(HaveLen(1))
				Expect(fakeCPICmdRunner.RunInputs[0]).To(Equal(fakebicloud.RunInput{
					Context: context,
					Method:  "detach_disk",
					Arguments: []interface{}{
						"fake-vm-cid",
						"fake-disk-cid",
					},
				}))
			})
		})

		Context("when the cpi command execution fails", func() {
			BeforeEach(func() {
				fakeCPICmdRunner.RunErr = errors.New("fake-run-error")
			})

			It("returns an error", func() {
				err := cloud.DetachDisk("fake-vm-cid", "fake-disk-cid")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-run-error"))
			})
		})

		itHandlesCPIErrors("detach_disk", func() error {
			return cloud.DetachDisk("fake-vm-cid", "fake-disk-cid")
		})
	})

	Describe("DeleteVM", func() {
		Context("when the cpi successfully deletes vm", func() {
			It("executes the cpi job script with the correct arguments", func() {
				err := cloud.DeleteVM("fake-vm-cid")
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeCPICmdRunner.RunInputs).To(HaveLen(1))
				Expect(fakeCPICmdRunner.RunInputs[0]).To(Equal(fakebicloud.RunInput{
					Context: context,
					Method:  "delete_vm",
					Arguments: []interface{}{
						"fake-vm-cid",
					},
				}))
			})
		})

		Context("when the cpi command execution fails", func() {
			BeforeEach(func() {
				fakeCPICmdRunner.RunErr = errors.New("fake-run-error")
			})

			It("returns an error", func() {
				err := cloud.DeleteVM("fake-vm-cid")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-run-error"))
			})
		})

		itHandlesCPIErrors("delete_vm", func() error {
			return cloud.DeleteVM("fake-vm-cid")
		})
	})

	Describe("DeleteDisk", func() {
		Context("when the cpi successfully deletes disk", func() {
			It("executes the cpi job script with the correct arguments", func() {
				err := cloud.DeleteDisk("fake-disk-cid")
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeCPICmdRunner.RunInputs).To(HaveLen(1))
				Expect(fakeCPICmdRunner.RunInputs[0]).To(Equal(fakebicloud.RunInput{
					Context: context,
					Method:  "delete_disk",
					Arguments: []interface{}{
						"fake-disk-cid",
					},
				}))
			})
		})

		Context("when the cpi command execution fails", func() {
			BeforeEach(func() {
				fakeCPICmdRunner.RunErr = errors.New("fake-run-error")
			})

			It("returns an error", func() {
				err := cloud.DeleteDisk("fake-disk-cid")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-run-error"))
			})
		})

		itHandlesCPIErrors("delete_disk", func() error {
			return cloud.DeleteDisk("fake-disk-cid")
		})
	})
})
