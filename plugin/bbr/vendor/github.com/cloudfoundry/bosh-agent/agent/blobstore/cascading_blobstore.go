package blobstore

import (
	boshUtilsBlobStore "github.com/cloudfoundry/bosh-utils/blobstore"
	boshcrypto "github.com/cloudfoundry/bosh-utils/crypto"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

const logTag = "cascadingBlobstore"

type cascadingBlobstore struct {
	innerBlobstore boshUtilsBlobStore.DigestBlobstore
	blobManager    boshUtilsBlobStore.BlobManagerInterface
	logger         boshlog.Logger
}

func NewCascadingBlobstore(
	innerBlobstore boshUtilsBlobStore.DigestBlobstore,
	blobManager boshUtilsBlobStore.BlobManagerInterface,
	logger boshlog.Logger) boshUtilsBlobStore.DigestBlobstore {
	return cascadingBlobstore{
		innerBlobstore: innerBlobstore,
		blobManager:    blobManager,
		logger:         logger,
	}
}

func (b cascadingBlobstore) Get(blobID string, digest boshcrypto.Digest) (string, error) {

	if b.blobManager.BlobExists(blobID) {
		blobPath, err := b.blobManager.GetPath(blobID, digest)

		if err != nil {
			return "", err
		}

		b.logger.Debug(logTag, "Found blob with BlobManager. BlobID: %s", blobID)
		return blobPath, nil
	}

	return b.innerBlobstore.Get(blobID, digest)
}

func (b cascadingBlobstore) CleanUp(fileName string) error {
	return b.innerBlobstore.CleanUp(fileName)
}

func (b cascadingBlobstore) Create(fileName string) (string, boshcrypto.MultipleDigest, error) {
	return b.innerBlobstore.Create(fileName)
}

func (b cascadingBlobstore) Validate() error {
	return b.innerBlobstore.Validate()
}

func (b cascadingBlobstore) Delete(blobID string) error {
	err := b.blobManager.Delete(blobID)

	if err != nil {
		return err
	}

	return b.innerBlobstore.Delete(blobID)
}
