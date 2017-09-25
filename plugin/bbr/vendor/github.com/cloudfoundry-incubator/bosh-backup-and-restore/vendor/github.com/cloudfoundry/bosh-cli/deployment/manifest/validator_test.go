package manifest_test

import (
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/deployment/manifest"
	boshinst "github.com/cloudfoundry/bosh-cli/installation"
	boshjob "github.com/cloudfoundry/bosh-cli/release/job"
	birelmanifest "github.com/cloudfoundry/bosh-cli/release/manifest"
	fakerel "github.com/cloudfoundry/bosh-cli/release/releasefakes"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
	birelsetmanifest "github.com/cloudfoundry/bosh-cli/release/set/manifest"
)

var _ = Describe("Validator", func() {
	var (
		logger         boshlog.Logger
		releaseManager boshinst.ReleaseManager
		validator      Validator

		validManifest           Manifest
		validReleaseSetManifest birelsetmanifest.Manifest
		release                 *fakerel.FakeRelease
	)

	BeforeEach(func() {
		logger = boshlog.NewLogger(boshlog.LevelNone)
		releaseManager = boshinst.NewReleaseManager(logger)
		validManifest = Manifest{
			Name: "fake-deployment-name",
			Networks: []Network{
				{
					Name: "fake-network-name",
					Type: "dynamic",
				},
			},
			ResourcePools: []ResourcePool{
				{
					Name:    "fake-resource-pool-name",
					Network: "fake-network-name",
					CloudProperties: biproperty.Map{
						"fake-prop-key": "fake-prop-value",
						"fake-prop-map-key": biproperty.Map{
							"fake-prop-key": "fake-prop-value",
						},
					},
					Stemcell: StemcellRef{
						URL: "file://fake-stemcell-url",
					},
				},
			},
			DiskPools: []DiskPool{
				{
					Name:     "fake-disk-pool-name",
					DiskSize: 1024,
					CloudProperties: biproperty.Map{
						"fake-prop-key": "fake-prop-value",
						"fake-prop-map-key": biproperty.Map{
							"fake-prop-key": "fake-prop-value",
						},
					},
				},
			},
			Jobs: []Job{
				{
					Name: "fake-job-name",
					Templates: []ReleaseJobRef{
						{
							Name:    "fake-job-name",
							Release: "fake-release-name",
						},
					},
					PersistentDisk: 1024,
					ResourcePool:   "fake-resource-pool-name",
					Networks: []JobNetwork{
						{
							Name:     "fake-network-name",
							Defaults: []NetworkDefault{NetworkDefaultDNS, NetworkDefaultGateway},
						},
					},
					Lifecycle: "service",
					Properties: biproperty.Map{
						"fake-prop-key": "fake-prop-value",
						"fake-prop-map-key": biproperty.Map{
							"fake-prop-key": "fake-prop-value",
						},
					},
				},
			},
			Properties: biproperty.Map{
				"fake-prop-key": "fake-prop-value",
				"fake-prop-map-key": biproperty.Map{
					"fake-prop-key": "fake-prop-value",
				},
			},
		}

		validReleaseSetManifest = birelsetmanifest.Manifest{
			Releases: []birelmanifest.ReleaseRef{
				{
					Name: "fake-release-name",
				},
			},
		}

		release = &fakerel.FakeRelease{
			NameStub:    func() string { return "fake-release-name" },
			VersionStub: func() string { return "1.0" },
		}
		release.JobsReturns([]*boshjob.Job{
			boshjob.NewJob(NewResource("fake-job-name", "", nil)),
		})
		releaseManager.Add(release)
		validator = NewValidator(logger)
	})

	Describe("Validate", func() {
		It("does not error if deployment is valid", func() {
			deploymentManifest := validManifest

			err := validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).ToNot(HaveOccurred())
		})

		It("validates name is not empty", func() {
			deploymentManifest := Manifest{
				Name: "",
			}

			err := validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("name must be provided"))
		})

		It("validates name is not blank", func() {
			deploymentManifest := Manifest{
				Name: "   \t",
			}

			err := validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("name must be provided"))
		})

		It("validates resource pool name", func() {
			deploymentManifest := Manifest{
				ResourcePools: []ResourcePool{
					{
						Name: "",
					},
				},
			}

			err := validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("resource_pools[0].name must be provided"))

			deploymentManifest = Manifest{
				ResourcePools: []ResourcePool{
					{
						Name: "not-blank",
					},
					{
						Name: "   \t",
					},
				},
			}

			err = validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("resource_pools[1].name must be provided"))
		})

		It("validates resource pool network", func() {
			deploymentManifest := Manifest{
				ResourcePools: []ResourcePool{
					{
						Network: "",
					},
				},
			}

			err := validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("resource_pools[0].network must be provided"))

			deploymentManifest = Manifest{
				Networks: []Network{
					{
						Name: "fake-network",
					},
				},
				ResourcePools: []ResourcePool{
					{
						Network: "other-network",
					},
				},
			}

			err = validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("resource_pools[0].network must be the name of a network"))
		})

		It("validates resource pool stemcell", func() {
			deploymentManifest := Manifest{
				ResourcePools: []ResourcePool{
					{
						Stemcell: StemcellRef{},
					},
				},
			}

			err := validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("resource_pools[0].stemcell.url must be provided"))

			deploymentManifest = Manifest{
				ResourcePools: []ResourcePool{
					{
						Stemcell: StemcellRef{
							URL: "invalid-url",
						},
					},
				},
			}

			err = validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("resource_pools[0].stemcell.url must be a valid URL (file:// or http(s)://)"))

			deploymentManifest = Manifest{
				ResourcePools: []ResourcePool{
					{
						Stemcell: StemcellRef{
							URL: "https://invalid-url",
						},
					},
				},
			}

			err = validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("resource_pools[0].stemcell.sha1 must be provided for http URL"))
		})

		It("validates disk pool name", func() {
			deploymentManifest := Manifest{
				DiskPools: []DiskPool{
					{
						Name: "",
					},
				},
			}

			err := validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("disk_pools[0].name must be provided"))

			deploymentManifest = Manifest{
				DiskPools: []DiskPool{
					{
						Name: "not-blank",
					},
					{
						Name: "   \t",
					},
				},
			}

			err = validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("disk_pools[1].name must be provided"))
		})

		It("validates disk pool size", func() {
			deploymentManifest := Manifest{
				DiskPools: []DiskPool{
					{
						Name: "fake-disk",
					},
				},
			}

			err := validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("disk_pools[0].disk_size must be > 0"))
		})

		Describe("networks", func() {
			It("validates name is present", func() {
				deploymentManifest := Manifest{
					Networks: []Network{
						{
							Name: "",
						},
					},
				}

				err := validator.Validate(deploymentManifest, validReleaseSetManifest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("networks[0].name must be provided"))

				deploymentManifest = Manifest{
					Networks: []Network{
						{
							Name: "not-blank",
						},
						{
							Name: "   \t",
						},
					},
				}

				err = validator.Validate(deploymentManifest, validReleaseSetManifest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("networks[1].name must be provided"))
			})

			It("validates network type is manual, dynamic, or vip", func() {
				typeError := "networks[0].type must be 'manual', 'dynamic', or 'vip'"

				err := validator.Validate(Manifest{
					Networks: []Network{
						{Type: "unknown-type"},
					},
				},
					validReleaseSetManifest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(typeError))

				err = validator.Validate(Manifest{
					Networks: []Network{
						{Type: "vip"},
					},
				},
					validReleaseSetManifest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).ToNot(ContainSubstring(typeError))

				err = validator.Validate(Manifest{
					Networks: []Network{
						{Type: "manual"},
					},
				},
					validReleaseSetManifest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).ToNot(ContainSubstring(typeError))

				err = validator.Validate(Manifest{
					Networks: []Network{
						{Type: "dynamic"},
					},
				},
					validReleaseSetManifest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).ToNot(ContainSubstring(typeError))
			})

			Context("manual networks", func() {
				It("validates that there is exactly 1 subnet", func() {
					deploymentManifest := Manifest{
						Networks: []Network{
							{
								Type:    "manual",
								Subnets: []Subnet{},
							},
						},
					}

					err := validator.Validate(deploymentManifest, validReleaseSetManifest)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("networks[0].subnets must be of size 1"))
				})

				It("validates that range is present", func() {
					deploymentManifest := Manifest{
						Networks: []Network{
							{
								Type:    "manual",
								Subnets: []Subnet{{}},
							},
						},
					}

					err := validator.Validate(deploymentManifest, validReleaseSetManifest)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("networks[0].subnets[0].range must be provided"))
				})

				It("validates that gateway is present", func() {
					deploymentManifest := Manifest{
						Networks: []Network{
							{
								Type:    "manual",
								Subnets: []Subnet{{}},
							},
						},
					}

					err := validator.Validate(deploymentManifest, validReleaseSetManifest)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("networks[0].subnets[0].gateway must be provided"))
				})

				It("validates that range is an ip range", func() {
					validationError := "networks[0].subnets[0].range must be an ip range"

					err := validator.Validate(Manifest{
						Networks: []Network{
							{
								Type: "manual",
								Subnets: []Subnet{{
									Range:   "not-an-ip",
									Gateway: "10.0.0.1",
								}},
							},
						},
					}, validReleaseSetManifest)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(validationError))

					err = validator.Validate(Manifest{
						Networks: []Network{
							{
								Type: "manual",
								Subnets: []Subnet{{
									Range:   "10.10.0.0",
									Gateway: "10.0.0.1",
								}},
							},
						},
					}, validReleaseSetManifest)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(validationError))
				})

				It("validates that gateway is an ip", func() {
					validationError := "networks[0].subnets[0].gateway must be an ip"

					err := validator.Validate(Manifest{
						Networks: []Network{
							{
								Type: "manual",
								Subnets: []Subnet{{
									Range:   "10.10.0.0/24",
									Gateway: "not-an-ip",
								}},
							},
						},
					}, validReleaseSetManifest)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(validationError))
				})

				It("validates that gateway is within the range", func() {
					validationError := "subnet gateway '10.0.0.1' must be within the specified range '10.10.0.0/24'"

					err := validator.Validate(Manifest{
						Networks: []Network{
							{
								Type: "manual",
								Subnets: []Subnet{{
									Range:   "10.10.0.0/24",
									Gateway: "10.0.0.1",
								}},
							},
						},
					}, validReleaseSetManifest)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(validationError))

					err = validator.Validate(Manifest{
						Networks: []Network{
							{
								Type: "manual",
								Subnets: []Subnet{{
									Range:   "10.10.0.0/24",
									Gateway: "10.10.0.1",
								}},
							},
						},
					}, validReleaseSetManifest)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).ToNot(ContainSubstring(validationError))
				})

				It("validates that the gateway is not the first ip in the range", func() {
					err := validator.Validate(Manifest{
						Networks: []Network{
							{
								Type: "manual",
								Subnets: []Subnet{{
									Range:   "10.10.0.0/24",
									Gateway: "10.10.0.0",
								}},
							},
						},
					}, validReleaseSetManifest)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("subnet gateway can't be the network address '10.10.0.0'"))
				})

				It("validates that the gateway is not the last ip in the range", func() {
					err := validator.Validate(Manifest{
						Networks: []Network{
							{
								Type: "manual",
								Subnets: []Subnet{{
									Range:   "10.10.0.0/24",
									Gateway: "10.10.0.255",
								}},
							},
						},
					}, validReleaseSetManifest)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("subnet gateway can't be the broadcast address '10.10.0.255'"))
				})
			})

			Context("dynamic networks", func() {
				It("does not validate that a static IP address is within the range", func() {
					validationError := "jobs[0].networks[0] static ip '10.10.0.42' must be within subnet range"
					err := validator.Validate(Manifest{
						Name: "fake-deployment-name",
						Networks: []Network{
							{
								Name: "fake-network-name",
								Type: "dynamic",
							},
						},
						Jobs: []Job{
							{
								Networks: []JobNetwork{
									{
										StaticIPs: []string{"10.10.0.42"},
									},
								},
							},
						},
					}, validReleaseSetManifest)
					Expect(err.Error()).ToNot(ContainSubstring(validationError))
				})
			})

			Context("VIP networks", func() {
				It("does not validate that a static IP address is within the range", func() {
					validationError := "jobs[0].networks[0] static ip '10.10.0.42' must be within subnet range"
					err := validator.Validate(Manifest{
						Name: "fake-deployment-name",
						Networks: []Network{
							{
								Name: "fake-network-name",
								Type: "vip",
							},
						},
						Jobs: []Job{
							{
								Networks: []JobNetwork{
									{
										StaticIPs: []string{"10.10.0.42"},
									},
								},
							},
						},
					}, validReleaseSetManifest)
					Expect(err.Error()).ToNot(ContainSubstring(validationError))
				})
			})
		})

		It("validates that there is only one job", func() {
			deploymentManifest := Manifest{
				Jobs: []Job{
					{},
					{},
				},
			}

			err := validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("jobs must be of size 1"))
		})

		It("validates job name", func() {
			deploymentManifest := Manifest{
				Jobs: []Job{
					{
						Name: "",
					},
				},
			}

			err := validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("jobs[0].name must be provided"))

			deploymentManifest = Manifest{
				Jobs: []Job{
					{
						Name: "not-blank",
					},
					{
						Name: "   \t",
					},
				},
			}

			err = validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("jobs[1].name must be provided"))
		})

		It("validates job persistent_disk", func() {
			deploymentManifest := Manifest{
				Jobs: []Job{
					{
						PersistentDisk: -1234,
					},
				},
			}

			err := validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("jobs[0].persistent_disk must be >= 0"))
		})

		It("validates job persistent_disk_pool", func() {
			deploymentManifest := Manifest{
				Jobs: []Job{
					{
						PersistentDiskPool: "non-existent-disk-pool",
					},
				},
				DiskPools: []DiskPool{
					{
						Name: "fake-disk-pool",
					},
				},
			}

			err := validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("jobs[0].persistent_disk_pool must be the name of a disk pool"))
		})

		It("validates job resource pool is provided", func() {
			deploymentManifest := Manifest{
				Jobs: []Job{{}},
			}

			err := validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("jobs[0].resource_pool must be provided"))
		})

		It("validates job resource pool is specified in resource pools", func() {
			deploymentManifest := Manifest{
				Jobs: []Job{
					{
						ResourcePool: "non-existent-resource-pool",
					},
				},
				ResourcePools: []ResourcePool{
					{
						Name: "fake-resource-pool",
					},
				},
			}

			err := validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("jobs[0].resource_pool must be the name of a resource pool"))
		})

		It("validates job instances", func() {
			deploymentManifest := Manifest{
				Jobs: []Job{
					{
						Instances: -1234,
					},
				},
			}

			err := validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("jobs[0].instances must be >= 0"))
		})

		Describe("job networks", func() {
			It("validates job networks", func() {
				deploymentManifest := Manifest{
					Jobs: []Job{
						{
							Networks: []JobNetwork{},
						},
					},
				}

				err := validator.Validate(deploymentManifest, validReleaseSetManifest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("jobs[0].networks must be a non-empty array"))
			})

			It("validates job network name", func() {
				deploymentManifest := Manifest{
					Jobs: []Job{
						{
							Networks: []JobNetwork{
								{
									Name: "",
								},
							},
						},
					},
				}

				err := validator.Validate(deploymentManifest, validReleaseSetManifest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("jobs[0].networks[0].name must be provided"))
			})

			It("validates job network static ips", func() {
				deploymentManifest := Manifest{
					Jobs: []Job{
						{
							Networks: []JobNetwork{
								{
									StaticIPs: []string{"non-ip"},
								},
							},
						},
					},
				}

				err := validator.Validate(deploymentManifest, validReleaseSetManifest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("jobs[0].networks[0].static_ips[0] must be a valid IP"))
			})

			It("validates job network default", func() {
				deploymentManifest := Manifest{
					Jobs: []Job{
						{
							Networks: []JobNetwork{
								{Defaults: []NetworkDefault{"non-dns-or-gateway"}},
								{Defaults: []NetworkDefault{"another-bad-string", "yet-another-bad-string"}},
							},
						},
					},
				}

				err := validator.Validate(deploymentManifest, validReleaseSetManifest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("jobs[0].networks[0].default[0] must be 'dns' or 'gateway'"))
				Expect(err.Error()).To(ContainSubstring("jobs[0].networks[1].default[0] must be 'dns' or 'gateway'"))
				Expect(err.Error()).To(ContainSubstring("jobs[0].networks[1].default[1] must be 'dns' or 'gateway'"))
			})

			It("validates job network corresponds to a specified network", func() {
				deploymentManifest := Manifest{
					Networks: []Network{
						{
							Name: "fake-network-name",
							Type: "manual",
							Subnets: []Subnet{{
								Range:   "10.10.0.0/24",
								Gateway: "10.0.0.1",
							}},
						},
					},
					Jobs: []Job{
						{
							Networks: []JobNetwork{
								{
									Name:      "different-network-name",
									StaticIPs: []string{"10.10.1.1"},
								},
							},
						},
					},
				}

				err := validator.Validate(deploymentManifest, validReleaseSetManifest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("jobs[0].networks[0] not found in networks"))
			})

			It("validates job network static ip is in the subnet range", func() {
				deploymentManifest := Manifest{
					Networks: []Network{
						{
							Name: "fake-network-name",
							Type: "manual",
							Subnets: []Subnet{{
								Range:   "10.10.0.0/24",
								Gateway: "10.10.0.1",
							}},
						},
					},
					Jobs: []Job{
						{
							Networks: []JobNetwork{
								{
									Name:      "fake-network-name",
									StaticIPs: []string{"10.10.1.1"},
								},
							},
						},
					},
				}

				err := validator.Validate(deploymentManifest, validReleaseSetManifest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("jobs[0].networks[0] static ip '10.10.1.1' must be within subnet range"))

				deploymentManifest = Manifest{
					Networks: []Network{
						{
							Name: "fake-network-name",
							Type: "manual",
							Subnets: []Subnet{{
								Range:   "10.10.0.0/24",
								Gateway: "10.10.0.1",
							}},
						},
					},
					Jobs: []Job{
						{
							Networks: []JobNetwork{
								{
									Name:      "fake-network-name",
									StaticIPs: []string{"10.10.0.2"},
								},
							},
						},
					},
				}

				err = validator.Validate(deploymentManifest, validReleaseSetManifest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).ToNot(ContainSubstring("static ip"))

			})

			Describe("defaults", func() {
				var deploymentManifest Manifest
				Context("with multiple networks", func() {
					BeforeEach(func() {
						deploymentManifest = Manifest{
							Networks: []Network{
								{Name: "fake-network-name1", Type: "manual"},
								{Name: "fake-network-name2", Type: "dynamic"},
							},
							Jobs: []Job{
								{
									Networks: []JobNetwork{
										{Name: "fake-network-name1"},
										{Name: "fake-network-name2"},
									},
								},
							},
						}
					})

					It("validates a default dns must be specified", func() {
						err := validator.Validate(deploymentManifest, validReleaseSetManifest)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("with multiple networks, a default for 'dns' must be specified"))
					})

					It("validates a default dns can only be specified for a single network", func() {
						deploymentManifest.Jobs[0].Networks[0].Defaults = []NetworkDefault{"dns", "gateway"}
						deploymentManifest.Jobs[0].Networks[1].Defaults = []NetworkDefault{"dns"}

						err := validator.Validate(deploymentManifest, validReleaseSetManifest)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("only one network can be the default for 'dns'"))
					})

					It("validates a default gateway must be specified", func() {
						err := validator.Validate(deploymentManifest, validReleaseSetManifest)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("with multiple networks, a default for 'gateway' must be specified"))
					})

					It("validates a default gateway can only be specified for a single network", func() {
						deploymentManifest.Jobs[0].Networks[0].Defaults = []NetworkDefault{"dns", "gateway"}
						deploymentManifest.Jobs[0].Networks[1].Defaults = []NetworkDefault{"gateway"}

						err := validator.Validate(deploymentManifest, validReleaseSetManifest)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("only one network can be the default for 'gateway'"))
					})
				})

				Context("with only one network", func() {
					BeforeEach(func() {
						deploymentManifest = Manifest{
							Networks: []Network{
								{Name: "fake-network-name1", Type: "manual"},
								{Name: "fake-network-name2", Type: "dynamic"},
							},
							Jobs: []Job{
								{
									Networks: []JobNetwork{
										{Name: "fake-network-name1"},
									},
								},
							},
						}
					})

					It("doesn't require any defaults to be set", func() {
						err := validator.Validate(deploymentManifest, validReleaseSetManifest)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).ToNot(ContainSubstring("default"))
					})

					It("is ok if defaults to are set", func() {
						deploymentManifest.Jobs[0].Networks[0].Defaults = []NetworkDefault{"dns"}
						err := validator.Validate(deploymentManifest, validReleaseSetManifest)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).ToNot(ContainSubstring("default"))
					})
				})
			})
		})

		It("validates job lifecycle", func() {
			deploymentManifest := Manifest{
				Jobs: []Job{
					{
						Lifecycle: "errand",
					},
				},
			}

			err := validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("jobs[0].lifecycle must be 'service' ('errand' not supported)"))
		})

		It("permits job templates to reference an undeclared release", func() {
			deploymentManifest := validManifest
			deploymentManifest.Jobs[0].Templates = []ReleaseJobRef{
				{
					Name:    "fake-job-name",
					Release: "fake-release-name",
				},
			}

			err := validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).NotTo(HaveOccurred())
		})

		It("validates job templates have a job name", func() {
			deploymentManifest := validManifest
			deploymentManifest.Jobs = []Job{
				{
					Templates: []ReleaseJobRef{{}},
				},
			}

			err := validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("jobs[0].templates[0].name must be provided"))
		})

		It("validates job templates have unique job names", func() {
			deploymentManifest := validManifest
			deploymentManifest.Jobs = []Job{
				{
					Templates: []ReleaseJobRef{
						{Name: "fake-job-name"},
						{Name: "fake-job-name"},
					},
				},
			}

			err := validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("jobs[0].templates[1].name 'fake-job-name' must be unique"))
		})

		It("validates job templates reference a release", func() {
			deploymentManifest := Manifest{
				Jobs: []Job{
					{
						Templates: []ReleaseJobRef{
							{Name: "fake-job-name"},
						},
					},
				},
			}

			err := validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("jobs[0].templates[0].release must be provided"))
		})

		It("validates job templates reference a release in releases list", func() {
			deploymentManifest := validManifest
			deploymentManifest.Jobs[0].Templates = []ReleaseJobRef{
				{Name: "fake-job-name", Release: "fake-other-release-name"},
			}

			err := validator.Validate(deploymentManifest, validReleaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("jobs[0].templates[0].release 'fake-other-release-name' must refer to release in releases"))
		})
	})

	Describe("ValidateReleaseJobs", func() {
		It("validates job templates reference a job declared within the release", func() {
			deploymentManifest := validManifest
			deploymentManifest.Jobs[0].Templates = []ReleaseJobRef{
				{Name: "fake-other-job-name", Release: "fake-release-name"},
			}

			release = &fakerel.FakeRelease{
				NameStub:    func() string { return "fake-release-name" },
				VersionStub: func() string { return "1.0" },
			}
			release.JobsReturns([]*boshjob.Job{
				boshjob.NewJob(NewResource("fake-job-name", "", nil)),
			})
			releaseManager.Add(release)

			err := validator.ValidateReleaseJobs(deploymentManifest, releaseManager)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("jobs[0].templates[0] must refer to a job in 'fake-release-name', but there is no job named 'fake-other-job-name'"))
		})
	})
})
