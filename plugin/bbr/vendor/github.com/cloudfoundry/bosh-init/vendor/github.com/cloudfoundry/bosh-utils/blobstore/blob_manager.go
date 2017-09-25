package blobstore

import (
	"io"
	"os"
	"path"
	"strings"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type BlobManager struct {
	fs            boshsys.FileSystem
	blobstorePath string
}

func NewBlobManager(fs boshsys.FileSystem, blobstorePath string) (manager BlobManager) {
	manager.fs = fs
	manager.blobstorePath = blobstorePath
	return
}

func (manager BlobManager) Fetch(blobID string) (boshsys.File, error, int) {
	blobPath := path.Join(manager.blobstorePath, blobID)

	readOnlyFile, err := manager.fs.OpenFile(blobPath, os.O_RDONLY, os.ModeDir)
	if err != nil {
		statusCode := 500
		if strings.Contains(err.Error(), "no such file") {
			statusCode = 404
		}
		return nil, bosherr.WrapError(err, "Reading blob"), statusCode
	}

	return readOnlyFile, nil, 200
}

func (manager BlobManager) Write(blobID string, reader io.Reader) error {
	blobPath := path.Join(manager.blobstorePath, blobID)

	writeOnlyFile, err := manager.fs.OpenFile(blobPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		err = bosherr.WrapError(err, "Opening blob store file")
		return err
	}

	defer func() {
		_ = writeOnlyFile.Close()
	}()
	_, err = io.Copy(writeOnlyFile, reader)
	if err != nil {
		err = bosherr.WrapError(err, "Updating blob")
	}
	return err
}
