package cmd_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	boshdir "github.com/cloudfoundry/bosh-cli/director"
)

var _ = Describe("InstanceTable", func() {
	Describe("ForVMInfo", func() {
		var (
			info boshdir.VMInfo
			tbl  InstanceTable
		)

		BeforeEach(func() {
			info = boshdir.VMInfo{}
			tbl = InstanceTable{Details: true, DNS: true, Vitals: true}
		})

		Describe("name, id", func() {
			It("returns ? name", func() {
				Expect(tbl.ForVMInfo(info).Name.String()).To(Equal("?"))
			})

			It("returns name", func() {
				info.JobName = "name"
				Expect(tbl.ForVMInfo(info).Name.String()).To(Equal("name"))
			})

			It("returns name with id", func() {
				info.JobName = "name"
				info.ID = "id"
				Expect(tbl.ForVMInfo(info).Name.String()).To(Equal("name/id"))
			})

			It("returns name with id, bootstrap and index", func() {
				idx := 1
				info.JobName = "name"
				info.ID = "id"
				info.Index = &idx
				info.Bootstrap = true
				Expect(tbl.ForVMInfo(info).Name.String()).To(Equal("name/id"))

				Expect(tbl.ForVMInfo(info).Bootstrap).ToNot(BeNil())
				Expect(tbl.ForVMInfo(info).Bootstrap.String()).To(Equal("true"))
				Expect(tbl.ForVMInfo(info).Index).ToNot(BeNil())
				Expect(tbl.ForVMInfo(info).Index.String()).To(Equal("1"))
			})

			It("returns name with id, without index", func() {
				info.JobName = "name"
				info.ID = "id"
				Expect(tbl.ForVMInfo(info).Name.String()).To(Equal("name/id"))
			})

			It("returns ? name with id", func() {
				info.JobName = ""
				info.ID = "id"
				Expect(tbl.ForVMInfo(info).Name.String()).To(Equal("?/id"))
			})
		})

		Describe("vm type, resource pool", func() {
			It("returns RP if vm type is empty", func() {
				info.ResourcePool = "rp"
				Expect(tbl.ForVMInfo(info).VMType.String()).To(Equal("rp"))
			})

			It("returns vm type if vm type is non-empty", func() {
				info.ResourcePool = "rp"
				info.VMType = "vm-type"
				Expect(tbl.ForVMInfo(info).VMType.String()).To(Equal("vm-type"))
			})
		})

		Describe("disk cids", func() {
			It("returns empty if disk cids is empty", func() {
				Expect(tbl.ForVMInfo(info).DiskCIDs.String()).To(Equal(""))
			})

			It("returns disk cid if disk cids is non-empty", func() {
				info.DiskIDs = []string{"disk-cid1", "disk-cid2"}
				Expect(tbl.ForVMInfo(info).DiskCIDs.String()).To(Equal("disk-cid1\ndisk-cid2"))
			})
		})
	})
})
