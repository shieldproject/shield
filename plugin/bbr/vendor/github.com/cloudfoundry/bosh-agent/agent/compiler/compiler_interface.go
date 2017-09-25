package compiler

import (
	boshmodels "github.com/cloudfoundry/bosh-agent/agent/applier/models"
	boshcrypto "github.com/cloudfoundry/bosh-utils/crypto"
)

type Compiler interface {
	Compile(pkg Package, deps []boshmodels.Package) (blobID string, digest boshcrypto.Digest, err error)
}

type Package struct {
	BlobstoreID string `json:"blobstore_id"`
	Name        string
	Sha1        boshcrypto.MultipleDigest
	Version     string
}

type Dependencies map[string]Package
