package vm_test

import (
	"errors"
	"time"

	. "github.com/cloudfoundry/bosh-cli/deployment/vm"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	biagentclient "github.com/cloudfoundry/bosh-agent/agentclient"
	bias "github.com/cloudfoundry/bosh-agent/agentclient/applyspec"
	bicloud "github.com/cloudfoundry/bosh-cli/cloud"
	biconfig "github.com/cloudfoundry/bosh-cli/config"
	bidisk "github.com/cloudfoundry/bosh-cli/deployment/disk"
	bideplmanifest "github.com/cloudfoundry/bosh-cli/deployment/manifest"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"

	fakebiagentclient "github.com/cloudfoundry/bosh-agent/agentclient/fakes"
	fakebicloud "github.com/cloudfoundry/bosh-cli/cloud/fakes"
	fakebiconfig "github.com/cloudfoundry/bosh-cli/config/fakes"
	fakebidisk "github.com/cloudfoundry/bosh-cli/deployment/disk/fakes"
	fakebivm "github.com/cloudfoundry/bosh-cli/deployment/vm/fakes"
	fakebiui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("VM", func() {
	var (
		vm               VM
		fakeVMRepo       *fakebiconfig.FakeVMRepo
		fakeStemcellRepo *fakebiconfig.FakeStemcellRepo
		fakeDiskDeployer *fakebivm.FakeDiskDeployer
		fakeAgentClient  *fakebiagentclient.FakeAgentClient
		fakeCloud        *fakebicloud.FakeCloud
		applySpec        bias.ApplySpec
		diskPool         bideplmanifest.DiskPool
		timeService      *FakeClock
		fs               *fakesys.FakeFileSystem
		logger           boshlog.Logger
	)

	BeforeEach(func() {
		fakeAgentClient = &fakebiagentclient.FakeAgentClient{}
		timeService = &FakeClock{Times: []time.Time{time.Now(), time.Now().Add(10 * time.Minute)}}

		// apply spec is only being passed to the agent client, so it doesn't need much content for testing
		applySpec = bias.ApplySpec{
			Deployment: "fake-deployment-name",
		}

		diskPool = bideplmanifest.DiskPool{
			Name:     "fake-persistent-disk-pool-name",
			DiskSize: 1024,
			CloudProperties: biproperty.Map{
				"fake-disk-pool-cloud-property-key": "fake-disk-pool-cloud-property-value",
			},
		}

		logger = boshlog.NewLogger(boshlog.LevelNone)
		fs = fakesys.NewFakeFileSystem()
		fakeCloud = fakebicloud.NewFakeCloud()
		fakeVMRepo = fakebiconfig.NewFakeVMRepo()
		fakeStemcellRepo = fakebiconfig.NewFakeStemcellRepo()
		fakeDiskDeployer = fakebivm.NewFakeDiskDeployer()
		vm = NewVM(
			"fake-vm-cid",
			fakeVMRepo,
			fakeStemcellRepo,
			fakeDiskDeployer,
			fakeAgentClient,
			fakeCloud,
			timeService,
			fs,
			logger,
		)
	})

	Describe("Exists", func() {
		It("returns true when the vm exists", func() {
			fakeCloud.HasVMFound = true

			exists, err := vm.Exists()
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("returns false when the vm does not exist", func() {
			fakeCloud.HasVMFound = false

			exists, err := vm.Exists()
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("returns error when checking fails", func() {
			fakeCloud.HasVMErr = errors.New("fake-has-vm-error")

			_, err := vm.Exists()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-has-vm-error"))
		})
	})

	Describe("UpdateDisks", func() {
		var expectedDisks []bidisk.Disk

		BeforeEach(func() {
			fakeDisk := fakebidisk.NewFakeDisk("fake-disk-cid")
			expectedDisks = []bidisk.Disk{fakeDisk}
			fakeDiskDeployer.SetDeployBehavior(expectedDisks, nil)
		})

		It("delegates to DiskDeployer.Deploy", func() {
			fakeStage := fakebiui.NewFakeStage()

			disks, err := vm.UpdateDisks(diskPool, fakeStage)
			Expect(err).NotTo(HaveOccurred())
			Expect(disks).To(Equal(expectedDisks))

			Expect(fakeDiskDeployer.DeployInputs).To(Equal([]fakebivm.DeployInput{
				{
					DiskPool:         diskPool,
					Cloud:            fakeCloud,
					VM:               vm,
					EventLoggerStage: fakeStage,
				},
			}))
		})
	})

	Describe("Stop", func() {
		It("stops agent services", func() {
			err := vm.Stop()
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeAgentClient.StopCallCount()).To(Equal(1))
		})

		Context("when stopping an agent fails", func() {
			BeforeEach(func() {
				fakeAgentClient.StopReturns(errors.New("fake-stop-error"))
			})

			It("returns an error", func() {
				err := vm.Stop()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-stop-error"))
			})
		})
	})

	Describe("Apply", func() {
		It("sends apply spec to the agent", func() {
			err := vm.Apply(applySpec)
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeAgentClient.ApplyArgsForCall(0)).To(Equal(applySpec))
		})

		Context("when sending apply spec to the agent fails", func() {
			BeforeEach(func() {
				fakeAgentClient.ApplyReturns(errors.New("fake-agent-apply-err"))
			})

			It("returns an error", func() {
				err := vm.Apply(applySpec)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-agent-apply-err"))
			})
		})
	})

	Describe("Start", func() {
		It("starts agent services", func() {
			err := vm.Start()
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeAgentClient.StartCallCount()).To(Equal(1))
		})

		Context("when starting an agent fails", func() {
			BeforeEach(func() {
				fakeAgentClient.StartReturns(errors.New("fake-start-error"))
			})

			It("returns an error", func() {
				err := vm.Start()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-start-error"))
			})
		})
	})

	Describe("WaitToBeRunning", func() {
		var invocations int
		BeforeEach(func() {
			responses := []struct {
				state biagentclient.AgentState
				err   error
			}{
				{biagentclient.AgentState{JobState: "pending"}, nil},
				{biagentclient.AgentState{JobState: "pending"}, nil},
				{biagentclient.AgentState{JobState: "running"}, nil},
			}
			fakeAgentClient.GetStateStub = func() (biagentclient.AgentState, error) {
				i := responses[invocations]
				invocations++
				return i.state, i.err
			}
		})

		It("waits until agent reports state as running", func() {
			err := vm.WaitToBeRunning(5, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(invocations).To(Equal(3))
		})
	})

	Describe("AttachDisk", func() {
		var disk *fakebidisk.FakeDisk

		BeforeEach(func() {
			disk = fakebidisk.NewFakeDisk("fake-disk-cid")
		})

		It("attaches disk to vm in the cloud", func() {
			err := vm.AttachDisk(disk)
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeCloud.AttachDiskInput).To(Equal(fakebicloud.AttachDiskInput{
				VMCID:   "fake-vm-cid",
				DiskCID: "fake-disk-cid",
			}))
		})

		It("sends mount disk to the agent after pinging the agent", func() {
			err := vm.AttachDisk(disk)
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeAgentClient.PingCallCount()).To(Equal(1))
			Expect(fakeAgentClient.MountDiskArgsForCall(0)).To(Equal("fake-disk-cid"))
		})

		Context("when attaching disk to cloud fails", func() {
			BeforeEach(func() {
				fakeCloud.AttachDiskErr = errors.New("fake-attach-error")
			})

			It("returns an error", func() {
				err := vm.AttachDisk(disk)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-attach-error"))

				Expect(fakeAgentClient.PingCallCount()).To(Equal(0))
			})
		})

		Context("when mounting disk fails", func() {
			BeforeEach(func() {
				fakeAgentClient.MountDiskReturns(errors.New("fake-mount-error"))
			})

			It("returns an error", func() {
				err := vm.AttachDisk(disk)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-mount-error"))
			})
		})

		It("returns an error if pinging fails", func() {
			fakeAgentClient.PingReturns("", errors.New("fake-error"))

			err := vm.AttachDisk(disk)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-error"))

			Expect(fakeAgentClient.MountDiskCallCount()).To(Equal(0))
		})
	})

	Describe("DetachDisk", func() {
		var disk *fakebidisk.FakeDisk

		BeforeEach(func() {
			disk = fakebidisk.NewFakeDisk("fake-disk-cid")
		})

		It("detaches disk from vm in the cloud", func() {
			err := vm.DetachDisk(disk)
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeCloud.DetachDiskInput).To(Equal(fakebicloud.DetachDiskInput{
				VMCID:   "fake-vm-cid",
				DiskCID: "fake-disk-cid",
			}))
			Expect(fakeAgentClient.PingCallCount()).To(Equal(1))
		})

		Context("when detaching disk to cloud fails", func() {
			BeforeEach(func() {
				fakeCloud.DetachDiskErr = errors.New("fake-detach-error")
			})

			It("returns an error", func() {
				err := vm.DetachDisk(disk)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-detach-error"))

				Expect(fakeAgentClient.PingCallCount()).To(Equal(0))
			})
		})

		It("returns an error if pinging fails", func() {
			fakeAgentClient.PingReturns("", errors.New("fake-error"))

			err := vm.DetachDisk(disk)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-error"))
		})
	})

	Describe("UnmountDisk", func() {
		var disk *fakebidisk.FakeDisk

		BeforeEach(func() {
			disk = fakebidisk.NewFakeDisk("fake-disk-cid")
		})

		It("sends unmount disk to the agent", func() {
			err := vm.UnmountDisk(disk)
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeAgentClient.UnmountDiskArgsForCall(0)).To(Equal("fake-disk-cid"))
		})

		Context("when unmounting disk fails", func() {
			BeforeEach(func() {
				fakeAgentClient.UnmountDiskReturns(errors.New("fake-unmount-error"))
			})

			It("returns an error", func() {
				err := vm.UnmountDisk(disk)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-unmount-error"))
			})
		})
	})

	Describe("Disks", func() {
		BeforeEach(func() {
			fakeAgentClient.ListDiskReturns([]string{"fake-disk-cid-1", "fake-disk-cid-2"}, nil)
		})

		It("returns disks that are reported by the agent", func() {
			disks, err := vm.Disks()
			Expect(err).ToNot(HaveOccurred())
			expectedFirstDisk := bidisk.NewDisk(biconfig.DiskRecord{CID: "fake-disk-cid-1"}, nil, nil)
			expectedSecondDisk := bidisk.NewDisk(biconfig.DiskRecord{CID: "fake-disk-cid-2"}, nil, nil)
			Expect(disks).To(Equal([]bidisk.Disk{expectedFirstDisk, expectedSecondDisk}))
		})

		Context("when listing disks fails", func() {
			BeforeEach(func() {
				fakeAgentClient.ListDiskReturns([]string{}, errors.New("fake-list-disk-error"))
			})

			It("returns an error", func() {
				_, err := vm.Disks()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-list-disk-error"))
			})
		})
	})

	Describe("Delete", func() {
		It("deletes vm in the cloud", func() {
			err := vm.Delete()
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeCloud.DeleteVMInput).To(Equal(fakebicloud.DeleteVMInput{
				VMCID: "fake-vm-cid",
			}))
		})

		It("deletes VM in the vm repo", func() {
			err := vm.Delete()
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeVMRepo.ClearCurrentCalled).To(BeTrue())
		})

		It("clears current stemcell in the stemcell repo", func() {
			err := vm.Delete()
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeStemcellRepo.ClearCurrentCalled).To(BeTrue())
		})

		Context("when deleting vm in the cloud fails", func() {
			BeforeEach(func() {
				fakeCloud.DeleteVMErr = errors.New("fake-delete-vm-error")
			})

			It("returns an error", func() {
				err := vm.Delete()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-delete-vm-error"))
			})
		})

		Context("when deleting vm in the cloud fails with VMNotFoundError", func() {
			var deleteErr = bicloud.NewCPIError("delete_vm", bicloud.CmdError{
				Type:    bicloud.VMNotFoundError,
				Message: "fake-vm-not-found-message",
			})

			BeforeEach(func() {
				fakeCloud.DeleteVMErr = deleteErr
			})

			It("deletes vm in the cloud", func() {
				err := vm.Delete()
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(deleteErr))
				Expect(fakeCloud.DeleteVMInput).To(Equal(fakebicloud.DeleteVMInput{
					VMCID: "fake-vm-cid",
				}))
			})

			It("deletes VM in the vm repo", func() {
				err := vm.Delete()
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(deleteErr))
				Expect(fakeVMRepo.ClearCurrentCalled).To(BeTrue())
			})

			It("clears current stemcell in the stemcell repo", func() {
				err := vm.Delete()
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(deleteErr))
				Expect(fakeStemcellRepo.ClearCurrentCalled).To(BeTrue())
			})
		})
	})

	Describe("MigrateDisk", func() {
		It("sends migrate_disk to the agent", func() {
			err := vm.MigrateDisk()
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeAgentClient.MigrateDiskCallCount()).To(Equal(1))
		})

		Context("when migrating disk fails", func() {
			BeforeEach(func() {
				fakeAgentClient.MigrateDiskReturns(errors.New("fake-migrate-error"))
			})

			It("returns an error", func() {
				err := vm.MigrateDisk()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-migrate-error"))
			})
		})
	})

	Describe("GetState", func() {
		BeforeEach(func() {
			fakeAgentClient.GetStateReturns(biagentclient.AgentState{JobState: "testing"}, nil)
		})

		It("sends get_state to the agent", func() {
			agentState, err := vm.GetState()
			Expect(err).ToNot(HaveOccurred())
			Expect(agentState).To(Equal(biagentclient.AgentState{JobState: "testing"}))
		})
	})
})

type FakeClock struct {
	Times []time.Time
}

func (c *FakeClock) Sleep(_ time.Duration) {}

func (c *FakeClock) Now() time.Time {
	t1 := c.Times[0]
	c.Times = c.Times[1:]
	return t1
}
