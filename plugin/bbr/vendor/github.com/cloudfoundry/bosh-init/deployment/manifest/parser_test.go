package manifest_test

import (
	"errors"

	. "github.com/cloudfoundry/bosh-init/deployment/manifest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("Parser", func() {
	var (
		comboManifestPath string
		fakeFs            *fakesys.FakeFileSystem
		parser            Parser
	)

	BeforeEach(func() {
		comboManifestPath = "fake-deployment-path"
		fakeFs = fakesys.NewFakeFileSystem()
		logger := boshlog.NewLogger(boshlog.LevelNone)
		parser = NewParser(fakeFs, logger)
	})

	Context("when combo manifest path does not exist", func() {
		BeforeEach(func() {
			err := fakeFs.RemoveAll(comboManifestPath)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error", func() {
			_, err := parser.Parse(comboManifestPath)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when parser fails to read the combo manifest file", func() {
		BeforeEach(func() {
			fakeFs.ReadFileError = errors.New("fake-read-file-error")
		})

		It("returns an error", func() {
			_, err := parser.Parse(comboManifestPath)
			Expect(err).To(HaveOccurred())
		})
	})

	BeforeEach(func() {
		contents := `
---
name: fake-deployment-name
update:
  update_watch_time: 2000-7000
resource_pools:
- name: fake-resource-pool-name
  cloud_properties:
    fake-property: fake-property-value
  env:
    bosh:
      password: secret
  stemcell:
    url: http://fake-stemcell-url
networks:
- name: fake-network-name
  type: dynamic
  dns:  [5.5.5.5, 6.6.6.6]
  subnets:
  - range: 1.2.3.0/22
    gateway: 1.1.1.1
    dns: [2.2.2.2]
    cloud_properties:
      cp_key: cp_value
  cloud_properties:
    subnet: fake-subnet
    a:
      b: value
- name: vip
  type: vip
disk_pools:
- name: fake-disk-pool-name
  disk_size: 2048
  cloud_properties:
    fake-disk-pool-cloud-property-key: fake-disk-pool-cloud-property-value
jobs:
- name: bosh
  networks:
  - name: vip
    static_ips: [1.2.3.4]
  - name: fake-network-name
    default: [dns]
  persistent_disk: 1024
  persistent_disk_pool: fake-disk-pool-name
  resource_pool: fake-resource-pool
  properties:
    fake-prop-key:
      nested-prop-key: fake-prop-value
properties:
  foo:
    bar: baz
`
		fakeFs.WriteFileString(comboManifestPath, contents)
	})

	It("parses deployment manifest from combo manifest file", func() {
		deploymentManifest, err := parser.Parse(comboManifestPath)
		Expect(err).ToNot(HaveOccurred())

		Expect(deploymentManifest).To(Equal(Manifest{
			Name: "fake-deployment-name",
			Update: Update{
				UpdateWatchTime: WatchTime{
					Start: 2000,
					End:   7000,
				},
			},
			Networks: []Network{
				{
					Name: "fake-network-name",
					Type: Dynamic,
					DNS:  []string{"5.5.5.5", "6.6.6.6"},
					Subnets: []Subnet{
						{
							Range:   "1.2.3.0/22",
							Gateway: "1.1.1.1",
							DNS:     []string{"2.2.2.2"},
							CloudProperties: biproperty.Map{
								"cp_key": "cp_value",
							},
						},
					},
					CloudProperties: biproperty.Map{
						"subnet": "fake-subnet",
						"a": biproperty.Map{
							"b": "value",
						},
					},
				},
				{
					Name:            "vip",
					Type:            VIP,
					CloudProperties: biproperty.Map{},
				},
			},
			ResourcePools: []ResourcePool{
				{
					Name: "fake-resource-pool-name",
					CloudProperties: biproperty.Map{
						"fake-property": "fake-property-value",
					},
					Env: biproperty.Map{
						"bosh": biproperty.Map{
							"password": "secret",
						},
					},
					Stemcell: StemcellRef{
						URL: "http://fake-stemcell-url",
					},
				},
			},
			DiskPools: []DiskPool{
				{
					Name:     "fake-disk-pool-name",
					DiskSize: 2048,
					CloudProperties: biproperty.Map{
						"fake-disk-pool-cloud-property-key": "fake-disk-pool-cloud-property-value",
					},
				},
			},
			Jobs: []Job{
				{
					Name: "bosh",
					Networks: []JobNetwork{
						{
							Name:      "vip",
							StaticIPs: []string{"1.2.3.4"},
						},
						{
							Name:     "fake-network-name",
							Defaults: []NetworkDefault{NetworkDefaultDNS},
						},
					},
					PersistentDisk:     1024,
					PersistentDiskPool: "fake-disk-pool-name",
					ResourcePool:       "fake-resource-pool",
					Properties: biproperty.Map{
						"fake-prop-key": biproperty.Map{
							"nested-prop-key": "fake-prop-value",
						},
					},
				},
			},
			Properties: biproperty.Map{
				"foo": biproperty.Map{
					"bar": "baz",
				},
			},
		}))
	})

	Context("when stemcell url begins with 'http'", func() {
		BeforeEach(func() {
			contents := `
---
name: fake-deployment-manifest

resource_pools:
- name: fake-resource-pool-name
  stemcell:
    url: http://fake-url
`
			fakeFs.WriteFileString(comboManifestPath, contents)
		})
		It("it does not change the url", func() {
			deploymentManifest, err := parser.Parse(comboManifestPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(deploymentManifest).To(Equal(Manifest{

				Name:       "fake-deployment-manifest",
				Properties: biproperty.Map{},
				Jobs:       []Job{},
				Networks:   []Network{},
				DiskPools:  []DiskPool{},
				ResourcePools: []ResourcePool{
					{
						Name:            "fake-resource-pool-name",
						Network:         "",
						CloudProperties: biproperty.Map{},
						Env:             biproperty.Map{},
						Stemcell: StemcellRef{
							URL:  "http://fake-url",
							SHA1: "",
						},
					},
				},
				Update: Update{
					UpdateWatchTime: WatchTime{Start: 0, End: 300000},
				},
			}))
		})
	})

	Context("when stemcell url is a file path", func() {
		Context("that begin with 'file:///'", func() {
			BeforeEach(func() {
				contents := `
---
name: fake-deployment-manifest

resource_pools:
- name: fake-resource-pool-name
  stemcell:
   url: file:///fake-absolute-path
`
				fakeFs.WriteFileString(comboManifestPath, contents)
			})
			It("it does not expand the path to be relative to the manifest path", func() {
				deploymentManifest, err := parser.Parse(comboManifestPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(deploymentManifest).To(Equal(Manifest{

					Name:       "fake-deployment-manifest",
					Properties: biproperty.Map{},
					Jobs:       []Job{},
					Networks:   []Network{},
					DiskPools:  []DiskPool{},
					ResourcePools: []ResourcePool{
						{
							Name:            "fake-resource-pool-name",
							Network:         "",
							CloudProperties: biproperty.Map{},
							Env:             biproperty.Map{},
							Stemcell: StemcellRef{
								URL:  "file:///fake-absolute-path",
								SHA1: "",
							},
						},
					},
					Update: Update{
						UpdateWatchTime: WatchTime{Start: 0, End: 300000},
					},
				}))
			})
		})

		Context("that begin with 'file://~/'", func() {
			BeforeEach(func() {
				contents := `
---
name: fake-deployment-manifest

resource_pools:
- name: fake-resource-pool-name
  stemcell:
   url: file://~/fake-absolute-path
`
				fakeFs.WriteFileString(comboManifestPath, contents)
			})
			It("it does not expand the path to be relative to the manifest path", func() {
				deploymentManifest, err := parser.Parse(comboManifestPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(deploymentManifest).To(Equal(Manifest{

					Name:       "fake-deployment-manifest",
					Properties: biproperty.Map{},
					Jobs:       []Job{},
					Networks:   []Network{},
					DiskPools:  []DiskPool{},
					ResourcePools: []ResourcePool{
						{
							Name:            "fake-resource-pool-name",
							Network:         "",
							CloudProperties: biproperty.Map{},
							Env:             biproperty.Map{},
							Stemcell: StemcellRef{
								URL:  "file://~/fake-absolute-path",
								SHA1: "",
							},
						},
					},
					Update: Update{
						UpdateWatchTime: WatchTime{Start: 0, End: 300000},
					},
				}))
			})
		})

		Context("that do not begin with 'file://~/' or  'file:///'", func() {
			BeforeEach(func() {
				comboManifestPath = "/path/to/fake-deployment-yml"
				contents := `
---
name: fake-deployment-manifest

resource_pools:
- name: fake-resource-pool-name
  stemcell:
   url: file://fake-relative-path
`
				fakeFs.WriteFileString(comboManifestPath, contents)
			})
			It("it does not expand the path to be relative to the manifest path", func() {
				deploymentManifest, err := parser.Parse(comboManifestPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(deploymentManifest).To(Equal(Manifest{

					Name:       "fake-deployment-manifest",
					Properties: biproperty.Map{},
					Jobs:       []Job{},
					Networks:   []Network{},
					DiskPools:  []DiskPool{},
					ResourcePools: []ResourcePool{
						{
							Name:            "fake-resource-pool-name",
							Network:         "",
							CloudProperties: biproperty.Map{},
							Env:             biproperty.Map{},
							Stemcell: StemcellRef{
								URL:  "file:///path/to/fake-relative-path",
								SHA1: "",
							},
						},
					},
					Update: Update{
						UpdateWatchTime: WatchTime{Start: 0, End: 300000},
					},
				}))
			})
		})
	})

	Context("when global property keys are not strings", func() {
		BeforeEach(func() {
			contents := `
---
properties:
  1: foo
`
			fakeFs.WriteFileString(comboManifestPath, contents)
		})

		It("returns an error", func() {
			_, err := parser.Parse(comboManifestPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Parsing global manifest properties"))
		})
	})

	Context("when job property keys are not strings", func() {
		BeforeEach(func() {
			contents := `
---
jobs:
- name: fake-deployment-job
  properties:
    1: foo
`
			fakeFs.WriteFileString(comboManifestPath, contents)
		})

		It("returns an error", func() {
			_, err := parser.Parse(comboManifestPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Parsing job 'fake-deployment-job' properties"))
		})
	})

	Context("when network cloud_properties keys are not strings", func() {
		BeforeEach(func() {
			contents := `
---
networks:
- name: fake-network
  cloud_properties:
    123: fake-property-value
`
			fakeFs.WriteFileString(comboManifestPath, contents)
		})

		It("returns an error", func() {
			_, err := parser.Parse(comboManifestPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Parsing network 'fake-network' cloud_properties"))
		})
	})

	Context("when resource_pool cloud_properties keys are not strings", func() {
		BeforeEach(func() {
			contents := `
---
resource_pools:
- name: fake-resource-pool
  cloud_properties:
    123: fake-property-value
`
			fakeFs.WriteFileString(comboManifestPath, contents)
		})

		It("returns an error", func() {
			_, err := parser.Parse(comboManifestPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Parsing resource_pool 'fake-resource-pool' cloud_properties"))
		})
	})

	Context("when resource_pool env keys are not strings", func() {
		BeforeEach(func() {
			contents := `
---
resource_pools:
- name: fake-resource-pool
  env:
    123: fake-property-value
`
			fakeFs.WriteFileString(comboManifestPath, contents)
		})

		It("returns an error", func() {
			_, err := parser.Parse(comboManifestPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Parsing resource_pool 'fake-resource-pool' env"))
		})
	})

	Context("when disk_pool cloud_properties keys are not strings", func() {
		BeforeEach(func() {
			contents := `
---
disk_pools:
- name: fake-disk-pool
  cloud_properties:
    123: fake-property-value
`
			fakeFs.WriteFileString(comboManifestPath, contents)
		})

		It("returns an error", func() {
			_, err := parser.Parse(comboManifestPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Parsing disk_pool 'fake-disk-pool' cloud_properties"))
		})
	})

	Context("when update watch time is not set", func() {
		BeforeEach(func() {
			contents := `
---
name: fake-deployment-name
`
			fakeFs.WriteFileString(comboManifestPath, contents)
		})

		It("uses default values", func() {
			deploymentManifest, err := parser.Parse(comboManifestPath)
			Expect(err).ToNot(HaveOccurred())

			Expect(deploymentManifest.Name).To(Equal("fake-deployment-name"))
			Expect(deploymentManifest.Update.UpdateWatchTime.Start).To(Equal(0))
			Expect(deploymentManifest.Update.UpdateWatchTime.End).To(Equal(300000))
		})
	})

	Context("when instance_groups is defined, treats it as jobs", func() {
		BeforeEach(func() {
			contents := `
---
instance_groups:
- name: jobby
`
			fakeFs.WriteFileString(comboManifestPath, contents)
		})

		It("treats instance groups as jobs", func() {
			deploymentManifest, err := parser.Parse(comboManifestPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(deploymentManifest.Jobs[0].Name).To(Equal("jobby"))
		})
	})

	Context("when jobs is defined inside an instance_group, treats it as templates", func() {
		BeforeEach(func() {
			contents := `
---
instance_groups:
- name: jobby
  jobs:
  - name: job1
`
			fakeFs.WriteFileString(comboManifestPath, contents)
		})

		It("treats instance groups as jobs", func() {
			deploymentManifest, err := parser.Parse(comboManifestPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(deploymentManifest.Jobs[0].Templates[0].Name).To(Equal("job1"))
		})
	})

	Context("when job is defined inside an instance_group with properties", func() {
		BeforeEach(func() {
			contents := `
---
instance_groups:
- name: jobby
  jobs:
  - name: job1
    properties:
      key1: value1
`
			fakeFs.WriteFileString(comboManifestPath, contents)
		})

		It("parses the property", func() {
			deploymentManifest, err := parser.Parse(comboManifestPath)
			Expect(err).ToNot(HaveOccurred())
			Expect((*deploymentManifest.Jobs[0].Templates[0].Properties)["key1"]).To(Equal("value1"))
		})
	})

	Context("when job is defined inside an instance_group with empty properties", func() {
		BeforeEach(func() {
			contents := `
---
instance_groups:
- name: jobby
  jobs:
  - name: job1
    properties: {}
`
			fakeFs.WriteFileString(comboManifestPath, contents)
		})

		It("parses the properties as an empty map", func() {
			deploymentManifest, err := parser.Parse(comboManifestPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(*deploymentManifest.Jobs[0].Templates[0].Properties).To(BeEmpty())
		})
	})

	Context("when job is defined inside an instance_group with no properties", func() {
		BeforeEach(func() {
			contents := `
---
instance_groups:
- name: jobby
  jobs:
  - name: job1
`
			fakeFs.WriteFileString(comboManifestPath, contents)
		})

		It("parses the properties as nil", func() {
			deploymentManifest, err := parser.Parse(comboManifestPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(deploymentManifest.Jobs[0].Templates[0].Properties).To(BeNil())
		})
	})

	Context("when both instance_groups and jobs are present at root level in deployment manifest", func() {
		BeforeEach(func() {
			contents := `
---
jobs:
- name: jobby

instance_groups:
- name: instancey

`
			fakeFs.WriteFileString(comboManifestPath, contents)
		})

		It("throws an error", func() {
			_, err := parser.Parse(comboManifestPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Deployment specifies both jobs and instance_groups keys, only one is allowed"))
		})
	})

	Context("when both templates and jobs are present at job level in deployment manifest", func() {
		BeforeEach(func() {
			contents := `
---
jobs:
- name: jobby
  templates:
  - name: temp1
  jobs:
  - name: job1

`
			fakeFs.WriteFileString(comboManifestPath, contents)
		})

		It("throws an error", func() {
			_, err := parser.Parse(comboManifestPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Deployment specifies both templates and jobs keys for instance_group jobby, only one is allowed"))
		})
	})
})
