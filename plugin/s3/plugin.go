package main

import (
	"crypto/tls"
	"fmt"
	"github.com/rlmcpherson/s3gof3r"
	"github.com/starkandwayne/shield/plugin"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	p := S3Plugin{
		Name:    "S3 Backup + Storage Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "no",
			Store:  "yes",
		},
	}

	plugin.Run(p)
}

type S3Plugin plugin.PluginInfo

type S3ConnectionInfo struct {
	Host              string
	SkipSSLValidation bool
	AccessKey         string
	SecretKey         string
	Bucket            string
	PathPrefix        string
	Name              string
}

func (p S3Plugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p S3Plugin) Backup(endpoint plugin.ShieldEndpoint) error {
	return plugin.UNIMPLEMENTED
}

func (p S3Plugin) Restore(endpoint plugin.ShieldEndpoint) error {
	return plugin.UNIMPLEMENTED
}

func (p S3Plugin) Store(endpoint plugin.ShieldEndpoint) (string, error) {
	s3, err := getS3ConnInfo(endpoint)
	if err != nil {
		return "", err
	}
	bucket := s3.GetBucket()

	path := s3.genBackupPath()
	plugin.DEBUG("Storing data in %s", path)
	writer, err := bucket.PutWriter(path, nil, bucket.Config)
	if err != nil {
		return "", err
	}
	if _, err = io.Copy(writer, os.Stdin); err != nil {
		return "", err
	}

	err = writer.Close()
	if err != nil {
		return "", err
	}

	return path, nil
}

func (p S3Plugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	s3, err := getS3ConnInfo(endpoint)
	if err != nil {
		return err
	}
	bucket := s3.GetBucket()

	reader, _, err := bucket.GetReader(file, bucket.Config)
	if err != nil {
		return err
	}
	if _, err = io.Copy(os.Stdout, reader); err != nil {
		return err
	}

	err = reader.Close()
	if err != nil {
		return err
	}

	return nil
}

func (p S3Plugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	s3, err := getS3ConnInfo(endpoint)
	if err != nil {
		return err
	}
	bucket := s3.GetBucket()

	err = bucket.Delete(file)
	if err != nil {
		return err
	}

	return nil
}

func getS3ConnInfo(e plugin.ShieldEndpoint) (S3ConnectionInfo, error) {
	host, err := e.StringValue("s3_host")
	if err != nil {
		return S3ConnectionInfo{}, err
	}

	insecure_ssl, err := e.BooleanValue("skip_ssl_validation")
	if err != nil {
		return S3ConnectionInfo{}, err
	}

	key, err := e.StringValue("access_key_id")
	if err != nil {
		return S3ConnectionInfo{}, err
	}

	secret, err := e.StringValue("secret_access_key")
	if err != nil {
		return S3ConnectionInfo{}, err
	}

	bucket, err := e.StringValue("bucket")
	if err != nil {
		return S3ConnectionInfo{}, err
	}

	prefix, err := e.StringValue("prefix")
	if err != nil {
		return S3ConnectionInfo{}, err
	}

	name, err := e.StringValue("name")
	if err != nil {
		return S3ConnectionInfo{}, err
	}

	return S3ConnectionInfo{
		Host:              host,
		SkipSSLValidation: insecure_ssl,
		AccessKey:         key,
		SecretKey:         secret,
		Bucket:            bucket,
		PathPrefix:        prefix,
		Name:              name,
	}, nil
}

func (s3 S3ConnectionInfo) genBackupPath() string {
	t := time.Now()
	year, mon, day := t.Date()
	hour, min, sec := t.Clock()
	uuid := plugin.GenUUID()
	return fmt.Sprintf("%s/%04d/%02d/%02d/%04d-%02d-%02d-%02d:%02d:%02d-%s-%s", s3.PathPrefix, year, mon, day, year, mon, day, hour, min, sec, s3.Name, uuid)
}

func (s3 S3ConnectionInfo) GetBucket() *s3gof3r.Bucket {
	keys := s3gof3r.Keys{
		AccessKey:     s3.AccessKey,
		SecretKey:     s3.SecretKey,
		SecurityToken: "",
	}
	conn := s3gof3r.New(s3.Host, keys)

	bucket := conn.Bucket(s3.Bucket)
	bucket.Client.Transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: s3.SkipSSLValidation}
	//	bucket.Config.Md5Check = false
	return bucket
}
