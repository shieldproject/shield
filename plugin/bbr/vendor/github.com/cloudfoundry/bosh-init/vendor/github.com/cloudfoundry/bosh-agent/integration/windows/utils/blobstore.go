package utils

import (
	"io"
	"net/http"
	"os"

	davclient "github.com/cloudfoundry/bosh-davcli/client"
	davconfig "github.com/cloudfoundry/bosh-davcli/config"
	"github.com/cloudfoundry/bosh-utils/uuid"
)

type BlobClient struct {
	dav           davclient.Client
	uuidGenerator uuid.Generator
}

func (b BlobClient) Create(filepath string) (string, error) {
	uuid, err := b.uuidGenerator.Generate()
	if err != nil {
		return "", err
	}
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	info, err := file.Stat()
	if err != nil {
		return "", err
	}

	err = b.dav.Put(uuid, file, info.Size())
	if err != nil {
		return "", err
	}
	return uuid, nil
}

func (b BlobClient) Get(uuid string, destinationPath string) error {
	readCloser, err := b.dav.Get(uuid)
	if err != nil {
		return err
	}
	defer readCloser.Close()

	targetFile, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer targetFile.Close()

	_, err = io.Copy(targetFile, readCloser)

	return err
}

func NewBlobstore(uri string) BlobClient {
	config := davconfig.Config{Endpoint: uri}
	client := davclient.NewClient(config, http.DefaultClient)

	return BlobClient{
		dav:           client,
		uuidGenerator: uuid.NewGenerator(),
	}
}
