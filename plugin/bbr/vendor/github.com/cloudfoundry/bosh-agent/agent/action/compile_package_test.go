package action_test

import (
	"encoding/json"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agent/action"
	boshmodels "github.com/cloudfoundry/bosh-agent/agent/applier/models"
	boshcomp "github.com/cloudfoundry/bosh-agent/agent/compiler"
	fakecomp "github.com/cloudfoundry/bosh-agent/agent/compiler/fakes"
	boshcrypto "github.com/cloudfoundry/bosh-utils/crypto"
)

func getCompileActionArguments() (blobID string, multiDigest boshcrypto.MultipleDigest, name, version string, deps boshcomp.Dependencies) {
	blobID = "fake-blobstore-id"
	multiDigest = boshcrypto.MustNewMultipleDigest(boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, "fake-sha1"))
	name = "fake-package-name"
	version = "fake-package-version"
	deps = boshcomp.Dependencies{
		"first_dep": boshcomp.Package{
			BlobstoreID: "first_dep_blobstore_id",
			Name:        "first_dep",
			Sha1:        boshcrypto.MustNewMultipleDigest(boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, "first_dep_sha1")),
			Version:     "first_dep_version",
		},
		"sec_dep": boshcomp.Package{
			BlobstoreID: "sec_dep_blobstore_id",
			Name:        "sec_dep",
			Sha1:        boshcrypto.MustNewMultipleDigest(boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, "sec_dep_sha1")),
			Version:     "sec_dep_version",
		},
	}
	return
}

var _ = Describe("CompilePackageAction", func() {
	var (
		compiler *fakecomp.FakeCompiler
		action   CompilePackageAction
	)

	BeforeEach(func() {
		compiler = fakecomp.NewFakeCompiler()
		action = NewCompilePackage(compiler)
	})

	AssertActionIsAsynchronous(action)
	AssertActionIsNotPersistent(action)
	AssertActionIsLoggable(action)

	AssertActionIsNotCancelable(action)
	AssertActionIsNotResumable(action)

	Describe("Run", func() {
		It("can unmarshal deps arguments", func() {
			depsJSON := `{"foo": {
      								"name":"foo",
      								"version":"0ee95716c58cf7aab3ef7301ff907118552c2dda.1",
      								"sha1":"9c7b167258b49ffa91c1689670bba9460808ad40",
      								"blobstore_id":"06f48a15-d739-4cca-4af1-ed95b5c791de"
      								}}`

			var deps boshcomp.Dependencies
			err := json.Unmarshal([]byte(depsJSON), &deps)
			Expect(err).ToNot(HaveOccurred())

			fooDep := deps["foo"]
			Expect(fooDep.Name).To(Equal("foo"))
			Expect(fooDep.Sha1.String()).To(Equal("9c7b167258b49ffa91c1689670bba9460808ad40"))
			Expect(fooDep.BlobstoreID).To(Equal("06f48a15-d739-4cca-4af1-ed95b5c791de"))
			Expect(fooDep.Version).To(Equal("0ee95716c58cf7aab3ef7301ff907118552c2dda.1"))
		})

		It("compile package compiles the package and returns blob id", func() {
			compiler.CompileBlobID = "my-blob-id"
			compiler.CompileDigest = boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, "some checksum")

			expectedPkg := boshcomp.Package{
				BlobstoreID: "fake-blobstore-id",
				Sha1:        boshcrypto.MustNewMultipleDigest(boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, "fake-sha1")),
				Name:        "fake-package-name",
				Version:     "fake-package-version",
			}

			expectedValue := map[string]interface{}{
				"result": map[string]string{
					"blobstore_id": "my-blob-id",
					"sha1":         "some checksum",
				},
			}

			expectedDeps := []boshmodels.Package{
				{
					Name:    "first_dep",
					Version: "first_dep_version",
					Source: boshmodels.Source{
						Sha1:        boshcrypto.MustNewMultipleDigest(boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, "first_dep_sha1")),
						BlobstoreID: "first_dep_blobstore_id",
					},
				},
				{
					Name:    "sec_dep",
					Version: "sec_dep_version",
					Source: boshmodels.Source{
						Sha1:        boshcrypto.MustNewMultipleDigest(boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, "sec_dep_sha1")),
						BlobstoreID: "sec_dep_blobstore_id",
					},
				},
			}

			value, err := action.Run(getCompileActionArguments())
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(Equal(expectedValue))

			Expect(compiler.CompilePkg).To(Equal(expectedPkg))

			// Using ConsistOf since package dependencies are specified as a hash (no order)
			Expect(compiler.CompileDeps).To(ConsistOf(expectedDeps))
		})

		It("returns error when compile fails", func() {
			compiler.CompileErr = errors.New("fake-compile-error")

			_, err := action.Run(getCompileActionArguments())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-compile-error"))
		})
	})
})
