package blobstore

import (
	"fmt"
	"path"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/cloudfoundry/bosh-utils/system"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
)

const (
	BlobstoreTypeDummy = "dummy"
	BlobstoreTypeLocal = "local"
)

type Provider struct {
	fs        system.FileSystem
	runner    system.CmdRunner
	configDir string
	uuidGen   boshuuid.Generator
	logger    boshlog.Logger
}

func NewProvider(
	fs system.FileSystem,
	runner system.CmdRunner,
	configDir string,
	logger boshlog.Logger,
) Provider {
	return Provider{
		uuidGen:   boshuuid.NewGenerator(),
		fs:        fs,
		runner:    runner,
		configDir: configDir,
		logger:    logger,
	}
}

func (p Provider) Get(storeType string, options map[string]interface{}) (blobstore Blobstore, err error) {
	configName := fmt.Sprintf("blobstore-%s.json", storeType)
	externalConfigFile := path.Join(p.configDir, configName)

	switch storeType {
	case BlobstoreTypeDummy:
		blobstore = newDummyBlobstore()

	case BlobstoreTypeLocal:
		blobstore = NewLocalBlobstore(
			p.fs,
			p.uuidGen,
			options,
		)

	default:
		blobstore = NewExternalBlobstore(
			storeType,
			options,
			p.fs,
			p.runner,
			p.uuidGen,
			externalConfigFile,
		)
	}

	blobstore = NewSHA1VerifiableBlobstore(blobstore)

	blobstore = NewRetryableBlobstore(blobstore, 3, p.logger)

	err = blobstore.Validate()
	if err != nil {
		err = bosherr.WrapError(err, "Validating blobstore")
	}
	return
}
