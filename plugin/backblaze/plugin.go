package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	gofmt "github.com/jhunt/go-ansi"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/shieldproject/shield/plugin"
	"github.com/shieldproject/shield/plugin/s3util"
)

const DefaultPrefix = ""

func validBucketName(v string) bool {
	ok, err := regexp.MatchString(`^[a-z0-9][a-z0-9\.\-]{1,61}[a-z0-9]$`, v)
	return ok && err == nil
}

func main() {
	p := BackblazePlugin{
		Name:    "Backblaze Storage Plugin",
		Author:  "SHIELD Core Team",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "no",
			Store:  "yes",
		},
		Example: `
{
  "access_key_id"       : "your-access-key-id",       # REQUIRED
  "secret_access_key"   : "your-secret-access-key",   # REQUIRED
  "bucket"              : "name-of-your-bucket",      # REQUIRED

  "prefix"              : "/path/in/bucket",     # where to store archives, inside the bucket
}
`,
		Defaults: `
{
}
`,
		Fields: []plugin.Field{
			plugin.Field{
				Mode:  "store",
				Name:  "access_key_id",
				Type:  "string",
				Title: "Access Key ID",
				Help:  "The Access Key ID to use when authenticating against B2.",
			},
			plugin.Field{
				Mode:  "store",
				Name:  "secret_access_key",
				Type:  "password",
				Title: "Secret Access Key",
				Help:  "The Secret Access Key to use when authenticating against B2.",
			},
			plugin.Field{
				Mode:     "store",
				Name:     "bucket",
				Type:     "string",
				Title:    "Bucket Name",
				Help:     "Name of the bucket to store backup archives in.",
				Example:  "my-aws-backups",
				Required: true,
			},
			plugin.Field{
				Mode:  "store",
				Name:  "prefix",
				Type:  "string",
				Title: "Bucket Path Prefix",
				Help:  "An optional sub-path of the bucket to use for storing archives.  By default, archives are stored in the root of the bucket.",
			},
		},
	}

	plugin.Run(p)
}

type BackblazePlugin plugin.PluginInfo

type backblazeEndpoint struct {
	AccessKey  string
	SecretKey  string
	Prefix     string
	Bucket     string
}

func (p BackblazePlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p BackblazePlugin) Validate(endpoint plugin.ShieldEndpoint) error {
	var (
		s    string
		err  error
		fail bool
	)

	//BEGIN AUTH VALIDATION
	s, err = endpoint.StringValue("access_key_id")
	if err != nil {
		gofmt.Printf("@R{\u2717 access_key_id        %s}\n", err)
		fail = true
	} else {
		gofmt.Printf("@G{\u2713 access_key_id}        @C{%s}\n", plugin.Redact(s))
	}

	s, err = endpoint.StringValue("secret_access_key")
	if err != nil {
		gofmt.Printf("@R{\u2717 secret_access_key    %s}\n", err)
		fail = true
	} else {
		gofmt.Printf("@G{\u2713 secret_access_key}    @C{%s}\n", plugin.Redact(s))
	}
	//END AUTH VALIDATION

	s, err = endpoint.StringValue("bucket")
	if err != nil {
		gofmt.Printf("@R{\u2717 bucket               %s}\n", err)
		fail = true
	} else if !validBucketName(s) {
		gofmt.Printf("@R{\u2717 bucket               '%s' is an invalid bucket name (must be all lowercase)}\n", s)
		fail = true
	} else {
		gofmt.Printf("@G{\u2713 bucket}               @C{%s}\n", plugin.Redact(s))
	}

	s, err = endpoint.StringValueDefault("prefix", DefaultPrefix)
	if err != nil {
		gofmt.Printf("@R{\u2717 prefix               %s}\n", err)
		fail = true
	} else if s == "" {
		gofmt.Printf("@G{\u2713 prefix}               (none)\n")
	} else {
		s = strings.TrimLeft(s, "/")
		gofmt.Printf("@G{\u2713 prefix}               @C{%s}\n", s)
	}

	if fail {
		return fmt.Errorf("b2: invalid configuration")
	}
	return nil
}

func (p BackblazePlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	return plugin.UNIMPLEMENTED
}

func (p BackblazePlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	return plugin.UNIMPLEMENTED
}

