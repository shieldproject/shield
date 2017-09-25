package director_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	semver "github.com/cppforlife/go-semi-semantic/version"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "github.com/cloudfoundry/bosh-cli/director"
)

var _ = Describe("Deployment", func() {
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

	Describe("Name", func() {
		It("returns name", func() {
			Expect(deployment.Name()).To(Equal("dep"))
		})
	})

	Describe("Releases", func() {
		It("returns releases", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[
	{"name": "dep", "releases":[{"name":"rel","version":"ver"}]}
]`),
				),
			)

			rels, err := deployment.Releases()
			Expect(err).ToNot(HaveOccurred())
			Expect(rels[0].Name()).To(Equal("rel"))
			Expect(rels[0].Version()).To(Equal(semver.MustNewVersionFromString("ver")))

			// idempotency check
			rels, err = deployment.Releases()
			Expect(err).ToNot(HaveOccurred())
			Expect(rels[0].Name()).To(Equal("rel"))
			Expect(rels[0].Version()).To(Equal(semver.MustNewVersionFromString("ver")))
		})

		It("returns an error for invalid release versions", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments"),
					ghttp.RespondWith(http.StatusOK, `[
	{"name": "dep", "releases":[{"name":"rel","version":"-"}]}
]`),
				),
			)

			_, err := deployment.Releases()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Parsing version for release"))
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("GET", "/deployments"), server)

			_, err := deployment.Releases()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Finding deployments"))
		})

		It("returns error if response cannot be unmarshalled", func() {
			ConfigureTaskResult(ghttp.VerifyRequest("GET", "/deployments"), "", server)

			_, err := deployment.Releases()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Finding deployments"))
		})
	})

	Describe("Stemcells", func() {
		It("returns stemcells", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[
	{"name": "dep", "stemcells":[{"name":"rel","version":"ver"}]}
]`),
				),
			)

			stems, err := deployment.Stemcells()
			Expect(err).ToNot(HaveOccurred())
			Expect(stems[0].Name()).To(Equal("rel"))
			Expect(stems[0].Version()).To(Equal(semver.MustNewVersionFromString("ver")))

			// idempotency check
			stems, err = deployment.Stemcells()
			Expect(err).ToNot(HaveOccurred())
			Expect(stems[0].Name()).To(Equal("rel"))
			Expect(stems[0].Version()).To(Equal(semver.MustNewVersionFromString("ver")))
		})

		It("returns an error for invalid stemcell versions", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments"),
					ghttp.RespondWith(http.StatusOK, `[
	{"name": "dep", "stemcells":[{"name":"rel","version":"-"}]}
]`),
				),
			)

			_, err := deployment.Stemcells()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Parsing version for stemcell"))
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("GET", "/deployments"), server)

			_, err := deployment.Stemcells()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Finding deployments"))
		})

		It("returns error if response cannot be unmarshalled", func() {
			ConfigureTaskResult(ghttp.VerifyRequest("GET", "/deployments"), "", server)

			_, err := deployment.Stemcells()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Finding deployments"))
		})
	})

	Describe("Manifest", func() {
		It("returns manifest", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments/dep"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `{"manifest":"content"}`),
				),
			)

			man, err := deployment.Manifest()
			Expect(err).ToNot(HaveOccurred())
			Expect(man).To(Equal("content"))
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("GET", "/deployments/dep"), server)

			_, err := deployment.Manifest()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Fetching manifest"))
		})
	})

	Describe("FetchLogs", func() {
		It("returns logs result", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments/dep/jobs/job/id/logs", "type=job"),
					ghttp.VerifyBasicAuth("username", "password"),
				),
				``,
				server,
			)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, `{"result":"logs-blob-id"}`),
				),
			)

			result, err := deployment.FetchLogs(NewAllOrInstanceGroupOrInstanceSlug("job", "id"), nil, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(LogsResult{BlobstoreID: "logs-blob-id", SHA1: ""}))
		})

		It("returns logs result for all deplotment", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments/dep/jobs/*/*/logs", "type=job"),
					ghttp.VerifyBasicAuth("username", "password"),
				),
				``,
				server,
			)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, `{"result":"logs-blob-id"}`),
				),
			)

			result, err := deployment.FetchLogs(NewAllOrInstanceGroupOrInstanceSlug("", ""), nil, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(LogsResult{BlobstoreID: "logs-blob-id", SHA1: ""}))
		})

		It("is able to apply filters and fetch agent logs", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments/dep/jobs/job/id/logs", "type=agent&filters=f1,f2"),
					ghttp.VerifyBasicAuth("username", "password"),
				),
				``,
				server,
			)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, `{"result":"logs-blob-id"}`),
				),
			)

			result, err := deployment.FetchLogs(
				NewAllOrInstanceGroupOrInstanceSlug("job", "id"), []string{"f1", "f2"}, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(LogsResult{BlobstoreID: "logs-blob-id", SHA1: ""}))
		})

		It("returns error if task response is non-200", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments/dep/jobs/job/id/logs", "type=job"),
					ghttp.VerifyBasicAuth("username", "password"),
				),
				``,
				server,
			)

			AppendBadRequest(ghttp.VerifyRequest("GET", "/tasks/123"), server)

			_, err := deployment.FetchLogs(NewAllOrInstanceGroupOrInstanceSlug("job", "id"), nil, false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Finding task '123'"))
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("GET", "/deployments/dep/jobs/job/id/logs", "type=job"), server)

			_, err := deployment.FetchLogs(NewAllOrInstanceGroupOrInstanceSlug("job", "id"), nil, false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Fetching logs"))
		})
	})

	Describe("EnableResurrection", func() {
		It("enables resurrection for all instances and returns without an error", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/deployments/dep/jobs/job/id/resurrection"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"application/json"},
					}),
					ghttp.VerifyBody([]byte(`{"resurrection_paused":false}`)),
				),
			)

			err := deployment.EnableResurrection(NewInstanceSlug("job", "id"), true)
			Expect(err).ToNot(HaveOccurred())
		})

		It("disables resurrection for all instances and returns without an error", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/deployments/dep/jobs/job/id/resurrection"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"application/json"},
					}),
					ghttp.VerifyBody([]byte(`{"resurrection_paused":true}`)),
				),
			)

			err := deployment.EnableResurrection(NewInstanceSlug("job", "id"), false)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("PUT", "/deployments/dep/jobs/job/id/resurrection"), server)

			err := deployment.EnableResurrection(NewInstanceSlug("job", "id"), true)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Changing VM resurrection state"))
		})
	})

	Describe("Ignore", func() {
		It("for an single instance, ignore instance returns without an error", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					///:deployment/instance_groups/:instancegroup/:id/ignore
					ghttp.VerifyRequest("PUT", "/deployments/dep/instance_groups/ig_name/id/ignore"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"application/json"},
					}),
					ghttp.VerifyBody([]byte(`{"ignore":true}`)),
				),
			)

			err := deployment.Ignore(NewInstanceSlug("ig_name", "id"), true)
			Expect(err).ToNot(HaveOccurred())
			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		It("unignores for an instance and returns without an error", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					///:deployment/instance_groups/:instancegroup/:id/ignore
					ghttp.VerifyRequest("PUT", "/deployments/dep/instance_groups/ig_name/id/ignore"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"application/json"},
					}),
					ghttp.VerifyBody([]byte(`{"ignore":false}`)),
				),
			)

			err := deployment.Ignore(NewInstanceSlug("ig_name", "id"), false)
			Expect(err).ToNot(HaveOccurred())
			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		It("should throw an error if an invalid instance slug provided", func() {
			err := deployment.Ignore(InstanceSlug{}, false)
			Expect(err).To(HaveOccurred())
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("PUT", "/deployments/dep/instance_groups/ig_name/id/ignore"), server)

			err := deployment.Ignore(NewInstanceSlug("ig_name", "id"), false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Changing ignore state for 'ig_name/id' in deployment 'dep'"))
		})
	})

	Describe("job states", func() {
		var (
			slug         AllOrInstanceGroupOrInstanceSlug
			force        bool
			dryRun       bool
			startOpts    StartOpts
			stopOpts     StopOpts
			detachedOpts StopOpts
			restartOpts  RestartOpts
			recreateOpts RecreateOpts
		)

		BeforeEach(func() {
			slug = AllOrInstanceGroupOrInstanceSlug{}
			force = false
			dryRun = false

			startOpts = StartOpts{}
			stopOpts = StopOpts{
				SkipDrain: false,
				Force:     force,
			}
			detachedOpts = StopOpts{
				Hard:      true,
				SkipDrain: false,
				Force:     force,
			}
			restartOpts = RestartOpts{}
			recreateOpts = RecreateOpts{}
		})

		states := map[string]func(Deployment) error{
			"started":  func(d Deployment) error { return d.Start(slug, startOpts) },
			"detached": func(d Deployment) error { return d.Stop(slug, detachedOpts) },
			"stopped":  func(d Deployment) error { return d.Stop(slug, stopOpts) },
			"restart":  func(d Deployment) error { return d.Restart(slug, restartOpts) },
			"recreate": func(d Deployment) error { return d.Recreate(slug, recreateOpts) },
		}

		for state, stateFunc := range states {
			state := state
			stateFunc := stateFunc

			Describe(fmt.Sprintf("change state to '%s'", state), func() {
				It("changes state for specific instance", func() {
					slug = NewAllOrInstanceGroupOrInstanceSlug("job", "id")

					ConfigureTaskResult(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("PUT", "/deployments/dep/jobs/job/id", fmt.Sprintf("state=%s", state)),
							ghttp.VerifyBasicAuth("username", "password"),
							ghttp.VerifyHeader(http.Header{
								"Content-Type": []string{"text/yaml"},
							}),
							ghttp.VerifyBody([]byte{}),
						),
						``,
						server,
					)
					err := stateFunc(deployment)
					Expect(err).ToNot(HaveOccurred())
				})

				It("changes state for the whole deployment", func() {
					slug = NewAllOrInstanceGroupOrInstanceSlug("", "")

					ConfigureTaskResult(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("PUT", "/deployments/dep/jobs/*", fmt.Sprintf("state=%s", state)),
							ghttp.VerifyBasicAuth("username", "password"),
							ghttp.VerifyHeader(http.Header{
								"Content-Type": []string{"text/yaml"},
							}),
							ghttp.VerifyBody([]byte{}),
						),
						``,
						server,
					)
					err := stateFunc(deployment)
					Expect(err).ToNot(HaveOccurred())
				})

				It("changes state for all indicies of a job", func() {
					slug = NewAllOrInstanceGroupOrInstanceSlug("job", "")

					ConfigureTaskResult(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("PUT", "/deployments/dep/jobs/job", fmt.Sprintf("state=%s", state)),
							ghttp.VerifyBasicAuth("username", "password"),
							ghttp.VerifyHeader(http.Header{
								"Content-Type": []string{"text/yaml"},
							}),
							ghttp.VerifyBody([]byte{}),
						),
						``,
						server,
					)
					err := stateFunc(deployment)
					Expect(err).ToNot(HaveOccurred())
				})

				It("changes state with canaries and max_in_flight set", func() {
					canaries := "50%"
					maxInFlight := "6"

					switch state {
					case "started":
						startOpts.Canaries = canaries
						startOpts.MaxInFlight = maxInFlight
					case "recreate":
						recreateOpts.Canaries = canaries
						recreateOpts.MaxInFlight = maxInFlight
					case "stopped":
						stopOpts.Canaries = canaries
						stopOpts.MaxInFlight = maxInFlight
					case "detached":
						detachedOpts.Canaries = canaries
						detachedOpts.MaxInFlight = maxInFlight
					case "restart":
						restartOpts.Canaries = canaries
						restartOpts.MaxInFlight = maxInFlight
					}

					query := fmt.Sprintf("state=%s&canaries=%s&max_in_flight=6", state, url.QueryEscape(canaries))

					ConfigureTaskResult(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("PUT", "/deployments/dep/jobs/*", query),
							ghttp.VerifyBasicAuth("username", "password"),
							ghttp.VerifyHeader(http.Header{
								"Content-Type": []string{"text/yaml"},
							}),
							ghttp.VerifyBody([]byte{}),
						),
						``,
						server,
					)
					err := stateFunc(deployment)
					Expect(err).ToNot(HaveOccurred())
				})

				if state == "recreate" {
					It("changes state with dry run", func() {
						recreateOpts.DryRun = true

						query := fmt.Sprintf("state=%s&dry_run=%t", state, recreateOpts.DryRun)

						ConfigureTaskResult(
							ghttp.CombineHandlers(
								ghttp.VerifyRequest("PUT", "/deployments/dep/jobs/*", query),
								ghttp.VerifyBasicAuth("username", "password"),
								ghttp.VerifyHeader(http.Header{
									"Content-Type": []string{"text/yaml"},
								}),
								ghttp.VerifyBody([]byte{}),
							),
							``,
							server,
						)
						err := stateFunc(deployment)
						Expect(err).ToNot(HaveOccurred())
					})

					It("changes state with fix", func() {
						recreateOpts.Fix = true

						query := fmt.Sprintf("state=%s&fix=%t", state, recreateOpts.Fix)

						ConfigureTaskResult(
							ghttp.CombineHandlers(
								ghttp.VerifyRequest("PUT", "/deployments/dep/jobs/*", query),
								ghttp.VerifyBasicAuth("username", "password"),
								ghttp.VerifyHeader(http.Header{
									"Content-Type": []string{"text/yaml"},
								}),
								ghttp.VerifyBody([]byte{}),
							),
							``,
							server,
						)
						err := stateFunc(deployment)
						Expect(err).ToNot(HaveOccurred())
					})
				}
				if state != "started" {
					It("changes state with skipping drain and forcing", func() {
						slug = NewAllOrInstanceGroupOrInstanceSlug("", "")
						force = true

						switch state {
						case "recreate":
							recreateOpts.SkipDrain = true
							recreateOpts.Force = force
						case "stopped":
							stopOpts.SkipDrain = true
							stopOpts.Force = force
						case "detached":
							detachedOpts.SkipDrain = true
							detachedOpts.Force = force
						case "restart":
							restartOpts.SkipDrain = true
							restartOpts.Force = force
						}

						query := fmt.Sprintf("state=%s&skip_drain=true&force=true", state)

						ConfigureTaskResult(
							ghttp.CombineHandlers(
								ghttp.VerifyRequest("PUT", "/deployments/dep/jobs/*", query),
								ghttp.VerifyBasicAuth("username", "password"),
								ghttp.VerifyHeader(http.Header{
									"Content-Type": []string{"text/yaml"},
								}),
								ghttp.VerifyBody([]byte{}),
							),
							``,
							server,
						)
						err := stateFunc(deployment)
						Expect(err).ToNot(HaveOccurred())
					})
				}

				It("returns an error if changing state response is non-200", func() {
					AppendBadRequest(ghttp.VerifyRequest("PUT", "/deployments/dep/jobs/*"), server)

					err := stateFunc(deployment)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Changing state"))
				})
			})
		}
	})

	Describe("ExportRelease", func() {
		var (
			relSlug ReleaseSlug
			osSlug  OSVersionSlug
		)

		BeforeEach(func() {
			relSlug = NewReleaseSlug("rel", "1")
			osSlug = NewOSVersionSlug("os", "2")
		})

		It("returns exported release result", func() {
			reqBody := `{
"deployment_name":"dep",
"release_name":"rel",
"release_version":"1",
"sha2":true,
"stemcell_os":"os",
"stemcell_version":"2"
}`

			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/releases/export"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"application/json"},
					}),
					ghttp.VerifyBody([]byte(strings.Replace(reqBody, "\n", "", -1))),
				),
				`{"blobstore_id":"release-blob-id","sha1":"release-sha1"}`,
				server,
			)

			result, err := deployment.ExportRelease(relSlug, osSlug)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ExportReleaseResult{
				BlobstoreID: "release-blob-id",
				SHA1:        "release-sha1",
			}))
		})

		It("returns error if task response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("POST", "/releases/export"), server)

			_, err := deployment.ExportRelease(relSlug, osSlug)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Exporting release"))
		})

		It("returns error if response cannot be unmarshalled", func() {
			ConfigureTaskResult(ghttp.VerifyRequest("POST", "/releases/export"), ``, server)

			_, err := deployment.ExportRelease(relSlug, osSlug)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unmarshaling export release result"))
		})
	})

	Describe("Update", func() {
		It("succeeds updating deployment", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/deployments", ""),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"text/yaml"},
					}),
					ghttp.VerifyBody([]byte("manifest")),
				),
				``,
				server,
			)

			err := deployment.Update([]byte("manifest"), UpdateOpts{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("succeeds updating deployment with recreate, fix and skip drain flags", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/deployments", "recreate=true&fix=true&skip_drain=*"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"text/yaml"},
					}),
					ghttp.VerifyBody([]byte("manifest")),
				),
				``,
				server,
			)

			updateOpts := UpdateOpts{
				Recreate:  true,
				Fix:       true,
				SkipDrain: SkipDrains{SkipDrain{All: true}},
			}
			err := deployment.Update([]byte("manifest"), updateOpts)
			Expect(err).ToNot(HaveOccurred())
		})

		It("succeeds updating deployment with canaries and max-in-flight flags", func() {
			canaries := "100%"

			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/deployments", fmt.Sprintf("canaries=%s&max_in_flight=5", url.QueryEscape(canaries))),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"text/yaml"},
					}),
					ghttp.VerifyBody([]byte("manifest")),
				),
				``,
				server,
			)

			updateOpts := UpdateOpts{
				Canaries:    canaries,
				MaxInFlight: "5",
			}
			err := deployment.Update([]byte("manifest"), updateOpts)
			Expect(err).ToNot(HaveOccurred())
		})

		It("succeeds updating deployment with dry-run flags", func() {
			dryRun := true

			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/deployments", fmt.Sprintf("dry_run=%t", dryRun)),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"text/yaml"},
					}),
					ghttp.VerifyBody([]byte("manifest")),
				),
				``,
				server,
			)

			updateOpts := UpdateOpts{
				DryRun: dryRun,
			}
			err := deployment.Update([]byte("manifest"), updateOpts)
			Expect(err).ToNot(HaveOccurred())
		})

		It("succeeds updating deployment with diff context values", func() {
			context := map[string]interface{}{
				"cloud_config_id":          "2",
				"runtime_config_id":        4,
				"some_other_context_field": "value",
			}

			requestParams := "context=%7B%22cloud_config_id%22%3A%222%22%2C%22runtime_config_id%22%3A4%2C%22some_other_context_field%22%3A%22value%22%7D"
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/deployments", requestParams),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"text/yaml"},
					}),
					ghttp.VerifyBody([]byte("manifest")),
				),
				``,
				server,
			)

			updateOpts := UpdateOpts{
				Diff: NewDeploymentDiff(nil, context),
			}

			err := deployment.Update([]byte("manifest"), updateOpts)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error if task response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("POST", "/deployments"), server)

			err := deployment.Update([]byte("manifest"), UpdateOpts{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Updating deployment"))
		})
	})

	Describe("Delete", func() {
		It("succeeds deleting", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("DELETE", "/deployments/dep", ""),
					ghttp.VerifyBasicAuth("username", "password"),
				),
				``,
				server,
			)

			Expect(deployment.Delete(false)).ToNot(HaveOccurred())
		})

		It("succeeds deleting with force flag", func() {
			ConfigureTaskResult(ghttp.VerifyRequest("DELETE", "/deployments/dep", "force=true"), ``, server)

			Expect(deployment.Delete(true)).ToNot(HaveOccurred())
		})

		It("succeeds even if error occurrs if deployment no longer exists", func() {
			AppendBadRequest(ghttp.VerifyRequest("DELETE", "/deployments/dep"), server)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[]`),
				),
			)

			Expect(deployment.Delete(false)).ToNot(HaveOccurred())
		})

		It("returns delete error if listing deployments fails", func() {
			AppendBadRequest(ghttp.VerifyRequest("DELETE", "/deployments/dep"), server)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, ``),
				),
			)

			err := deployment.Delete(false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Deleting deployment 'dep': Director responded with non-successful status code"))
		})

		It("returns delete error if response is non-200 and deployment still exists", func() {
			AppendBadRequest(ghttp.VerifyRequest("DELETE", "/deployments/dep"), server)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[{"name": "dep"}]`),
				),
			)

			err := deployment.Delete(false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Deleting deployment 'dep': Director responded with non-successful status code"))
		})
	})

	Describe("AttachDisk", func() {
		It("calls attachdisk director api", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/disks/disk_cid/attachments", "deployment=dep&job=dea&instance_id=17f01a35-bf9c-4949-bcf2-c07a95e4df33"),
					ghttp.VerifyBasicAuth("username", "password"),
				),
				"",
				server,
			)

			err := deployment.AttachDisk(NewInstanceSlug("dea", "17f01a35-bf9c-4949-bcf2-c07a95e4df33"), "disk_cid")
			Expect(err).ToNot(HaveOccurred())
		})

		Context("director returns a non-200 response", func() {
			It("should return an error", func() {
				ConfigureTaskResult(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("PUT", "/disks/disk_cid/attachments", "deployment=dep&job=dea&instance_id=17f01a35-bf9c-4949-bcf2-c07a95e4df33"),
						ghttp.VerifyBasicAuth("username", "password"),
						ghttp.RespondWith(500, "Internal Server Error"),
					),
					"",
					server,
				)

				err := deployment.AttachDisk(NewInstanceSlug("dea", "17f01a35-bf9c-4949-bcf2-c07a95e4df33"), "disk_cid")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Variables", func() {
		It("returns the list of placeholder variables", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments/dep/variables"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[
						{"id": "1", "name": "foo-1"},
						{"id": "2", "name": "foo-2"}
					]`),
				),
			)

			result, err := deployment.Variables()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(result)).To(Equal(2))

			Expect(result[0].ID).To(Equal("1"))
			Expect(result[0].Name).To(Equal("foo-1"))

			Expect(result[1].ID).To(Equal("2"))
			Expect(result[1].Name).To(Equal("foo-2"))
		})

		It("returns an empty list if there are no placeholder variables", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments/dep/variables"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[]`),
				),
			)

			result, err := deployment.Variables()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(result)).To(Equal(0))
		})

		It("errors if fetching placeholder variables fails", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments/dep/variables"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusInternalServerError, ""),
				))

			_, err := deployment.Variables()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("Error fetching variables for deployment 'dep'"))
		})
	})

	Describe("using a director with context", func() {
		contextId := "example-context-id"

		BeforeEach(func() {
			var err error
			directorWithContextId := director.WithContext(contextId)

			deployment, err = directorWithContextId.FindDeployment("dep")
			Expect(err).ToNot(HaveOccurred())
		})

		It("adds context to request headers", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/deployments", ""),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type":      []string{"text/yaml"},
						"X-Bosh-Context-Id": []string{contextId},
					}),
					ghttp.VerifyBody([]byte("manifest")),
				),
				``,
				server,
			)

			err := deployment.Update([]byte("manifest"), UpdateOpts{})
			Expect(err).ToNot(HaveOccurred())
		})

	})

})
