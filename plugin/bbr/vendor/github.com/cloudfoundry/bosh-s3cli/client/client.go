package client

import (
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/cloudfoundry/bosh-s3cli/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// S3Blobstore encapsulates interactions with an S3 compatible blobstore
type S3Blobstore struct {
	s3Client    *s3.S3
	s3cliConfig *config.S3Cli
}

var errorInvalidCredentialsSourceValue = errors.New("the client operates in read only mode. Change 'credentials_source' parameter value ")
var oneTB = int64(1000 * 1024 * 1024 * 1024)

// New returns a BlobstoreClient if the configuration file backing configFile is valid
func New(s3Client *s3.S3, s3cliConfig *config.S3Cli) (S3Blobstore, error) {
	return S3Blobstore{s3Client: s3Client, s3cliConfig: s3cliConfig}, nil
}

// Get fetches a blob from an S3 compatible blobstore
// Destination will be overwritten if exists
func (client *S3Blobstore) Get(src string, dest io.WriterAt) error {
	downloader := s3manager.NewDownloaderWithClient(client.s3Client)

	_, err := downloader.Download(dest, &s3.GetObjectInput{
		Bucket: aws.String(client.s3cliConfig.BucketName),
		Key:    client.key(src),
	})

	if err != nil {
		return err
	}

	return nil
}

// Put uploads a blob to an S3 compatible blobstore
func (client *S3Blobstore) Put(src io.ReadSeeker, dest string) error {
	cfg := client.s3cliConfig
	if cfg.CredentialsSource == config.NoneCredentialsSource {
		return errorInvalidCredentialsSourceValue
	}

	uploader := s3manager.NewUploaderWithClient(client.s3Client, func(u *s3manager.Uploader) {
		u.LeavePartsOnError = false

		if !cfg.MultipartUpload {
			// disable multipart uploads by way of large PartSize configuration
			u.PartSize = oneTB
		}
	})
	uploadInput := &s3manager.UploadInput{
		Body:   src,
		Bucket: aws.String(cfg.BucketName),
		Key:    client.key(dest),
	}
	if cfg.ServerSideEncryption != "" {
		uploadInput.ServerSideEncryption = aws.String(cfg.ServerSideEncryption)
	}
	if cfg.SSEKMSKeyID != "" {
		uploadInput.SSEKMSKeyId = aws.String(cfg.SSEKMSKeyID)
	}

	retry := 0
	maxRetries := 3
	for {
		putResult, err := uploader.Upload(uploadInput)
		if err != nil {
			if _, ok := err.(s3manager.MultiUploadFailure); ok {
				if retry == maxRetries {
					log.Println("Upload retry limit exceeded:", err.Error())
					return fmt.Errorf("upload retry limit exceeded: %s", err.Error())
				}
				retry++
				time.Sleep(time.Second * time.Duration(retry))
				continue
			}
			log.Println("Upload failed:", err.Error())
			return fmt.Errorf("upload failure: %s", err.Error())
		}

		log.Println("Successfully uploaded file to", putResult.Location)
		return nil
	}
}

// Delete removes a blob from an S3 compatible blobstore. If the object does
// not exist, Delete does not return an error.
func (client *S3Blobstore) Delete(dest string) error {
	if client.s3cliConfig.CredentialsSource == config.NoneCredentialsSource {
		return errorInvalidCredentialsSourceValue
	}

	deleteParams := &s3.DeleteObjectInput{
		Bucket: aws.String(client.s3cliConfig.BucketName),
		Key:    client.key(dest),
	}

	_, err := client.s3Client.DeleteObject(deleteParams)

	if err == nil {
		return nil
	}

	if reqErr, ok := err.(awserr.RequestFailure); ok {
		if reqErr.StatusCode() == 404 {
			return nil
		}
	}
	return err
}

// Exists checks if blob exists in an S3 compatible blobstore
func (client *S3Blobstore) Exists(dest string) (bool, error) {

	existsParams := &s3.HeadObjectInput{
		Bucket: aws.String(client.s3cliConfig.BucketName),
		Key:    client.key(dest),
	}

	_, err := client.s3Client.HeadObject(existsParams)

	if err == nil {
		log.Printf("File '%s' exists in bucket '%s'\n", dest, client.s3cliConfig.BucketName)
		return true, nil
	}

	if reqErr, ok := err.(awserr.RequestFailure); ok {
		if reqErr.StatusCode() == 404 {
			log.Printf("File '%s' does not exist in bucket '%s'\n", dest, client.s3cliConfig.BucketName)
			return false, nil
		}
	}
	return false, err
}

func (client *S3Blobstore) key(srcOrDest string) *string {
	formattedKey := aws.String(srcOrDest)
	if len(client.s3cliConfig.FolderName) != 0 {
		formattedKey = aws.String(fmt.Sprintf("%s/%s", client.s3cliConfig.FolderName, srcOrDest))
	}

	return formattedKey
}