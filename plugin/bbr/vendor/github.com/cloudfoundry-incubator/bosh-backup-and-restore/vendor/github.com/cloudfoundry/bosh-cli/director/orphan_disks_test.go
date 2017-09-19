package director_test

import (
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "github.com/cloudfoundry/bosh-cli/director"
)

var _ = Describe("Director", func() {
	var (
		director Director
		server   *ghttp.Server
	)

	BeforeEach(func() {
		director, server = BuildServer()
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("OrphanDisks", func() {
		It("returns orphaned disks", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/disks"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[
	{
		"disk_cid": "cid1",
		"size": 1000,

		"deployment_name": "dep1",
		"instance_name": "instance1",
		"az": "az1",

		"orphaned_at": "2016-01-09 06:23:25 +0000"
	},
	{
		"disk_cid": "cid2",
		"size": 2000,

		"deployment_name": "dep2",
		"instance_name": "instance2",
		"az": "az2",

		"orphaned_at": "2016-08-25 00:17:16 UTC"
	}
]`),
				),
			)

			dep1, err := director.FindDeployment("dep1")
			Expect(err).ToNot(HaveOccurred())

			dep2, err := director.FindDeployment("dep2")
			Expect(err).ToNot(HaveOccurred())

			disks, err := director.OrphanDisks()
			Expect(err).ToNot(HaveOccurred())
			Expect(disks).To(HaveLen(2))

			Expect(disks[0].CID()).To(Equal("cid1"))
			Expect(disks[0].Size()).To(Equal(uint64(1000)))
			Expect(disks[0].Deployment()).To(Equal(dep1))
			Expect(disks[0].InstanceName()).To(Equal("instance1"))
			Expect(disks[0].AZName()).To(Equal("az1"))
			Expect(disks[0].OrphanedAt()).To(Equal(time.Date(2016, time.January, 9, 6, 23, 25, 0, time.UTC)))

			Expect(disks[1].CID()).To(Equal("cid2"))
			Expect(disks[1].Size()).To(Equal(uint64(2000)))
			Expect(disks[1].Deployment()).To(Equal(dep2))
			Expect(disks[1].InstanceName()).To(Equal("instance2"))
			Expect(disks[1].AZName()).To(Equal("az2"))
			Expect(disks[1].OrphanedAt()).To(Equal(time.Date(2016, time.August, 25, 0, 17, 16, 0, time.UTC)))
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("GET", "/disks"), server)

			_, err := director.OrphanDisks()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Finding orphaned disks: Director responded with non-successful status code"))
		})

		It("returns error if response cannot be unmarshalled", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/disks"),
					ghttp.RespondWith(http.StatusOK, ``),
				),
			)

			_, err := director.OrphanDisks()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Finding orphaned disks: Unmarshaling Director response"))
		})
	})
})

var _ = Describe("OrphanDisk", func() {
	var (
		director Director
		disk     OrphanDisk
		server   *ghttp.Server
	)

	BeforeEach(func() {
		director, server = BuildServer()

		var err error

		disk, err = director.FindOrphanDisk("cid")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("Delete", func() {
		It("deletes orphaned disk", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("DELETE", "/disks/cid"),
					ghttp.VerifyBasicAuth("username", "password"),
				),
				"",
				server,
			)

			Expect(disk.Delete()).ToNot(HaveOccurred())
		})

		It("succeeds even if error occurrs if disk no longer exists", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("DELETE", "/disks/cid"),
					ghttp.RespondWith(http.StatusBadRequest, ``),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/disks"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[]`),
				),
			)

			Expect(disk.Delete()).ToNot(HaveOccurred())
		})

		It("returns delete error if listing disks fails", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("DELETE", "/disks/cid"),
					ghttp.RespondWith(http.StatusBadRequest, ``),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/disks"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, ``),
				),
			)

			err := disk.Delete()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Deleting orphaned disk 'cid': Director responded with non-successful status code"))
		})

		It("returns error if response is non-200 and disk still exists", func() {
			AppendBadRequest(ghttp.VerifyRequest("DELETE", "/disks/cid"), server)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/disks"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[
	{ "disk_cid": "cid", "orphaned_at": "2016-01-09 06:23:25 +0000" }
]`),
				),
			)

			err := disk.Delete()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Deleting orphaned disk 'cid': Director responded with non-successful status code"))
		})

		It("returns error if response cannot be unmarshalled and disk still exists", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("DELETE", "/disks/cid"),
					ghttp.RespondWith(http.StatusOK, ``),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/disks"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[
	{ "disk_cid": "cid", "orphaned_at": "2016-01-09 06:23:25 +0000" }
]`),
				),
			)

			err := disk.Delete()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Deleting orphaned disk 'cid': Unmarshaling Director response"))
		})
	})

	Describe("OrphanDisk", func() {
		It("orphans disk", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("DELETE", "/disks/cid", "orphan=true"),
					ghttp.VerifyBasicAuth("username", "password"),
				),
				"",
				server,
			)

			Expect(director.OrphanDisk("cid")).ToNot(HaveOccurred())
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("DELETE", "/disks/cid", "orphan=true"), server)

			err := director.OrphanDisk("cid")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Orphaning disk 'cid': Director responded with non-successful status code"))
		})

		It("returns error if response cannot be unmarshalled", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("DELETE", "/disks/cid", "orphan=true"),
					ghttp.RespondWith(http.StatusOK, ``),
				),
			)

			err := director.OrphanDisk("cid")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Orphaning disk 'cid': Unmarshaling Director response"))
		})
	})

})
