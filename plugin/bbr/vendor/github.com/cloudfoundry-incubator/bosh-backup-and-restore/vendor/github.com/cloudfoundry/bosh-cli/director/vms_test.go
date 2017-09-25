package director_test

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "github.com/cloudfoundry/bosh-cli/director"
)

var _ = Describe("VMs", func() {
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

	Describe("VMInfos", func() {
		It("returns vm infos", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments/dep/vms", "format=full"),
					ghttp.VerifyBasicAuth("username", "password"),
				),
				strings.Replace(`{
	"agent_id": "agent-id",
	"job_name": "job",
	"id": "id",
	"index": 1,
	"job_state": "running",
	"bootstrap": true,
	"ips": [ "ip" ],
	"dns": [ "dns" ],
	"az": "az",
	"ignore": true,
	"vm_cid": "vm-cid",
	"disk_cid": "disk-cid",
	"disk_cids": ["disk-cid1", "disk-cid2"],
	"vm_type": "vm-type",
	"resource_pool": "rp",
	"processes": [{
		"name": "service",
		"state": "running",
		"uptime": { "secs": 343020 },
		"cpu": { "total": 10 },
		"mem": { "percent": 0.5, "kb": 23952 }
	}],
	"vitals": {
		"cpu": { "wait": "0.8", "user": "65.7", "sys": "4.5" },
		"swap": { "percent": "5", "kb": "53580" },
		"mem": { "percent": "33", "kb": "1342088" },
		"uptime": { "secs": 10020 },
		"load": [ "2.20", "1.63", "1.53" ],
		"disk": {
			"system": { "percent": "47", "inode_percent": "19" },
			"ephemeral": { "percent": "47", "inode_percent": "19" }
		}
	},
	"resurrection_paused": true
}`, "\n", "", -1),
				server,
			)

			infos, err := deployment.VMInfos()
			Expect(err).ToNot(HaveOccurred())
			Expect(infos).To(HaveLen(1))

			index := 1
			uptime := uint64(10020)
			procCPUTotal := 10.0
			procMemPer := 0.5
			procMemKB := uint64(23952)
			procUptime := uint64(343020)

			Expect(infos[0]).To(Equal(VMInfo{
				AgentID: "agent-id",

				JobName:      "job",
				ID:           "id",
				Index:        &index,
				ProcessState: "running",
				Bootstrap:    true,

				IPs: []string{"ip"},
				DNS: []string{"dns"},

				AZ:           "az",
				Ignore:       true,
				VMID:         "vm-cid",
				VMType:       "vm-type",
				ResourcePool: "rp",
				DiskID:       "disk-cid",
				DiskIDs:      []string{"disk-cid1", "disk-cid2"},

				Processes: []VMInfoProcess{
					VMInfoProcess{
						Name:   "service",
						State:  "running",
						CPU:    VMInfoVitalsCPU{Total: &procCPUTotal},
						Mem:    VMInfoVitalsMemIntSize{KB: &procMemKB, Percent: &procMemPer},
						Uptime: VMInfoVitalsUptime{Seconds: &procUptime},
					},
				},

				Vitals: VMInfoVitals{
					CPU:    VMInfoVitalsCPU{Sys: "4.5", User: "65.7", Wait: "0.8"},
					Mem:    VMInfoVitalsMemSize{KB: "1342088", Percent: "33"},
					Swap:   VMInfoVitalsMemSize{KB: "53580", Percent: "5"},
					Uptime: VMInfoVitalsUptime{Seconds: &uptime},
					Load:   []string{"2.20", "1.63", "1.53"},
					Disk: map[string]VMInfoVitalsDiskSize{
						"system":    VMInfoVitalsDiskSize{InodePercent: "19", Percent: "47"},
						"ephemeral": VMInfoVitalsDiskSize{InodePercent: "19", Percent: "47"},
					},
				},

				ResurrectionPaused: true,
			}))
		})

		It("correctly parses disk cids", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments/dep/vms", "format=full"),
					ghttp.VerifyBasicAuth("username", "password"),
				),
				strings.Replace(`{
	"agent_id": "agent-id",
	"job_name": "job",
	"id": "id",
	"index": 1,
	"job_state": "running",
	"bootstrap": true,
	"ips": [ "ip" ],
	"dns": [ "dns" ],
	"az": "az",
	"vm_cid": "vm-cid",
	"disk_cid": "disk-cid",
	"disk_cids": [],
	"vm_type": "vm-type",
	"resource_pool": "rp",
	"processes": [{
		"name": "service",
		"state": "running",
		"uptime": { "secs": 343020 },
		"cpu": { "total": 10 },
		"mem": { "percent": 0.5, "kb": 23952 }
	}],
	"vitals": {
		"cpu": { "wait": "0.8", "user": "65.7", "sys": "4.5" },
		"swap": { "percent": "5", "kb": "53580" },
		"mem": { "percent": "33", "kb": "1342088" },
		"uptime": { "secs": 10020 },
		"load": [ "2.20", "1.63", "1.53" ],
		"disk": {
			"system": { "percent": "47", "inode_percent": "19" },
			"ephemeral": { "percent": "47", "inode_percent": "19" }
		}
	},
	"resurrection_paused": true
}`, "\n", "", -1),
				server,
			)

			infos, err := deployment.VMInfos()
			Expect(err).ToNot(HaveOccurred())
			Expect(infos).To(HaveLen(1))

			index := 1
			uptime := uint64(10020)
			procCPUTotal := 10.0
			procMemPer := 0.5
			procMemKB := uint64(23952)
			procUptime := uint64(343020)

			Expect(infos[0]).To(Equal(VMInfo{
				AgentID: "agent-id",

				JobName:      "job",
				ID:           "id",
				Index:        &index,
				ProcessState: "running",
				Bootstrap:    true,

				IPs: []string{"ip"},
				DNS: []string{"dns"},

				AZ:           "az",
				VMID:         "vm-cid",
				VMType:       "vm-type",
				ResourcePool: "rp",
				DiskID:       "disk-cid",
				DiskIDs:      []string{"disk-cid"},

				Processes: []VMInfoProcess{
					VMInfoProcess{
						Name:   "service",
						State:  "running",
						CPU:    VMInfoVitalsCPU{Total: &procCPUTotal},
						Mem:    VMInfoVitalsMemIntSize{KB: &procMemKB, Percent: &procMemPer},
						Uptime: VMInfoVitalsUptime{Seconds: &procUptime},
					},
				},

				Vitals: VMInfoVitals{
					CPU:    VMInfoVitalsCPU{Sys: "4.5", User: "65.7", Wait: "0.8"},
					Mem:    VMInfoVitalsMemSize{KB: "1342088", Percent: "33"},
					Swap:   VMInfoVitalsMemSize{KB: "53580", Percent: "5"},
					Uptime: VMInfoVitalsUptime{Seconds: &uptime},
					Load:   []string{"2.20", "1.63", "1.53"},
					Disk: map[string]VMInfoVitalsDiskSize{
						"system":    VMInfoVitalsDiskSize{InodePercent: "19", Percent: "47"},
						"ephemeral": VMInfoVitalsDiskSize{InodePercent: "19", Percent: "47"},
					},
				},

				ResurrectionPaused: true,
			}))

		})

		It("correctly parses disk cids when no persistent disks are present", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments/dep/vms", "format=full"),
					ghttp.VerifyBasicAuth("username", "password"),
				),
				strings.Replace(`{
	"agent_id": "agent-id",
	"job_name": "job",
	"id": "id",
	"index": 1,
	"job_state": "running",
	"bootstrap": true,
	"ips": [ "ip" ],
	"dns": [ "dns" ],
	"az": "az",
	"vm_cid": "vm-cid",
	"disk_cid": "",
	"disk_cids": [],
	"vm_type": "vm-type",
	"resource_pool": "rp",
	"processes": [{
		"name": "service",
		"state": "running",
		"uptime": { "secs": 343020 },
		"cpu": { "total": 10 },
		"mem": { "percent": 0.5, "kb": 23952 }
	}],
	"vitals": {
		"cpu": { "wait": "0.8", "user": "65.7", "sys": "4.5" },
		"swap": { "percent": "5", "kb": "53580" },
		"mem": { "percent": "33", "kb": "1342088" },
		"uptime": { "secs": 10020 },
		"load": [ "2.20", "1.63", "1.53" ],
		"disk": {
			"system": { "percent": "47", "inode_percent": "19" },
			"ephemeral": { "percent": "47", "inode_percent": "19" }
		}
	},
	"resurrection_paused": true
}`, "\n", "", -1),
				server,
			)

			infos, err := deployment.VMInfos()
			Expect(err).ToNot(HaveOccurred())
			Expect(infos).To(HaveLen(1))

			index := 1
			uptime := uint64(10020)
			procCPUTotal := 10.0
			procMemPer := 0.5
			procMemKB := uint64(23952)
			procUptime := uint64(343020)

			Expect(infos[0]).To(Equal(VMInfo{
				AgentID: "agent-id",

				JobName:      "job",
				ID:           "id",
				Index:        &index,
				ProcessState: "running",
				Bootstrap:    true,

				IPs: []string{"ip"},
				DNS: []string{"dns"},

				AZ:           "az",
				VMID:         "vm-cid",
				VMType:       "vm-type",
				ResourcePool: "rp",
				DiskID:       "",
				DiskIDs:      []string{},

				Processes: []VMInfoProcess{
					VMInfoProcess{
						Name:   "service",
						State:  "running",
						CPU:    VMInfoVitalsCPU{Total: &procCPUTotal},
						Mem:    VMInfoVitalsMemIntSize{KB: &procMemKB, Percent: &procMemPer},
						Uptime: VMInfoVitalsUptime{Seconds: &procUptime},
					},
				},

				Vitals: VMInfoVitals{
					CPU:    VMInfoVitalsCPU{Sys: "4.5", User: "65.7", Wait: "0.8"},
					Mem:    VMInfoVitalsMemSize{KB: "1342088", Percent: "33"},
					Swap:   VMInfoVitalsMemSize{KB: "53580", Percent: "5"},
					Uptime: VMInfoVitalsUptime{Seconds: &uptime},
					Load:   []string{"2.20", "1.63", "1.53"},
					Disk: map[string]VMInfoVitalsDiskSize{
						"system":    VMInfoVitalsDiskSize{InodePercent: "19", Percent: "47"},
						"ephemeral": VMInfoVitalsDiskSize{InodePercent: "19", Percent: "47"},
					},
				},

				ResurrectionPaused: true,
			}))

		})

		It("returns vm infos with running vms", func() {
			ConfigureTaskResult(
				ghttp.VerifyRequest("GET", "/deployments/dep/vms", "format=full"),
				`
{"job_state":"running"}
{"job_state":"running","processes":[{"state": "running"}]}
{"job_state":"running","processes":[{"state": "running"},{"state": "failing"}]}
{"job_state":"failing","processes":[{"state": "running"},{"state": "running"}]}
`,
				server,
			)

			infos, err := deployment.VMInfos()
			Expect(err).ToNot(HaveOccurred())
			Expect(infos[0].IsRunning()).To(BeTrue())
			Expect(infos[1].IsRunning()).To(BeTrue())
			Expect(infos[2].IsRunning()).To(BeFalse())
			Expect(infos[3].IsRunning()).To(BeFalse())
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("GET", "/deployments/dep/vms", "format=full"), server)

			_, err := deployment.VMInfos()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Listing deployment 'dep' vms infos: Director responded with non-successful status code"))
		})

		It("returns error if infos cannot be unmarshalled", func() {
			ConfigureTaskResult(ghttp.VerifyRequest("GET", "/deployments/dep/vms", "format=full"), `-`, server)

			_, err := deployment.VMInfos()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unmarshaling vm info response"))
		})
	})
})
