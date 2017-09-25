package state_test

import (
	"errors"

	. "github.com/cloudfoundry/bosh-agent/agent/action/state"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"

	fakeplatform "github.com/cloudfoundry/bosh-agent/platform/fakes"
	fakeuuidgen "github.com/cloudfoundry/bosh-utils/uuid/fakes"
)

var _ = Describe("SyncDNSState", func() {
	var (
		localDNSState     []byte
		syncDNSState      SyncDNSState
		fakeFileSystem    *fakesys.FakeFileSystem
		fakeUUIDGenerator *fakeuuidgen.FakeGenerator
		fakePlatform      *fakeplatform.FakePlatform

		path string
		err  error
	)

	BeforeEach(func() {
		fakePlatform = fakeplatform.NewFakePlatform()
		fakeFileSystem = fakePlatform.GetFs().(*fakesys.FakeFileSystem)
		fakeUUIDGenerator = fakeuuidgen.NewFakeGenerator()
		path = "/blobstore-dns-records.json"
		syncDNSState = NewSyncDNSState(fakePlatform, path, fakeUUIDGenerator)
		err = nil
		localDNSState = []byte(`{
					"version": 1234,
					"records": [
						["rec", "ip"]
					],
					"record_keys": ["id", "instance_group", "az", "network", "deployment", "ip"],
					"record_infos": [
						["id-1", "instance-group-1", "az1", "network1", "deployment1", "ip1"]
					]
				}`)
	})

	Describe("#SaveState", func() {
		Context("when there are failures", func() {
			Context("when saving the marshalled DNS state", func() {
				It("fails saving the DNS state", func() {
					fakeFileSystem.WriteFileError = errors.New("fake fail saving error")

					err = syncDNSState.SaveState(localDNSState)
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("writing the blobstore DNS state: fake fail saving error"))

				})
			})

			Context("when writing to a temp file fails", func() {
				It("does not override the existing records.json", func() {
					fakeFileSystem.WriteFile(path, []byte("{}"))

					fakeUUIDGenerator.GeneratedUUID = "fake-generated-uuid"
					fakeFileSystem.WriteFileErrors[path+"fake-generated-uuid"] = errors.New("failed to write tmp file")

					err = syncDNSState.SaveState(localDNSState)
					Expect(err).To(MatchError("writing the blobstore DNS state: failed to write tmp file"))

					contents, err := fakeFileSystem.ReadFile(path)
					Expect(err).ToNot(HaveOccurred())

					Expect(contents).To(MatchJSON("{}"))
				})
			})

			Context("when generating a uuid fails", func() {
				It("returns an error", func() {
					fakeUUIDGenerator.GenerateError = errors.New("failed to generate a uuid")

					err = syncDNSState.SaveState(localDNSState)
					Expect(err).To(MatchError("generating uuid for temp file: failed to generate a uuid"))
				})
			})

			Context("when the rename fails", func() {
				It("returns an error", func() {
					fakeFileSystem.RenameError = errors.New("failed to rename")

					err = syncDNSState.SaveState(localDNSState)
					Expect(err).To(MatchError("renaming: failed to rename"))
				})
			})

			Context("when setting the file permissions fails", func() {
				It("returns an error", func() {
					fakePlatform.SetupRecordsJSONPermissionErr = errors.New("failed to set permissions")
					err = syncDNSState.SaveState(localDNSState)
					Expect(err).To(MatchError("setting permissions of blobstore DNS state: failed to set permissions"))
				})
			})
		})

		Context("when there are no failures", func() {
			BeforeEach(func() {
				fakeUUIDGenerator.GeneratedUUID = "fake-generated-uuid"
			})

			It("quietly saves the state in the path", func() {
				err = syncDNSState.SaveState(localDNSState)
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeFileSystem.RenameOldPaths[0]).To(Equal(path + "fake-generated-uuid"))
				Expect(fakeFileSystem.RenameNewPaths[0]).To(Equal(path))
				Expect(fakeFileSystem.WriteFileQuietlyCallCount).To(Equal(1))
				Expect(fakeFileSystem.WriteFileCallCount).To(Equal(0))

				contents, err := fakeFileSystem.ReadFile(path)
				Expect(err).ToNot(HaveOccurred())

				Expect(contents).To(MatchJSON(`{
					"version": 1234,
					"records": [
						["rec", "ip"]
					],
					"record_keys": ["id", "instance_group", "az", "network", "deployment", "ip"],
					"record_infos": [
						["id-1", "instance-group-1", "az1", "network1", "deployment1", "ip1"]
					]
				}`))
			})

			It("should set platorm specific permissions", func() {
				err = syncDNSState.SaveState(localDNSState)
				Expect(err).ToNot(HaveOccurred())

				Expect(fakePlatform.SetupRecordsJSONPermissionPath).To(Equal(path + "fake-generated-uuid"))
			})
		})
	})

	Describe("#NeedsUpdate", func() {
		It("returns true when state file does not exist", func() {
			Expect(syncDNSState.NeedsUpdate(0)).To(BeTrue())
		})

		Context("when state file exists", func() {
			BeforeEach(func() {
				fakeFileSystem.WriteFile(path, []byte(`{"version":1}`))
			})

			It("returns true when the state file version is less than the supplied version", func() {
				Expect(syncDNSState.NeedsUpdate(2)).To(BeTrue())
			})

			It("returns false when the state file version is equal to the supplied version", func() {
				Expect(syncDNSState.NeedsUpdate(1)).To(BeFalse())
			})

			It("returns false when the state file version is greater than the supplied version", func() {
				Expect(syncDNSState.NeedsUpdate(0)).To(BeFalse())
			})

			It("returns true there is an error loading the state", func() {
				fakeFileSystem.ReadFileError = errors.New("fake fail reading error")

				Expect(syncDNSState.NeedsUpdate(2)).To(BeTrue())
			})

			It("returns true when unmarshalling the version fails", func() {
				fakeFileSystem.WriteFile(path, []byte(`garbage`))

				Expect(syncDNSState.NeedsUpdate(2)).To(BeTrue())
			})
		})
	})
})
