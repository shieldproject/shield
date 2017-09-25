package applyspec_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agent/applier/applyspec"
	models "github.com/cloudfoundry/bosh-agent/agent/applier/models"
	"github.com/cloudfoundry/bosh-utils/crypto"
)

var _ = Describe("V1ApplySpec", func() {
	Describe("json unmarshalling", func() {
		It("returns parsed apply spec from json with multi sha digests", func() {
			specJSON := `{
				"id": "node-id",
				"index": 4,
				"properties": {
					"logging": {"max_log_file_size": "10M"}
				},
				"job": {
					"name": "router",
					"template": "router template",
					"version": "1.0",
					"sha1": "sha1:routersha1;sha256:routersha256",
					"blobstore_id": "router-blob-id-1",
					"templates": [
						{"name": "template 1", "version": "0.1"},
						{"name": "template 2", "version": "0.2"}
					]
				},
				"packages": {
					"package 1": {"name": "package 1", "version": "0.1", "sha1": "sha1:package1sha1;sha256:package1sha256", "blobstore_id": "package-blob-id-1"},
					"package 2": {"name": "package 2", "version": "0.2", "sha1": "sha1:package2sha1;sha256:package2sha256", "blobstore_id": "package-blob-id-2"}
				},
				"networks": {
					"manual-net": {
						"cloud_properties": {
							"subnet": "subnet-xxxxxx"
						},
						"default": [
							"dns",
							"gateway"
						],
						"dns": [
							"xx.xx.xx.xx"
						],
						"dns_record_name": "job-index.job-name.manual-net.deployment-name.bosh",
						"gateway": "xx.xx.xx.xx",
						"ip": "xx.xx.xx.xx",
						"netmask": "xx.xx.xx.xx"
					},
					"vip-net": {
						"cloud_properties": {
							"security_groups": [
								"bosh"
							]
						},
						"dns_record_name": "job-index.job-name.vip-net.deployment-name.bosh",
						"ip": "xx.xx.xx.xx",
						"type": "vip"
					}
				},
				"rendered_templates_archive": {
					"sha1": "sha1:archivesha1;sha256:archivesha256",
					"blobstore_id": "archive-blob-id-1"
				}
			}`

			spec := V1ApplySpec{}
			err := json.Unmarshal([]byte(specJSON), &spec)
			Expect(err).ToNot(HaveOccurred())

			jobName := "router"
			expectedIndex := 4
			sha1 := crypto.MustParseMultipleDigest("sha1:archivesha1;sha256:archivesha256")
			expectedSpec := V1ApplySpec{
				Index:  &expectedIndex,
				NodeID: "node-id",
				PropertiesSpec: PropertiesSpec{
					LoggingSpec: LoggingSpec{MaxLogFileSize: "10M"},
				},
				JobSpec: JobSpec{
					Name:     &jobName,
					Template: "router template",
					Version:  "1.0",
					JobTemplateSpecs: []JobTemplateSpec{
						{Name: "template 1", Version: "0.1"},
						{Name: "template 2", Version: "0.2"},
					},
				},
				PackageSpecs: map[string]PackageSpec{
					"package 1": {Name: "package 1", Version: "0.1", Sha1: crypto.MustParseMultipleDigest("sha1:package1sha1;sha256:package1sha256"), BlobstoreID: "package-blob-id-1"},
					"package 2": {Name: "package 2", Version: "0.2", Sha1: crypto.MustParseMultipleDigest("sha1:package2sha1;sha256:package2sha256"), BlobstoreID: "package-blob-id-2"},
				},
				RenderedTemplatesArchiveSpec: &RenderedTemplatesArchiveSpec{
					Sha1:        &sha1,
					BlobstoreID: "archive-blob-id-1",
				},
				NetworkSpecs: map[string]NetworkSpec{
					"manual-net": {
						Fields: map[string]interface{}{
							"cloud_properties": map[string]interface{}{"subnet": "subnet-xxxxxx"},
							"default":          []interface{}{"dns", "gateway"},
							"dns":              []interface{}{"xx.xx.xx.xx"},
							"dns_record_name":  "job-index.job-name.manual-net.deployment-name.bosh",
							"gateway":          "xx.xx.xx.xx",
							"ip":               "xx.xx.xx.xx",
							"netmask":          "xx.xx.xx.xx",
						},
					},
					"vip-net": {
						Fields: map[string]interface{}{
							"cloud_properties": map[string]interface{}{"security_groups": []interface{}{"bosh"}},
							"dns_record_name":  "job-index.job-name.vip-net.deployment-name.bosh",
							"ip":               "xx.xx.xx.xx",
							"type":             "vip",
						},
					},
				},
			}

			Expect(spec).To(Equal(expectedSpec))

			specBytes, err := json.Marshal(spec)
			Expect(err).ToNot(HaveOccurred())

			reloadedSpec := V1ApplySpec{}
			err = json.Unmarshal([]byte(specBytes), &reloadedSpec)
			Expect(err).ToNot(HaveOccurred())

			Expect(reloadedSpec).To(Equal(spec))
		})

		It("marshals partial apply specs, like compilation apply specs", func() {
			specJSON := `{
				"deployment": "simple",
				"job": {
					"name": "compilation-160f4005"
				},
				"index": 0,
				"id": "ef7a1af2",
				"rendered_templates_archive": {
					"blobstore_id": "",
					"sha1": ""
				},
				"networks": {
					"a": {
						"ip": "192.168.1.5",
						"netmask": "255.255.255.0",
						"cloud_properties": {},
						"default": [
							"dns",
							"gateway"
						],
						"dns": [
							"192.168.1.1",
							"192.168.1.2"
						],
						"gateway": "192.168.1.1"
					}
				}
			}`

			spec := V1ApplySpec{}
			err := json.Unmarshal([]byte(specJSON), &spec)
			Expect(err).ToNot(HaveOccurred())
			specBytes, err := json.Marshal(spec)
			Expect(err).ToNot(HaveOccurred())
			reloadedSpec := V1ApplySpec{}
			err = json.Unmarshal([]byte(specBytes), &reloadedSpec)
			Expect(err).ToNot(HaveOccurred())

			Expect(reloadedSpec).To(Equal(spec))
		})
	})

	Describe("Jobs", func() {
		It("returns jobs specified in job specs", func() {
			jobName := "fake-job-legacy-name"
			sha1 := crypto.MustParseMultipleDigest("sha1:fakerenderedtemplatesarchivesha1")
			spec := V1ApplySpec{
				JobSpec: JobSpec{
					Name:    &jobName,
					Version: "fake-job-legacy-version",
					JobTemplateSpecs: []JobTemplateSpec{
						{
							Name:    "fake-job1-name",
							Version: "fake-job1-version",
						},
						{
							Name:    "fake-job2-name",
							Version: "fake-job2-version",
						},
					},
				},
				PackageSpecs: map[string]PackageSpec{
					"fake-package1": {
						Name:        "fake-package1-name",
						Version:     "fake-package1-version",
						Sha1:        crypto.MustParseMultipleDigest("sha1:fakepackage1sha1"),
						BlobstoreID: "fake-package1-blob-id",
					},
					"fake-package2": {
						Name:        "fake-package2-name",
						Version:     "fake-package2-version",
						Sha1:        crypto.MustParseMultipleDigest("sha1:fakepackage2sha1"),
						BlobstoreID: "fake-package2-blob-id",
					},
				},
				RenderedTemplatesArchiveSpec: &RenderedTemplatesArchiveSpec{
					Sha1:        &sha1,
					BlobstoreID: "fake-rendered-templates-archive-blobstore-id",
				},
			}

			expectedPackagesOnEachJob := []models.Package{
				{
					Name:    "fake-package1-name",
					Version: "fake-package1-version",
					Source: models.Source{
						Sha1:          crypto.MustParseMultipleDigest("sha1:fakepackage1sha1"),
						BlobstoreID:   "fake-package1-blob-id",
						PathInArchive: "",
					},
				},
				{
					Name:    "fake-package2-name",
					Version: "fake-package2-version",
					Source: models.Source{
						Sha1:          crypto.MustParseMultipleDigest("sha1:fakepackage2sha1"),
						BlobstoreID:   "fake-package2-blob-id",
						PathInArchive: "",
					},
				},
			}

			// Test Packages separately since it has to be done via ConsistOf
			// because apply spec uses a hash to specify package dependencies
			actualJobs := spec.Jobs()
			Expect(actualJobs[0].Packages).To(ConsistOf(expectedPackagesOnEachJob))
			Expect(actualJobs[1].Packages).To(ConsistOf(expectedPackagesOnEachJob))

			Expect(actualJobs).To(Equal([]models.Job{
				{
					Name:    "fake-job1-name",
					Version: "fake-job1-version",
					Source: models.Source{
						Sha1:          crypto.MustParseMultipleDigest("sha1:fakerenderedtemplatesarchivesha1"),
						BlobstoreID:   "fake-rendered-templates-archive-blobstore-id",
						PathInArchive: "fake-job1-name",
					},
					Packages: actualJobs[0].Packages, // tested above
				},
				{
					Name:    "fake-job2-name",
					Version: "fake-job2-version",
					Source: models.Source{
						Sha1:          crypto.MustParseMultipleDigest("sha1:fakerenderedtemplatesarchivesha1"),
						BlobstoreID:   "fake-rendered-templates-archive-blobstore-id",
						PathInArchive: "fake-job2-name",
					},
					Packages: actualJobs[1].Packages, // tested above
				},
			}))
		})

		It("returns no jobs when no jobs specified", func() {
			spec := V1ApplySpec{}
			Expect(spec.Jobs()).To(Equal([]models.Job{}))
		})
	})

	Describe("Packages", func() {
		It("retuns packages", func() {
			spec := V1ApplySpec{
				PackageSpecs: map[string]PackageSpec{
					"fake-package1-name-key": {
						Name:        "fake-package1-name",
						Version:     "fake-package1-version",
						Sha1:        crypto.MustParseMultipleDigest("sha1:fakepackage1sha1"),
						BlobstoreID: "fake-package1-blobstore-id",
					},
				},
			}

			Expect(spec.Packages()).To(Equal([]models.Package{
				{
					Name:    "fake-package1-name",
					Version: "fake-package1-version",
					Source: models.Source{
						Sha1:        crypto.MustParseMultipleDigest("sha1:fakepackage1sha1"),
						BlobstoreID: "fake-package1-blobstore-id",
					},
				},
			}))
		})

		It("returns no packages when no packages specified", func() {
			spec := V1ApplySpec{}
			Expect(spec.Packages()).To(Equal([]models.Package{}))
		})
	})

	Describe("MaxLogFileSize", func() {
		It("returns 50M if size is not provided", func() {
			spec := V1ApplySpec{}
			Expect(spec.MaxLogFileSize()).To(Equal("50M"))
		})

		It("returns provided size", func() {
			spec := V1ApplySpec{}
			spec.PropertiesSpec.LoggingSpec.MaxLogFileSize = "fake-size"
			Expect(spec.MaxLogFileSize()).To(Equal("fake-size"))
		})
	})
})

var _ = Describe("NetworkSpec", func() {
	Describe("PopulateIPInfo", func() {
		It("populates network spec with ip, netmask and gateway addressess", func() {
			networkSpec := NetworkSpec{}

			networkSpec = networkSpec.PopulateIPInfo("fake-ip", "fake-netmask", "fake-gateway")

			Expect(networkSpec).To(Equal(NetworkSpec{
				Fields: map[string]interface{}{
					"ip":      "fake-ip",
					"netmask": "fake-netmask",
					"gateway": "fake-gateway",
				},
			}))
		})

		It("overwrites network spec with ip, netmask and gateway addressess", func() {
			networkSpec := NetworkSpec{
				Fields: map[string]interface{}{
					"ip":      "fake-old-ip",
					"netmask": "fake-old-netmask",
					"gateway": "fake-old-gateway",
				},
			}

			networkSpec = networkSpec.PopulateIPInfo("fake-ip", "fake-netmask", "fake-gateway")

			Expect(networkSpec).To(Equal(NetworkSpec{
				Fields: map[string]interface{}{
					"ip":      "fake-ip",
					"netmask": "fake-netmask",
					"gateway": "fake-gateway",
				},
			}))
		})
	})
})
