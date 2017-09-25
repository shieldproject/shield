package models

import "github.com/cloudfoundry/bosh-utils/crypto"

type Source struct {
	Sha1          crypto.Digest
	BlobstoreID   string
	PathInArchive string
}
