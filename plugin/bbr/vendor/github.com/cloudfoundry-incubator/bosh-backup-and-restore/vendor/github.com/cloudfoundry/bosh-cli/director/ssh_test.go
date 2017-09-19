package director_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "github.com/cloudfoundry/bosh-cli/director"
)

var _ = Describe("NewSSHOpts", func() {
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

	Describe("SetUpSSH", func() {
		It("sets up SSH sessions without gateway configuration", func() {
			respBody := `[
	{
		"index": 1,
		"host_public_key": "host1-pub-key",
		"ip": "host1-ip",
		"status": "success",
		"command": "setup"
	},
	{
		"index": 2,
		"host_public_key": "host2-pub-key",
		"ip": "host2-ip",
		"status": "success",
		"command": "setup"
	}
]`

			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/deployments/dep/ssh"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyJSONRepresenting(map[string]interface{}{
						"command": "setup",
						"params": map[string]string{
							"user":       "user",
							"public_key": "pub-key",
						},
						"deployment_name": "dep",
						"target": map[string]interface{}{
							"job":     "job",
							"indexes": []string{"index"},
							"ids":     []string{"index"},
						},
					}),
				),
				respBody,
				server,
			)

			slug := NewAllOrInstanceGroupOrInstanceSlug("job", "index")
			opts := SSHOpts{
				Username:  "user",
				PublicKey: "pub-key",
			}

			result, err := deployment.SetUpSSH(slug, opts)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(SSHResult{
				Hosts: []Host{
					{
						Job:       "",
						IndexOrID: "1",

						Username:      "user",
						Host:          "host1-ip",
						HostPublicKey: "host1-pub-key",
					},
					{
						Job:       "",
						IndexOrID: "2",

						Username:      "user",
						Host:          "host2-ip",
						HostPublicKey: "host2-pub-key",
					},
				},

				GatewayUsername: "",
				GatewayHost:     "",
			}))
		})

		It("sets up SSH sessions with Director provided gateway host and username", func() {
			respBody := `[
	{
		"index": 1,
		"host_public_key": "host1-pub-key",
		"ip": "host1-ip",
		"status": "success",
		"command": "setup",

		"gateway_user": "gw-user",
		"gateway_host": "gw-host"
	},
	{
		"index": 2,
		"host_public_key": "host2-pub-key",
		"ip": "host2-ip",
		"status": "success",
		"command": "setup"
	}
]`

			ConfigureTaskResult(ghttp.VerifyRequest("POST", "/deployments/dep/ssh"), respBody, server)

			slug := NewAllOrInstanceGroupOrInstanceSlug("job", "index")
			opts := SSHOpts{
				Username:  "user",
				PublicKey: "pub-key",
			}

			result, err := deployment.SetUpSSH(slug, opts)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(SSHResult{
				Hosts: []Host{
					{
						Job:       "",
						IndexOrID: "1",

						Username:      "user",
						Host:          "host1-ip",
						HostPublicKey: "host1-pub-key",
					},
					{
						Job:       "",
						IndexOrID: "2",

						Username:      "user",
						Host:          "host2-ip",
						HostPublicKey: "host2-pub-key",
					},
				},

				// Assumes that Director returns same gateway information for each one of the hosts
				GatewayUsername: "gw-user",
				GatewayHost:     "gw-host",
			}))
		})

		It("allows to use it with multiple jobs and indicies", func() {
			respBody := `[
	{
		"index": 1,
		"host_public_key": "host1-pub-key",
		"ip": "host1-ip",
		"status": "success",
		"command": "setup"
	}
]`

			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/deployments/dep/ssh"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyJSONRepresenting(map[string]interface{}{
						"command": "setup",
						"params": map[string]string{
							"user":       "user",
							"public_key": "pub-key",
						},
						"deployment_name": "dep",
						"target": map[string]interface{}{
							// Empty string arrays are necessary for Director
							"indexes": []string{},
							"ids":     []string{},
						},
					}),
				),
				respBody,
				server,
			)

			slug := NewAllOrInstanceGroupOrInstanceSlug("", "")
			opts := SSHOpts{
				Username:  "user",
				PublicKey: "pub-key",
			}

			result, err := deployment.SetUpSSH(slug, opts)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(SSHResult{
				Hosts: []Host{
					{
						Job:       "",
						IndexOrID: "1",

						Username:      "user",
						Host:          "host1-ip",
						HostPublicKey: "host1-pub-key",
					},
				},
			}))
		})

		It("picks up ID over index and ignores if index is null", func() {
			respBody := `[
	{
		"index": 1,
		"id": "host1-id",
		"host_public_key": "host1-pub-key",
		"ip": "host1-ip",
		"status": "success",
		"command": "setup"
	},
	{
		"id": "host2-id",
		"host_public_key": "host2-pub-key",
		"ip": "host2-ip",
		"status": "success",
		"command": "setup"
	}
]`

			ConfigureTaskResult(ghttp.VerifyRequest("POST", "/deployments/dep/ssh"), respBody, server)

			slug := NewAllOrInstanceGroupOrInstanceSlug("job", "index")
			opts := SSHOpts{
				Username:  "user",
				PublicKey: "pub-key",
			}

			result, err := deployment.SetUpSSH(slug, opts)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(SSHResult{
				Hosts: []Host{
					{
						Job:       "",
						IndexOrID: "host1-id",

						Username:      "user",
						Host:          "host1-ip",
						HostPublicKey: "host1-pub-key",
					},
					{
						Job:       "",
						IndexOrID: "host2-id",

						Username:      "user",
						Host:          "host2-ip",
						HostPublicKey: "host2-pub-key",
					},
				},
			}))
		})

		It("returns error if no sessions were created", func() {
			ConfigureTaskResult(ghttp.VerifyRequest("POST", "/deployments/dep/ssh"), "[]", server)

			slug := NewAllOrInstanceGroupOrInstanceSlug("job", "index")

			_, err := deployment.SetUpSSH(slug, SSHOpts{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Did not create any SSH sessions for the instances"))
		})

		It("returns error if any session creation fails", func() {
			respBody := `[
	{
		"index": 1,
		"host_public_key": "host1-pub-key",
		"ip": "host1-ip",
		"status": "success",
		"command": "setup"
	},
	{
		"index": 2,
		"host_public_key": "host2-pub-key",
		"ip": "host2-ip",
		"status": "not-success",
		"command": "setup"
	}
]`

			ConfigureTaskResult(ghttp.VerifyRequest("POST", "/deployments/dep/ssh"), respBody, server)

			slug := NewAllOrInstanceGroupOrInstanceSlug("job", "index")

			_, err := deployment.SetUpSSH(slug, SSHOpts{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Failed to set up SSH session for one of the instances"))
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("POST", "/deployments/dep/ssh"), server)

			slug := NewAllOrInstanceGroupOrInstanceSlug("job", "index")

			_, err := deployment.SetUpSSH(slug, SSHOpts{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Setting up SSH in deployment"))
		})

		It("returns error if response cannot be unmarshalled", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/deployments/dep/ssh"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)

			slug := NewAllOrInstanceGroupOrInstanceSlug("job", "index")

			_, err := deployment.SetUpSSH(slug, SSHOpts{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Setting up SSH in deployment 'dep': Unmarshaling Director response"))
		})
	})

	Describe("CleanUpSSH", func() {
		It("cleans up SSH for specific job and index-or-id", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/deployments/dep/ssh"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyJSONRepresenting(map[string]interface{}{
						"command": "cleanup",
						"params": map[string]string{
							"user_regex": "^user",
						},
						"deployment_name": "dep",
						"target": map[string]interface{}{
							"job":     "job",
							"indexes": []string{"index"},
							"ids":     []string{"index"},
						},
					}),
				),
			)

			slug := NewAllOrInstanceGroupOrInstanceSlug("job", "index")
			opts := SSHOpts{
				Username:  "user",
				PublicKey: "pub-key",
			}

			err := deployment.CleanUpSSH(slug, opts)
			Expect(err).ToNot(HaveOccurred())
		})

		It("allows to use it with multiple jobs and indicies", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/deployments/dep/ssh"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyJSONRepresenting(map[string]interface{}{
						"command": "cleanup",
						"params": map[string]string{
							"user_regex": "^user",
						},
						"deployment_name": "dep",
						"target": map[string]interface{}{
							// Empty string arrays are necessary for Director
							"indexes": []string{},
							"ids":     []string{},
						},
					}),
				),
			)

			slug := NewAllOrInstanceGroupOrInstanceSlug("", "")
			opts := SSHOpts{
				Username:  "user",
				PublicKey: "pub-key",
			}

			err := deployment.CleanUpSSH(slug, opts)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("POST", "/deployments/dep/ssh"), server)

			slug := NewAllOrInstanceGroupOrInstanceSlug("job", "index")

			err := deployment.CleanUpSSH(slug, SSHOpts{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Cleaning up SSH in deployment"))
		})
	})
})