func (p BackblazePlugin) Store(endpoint plugin.ShieldEndpoint) (string, int64, error) {
	e, err := getB2ConnInfo(endpoint)
	if err != nil {
		return "", 0, err
	}

	client, err := e.Connect()
	if err != nil {
		return "", 0, err
	}

	path := s3util.GenBackupPath(e.Prefix)
	plugin.DEBUG("Storing data in %s", path)

	ctx := context.Background()
	cr := s3util.NewCountingReader(os.Stdin)
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(e.Bucket),
		Key:    aws.String(path),
		Body:   cr,
	})
	if err != nil {
		return "", 0, err
	}

	return path, cr.N, nil
}

func (p BackblazePlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	e, err := getB2ConnInfo(endpoint)
	if err != nil {
		return err
	}

	client, err := e.Connect()
	if err != nil {
		return err
	}

	ctx := context.Background()
	result, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(e.Bucket),
		Key:    aws.String(file),
	})
	if err != nil {
		return err
	}
	defer result.Body.Close()

	_, err = io.Copy(os.Stdout, result.Body)
	return err
}

func (p BackblazePlugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	e, err := getB2ConnInfo(endpoint)
	if err != nil {
		return err
	}

	client, err := e.Connect()
	if err != nil {
		return err
	}

	ctx := context.Background()
	_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(e.Bucket),
		Key:    aws.String(file),
	})
	return err
}

func getB2ConnInfo(e plugin.ShieldEndpoint) (backblazeEndpoint, error) {
	key, err := e.StringValue("access_key_id")
	if err != nil {
		return backblazeEndpoint{}, err
	}

	secret, err := e.StringValue("secret_access_key")
	if err != nil {
		return backblazeEndpoint{}, err
	}

	bucket, err := e.StringValue("bucket")
	if err != nil {
		return backblazeEndpoint{}, err
	}

	prefix, err := e.StringValueDefault("prefix", DefaultPrefix)
	if err != nil {
		return backblazeEndpoint{}, err
	}
	prefix = strings.TrimLeft(prefix, "/")

	return backblazeEndpoint{
		AccessKey: key,
		SecretKey: secret,
		Prefix:    prefix,
		Bucket:    bucket,
	}, nil
}

// detectB2Region calls the B2 native API to determine the S3-compatible
// region and endpoint URL for the given bucket.  The b2_authorize_account
// response contains an s3ApiUrl like https://s3.us-west-004.backblazeb2.com;
// the region is extracted from that hostname.
func detectB2Region(keyID, appKey, bucketName string) (region, s3Endpoint string, err error) {
	// Step 1: authorize
	req, err := http.NewRequest(http.MethodPost,
		"https://api.backblazeb2.com/b2api/v2/b2_authorize_account", nil)
	if err != nil {
		return "", "", fmt.Errorf("b2: authorize request: %w", err)
	}
	req.SetBasicAuth(keyID, appKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("b2: authorize: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("b2: authorize HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var authResp struct {
		AccountID string `json:"accountId"`
		APIURL    string `json:"apiUrl"`
		S3APIURL  string `json:"s3ApiUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", "", fmt.Errorf("b2: authorize decode: %w", err)
	}

	// Extract region from s3ApiUrl hostname, e.g. s3.us-west-004.backblazeb2.com
	s3Endpoint = authResp.S3APIURL
	host := strings.TrimPrefix(s3Endpoint, "https://")
	host = strings.TrimPrefix(host, "http://")
	// remove any trailing path
	if idx := strings.Index(host, "/"); idx >= 0 {
		host = host[:idx]
	}
	parts := strings.SplitN(host, ".", 3)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("b2: unexpected s3ApiUrl format: %s", authResp.S3APIURL)
	}
	region = parts[1] // e.g. "us-west-004"

	return region, s3Endpoint, nil
}

func (e backblazeEndpoint) Connect() (*s3.Client, error) {
	region, s3Endpoint, err := detectB2Region(e.AccessKey, e.SecretKey, e.Bucket)
	if err != nil {
		return nil, err
	}
	return s3util.NewClient(s3util.S3Config{
		Region:          region,
		AccessKeyID:     e.AccessKey,
		SecretAccessKey: e.SecretKey,
		Endpoint:        s3Endpoint,
		UsePathStyle:    true,
	})
}
