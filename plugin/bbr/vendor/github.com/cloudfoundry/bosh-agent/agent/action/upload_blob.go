package action

import (
	"bytes"
	"encoding/base64"
	"errors"
	"github.com/cloudfoundry/bosh-utils/blobstore"
	boshcrypto "github.com/cloudfoundry/bosh-utils/crypto"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type UploadBlobSpec struct {
	BlobID   string                    `json:"blob_id"`
	Checksum boshcrypto.MultipleDigest `json:"checksum"`
	Payload  string                    `json:"payload"`
}

type UploadBlobAction struct {
	blobManager blobstore.BlobManagerInterface
}

func NewUploadBlobAction(blobManager blobstore.BlobManagerInterface) UploadBlobAction {
	return UploadBlobAction{blobManager: blobManager}
}

func (a UploadBlobAction) IsAsynchronous(_ ProtocolVersion) bool {
	return true
}

func (a UploadBlobAction) IsPersistent() bool {
	return false
}

func (a UploadBlobAction) IsLoggable() bool {
	return false
}

func (a UploadBlobAction) Run(content UploadBlobSpec) (string, error) {

	decodedPayload, err := base64.StdEncoding.DecodeString(content.Payload)
	if err != nil {
		return content.BlobID, err
	}

	if err = a.validatePayload(decodedPayload, content.Checksum); err != nil {
		return content.BlobID, err
	}

	reader := bytes.NewReader(decodedPayload)

	err = a.blobManager.Write(content.BlobID, reader)

	return content.BlobID, err
}

func (a UploadBlobAction) validatePayload(payload []byte, payloadDigest boshcrypto.Digest) error {
	err := payloadDigest.Verify(bytes.NewReader(payload))
	if err != nil {
		return bosherr.WrapErrorf(err, "Payload corrupted. Checksum mismatch. Expected '%s'", payloadDigest.String())
	}

	return nil
}

func (a UploadBlobAction) Resume() (interface{}, error) {
	return nil, errors.New("not supported")
}

func (a UploadBlobAction) Cancel() error {
	return errors.New("not supported")
}
