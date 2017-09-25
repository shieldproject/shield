package config_test

import (
	. "github.com/cloudfoundry/bosh-cli/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	//	"encoding/json"
	//	"errors"
	//
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
)

var _ = Describe("legacyDeploymentStateMigrator", func() {
	var (
		migrator                      LegacyDeploymentStateMigrator
		deploymentStateService        DeploymentStateService
		legacyDeploymentStateFilePath string
		modernDeploymentStateFilePath string
		fakeFs                        *fakesys.FakeFileSystem
		fakeUUIDGenerator             *fakeuuid.FakeGenerator
	)

	BeforeEach(func() {
		fakeFs = fakesys.NewFakeFileSystem()
		fakeUUIDGenerator = fakeuuid.NewFakeGenerator()
		legacyDeploymentStateFilePath = "/path/to/legacy/bosh-deployment.yml"
		modernDeploymentStateFilePath = "/path/to/legacy/deployment.json"
		logger := boshlog.NewLogger(boshlog.LevelNone)
		deploymentStateService = NewFileSystemDeploymentStateService(fakeFs, fakeUUIDGenerator, logger, modernDeploymentStateFilePath)
		migrator = NewLegacyDeploymentStateMigrator(deploymentStateService, fakeFs, fakeUUIDGenerator, logger)
	})

	Describe("MigrateIfExists", func() {
		Context("when no legacy deployment config file exists", func() {
			It("does nothing", func() {
				migrated, err := migrator.MigrateIfExists(legacyDeploymentStateFilePath)
				Expect(migrated).To(BeFalse())
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeFs.FileExists(modernDeploymentStateFilePath)).To(BeFalse())
			})
		})

		Context("when legacy deployment config file exists (but is unparseable)", func() {
			BeforeEach(func() {
				fakeFs.WriteFileString(legacyDeploymentStateFilePath, `xyz`)
			})

			It("does not delete the legacy deployment state file", func() {
				migrated, err := migrator.MigrateIfExists(legacyDeploymentStateFilePath)
				Expect(migrated).To(BeFalse())
				Expect(err).To(HaveOccurred())

				Expect(fakeFs.FileExists(modernDeploymentStateFilePath)).To(BeFalse())
				Expect(fakeFs.FileExists(legacyDeploymentStateFilePath)).To(BeTrue())
			})
		})

		Context("when legacy deployment config file exists (and is empty)", func() {
			BeforeEach(func() {
				fakeFs.WriteFileString(legacyDeploymentStateFilePath, `--- {}`)
			})

			It("deletes the legacy deployment state file", func() {
				migrated, err := migrator.MigrateIfExists(legacyDeploymentStateFilePath)
				Expect(migrated).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeFs.FileExists(legacyDeploymentStateFilePath)).To(BeFalse())
			})
		})

		Context("when legacy deployment config file exists and UUID exists", func() {

			BeforeEach(func() {
				fakeFs.WriteFileString(legacyDeploymentStateFilePath, `---
instances:
- :id: 1
  :name: micro-robinson
  :uuid: bm-5480c6bb-3ba8-449a-a262-a2e75fbe5daf
  :stemcell_cid:
  :stemcell_sha1:
  :stemcell_name:
  :config_sha1: f9bdbc6cf6bf922f520ee9c45ed94a16a46dd972
  :vm_cid:
  :disk_cid:
disks: []
registry_instances:
- :id: 1
  :instance_id: i-a1624150
  :settings: '{}'
`)
			})

			It("deletes the legacy deployment state file", func() {
				migrated, err := migrator.MigrateIfExists(legacyDeploymentStateFilePath)
				Expect(migrated).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeFs.FileExists(legacyDeploymentStateFilePath)).To(BeFalse())
			})

			It("uses the legacy UUID as the director_uuid in the new deployment manifest", func() {
				migrated, err := migrator.MigrateIfExists(legacyDeploymentStateFilePath)
				Expect(migrated).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())

				content, err := fakeFs.ReadFileString(modernDeploymentStateFilePath)
				Expect(err).ToNot(HaveOccurred())

				Expect(content).To(MatchRegexp(`{
    "director_id": "bm-5480c6bb-3ba8-449a-a262-a2e75fbe5daf",
    "installation_id": "",
    "current_vm_cid": "",
    "current_stemcell_id": "",
    "current_disk_id": "",
    "current_release_ids": null,
    "current_manifest_sha": "",
    "disks": \[\],
    "stemcells": \[\],
    "releases": \[\]
}`))
			})
		})

		Context("when legacy deployment config file exists and it does not contain a UUID", func() {

			BeforeEach(func() {
				fakeFs.WriteFileString(legacyDeploymentStateFilePath, `---
instances:
- :id: 1
  :name: micro-robinson
  :stemcell_cid:
  :stemcell_sha1:
  :stemcell_name:
  :config_sha1: f9bdbc6cf6bf922f520ee9c45ed94a16a46dd972
  :vm_cid:
  :disk_cid:
disks: []
registry_instances:
- :id: 1
  :instance_id: i-a1624150
  :settings: '{}'
`)
			})

			It("deletes the legacy deployment state file", func() {
				migrated, err := migrator.MigrateIfExists(legacyDeploymentStateFilePath)
				Expect(migrated).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeFs.FileExists(legacyDeploymentStateFilePath)).To(BeFalse())
			})

			It("generates a new UUID to use as the director_uuid in the new deployment manifest", func() {
				migrated, err := migrator.MigrateIfExists(legacyDeploymentStateFilePath)
				Expect(migrated).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())

				content, err := fakeFs.ReadFileString(modernDeploymentStateFilePath)
				Expect(err).ToNot(HaveOccurred())

				Expect(content).To(MatchRegexp(`{
    "director_id": "fake-uuid-0",
    "installation_id": "",
    "current_vm_cid": "",
    "current_stemcell_id": "",
    "current_disk_id": "",
    "current_release_ids": null,
    "current_manifest_sha": "",
    "disks": \[\],
    "stemcells": \[\],
    "releases": \[\]
}`))
			})
		})

		Context("when legacy deployment config file exists (without vm, disk, or stemcell)", func() {
			BeforeEach(func() {
				fakeFs.WriteFileString(legacyDeploymentStateFilePath, `---
instances:
- :id: 1
  :name: micro-robinson
  :uuid: bm-5480c6bb-3ba8-449a-a262-a2e75fbe5daf
  :stemcell_cid:
  :stemcell_sha1:
  :stemcell_name:
  :config_sha1: f9bdbc6cf6bf922f520ee9c45ed94a16a46dd972
  :vm_cid:
  :disk_cid:
disks: []
registry_instances:
- :id: 1
  :instance_id: i-a1624150
  :settings: '{}'
`)
			})

			It("deletes the legacy deployment state file", func() {
				migrated, err := migrator.MigrateIfExists(legacyDeploymentStateFilePath)
				Expect(migrated).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeFs.FileExists(legacyDeploymentStateFilePath)).To(BeFalse())
			})

			It("creates a new deployment state file without vm, disk, or stemcell", func() {
				migrated, err := migrator.MigrateIfExists(legacyDeploymentStateFilePath)
				Expect(migrated).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())

				content, err := fakeFs.ReadFileString(modernDeploymentStateFilePath)
				Expect(err).ToNot(HaveOccurred())

				Expect(content).To(MatchRegexp(`{
    "director_id": "bm-5480c6bb-3ba8-449a-a262-a2e75fbe5daf",
    "installation_id": "",
    "current_vm_cid": "",
    "current_stemcell_id": "",
    "current_disk_id": "",
    "current_release_ids": null,
    "current_manifest_sha": "",
    "disks": \[\],
    "stemcells": \[\],
    "releases": \[\]
}`))
			})
		})

		Context("when legacy deployment config file exists (with vm, disk & stemcell)", func() {
			BeforeEach(func() {
				fakeFs.WriteFileString(legacyDeploymentStateFilePath, `---
instances:
- :id: 1
  :name: micro-robinson
  :uuid: bm-5480c6bb-3ba8-449a-a262-a2e75fbe5daf
  :stemcell_cid: ami-f2503e9a light
  :stemcell_sha1: 561b73dafc86454751db09855b0de7a89f0b4337
  :stemcell_name: light-bosh-stemcell-2807-aws-xen-ubuntu-trusty-go_agent
  :config_sha1: f9bdbc6cf6bf922f520ee9c45ed94a16a46dd972
  :vm_cid: i-a1624150
  :disk_cid: vol-565ed74d
disks: []
registry_instances:
- :id: 1
  :instance_id: i-a1624150
  :settings: '{}'
`)
			})

			It("deletes the legacy deployment state file", func() {
				migrated, err := migrator.MigrateIfExists(legacyDeploymentStateFilePath)
				Expect(migrated).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeFs.FileExists(legacyDeploymentStateFilePath)).To(BeFalse())
			})

			It("creates a new deployment state file with vm, disk & stemcell", func() {
				migrated, err := migrator.MigrateIfExists(legacyDeploymentStateFilePath)
				Expect(migrated).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())

				content, err := fakeFs.ReadFileString(modernDeploymentStateFilePath)
				Expect(err).ToNot(HaveOccurred())

				Expect(content).To(MatchRegexp(`{
    "director_id": "bm-5480c6bb-3ba8-449a-a262-a2e75fbe5daf",
    "installation_id": "",
    "current_vm_cid": "i-a1624150",
    "current_stemcell_id": "",
    "current_disk_id": "fake-uuid-0",
    "current_release_ids": null,
    "current_manifest_sha": "",
    "disks": \[
        {
            "id": "fake-uuid-0",
            "cid": "vol-565ed74d",
            "size": 0,
            "cloud_properties": {}
        }
    \],
    "stemcells": \[
        {
            "id": "fake-uuid-1",
            "name": "light-bosh-stemcell-2807-aws-xen-ubuntu-trusty-go_agent",
            "version": "",
            "cid": "ami-f2503e9a light"
        }
    \],
    "releases": \[\]
}`))
			})
		})

		Context("when legacy deployment config file exists (with vm only)", func() {
			BeforeEach(func() {
				fakeFs.WriteFileString(legacyDeploymentStateFilePath, `---
instances:
- :id: 1
  :name: micro-robinson
  :uuid: bm-5480c6bb-3ba8-449a-a262-a2e75fbe5daf
  :stemcell_cid:
  :stemcell_sha1:
  :stemcell_name:
  :config_sha1: f9bdbc6cf6bf922f520ee9c45ed94a16a46dd972
  :vm_cid: i-a1624150
  :disk_cid:
disks: []
registry_instances:
- :id: 1
  :instance_id: i-a1624150
  :settings: '{}'
`)
			})

			It("deletes the legacy deployment state file", func() {
				migrated, err := migrator.MigrateIfExists(legacyDeploymentStateFilePath)
				Expect(migrated).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeFs.FileExists(legacyDeploymentStateFilePath)).To(BeFalse())
			})

			It("creates a new deployment state file with vm", func() {
				migrated, err := migrator.MigrateIfExists(legacyDeploymentStateFilePath)
				Expect(migrated).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())

				content, err := fakeFs.ReadFileString(modernDeploymentStateFilePath)
				Expect(err).ToNot(HaveOccurred())

				Expect(content).To(MatchRegexp(`{
    "director_id": "bm-5480c6bb-3ba8-449a-a262-a2e75fbe5daf",
    "installation_id": "",
    "current_vm_cid": "i-a1624150",
    "current_stemcell_id": "",
    "current_disk_id": "",
    "current_release_ids": null,
    "current_manifest_sha": "",
    "disks": \[\],
    "stemcells": \[\],
    "releases": \[\]
}`))
			})
		})

		Context("when legacy deployment config file exists (with disk only)", func() {
			BeforeEach(func() {
				fakeFs.WriteFileString(legacyDeploymentStateFilePath, `---
instances:
- :id: 1
  :name: micro-robinson
  :uuid: bm-5480c6bb-3ba8-449a-a262-a2e75fbe5daf
  :stemcell_cid:
  :stemcell_sha1:
  :stemcell_name:
  :config_sha1: f9bdbc6cf6bf922f520ee9c45ed94a16a46dd972
  :vm_cid:
  :disk_cid: vol-565ed74d
disks: []
registry_instances:
- :id: 1
  :instance_id: i-a1624150
  :settings: '{}'
`)
			})

			It("deletes the legacy deployment state file", func() {
				migrated, err := migrator.MigrateIfExists(legacyDeploymentStateFilePath)
				Expect(migrated).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeFs.FileExists(legacyDeploymentStateFilePath)).To(BeFalse())
			})

			It("creates a new deployment state file with disk only", func() {
				migrated, err := migrator.MigrateIfExists(legacyDeploymentStateFilePath)
				Expect(migrated).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())

				content, err := fakeFs.ReadFileString(modernDeploymentStateFilePath)
				Expect(err).ToNot(HaveOccurred())

				Expect(content).To(MatchRegexp(`{
    "director_id": "bm-5480c6bb-3ba8-449a-a262-a2e75fbe5daf",
    "installation_id": "",
    "current_vm_cid": "",
    "current_stemcell_id": "",
    "current_disk_id": "fake-uuid-0",
    "current_release_ids": null,
    "current_manifest_sha": "",
    "disks": \[
        {
            "id": "fake-uuid-0",
            "cid": "vol-565ed74d",
            "size": 0,
            "cloud_properties": {}
        }
    \],
    "stemcells": \[\],
    "releases": \[\]
}`))
			})
		})

		Context("when legacy deployment config file exists and contains none-specific node tag (!)", func() {
			BeforeEach(func() {
				fakeFs.WriteFileString(legacyDeploymentStateFilePath, `---
instances:
- :id: 1
  :name: micro-robinson
  :uuid: bm-5480c6bb-3ba8-449a-a262-a2e75fbe5daf
  :stemcell_cid:
  :stemcell_sha1:
  :stemcell_name:
  :config_sha1: f9bdbc6cf6bf922f520ee9c45ed94a16a46dd972
  :vm_cid:
  :disk_cid:
disks: []
registry_instances:
- :id: 1
  :instance_id: i-a1624150
  :settings: ! '{"vm":{"name":"fake-name"}}'
`)
			})

			It("deletes the legacy deployment state file", func() {
				migrated, err := migrator.MigrateIfExists(legacyDeploymentStateFilePath)
				Expect(migrated).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeFs.FileExists(legacyDeploymentStateFilePath)).To(BeFalse())
			})

			It("creates a new deployment state file", func() {
				migrated, err := migrator.MigrateIfExists(legacyDeploymentStateFilePath)
				Expect(migrated).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())

				content, err := fakeFs.ReadFileString(modernDeploymentStateFilePath)
				Expect(err).ToNot(HaveOccurred())

				Expect(content).To(MatchRegexp(`{
    "director_id": "bm-5480c6bb-3ba8-449a-a262-a2e75fbe5daf",
    "installation_id": "",
    "current_vm_cid": "",
    "current_stemcell_id": "",
    "current_disk_id": "",
    "current_release_ids": null,
    "current_manifest_sha": "",
    "disks": \[\],
    "stemcells": \[\],
    "releases": \[\]
}`))
			})
		})

		Context("when legacy deployment config file exists (with stemcell only)", func() {
			BeforeEach(func() {
				fakeFs.WriteFileString(legacyDeploymentStateFilePath, `---
instances:
- :id: 1
  :name: micro-robinson
  :uuid: bm-5480c6bb-3ba8-449a-a262-a2e75fbe5daf
  :stemcell_cid: ami-f2503e9a light
  :stemcell_sha1: 561b73dafc86454751db09855b0de7a89f0b4337
  :stemcell_name: light-bosh-stemcell-2807-aws-xen-ubuntu-trusty-go_agent
  :config_sha1: f9bdbc6cf6bf922f520ee9c45ed94a16a46dd972
  :vm_cid:
  :disk_cid:
disks: []
registry_instances:
- :id: 1
  :instance_id: i-a1624150
  :settings: '{}'
`)
			})

			It("deletes the legacy deployment state file", func() {
				migrated, err := migrator.MigrateIfExists(legacyDeploymentStateFilePath)
				Expect(migrated).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeFs.FileExists(legacyDeploymentStateFilePath)).To(BeFalse())
			})

			It("creates a new deployment state file with stemcell only (marked as unused)", func() {
				migrated, err := migrator.MigrateIfExists(legacyDeploymentStateFilePath)
				Expect(migrated).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())

				content, err := fakeFs.ReadFileString(modernDeploymentStateFilePath)
				Expect(err).ToNot(HaveOccurred())

				Expect(content).To(MatchRegexp(`{
    "director_id": "bm-5480c6bb-3ba8-449a-a262-a2e75fbe5daf",
    "installation_id": "",
    "current_vm_cid": "",
    "current_stemcell_id": "",
    "current_disk_id": "",
    "current_release_ids": null,
    "current_manifest_sha": "",
    "disks": \[\],
    "stemcells": \[
        {
            "id": "fake-uuid-0",
            "name": "light-bosh-stemcell-2807-aws-xen-ubuntu-trusty-go_agent",
            "version": "",
            "cid": "ami-f2503e9a light"
        }
    \],
    "releases": \[\]
}`))
			})
		})
	})
})
