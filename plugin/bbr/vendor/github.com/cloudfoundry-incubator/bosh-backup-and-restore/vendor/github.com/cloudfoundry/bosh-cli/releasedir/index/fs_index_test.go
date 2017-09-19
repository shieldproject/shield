package index_test

import (
	"errors"
	"path/filepath"
	"strings"

	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshidx "github.com/cloudfoundry/bosh-cli/releasedir/index"
	fakeidx "github.com/cloudfoundry/bosh-cli/releasedir/index/indexfakes"
)

var _ = Describe("FSIndex", func() {
	var (
		reporter *fakeidx.FakeReporter
		blobs    *fakeidx.FakeIndexBlobs
		fs       *fakesys.FakeFileSystem
		index    boshidx.FSIndex
	)

	BeforeEach(func() {
		reporter = &fakeidx.FakeReporter{}
		blobs = &fakeidx.FakeIndexBlobs{}
		fs = fakesys.NewFakeFileSystem()
		index = boshidx.NewFSIndex("index-name", filepath.Join("/", "dir"), true, true, reporter, blobs, fs)
	})

	Describe("Find", func() {
		It("returns nothing if entry with fingerprint is not found", func() {
			path, sha1, err := index.Find("name", "fp")
			Expect(err).ToNot(HaveOccurred())
			Expect(path).To(BeEmpty())
			Expect(sha1).To(BeEmpty())
		})

		It("returns path and sha1 based on sha1 if entry with fingerprint is found", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "name", "index.yml"), `---
builds:
  fp2: {version: fp2, sha1: fp2-sha1}
  fp: {version: fp, sha1: fp-sha1}
format-version: "2"`)

			blobs.GetStub = func(name string, blobID string, sha1 string) (string, error) {
				Expect(name).To(Equal("name/fp"))
				Expect(blobID).To(Equal(""))
				Expect(sha1).To(Equal("fp-sha1"))
				return "path", nil
			}

			path, sha1, err := index.Find("name", "fp")
			Expect(err).ToNot(HaveOccurred())
			Expect(path).To(Equal("path"))
			Expect(sha1).To(Equal("fp-sha1"))
		})

		It("returns path and sha1 based on sha1 if entry with fingerprint is found in non-prefixed index file", func() {
			index = boshidx.NewFSIndex("index-name", filepath.Join("/", "dir"), false, true, reporter, blobs, fs)

			fs.WriteFileString(filepath.Join("/", "dir", "index.yml"), `---
builds:
  fp2: {version: fp2, sha1: fp2-sha1}
  fp: {version: fp, sha1: fp-sha1}
format-version: "2"`)

			blobs.GetReturns("path", nil)

			path, sha1, err := index.Find("name", "fp")
			Expect(err).ToNot(HaveOccurred())
			Expect(path).To(Equal("path"))
			Expect(sha1).To(Equal("fp-sha1"))
		})

		It("returns path and sha1 based on blob id and sha1 if entry with fingerprint is found", func() {
			fs.WriteFileString("/dir/name/index.yml", `---
builds:
  fp2: {version: fp2, sha1: fp2-sha1, blobstore_id: fp2-blob-id}
  fp: {version: fp, sha1: fp-sha1, blobstore_id: fp-blob-id}
format-version: "2"`)

			blobs.GetStub = func(name string, blobID string, sha1 string) (string, error) {
				Expect(name).To(Equal("name/fp"))
				Expect(blobID).To(Equal("fp-blob-id"))
				Expect(sha1).To(Equal("fp-sha1"))
				return "path", nil
			}

			path, sha1, err := index.Find("name", "fp")
			Expect(err).ToNot(HaveOccurred())
			Expect(path).To(Equal("path"))
			Expect(sha1).To(Equal("fp-sha1"))
		})

		It("returns error if found entry cannot be fetched", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "name", "index.yml"), `---
builds:
  fp: {version: fp, sha1: fp-sha1, blobstore_id: fp-blob-id}
format-version: "2"`)

			blobs.GetReturns("", errors.New("fake-err"))

			_, _, err := index.Find("name", "fp")
			Expect(err).To(Equal(errors.New("fake-err")))
		})

		It("does not require version to equal entry key", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "name", "index.yml"), `---
builds:
  fp: {version: other-fp}
format-version: "2"`)

			_, _, err := index.Find("name", "fp")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error if name is empty", func() {
			_, _, err := index.Find("", "fp")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected non-empty name"))
		})

		It("returns error if fingerprint is empty", func() {
			_, _, err := index.Find("name", "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected non-empty fingerprint"))
		})

		It("returns error if index file cannot be unmarshalled", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "name", "index.yml"), "-")

			_, _, err := index.Find("name", "fp")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("line 1"))
		})

		It("returns error if reading index file fails", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "name", "index.yml"), "")
			fs.ReadFileError = errors.New("fake-err")

			_, _, err := index.Find("name", "fp")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})

	Describe("Add", func() {
		It("adds new entry when no index file exists", func() {
			blobs.AddStub = func(name, path, sha1 string) (string, string, error) {
				Expect(name).To(Equal("name/fp"))
				Expect(path).To(Equal("path"))
				Expect(sha1).To(Equal("sha1"))
				return "blob-id", "blob-path", nil
			}

			path, sha1, err := index.Add("name", "fp", "path", "sha1")
			Expect(err).ToNot(HaveOccurred())
			Expect(path).To(Equal("blob-path"))
			Expect(sha1).To(Equal("sha1"))

			Expect(fs.ReadFileString(filepath.Join("/", "dir", "name", "index.yml"))).To(Equal(`builds:
  fp:
    version: fp
    blobstore_id: blob-id
    sha1: sha1
format-version: "2"
`))
		})

		It("adds new entry to a non-prefixed index file", func() {
			index = boshidx.NewFSIndex("index-name", filepath.Join("/", "dir"), false, true, reporter, blobs, fs)

			blobs.AddReturns("blob-id", "blob-path", nil)

			path, sha1, err := index.Add("name", "fp", "path", "sha1")
			Expect(err).ToNot(HaveOccurred())
			Expect(path).To(Equal("blob-path"))
			Expect(sha1).To(Equal("sha1"))

			Expect(fs.ReadFileString(filepath.Join("/", "dir", "index.yml"))).To(Equal(`builds:
  fp:
    version: fp
    blobstore_id: blob-id
    sha1: sha1
format-version: "2"
`))
		})

		It("adds new entry with blobstore id to existing index if index allows blobs ids", func() {
			index = boshidx.NewFSIndex("index-name", filepath.Join("/", "dir"), true, true, reporter, blobs, fs)

			fs.WriteFileString(filepath.Join("/", "dir", "name", "index.yml"), `---
builds:
  fp2: {version: fp2, sha1: fp2-sha1, blobstore_id: fp2-blob-id}
format-version: "2"`)

			blobs.AddReturns("blob-id", "blob-path", nil)

			path, sha1, err := index.Add("name", "fp", "path", "sha1")
			Expect(err).ToNot(HaveOccurred())
			Expect(path).To(Equal("blob-path"))
			Expect(sha1).To(Equal("sha1"))

			Expect(fs.ReadFileString(filepath.Join("/", "dir", "name", "index.yml"))).To(Equal(`builds:
  fp:
    version: fp
    blobstore_id: blob-id
    sha1: sha1
  fp2:
    version: fp2
    blobstore_id: fp2-blob-id
    sha1: fp2-sha1
format-version: "2"
`))
		})

		It("adds new entry without blobstore id if index disallows blobs ids", func() {
			index = boshidx.NewFSIndex("index-name", "/dir", true, false, reporter, blobs, fs)

			blobs.AddReturns("", "blob-path", nil)

			path, sha1, err := index.Add("name", "fp", "path", "sha1")
			Expect(err).ToNot(HaveOccurred())
			Expect(path).To(Equal("blob-path"))
			Expect(sha1).To(Equal("sha1"))

			Expect(fs.ReadFileString(filepath.Join("/", "dir", "name", "index.yml"))).To(Equal(`builds:
  fp:
    version: fp
    sha1: sha1
format-version: "2"
`))
		})

		It("returns error when adding entry with blobstore id if index disallows blob ids", func() {
			index = boshidx.NewFSIndex("index-name", "/dir", true, false, reporter, blobs, fs)

			blobs.AddReturns("blob-id", "blob-path", nil)

			_, _, err := index.Add("name", "fp", "path", "sha1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(
				`Internal inconsistency: entry must not include blob ID 'index.indexEntry{Key:"fp", Version:"fp", BlobstoreID:"blob-id", SHA1:"sha1"}'`))
		})

		It("returns error when adding new entry without blobstore id if index allows blob ids", func() {
			index = boshidx.NewFSIndex("index-name", filepath.Join("/", "dir"), true, true, reporter, blobs, fs)

			blobs.AddReturns("", "blob-path", nil)

			_, _, err := index.Add("name", "fp", "path", "sha1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(
				`Internal inconsistency: entry must include blob ID 'index.indexEntry{Key:"fp", Version:"fp", BlobstoreID:"", SHA1:"sha1"}'`))
		})

		It("reports addition", func() {
			blobs.AddReturns("blob-id", "blob-path", nil)

			_, _, err := index.Add("name", "fp", "path", "sha1")
			Expect(err).ToNot(HaveOccurred())

			Expect(reporter.IndexEntryStartedAddingCallCount()).To(Equal(1))
			Expect(reporter.IndexEntryFinishedAddingCallCount()).To(Equal(1))

			kind, desc := reporter.IndexEntryStartedAddingArgsForCall(0)
			Expect(kind).To(Equal("index-name"))
			Expect(desc).To(Equal("name/fp"))

			kind, desc, err = reporter.IndexEntryFinishedAddingArgsForCall(0)
			Expect(kind).To(Equal("index-name"))
			Expect(desc).To(Equal("name/fp"))
			Expect(err).To(BeNil())
		})

		It("reports addition error if blob cannot be added", func() {
			blobs.AddReturns("", "", errors.New("fake-err"))

			_, _, err := index.Add("name", "fp", "path", "sha1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			Expect(reporter.IndexEntryStartedAddingCallCount()).To(Equal(1))
			Expect(reporter.IndexEntryFinishedAddingCallCount()).To(Equal(1))

			kind, desc := reporter.IndexEntryStartedAddingArgsForCall(0)
			Expect(kind).To(Equal("index-name"))
			Expect(desc).To(Equal("name/fp"))

			kind, desc, err = reporter.IndexEntryFinishedAddingArgsForCall(0)
			Expect(kind).To(Equal("index-name"))
			Expect(desc).To(Equal("name/fp"))
			Expect(err).ToNot(BeNil())
		})

		It("reports addition error if writing index fails", func() {
			blobs.AddReturns("blob-id", "blob-path", nil)

			fs.WriteFileError = errors.New("fake-err")

			_, _, err := index.Add("name", "fp", "path", "sha1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			Expect(reporter.IndexEntryStartedAddingCallCount()).To(Equal(1))
			Expect(reporter.IndexEntryFinishedAddingCallCount()).To(Equal(1))

			kind, desc := reporter.IndexEntryStartedAddingArgsForCall(0)
			Expect(kind).To(Equal("index-name"))
			Expect(desc).To(Equal("name/fp"))

			kind, desc, err = reporter.IndexEntryFinishedAddingArgsForCall(0)
			Expect(kind).To(Equal("index-name"))
			Expect(desc).To(Equal("name/fp"))
			Expect(err).ToNot(BeNil())
		})

		It("returns error if there is already an entry with same fingerprint", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "name", "index.yml"), `---
builds:
  fp: {version: fp}
format-version: "2"`)

			_, _, err := index.Add("name", "fp", "path", "sha1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(
				`Trying to add duplicate index entry 'name/fp' and SHA1 'sha1' (conflicts with 'index.indexEntry{Key:"fp", Version:"fp", BlobstoreID:"", SHA1:""}')`))
		})

		It("does not reorder keys needlessly", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "name", "index.yml"), fsIndexSortingFixture)

			blobs.AddReturns("blob-id", "blob-path", nil)
			_, _, err := index.Add("name", "fp", "path", "sha1")
			Expect(err).ToNot(HaveOccurred())

			afterFirstSort, err := fs.ReadFileString(filepath.Join("/", "dir", "name", "index.yml"))
			Expect(err).ToNot(HaveOccurred())

			Expect(afterFirstSort).ToNot(Equal(fsIndexSortingFixture)) // sanity check

			blobs.AddReturns("another-blob-id", "another-blob-path", nil)
			_, _, err = index.Add("name", "another-fp", "another-path", "another-sha1")
			Expect(err).ToNot(HaveOccurred())

			after, err := fs.ReadFileString(filepath.Join("/", "dir", "name", "index.yml"))
			Expect(err).ToNot(HaveOccurred())

			Expect(after).ToNot(Equal(afterFirstSort)) // sanity check
			Expect(strings.Replace(after,
				`  another-fp:
    version: another-fp
    blobstore_id: another-blob-id
    sha1: another-sha1
`, "", 1)).To(Equal(afterFirstSort))
		})

		It("returns error if name is empty", func() {
			_, _, err := index.Add("", "fp", "path", "sha1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected non-empty name"))
		})

		It("returns error if fingerprint is empty", func() {
			_, _, err := index.Add("name", "", "path", "sha1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected non-empty fingerprint"))
		})

		It("returns error if path is empty", func() {
			_, _, err := index.Add("name", "fp", "", "sha1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected non-empty archive path"))
		})

		It("returns error if sha1 is empty", func() {
			_, _, err := index.Add("name", "fp", "path", "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected non-empty archive SHA1"))
		})

		It("returns error if index file cannot be unmarshalled", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "name", "index.yml"), "-")

			_, _, err := index.Add("name", "fp", "path", "sha1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("line 1"))
		})

		It("returns error if reading index file fails", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "name", "index.yml"), "")
			fs.ReadFileError = errors.New("fake-err")

			_, _, err := index.Add("name", "fp", "path", "sha1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})

// Fixture needs to be long because natural sort may succeed for smaller sizes
const fsIndexSortingFixture = `
builds:
  0f99de1633d54d6c66dac8f6467daa28291fb166:
    version: 0f99de1633d54d6c66dac8f6467daa28291fb166
    blobstore_id: 80cf0db2-1e22-4dff-9d4d-e074cccac754
    sha1: 0259e66c440ed57d10dd9d8461ed067b22008f72
  1ba995dda7bd8eb914739f8aa801b6bd4b13aa10:
    version: 1ba995dda7bd8eb914739f8aa801b6bd4b13aa10
    blobstore_id: 158b92b6-5802-4111-8d31-2fe1c9afe573
    sha1: f6f30b07e1b9bc1e5066efcbbf3e1187fd0241fe
  1e44d704916893f16c988a10f7a5de343559eb87:
    version: 1e44d704916893f16c988a10f7a5de343559eb87
    blobstore_id: 5760b739-4db4-4251-998b-981874525574
    sha1: 2f461b8e0c436b349359ce37ab51d1f5334da3fc
  1f3ec928aa65f43de8d00045d37cb3e92f3be2bb:
    version: "11"
    blobstore_id: rest/objects/4e4e78bca21e122204e4e9863926b1051310ea168eb3
    sha1: dc717cd17e0e86b48ae0453ccf73c4d4388cfdd1
  2c7b2134c298011776b7a4b882244d3efdf5c0b9:
    version: 2c7b2134c298011776b7a4b882244d3efdf5c0b9
    blobstore_id: b2e11101-46e3-4923-919e-23daffd11297
    sha1: 1847402ee1cfb15fa09f541265d2dfe7cccdf5df
  34bb529997a5ad31292a425c1a19f85bffc4e249:
    version: "20"
    blobstore_id: f4f08f5a-df85-4920-a367-0ae9c76eec2c
    sha1: de2cda415873dbf7bc4464ed5e00e546671b1b64
  3d82f6e9d7451941e5abb35c60287323adfcb0d4:
    version: 3d82f6e9d7451941e5abb35c60287323adfcb0d4
    blobstore_id: d40c09a7-4dd1-4966-9fb5-7f18faa1d41f
    sha1: e2ac2b9334a76cc3a210f0c44159616189d21a36
  3f7479cc2653d8b95ee97900e60775594ef7c05b:
    version: 3f7479cc2653d8b95ee97900e60775594ef7c05b
    blobstore_id: df15710e-949b-4d7a-adf2-6816357edf73
    sha1: cb7e5a6de926b988ea888e4ce4507261c0b936d2
  4d250998bf31db1fc28f88f243a895562cfff387:
    version: 4d250998bf31db1fc28f88f243a895562cfff387
    blobstore_id: 02472f69-363d-46a1-9248-41de1d525a6c
    sha1: a9bdaf22368bbdb48c09b3050fe5cf9b54e15ab4
  4d30424182737b18d4a65d0e96c4e7c1ac548d1d:
    version: 4d30424182737b18d4a65d0e96c4e7c1ac548d1d
    blobstore_id: 2aed46cc-6b3e-45bf-92d1-814741c70f6d
    sha1: d73b8086faab20951b5a27a1bae8511961321949
  06a47e4bffdad38c76c6f1069d1f477635800a5e:
    version: 06a47e4bffdad38c76c6f1069d1f477635800a5e
    blobstore_id: 9e4e46b6-a845-4fb8-bd6e-26869b7af3ee
    sha1: 462fd4c88b63b8027b610cd28ad36a23def61775
  8a334e2d4a725f263cf1dbcb59ffd3c37d0b3ad0:
    version: "7"
    blobstore_id: rest/objects/4e4e78bca41e121204e4e86ee5392105085fd1062b61
    sha1: e6a9a3e8def5c7691405f1aadc4322f4e584a185
  8b7977b57a31aea379d10ab04851d0ea57009b07:
    version: 8b7977b57a31aea379d10ab04851d0ea57009b07
    blobstore_id: 189ad1bf-e3fc-44f4-b173-33be479973a5
    sha1: 7b40a601834629a8e197a4bb5c01621055075c8f
  12db7a19f93a311b821cc476fe8c3c46fae2ef3c:
    version: 12db7a19f93a311b821cc476fe8c3c46fae2ef3c
    blobstore_id: 79046f68-c01f-45b0-a6a1-d4b1fd4f7efe
    sha1: c6489d0461805a26ccbfc7214e390e9e3b89b218
  45aa8e7235538dc00a34461f23ccd821de11192f:
    version: 45aa8e7235538dc00a34461f23ccd821de11192f
    blobstore_id: 58c4f159-ec9b-433a-8c53-f9c951b7acbb
    sha1: 7ee5c8b50710db7d1bac04515d92ce828f8c811a
  50f0a11bd41eb9c6e45a20428dcf421b78a02c2b:
    version: 50f0a11bd41eb9c6e45a20428dcf421b78a02c2b
    blobstore_id: 56777d2a-114a-4e74-9e9a-aa367b79ed9d
    sha1: f35c7a4d8c9b093fc403b70f827f24848e80734a
  55f69fb2492a5e5ba86d3c28ea6127f31227169c:
    version: 55f69fb2492a5e5ba86d3c28ea6127f31227169c
    blobstore_id: 0184ee37-f744-4901-889b-a49f727a0a5e
    sha1: f8ec5f3f722b17fed4093cd244fef2a2f4f36cff
  5db2ee56b543227bf9684be7f25fb42b0ee5d161:
    version: 5db2ee56b543227bf9684be7f25fb42b0ee5d161
    blobstore_id: c0d3ab18-9928-4f47-b21b-b73f2257c170
    sha1: bf1ce012cf0a9bf6729b3af0360e426a5e8a4e4a
  61f576fafb72e856ef4765f59ed5b387f40d892e:
    version: 61f576fafb72e856ef4765f59ed5b387f40d892e
    blobstore_id: 35429e5b-7efc-48e7-a7a9-8bbb51d71458
    sha1: b72b42c9625f46ce0f9d4a2b6c71f592fa9b8ebb
  6dcccf7116fbe40f63001dee8dbf61f53a6464ab:
    version: 6dcccf7116fbe40f63001dee8dbf61f53a6464ab
    blobstore_id: 8abea158-1e06-4f6b-bb1b-d3b0515430c5
    sha1: fdf458dbd4c65087078bc7a633f5ad71e3da554a
  6f7ff6b7ea5ce202a6b38d522ab8ea55d503a58e:
    version: 6f7ff6b7ea5ce202a6b38d522ab8ea55d503a58e
    blobstore_id: 5bef0783-06c1-45f1-b878-0fd12c6675e8
    sha1: 32ae127cc0ba6f97d1f311390652b2a12165f656
  85c90111b019bf59b3274bff1e5801da2c85b694:
    version: 85c90111b019bf59b3274bff1e5801da2c85b694
    blobstore_id: 848ca546-1c0b-40a7-9ff7-4f2b9a247174
    sha1: 3f626ca679a9126b6d508e4dcd02439bceb72304
  91bf180ebea3f2e16622c48215476ebf2f75a18e:
    version: 91bf180ebea3f2e16622c48215476ebf2f75a18e
    blobstore_id: f28e4f32-1b74-473e-8aca-af7e5eea8613
    sha1: 0f7a13e96137c6a3e340af2d08219a6410576477
  93aa5c2fd4b7a0cae4c3db8751497b1b27097a9d:
    version: 93aa5c2fd4b7a0cae4c3db8751497b1b27097a9d
    blobstore_id: 06c8321a-d725-46ed-a1eb-9b6c3ccabc3c
    sha1: 815cde9cd22882fa678d44714d322cb5f3ab5727
  9b94fcd3906fa33059def78a697c223ba46286df:
    version: "16"
    blobstore_id: 991e83e2-8d96-40c2-9152-04f584a590ab
    sha1: f3d6126801c0ff05ee2c871e4c01f10f46877d08
  9b93385af4e3191db346478415306cf7d4caef0c:
    version: 9b93385af4e3191db346478415306cf7d4caef0c
    blobstore_id: 9a171e4f-afcf-432d-94d5-3270ff2e8a59
    sha1: e99027633d2c34aef0b77cd1e956f3db81ef6908
  9e3f52181cfdc470517b1af08a342d8e901820d7:
    version: 9e3f52181cfdc470517b1af08a342d8e901820d7
    blobstore_id: 5dd27264-d766-4597-9d95-c45f021d0871
    sha1: d09b37d63b0a65c87503e106a1801129fed477f5
  9e25cc97867215f528503c4bd15596dccace46a1:
    version: "8"
    blobstore_id: rest/objects/4e4e78bca21e121004e4e7d511f5530509433932a231
    sha1: a1b82f4fbc843cbe931312ae5db369ff7a8dc748
  0132daaa1c77b18abff166c3087cf2d12203fba7:
    version: "10"
    blobstore_id: rest/objects/4e4e78bca31e121204e4e86ee39692050f8500043616
    sha1: c821bdb0e0c539a663bb3323babd7fe8f7c23df1
  157a8ffeaed8dab6316d87510d9503195c9c2692:
    version: "2"
    blobstore_id: rest/objects/4e4e78bca31e121204e4e86ee3969204f8c978dcbb98
    sha1: b3741954ca0fb7c50ab9135173c057377e408f63
  218c4b5d6914eaf7f3ef2bf768f7255e83165194:
    version: 218c4b5d6914eaf7f3ef2bf768f7255e83165194
    blobstore_id: 12681469-60d5-4c46-a602-45589ab21af4
    sha1: 027a46bef7cab2188693eb2e06ddc866db2f00cc
  251d59de092b660d4b905e359fd0ae3c7fb0187d:
    version: 251d59de092b660d4b905e359fd0ae3c7fb0187d
    blobstore_id: 1c3ffaef-aec5-4058-a560-a46549e28593
    sha1: b3913d9cf10c207f50e576d0d9d4630c90d3f6ef
  264e5bada8ea3c5f7a52e4b80db9d7d04d8cb928:
    version: 264e5bada8ea3c5f7a52e4b80db9d7d04d8cb928
    blobstore_id: 7b46788e-4127-4379-b70e-a985205e7128
    sha1: edaf8a9448acfdd3d1ca7029818410ed333bc40a
  334c550dfbbfa4aec09bedb86c0155c14bb4acae:
    version: "9"
    blobstore_id: rest/objects/4e4e78bca51e122204e4e9863f28f3050d308d467efa
    sha1: b82406636fe4db39b257feee8e57ac438734b2df
  742d76d4ff3277adef24890a4c79c02480b4dbe3:
    version: "3"
    blobstore_id: rest/objects/4e4e78bca31e121204e4e86ee3969204fc65ffbf023d
    sha1: 6be5dbc64309c7d0a55f2864f48d14824e53d7cc
  963e768cf08a4f319367a50d91e0308e80279e61:
    version: "14"
    blobstore_id: 669696b4-22bf-47eb-b501-0c424f474035
    sha1: 80d4d39f77a0bccfa7cbe6343ebe7ec72accb265
  2074b22decbb3c5bed4a5d65a27bf4fe24807b6f:
    version: 2074b22decbb3c5bed4a5d65a27bf4fe24807b6f
    blobstore_id: 0a18764e-11d4-497b-a307-c3c200e58879
    sha1: 767da814db786d53bd05ee519e842a7dfa5958f3
  2192fd1829a04bca4525427c5d2f5dc995c9b5c3:
    version: "12"
    blobstore_id: 8c61b87c-4e59-450a-bc63-760edfe9de9a
    sha1: 503c79f60e20c1f198fe0885ab75340910544913
  7608bbd9de61cc4de0aba8232fcd49aea24bc588:
    version: 7608bbd9de61cc4de0aba8232fcd49aea24bc588
    blobstore_id: d0f15a4c-e76c-45cd-76f3-b6ebf40e0f4f
    sha1: fa86c868172f49f52136b562e384f1a383e611b0
  12158a8997d7447d8cd65e3766d0ef657140b7a9:
    version: 12158a8997d7447d8cd65e3766d0ef657140b7a9
    blobstore_id: 0ed44a3f-992c-4fc8-acae-d093cc56b6df
    sha1: 7f970d9d417171c70c689a9aaafcd11d4064e28d
  36416d30c7ffc1cb9efc772431ead891cb4f0e43:
    version: "19"
    blobstore_id: 329d4bb6-e03a-4268-8224-2adb8c245d08
    sha1: f102c7224d165395a828bf3ba164159d5d757c84
  57428a045c5c1dd64dc95226f06696b462e87dbd:
    version: 57428a045c5c1dd64dc95226f06696b462e87dbd
    blobstore_id: cf452113-9861-4030-8a88-b4a7da280174
    sha1: 06a4653c53422fcae5b5129330ed589547c99e09
  068358f24cadb45e8e8d617b0fcb76e6b52ce320:
    version: 068358f24cadb45e8e8d617b0fcb76e6b52ce320
    blobstore_id: e903b6cc-3f8a-4581-acdf-3e02e88f282d
    sha1: 123137d34ce145f51b0b7b9569bcdc79806ed343
  76183c9ef5f388d2d3d8e08ed2040e250014ed06:
    version: "13"
    blobstore_id: 93e533b5-3232-4110-9083-1cce9d3ce482
    sha1: dc6ec15cb8546b3f72ed937e8b9e5e7a7efa58d7
  749031e80f1595aa0ad0553c6b52789dc42e96b0:
    version: "4"
    blobstore_id: rest/objects/4e4e78bca61e122204e4e98643d9ae04fc693900c97f
    sha1: 03cc98bb27638168bf0072e1677899f76235e256
  58738595b7c1eb8e948c5bcf1356a09291323f65:
    version: 58738595b7c1eb8e948c5bcf1356a09291323f65
    blobstore_id: 91079071-f197-4b22-b204-1cd032b6a63e
    sha1: ad6ace266df09792dc341a545803cbbea0c7615d
  5435856817fef53713511c2ee35226c43414c4c0:
    version: 5435856817fef53713511c2ee35226c43414c4c0
    blobstore_id: d2c6be15-eb6a-4540-b7bd-3440c37015f0
    sha1: b927ccbec3ef5537b9ad8dcd7cca74043ed4639a
  a1f47e3bde25a9ffdcf2dc47fc66e6116fb83561:
    version: "18"
    blobstore_id: 74bd6ab9-feea-4568-aeb3-3a9992b61fff
    sha1: 0ad617507705225345f47fac9e0966b9c25675bb
  a4c62269df79c70f60a13ef4e61fefba65e9fde6:
    version: "15"
    blobstore_id: a08e3c0d-50c5-434f-84c2-31572e91b0d8
    sha1: fa44de19bc8d9eb2f7c5fd01f21eec6e50f4db9f
  a5d38ec2bebcc0c4b80868b41028ce015021d10a:
    version: a5d38ec2bebcc0c4b80868b41028ce015021d10a
    blobstore_id: 4a143e92-95ab-4aaa-8b7f-fcdffc441013
    sha1: 347f6baa879d7c5ba23812986e0b3ed168a1d42e
  a71b0296ec100edb6a69e410defe39c637a63c3f:
    version: a71b0296ec100edb6a69e410defe39c637a63c3f
    blobstore_id: 4d675831-31d1-4766-4575-89410e1d02b7
    sha1: d269c40adfa752539f1f02fda3383cdd987a9652
  abad62506b6f8b419fdc8f42a9e9967045057b45:
    version: abad62506b6f8b419fdc8f42a9e9967045057b45
    blobstore_id: 130cd1f9-415d-4bd5-9d3b-f638d5c7063f
    sha1: 49a084daa2311be6d5aae6839effe41588a8897a
  ad68c69327a2a842bc1cce312556c10d42350fcb:
    version: ad68c69327a2a842bc1cce312556c10d42350fcb
    blobstore_id: ba51fa8d-2c12-4c26-aa9f-3ca3a5b7c15b
    sha1: b32560e90a020d30280463a593e3bd985598eb15
  b0a355bec667ad7c7afb439fbb275d7e28f32cf4:
    version: "17"
    blobstore_id: e6dcf021-645a-4bd2-b17b-1deb4df09482
    sha1: 7e8e5e2db383bd17a9012a179ee518edb48db0ac
  b17fefeaaed8848fe336ab382dde09b0ad044500:
    version: "5"
    blobstore_id: rest/objects/4e4e78bca21e121204e4e86ee151bc05010a66fa1d37
    sha1: a876e590b7b49156bac396de595f64eaf8216bc0
  b753c1f340b2a96663043163b30268c2a73f8861:
    version: b753c1f340b2a96663043163b30268c2a73f8861
    blobstore_id: 6deeb540-6809-4e8c-a4a7-386f24cee9bc
    sha1: 5b505fb226fd25a0f38152ecd8effea02e6df06b
  b994f6bfc668b2932509f51b6b173127fbf53806:
    version: b994f6bfc668b2932509f51b6b173127fbf53806
    blobstore_id: ee940d1d-1aff-4306-b1ea-487fecc2068e
    sha1: 86bf03276a117778b6785dbe1006db353b3c21d6
  b3721d5b21cb50b831e43b460ec90479f44a62ff:
    version: b3721d5b21cb50b831e43b460ec90479f44a62ff
    blobstore_id: 35e7cfa9-aa41-4958-8369-189e9e0478df
    sha1: 23d7f6fd8c0a427f4c548d4e95e2f2f01311d8cd
  bd1489d7ec55cf261f9b400ebeaa4361b91c3ec4:
    version: bd1489d7ec55cf261f9b400ebeaa4361b91c3ec4
    blobstore_id: e2e88067-46b6-4603-ab33-c9fe641a7692
    sha1: 744869ebb74bdb37e7a9a3602d19d51b88b7a94d
  c09dd3b869794af1cc097bbd0d7b1bb9aa23e7d9:
    version: c09dd3b869794af1cc097bbd0d7b1bb9aa23e7d9
    blobstore_id: e981fe1f-4e9d-4d5e-81d6-ec482c861b17
    sha1: 17ff1fccdb4832f9119d3f0c70bffb3be589dc81
  c18b684f2c5ce48682482f90fedbb71ae628d43d:
    version: c18b684f2c5ce48682482f90fedbb71ae628d43d
    blobstore_id: 7b6272a6-8e09-4bb1-7b13-26ff16d51627
    sha1: c081808ee510201d16868ee02bd8447235b1a37d
  c50e80ce722c92b9dac722d461f3de14bbdf131a:
    version: c50e80ce722c92b9dac722d461f3de14bbdf131a
    blobstore_id: 1a17151d-968b-4132-87ec-1b57de5281e4
    sha1: e7ba24f01c6512d62855647d6080476b43d02d3c
  cc1da8b6e4aefd733c9bf4a9c49d7890cc1318d0:
    version: cc1da8b6e4aefd733c9bf4a9c49d7890cc1318d0
    blobstore_id: 886ea3b7-46fa-4e22-9f2a-4d19c5e7a011
    sha1: 42dfec8e088c1511a66b47f0886808f3d08a20a5
  cf8f488099089caebca28c24b7a5caf8108fead5:
    version: "1"
    blobstore_id: rest/objects/4e4e78bca41e121004e4e7d517618f04f7f825dee51d
    sha1: 840c5b8571fbf17e69d3b2d92b434bcc9af6900c
  d5f6e201b5a37a023acbd557b7be402df85c5143:
    version: d5f6e201b5a37a023acbd557b7be402df85c5143
    blobstore_id: 826feffe-321a-4a99-a664-80d4830a67e3
    sha1: b2cdd14a2607b7ae3faee3a21fd2b7f835bb64ae
  d28bfc37e8b3b285a0edbd7d174fd1f3924e0324:
    version: d28bfc37e8b3b285a0edbd7d174fd1f3924e0324
    blobstore_id: feacf1c1-7b32-4058-9c21-45f857fab5c6
    sha1: 7b000f469a411913ac2efedc49fa2ad34d30d0da
  d14351a2076a5879ad2a3a78d7eab5f9b4a0cdd8:
    version: d14351a2076a5879ad2a3a78d7eab5f9b4a0cdd8
    blobstore_id: 42024172-904d-4cfc-a52a-87a91309d4a3
    sha1: a2d3f8631e67a3607447d98dea5db82d7e1fe87a
  d81683871534eec6eb2d523a430b336c82ce38bc:
    version: d81683871534eec6eb2d523a430b336c82ce38bc
    blobstore_id: 42f3ee24-46f6-4526-9b28-cd8528337dba
    sha1: cdedc508c301ef7401ac099fa3588a265bebdfa0
  e818d79d3d1b71d804589e69af854e077e834b81:
    version: e818d79d3d1b71d804589e69af854e077e834b81
    blobstore_id: 191380a7-b318-48b5-9a57-e2307fd76b85
    sha1: dbe9cd84c666a44c27f5971c68891ea0e12f9b60
  e8e8938f17213bc046639933c10f2ce386584eab:
    version: "6"
    blobstore_id: rest/objects/4e4e78bca51e121204e4e86ee8e2c90502c2b6cb4627
    sha1: 07c4338c3d5ff39f1f1677b944adb59be73774f1
  e21e06a64f94124445d35b2e7dd11efaa451a16c:
    version: e21e06a64f94124445d35b2e7dd11efaa451a16c
    blobstore_id: 16f18ff1-ab40-4a57-bfca-8098609fbbef
    sha1: 612c6883545492f301ce40a0104ae0e61e0857eb
  ee6bb2e117ffa1a2acf3be5f5c60d21c064e4cf3:
    version: ee6bb2e117ffa1a2acf3be5f5c60d21c064e4cf3
    blobstore_id: 3d0a0363-2146-4126-a1e0-4f6650a7727e
    sha1: d9d3c55caece066d98fdd7e4e4bf8bd0b8da5151
  f1c4ede10a6368c0b8e90203034cb8f43e6cbdf9:
    version: f1c4ede10a6368c0b8e90203034cb8f43e6cbdf9
    blobstore_id: 57891326-6499-4dd9-9e65-9267c2a79de4
    sha1: 5f69d3c45aa83951397c250171d0d8f9ea1b1671
  f5bd42f56fb0eeac8e708cf3d5f3ad168b0ddfea:
    version: f5bd42f56fb0eeac8e708cf3d5f3ad168b0ddfea
    blobstore_id: fd677040-6243-4266-a057-a25235649d24
    sha1: 6fb85d6e50420de6f818534fea7fe78ef2bb1f1d
  fe06b07fd3f0e4794b8d0e83c9b670d25f7ee4e4:
    version: fe06b07fd3f0e4794b8d0e83c9b670d25f7ee4e4
    blobstore_id: 0fa3c3fc-6124-46ec-8aa9-7376bf23db9d
    sha1: 590d94844a1f5e61a150c0a673142adaf917007c
  ff9a413d3187692f23f0d70bcef7e99a8a9ac450:
    version: ff9a413d3187692f23f0d70bcef7e99a8a9ac450
    blobstore_id: 6ef27507-3a2c-4a6f-b7b6-36d5088d9ca3
    sha1: 13ef011e052ed89444533c3866fa9124538b6df7
  ff9dbb3487f9ead07dc6d90e13d62da1c6b79927:
    version: ff9dbb3487f9ead07dc6d90e13d62da1c6b79927
    blobstore_id: 40591972-17b2-445f-9cd8-57a3b353d7ff
    sha1: ad27674d49e9960a989fd2cbaacf023635cf7916
  ff35b9bf29cf99d5c491ae552ab704cb633ef095:
    version: ff35b9bf29cf99d5c491ae552ab704cb633ef095
    blobstore_id: e5b0ecbc-7d8a-4770-8be9-1d9bd6ec0b40
    sha1: dd64c021db6062999a5227e4832c7bf80cd682d8
format-version: "2"
`
