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
		director   Director
		deployment Deployment
		server     *ghttp.Server
	)

	BeforeEach(func() {
		director, server = BuildServer()

		var err error

		deployment, err = director.FindDeployment("dep")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("Snapshots", func() {
		It("returns snapshots", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments/dep/snapshots"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[
  {
    "job": "job1",
    "index": 1,
    "snapshot_cid": "snap1-cid",
    "created_at": "2015-01-02 15:04:05 -0000",
    "clean": true
  },
  {
    "job": "job2",
    "index": 2,
    "snapshot_cid": "snap2-cid",
    "created_at": "2016-01-02 15:04:05 -0000",
    "clean": false
  }
]`),
				),
			)

			snaps, err := deployment.Snapshots()
			Expect(err).ToNot(HaveOccurred())
			Expect(snaps).To(HaveLen(2))

			Expect(snaps[0].Job).To(Equal("job1"))
			Expect(*snaps[0].Index).To(Equal(1))
			Expect(snaps[0].CID).To(Equal("snap1-cid"))
			Expect(snaps[0].CreatedAt).To(Equal(time.Date(2015, time.January, 2, 15, 4, 5, 0, time.UTC)))
			Expect(snaps[0].Clean).To(BeTrue())

			Expect(snaps[1].Job).To(Equal("job2"))
			Expect(*snaps[1].Index).To(Equal(2))
			Expect(snaps[1].CID).To(Equal("snap2-cid"))
			Expect(snaps[1].CreatedAt).To(Equal(time.Date(2016, time.January, 2, 15, 4, 5, 0, time.UTC)))
			Expect(snaps[1].Clean).To(BeFalse())
		})

		It("returns an error for invalid created at", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments/dep/snapshots"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[{"created_at":"-"}]`),
				),
			)

			_, err := deployment.Snapshots()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Converting created at"))
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("GET", "/deployments/dep/snapshots"), server)

			_, err := deployment.Snapshots()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Finding snapshots: Director responded with non-successful status code"))
		})

		It("returns error if response cannot be unmarshalled", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments/dep/snapshots"),
					ghttp.RespondWith(http.StatusOK, ``),
				),
			)

			_, err := deployment.Snapshots()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Finding snapshots: Unmarshaling Director response"))
		})
	})

	Describe("TakeSnapshot", func() {
		It("takes snapshot of an instance", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/deployments/dep/jobs/job/id/snapshots"),
					ghttp.VerifyBasicAuth("username", "password"),
				),
				"",
				server,
			)

			err := deployment.TakeSnapshot(NewInstanceSlug("job", "id"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("POST", "/deployments/dep/jobs/job/id/snapshots"), server)

			err := deployment.TakeSnapshot(NewInstanceSlug("job", "id"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Taking snapshot for instance"))
		})
	})

	Describe("DeleteSnapshot", func() {
		It("deletes snapshot", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("DELETE", "/deployments/dep/snapshots/cid"),
					ghttp.VerifyBasicAuth("username", "password"),
				),
				"",
				server,
			)

			err := deployment.DeleteSnapshot("cid")
			Expect(err).ToNot(HaveOccurred())
		})

		It("succeeds even if error occurrs if snapshot no longer exists", func() {
			AppendBadRequest(ghttp.VerifyRequest("DELETE", "/deployments/dep/snapshots/cid"), server)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments/dep/snapshots"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[]`),
				),
			)

			Expect(deployment.DeleteSnapshot("cid")).ToNot(HaveOccurred())
		})

		It("returns delete error if listing snapshots fails", func() {
			AppendBadRequest(ghttp.VerifyRequest("DELETE", "/deployments/dep/snapshots/cid"), server)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments/dep/snapshots"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, ``),
				),
			)

			err := deployment.DeleteSnapshot("cid")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Deleting snapshot 'cid' from deployment"))
		})

		It("returns delete error if response is non-200 and snapshot still exists", func() {
			AppendBadRequest(ghttp.VerifyRequest("DELETE", "/deployments/dep/snapshots/cid"), server)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments/dep/snapshots"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[{"snapshot_cid": "cid"}]`),
				),
			)

			err := deployment.DeleteSnapshot("cid")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Deleting snapshot 'cid' from deployment"))
		})
	})

	Describe("TakeSnapshots", func() {
		It("takes snapshots of the whole deployment", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/deployments/dep/snapshots"),
					ghttp.VerifyBasicAuth("username", "password"),
				),
				"",
				server,
			)

			Expect(deployment.TakeSnapshots()).ToNot(HaveOccurred())
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("POST", "/deployments/dep/snapshots"), server)

			err := deployment.TakeSnapshots()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Taking snapshots for deployment"))
		})
	})

	Describe("DeleteSnapshots", func() {
		It("deletes all deployment snapshots", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("DELETE", "/deployments/dep/snapshots"),
					ghttp.VerifyBasicAuth("username", "password"),
				),
				"",
				server,
			)

			Expect(deployment.DeleteSnapshots()).ToNot(HaveOccurred())
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("DELETE", "/deployments/dep/snapshots"), server)

			err := deployment.DeleteSnapshots()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Deleting snapshots for deployment"))
		})
	})
})
